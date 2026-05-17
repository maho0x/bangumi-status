package main

import (
	"bangumi-status/internal/notifier"
	"bangumi-status/internal/region"
	"bangumi-status/internal/store"
	"bangumi-status/internal/types"
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

)

//go:embed static
var staticFS embed.FS

type server struct {
	store *store.Store
	// secret is the legacy single shared token (admin: accepts any probe-id).
	secret string
	// tokenPrefixes maps a third-party token → required probe-id prefix.
	// Anything reported under this token must have a probe-id that
	// HasPrefix(prefix). The operator can self-pick probe-ids inside their
	// namespace without contacting the maintainer.
	tokenPrefixes map[string]string
	// onlineSourceProbe is the canonical probe whose OnlineCount samples are
	// recorded. Multiple probes scrape bangumi.tv at different offsets and see
	// slightly different values; trusting one source avoids systematic bias.
	onlineSourceProbe string
	notifier          *notifier.Telegram

	mu       sync.RWMutex
	cached   *types.Overall
	cachedAt time.Time
	sfGroup  singleflight.Group

	rxHub *reactionHub
}

// reactionHub fans out "something changed" pings to active SSE subscribers.
// Subscribers re-query the DB on their own so each client receives counts
// already filtered by its own user-id.
type reactionHub struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func newReactionHub() *reactionHub {
	return &reactionHub{subs: map[chan struct{}]struct{}{}}
}

func (h *reactionHub) subscribe() (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

func (h *reactionHub) notify() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default:
			// Subscriber already has a pending tick — coalesce.
		}
	}
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbDSN := flag.String("db-dsn", os.Getenv("DB_DSN"), "database DSN (postgres://...)")
	flag.Parse()

	secret := os.Getenv("INGEST_SECRET")
	tokenPrefixes, err := parseTokenPrefixes(os.Getenv("INGEST_SECRETS"))
	if err != nil {
		log.Fatalf("parse INGEST_SECRETS: %v", err)
	}
	if secret == "" && len(tokenPrefixes) == 0 {
		log.Fatal("INGEST_SECRET or INGEST_SECRETS env var is required")
	}
	if len(tokenPrefixes) > 0 {
		log.Printf("loaded %d third-party token(s) (prefix-namespaced)", len(tokenPrefixes))
	}

	if *dbDSN == "" {
		log.Fatal("DB_DSN env var or -db-dsn flag is required")
	}

	st, err := store.Open(*dbDSN)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	pageURL := os.Getenv("STATUS_PAGE_URL")
	if pageURL == "" {
		pageURL = "https://bgm-status.ry.mk"
	}
	tg := notifier.NewTelegram(
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		os.Getenv("TELEGRAM_CHAT_ID"),
		pageURL,
		st,
	)
	if tg.Enabled() {
		log.Printf("telegram notifications enabled")
	}

	onlineSourceProbe := os.Getenv("ONLINE_SOURCE_PROBE")
	if onlineSourceProbe == "" {
		onlineSourceProbe = "wataame-tokyo-1"
	}
	log.Printf("online counter source: %s", onlineSourceProbe)

	s := &server{store: st, secret: secret, tokenPrefixes: tokenPrefixes, onlineSourceProbe: onlineSourceProbe, notifier: tg, rxHub: newReactionHub()}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/ingest", s.handleIngest)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/mini", s.handleMini)
	mux.HandleFunc("GET /api/online", s.handleOnline)
	mux.HandleFunc("GET /api/probes", s.handleProbes)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/feed.atom", s.handleFeed)
	mux.HandleFunc("GET /api/reactions", s.handleReactionsList)
	mux.HandleFunc("POST /api/reactions", s.handleReactionsToggle)
	mux.HandleFunc("GET /api/reactions/stream", s.handleReactionsStream)

	// Serve embedded static site at /.
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("embed sub: %v", err)
	}
	mux.Handle("/", spaHandler{fs: sub})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withLogging(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go s.retentionLoop(ctx)
	go s.cacheRefreshLoop(ctx)
	go s.dailyReportLoop(ctx)
	go s.reactionPurgeLoop(ctx)

	go func() {
		log.Printf("listening on %s db=%s", *addr, *dbDSN)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}

func (s *server) handleIngest(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	token := auth[len("Bearer "):]
	var p types.IngestPayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&p); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if p.Probe == "" {
		http.Error(w, "missing probe", http.StatusBadRequest)
		return
	}
	// Normalize + validate region as ISO 3166-1 alpha-2 (e.g. "jp", "us").
	// Reject unknown codes so a misconfigured probe surfaces immediately
	// instead of polluting the data set with garbage region labels.
	if normalized, ok := region.Normalize(p.Region); ok {
		p.Region = normalized
	} else {
		http.Error(w, "invalid region (must be ISO 3166-1 alpha-2 country code)", http.StatusBadRequest)
		return
	}
	// Auth: either the legacy admin token (any probe-id), or a third-party
	// token whose required prefix matches the payload's probe-id. Constant-
	// time compare on every token to avoid timing leaks.
	authorized := false
	if s.secret != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.secret)) == 1 {
		authorized = true
	} else {
		matchedPrefix := ""
		matched := false
		for tok, prefix := range s.tokenPrefixes {
			if subtle.ConstantTimeCompare([]byte(token), []byte(tok)) == 1 {
				matchedPrefix = prefix
				matched = true
			}
		}
		if matched && strings.HasPrefix(p.Probe, matchedPrefix) && len(p.Probe) > len(matchedPrefix) {
			authorized = true
		}
	}
	if !authorized {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	kept := make([]types.Result, 0, len(p.Results))
	for _, res := range p.Results {
		if res.Probe == "" {
			res.Probe = p.Probe
		}
		if res.Region == "" {
			res.Region = p.Region
		}
		if !res.Valid() {
			continue
		}
		// marker missing means the probe's cookie is invalid, not a service fault.
		if res.Kind == types.KindAuth && res.Err == "marker missing" {
			continue
		}
		kept = append(kept, res)
	}
	if err := s.store.Insert(r.Context(), kept); err != nil {
		http.Error(w, "insert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.store.UpsertProbe(r.Context(), p.Probe, p.Region, time.Now().Unix())
	if p.OnlineCount > 0 && p.Probe == s.onlineSourceProbe {
		ts := p.OnlineTS
		if ts == 0 {
			ts = time.Now().Unix()
		}
		_ = s.store.InsertOnline(r.Context(), ts, p.OnlineCount)
	}

	// Note: we do NOT invalidate the cache here. The 14 probes ingest every
	// ~4 seconds combined; nil-ing the cache forces the next /api/status
	// caller to block on a 10+ second recomputation. cacheRefreshLoop refreshes
	// in the background every 20s, which is fresh enough for a status page.

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "accepted": len(kept)})
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	overall, err := s.getOverall(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=20")
	_ = json.NewEncoder(w).Encode(overall)
}

func (s *server) handleMini(w http.ResponseWriter, r *http.Request) {
	overall, err := s.getOverall(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Core sites: include both guest and auth endpoints.
	var worstStatus = types.StatusOK
	for _, c := range overall.Components {
		switch c.Domain {
		case "bgm.tv", "bangumi.tv", "chii.in":
			if worseThan(c.Status, worstStatus) {
				worstStatus = c.Status
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=20")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     worstStatus,
		"message":    messageFor(worstStatus),
		"updated_at": overall.UpdatedAt,
	})
}

func (s *server) handleOnline(w http.ResponseWriter, r *http.Request) {
	var since time.Time
	var bucket int64
	switch r.URL.Query().Get("range") {
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
		bucket = 600
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
		bucket = 3600
	case "all":
		since = time.Unix(0, 0)
		bucket = 21600
	default:
		since = time.Now().Add(-24 * time.Hour)
	}
	pts, err := s.store.OnlineSeriesBucketed(r.Context(), since, bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pts == nil {
		pts = []types.OnlinePoint{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	json.NewEncoder(w).Encode(pts)
}

func (s *server) handleProbes(w http.ResponseWriter, r *http.Request) {
	ps, err := s.store.Probes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ps)
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	Link    []atomLink  `xml:"link"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Author  atomAuthor  `xml:"author"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	ID      string    `xml:"id"`
	Title   string    `xml:"title"`
	Link    atomLink  `xml:"link"`
	Updated string    `xml:"updated"`
	Summary atomText  `xml:"summary"`
	Content atomText  `xml:"content"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

func (s *server) handleFeed(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	base := scheme + "://" + host

	entries := make([]atomEntry, 0)
	for _, c := range types.AllComponents() {
		buckets, err := s.store.DailyBuckets(r.Context(), c.Domain, c.Kind, 30)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, b := range buckets {
			if b.Total == 0 || b.Status == types.StatusOK {
				continue
			}
			day, err := time.Parse("2006-01-02", b.Day)
			if err != nil {
				continue
			}
			sev := "Degraded performance"
			if b.Status == types.StatusDown {
				sev = "Service disruption"
			}
			title := fmt.Sprintf("%s — %s", sev, c.Label)
			id := fmt.Sprintf("tag:%s,%s:%s/%s", host, b.Day, c.Domain, c.Kind)
			okCount := b.Total - b.Down - b.Degrade
			summary := fmt.Sprintf("%s on %s. %d down, %d degraded, %d ok of %d checks (%.1f%% uptime).",
				sev, b.Day, b.Down, b.Degrade, okCount, b.Total, b.Uptime)
			// Entry "updated" is end-of-day UTC so latest incident day sorts first.
			updated := day.Add(24 * time.Hour).Add(-1 * time.Second).UTC()
			entries = append(entries, atomEntry{
				ID:      id,
				Title:   title,
				Link:    atomLink{Href: base + "/", Rel: "alternate", Type: "text/html"},
				Updated: updated.Format(time.RFC3339),
				Summary: atomText{Type: "text", Body: summary},
				Content: atomText{Type: "text", Body: summary},
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Updated > entries[j].Updated })
	if len(entries) > 50 {
		entries = entries[:50]
	}

	feed := atomFeed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Title:   "Bangumi Status · Incidents",
		Link:    []atomLink{
			{Href: base + "/api/feed.atom", Rel: "self", Type: "application/atom+xml"},
			{Href: base + "/", Rel: "alternate", Type: "text/html"},
		},
		ID:      "tag:" + host + ":feed",
		Updated: time.Now().UTC().Format(time.RFC3339),
		Author:  atomAuthor{Name: "Bangumi Status"},
		Entries: entries,
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(feed)
	_ = enc.Flush()
	_, _ = w.Write([]byte("\n"))
}

// allowedReactions is the canonical set of reaction emoji IDs (chii.in TV smiles).
var allowedReactions = map[int]bool{
	44: true, 40: true, 15: true, 23: true, 83: true, 65: true,
	41: true, 102: true, 49: true, 46: true, 51: true, 101: true,
}

// reactionRate throttles reaction calls per source IP (token bucket: 300/min).
var reactionRate = newRateLimiter(300, time.Minute)

func (s *server) handleReactionsList(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if len(userID) > 128 {
		userID = ""
	}
	counts, err := s.store.ListReactions(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Ensure every allowed emoji appears even when count is 0.
	byID := map[int]store.ReactionCount{}
	for _, c := range counts {
		byID[c.EmojiID] = c
	}
	out := make([]store.ReactionCount, 0, len(allowedReactions))
	for id := range allowedReactions {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		} else {
			out = append(out, store.ReactionCount{EmojiID: id})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EmojiID < out[j].EmojiID })
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *server) handleReactionsToggle(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" || len(userID) < 8 || len(userID) > 128 {
		http.Error(w, "missing or invalid X-User-ID", http.StatusBadRequest)
		return
	}
	var body struct {
		EmojiID int `json:"emoji_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !allowedReactions[body.EmojiID] {
		http.Error(w, "unknown emoji_id", http.StatusBadRequest)
		return
	}
	ip := clientIP(r)
	if !reactionRate.allow(ip) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	if err := s.store.AddReaction(r.Context(), body.EmojiID, userID, ip); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.rxHub.notify()
	w.WriteHeader(http.StatusNoContent)
}

// handleReactionsStream pushes the full reaction count list whenever the
// global state changes (and as a periodic heartbeat). The user id is taken
// from the `uid` query param because EventSource cannot set custom headers.
func (s *server) handleReactionsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("uid"))
	if len(userID) > 128 {
		userID = ""
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx/caddy)
	flusher.Flush()

	ch, cleanup := s.rxHub.subscribe()
	defer cleanup()

	send := func() bool {
		counts, err := s.store.ListReactions(r.Context(), userID)
		if err != nil {
			return false
		}
		byID := map[int]store.ReactionCount{}
		for _, c := range counts {
			byID[c.EmojiID] = c
		}
		out := make([]store.ReactionCount, 0, len(allowedReactions))
		for id := range allowedReactions {
			if c, ok := byID[id]; ok {
				out = append(out, c)
			} else {
				out = append(out, store.ReactionCount{EmojiID: id})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].EmojiID < out[j].EmojiID })
		buf, err := json.Marshal(out)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", buf); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !send() {
		return
	}

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			if !send() {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i > 0 {
		host = host[:i]
	}
	return host
}

// rateLimiter is a tiny per-key token bucket. Not designed for high concurrency
// but adequate for low-rate UI actions like reaction toggles.
type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	xs := l.hits[key]
	out := xs[:0]
	for _, t := range xs {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	if len(out) >= l.limit {
		l.hits[key] = out
		return false
	}
	l.hits[key] = append(out, now)
	// Opportunistic GC: drop empty entries to keep the map bounded.
	if len(l.hits) > 4096 {
		for k, v := range l.hits {
			if len(v) == 0 || v[len(v)-1].Before(cutoff) {
				delete(l.hits, k)
			}
		}
	}
	return true
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.Stats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stats["now"] = time.Now().Unix()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *server) getOverall(ctx context.Context) (*types.Overall, error) {
	s.mu.RLock()
	cached := s.cached
	stale := cached != nil && time.Since(s.cachedAt) >= 30*time.Second
	s.mu.RUnlock()

	if cached != nil {
		if stale {
			// Return stale data immediately; refresh in background.
			go s.sfGroup.Do("refresh", func() (any, error) {
				out, err := s.computeOverall(context.Background())
				if err == nil {
					s.mu.Lock()
					s.cached = out
					s.cachedAt = time.Now()
					s.mu.Unlock()
					s.notifier.Process(out.Components)
					s.notifier.UpdateSummary(out)
				}
				return nil, err
			})
		}
		return cached, nil
	}

	// No cache yet (first run) — sfGroup.Do blocks all callers until one computation finishes.
	v, err, _ := s.sfGroup.Do("refresh", func() (any, error) {
		out, err := s.computeOverall(context.Background())
		if err == nil {
			s.mu.Lock()
			s.cached = out
			s.cachedAt = time.Now()
			s.mu.Unlock()
			s.notifier.Process(out.Components)
			s.notifier.UpdateSummary(out)
		}
		return out, err
	})
	if err != nil {
		return nil, err
	}
	return v.(*types.Overall), nil
}

func (s *server) computeOverall(ctx context.Context) (*types.Overall, error) {
	allComponents := types.AllComponents()
	results := make([]types.ComponentStatus, len(allComponents))

	g, gctx := errgroup.WithContext(ctx)
	incidentSince := time.Now().AddDate(0, 0, -14)

	for i, c := range allComponents {
		i, c := i, c
		g.Go(func() error {
			buckets, err := s.store.DailyBuckets(gctx, c.Domain, c.Kind, 30)
			if err != nil {
				return err
			}
			views, err := s.store.LatestPerProbe(gctx, c.Domain, c.Kind)
			if err != nil {
				return err
			}
			currentStatus, last := store.RollupStatus(views)
			incidents, err := s.store.Incidents(gctx, c.Domain, c.Kind, incidentSince)
			if err != nil {
				return err
			}
			store.OverlayIncidentsOnBuckets(buckets, incidents)
			var totalOK, totalAll int
			for _, b := range buckets {
				totalAll += b.Total
				totalOK += b.Total - b.Down - b.Degrade
			}
			var up float64
			if totalAll > 0 {
				up = float64(totalOK) / float64(totalAll) * 100
			}
			results[i] = types.ComponentStatus{
				Domain:     c.Domain,
				Kind:       c.Kind,
				Label:      c.Label,
				Status:     currentStatus,
				Uptime:     up,
				Days:       buckets,
				LastCheck:  last,
				ProbeViews: views,
				Incidents:  incidents,
			}
			return nil
		})
	}

	// Fetch probes and online series concurrently with component queries.
	var probes []types.Probe
	var online []types.OnlinePoint
	g.Go(func() error {
		var err error
		probes, err = s.store.Probes(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		online, err = s.store.OnlineSeries(gctx, time.Now().Add(-24*time.Hour))
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	out := &types.Overall{UpdatedAt: time.Now().Unix()}
	var worstStatus = types.StatusOK
	for _, cs := range results {
		out.Components = append(out.Components, cs)
		if worseThan(cs.Status, worstStatus) {
			worstStatus = cs.Status
		}
	}
	out.Probes = probes
	out.Online = online
	out.Status = worstStatus
	out.Message = messageFor(worstStatus)

	return out, nil
}

func worseThan(a, b types.Status) bool {
	rank := map[types.Status]int{types.StatusOK: 0, types.StatusDegraded: 1, types.StatusDown: 2}
	return rank[a] > rank[b]
}

func messageFor(s types.Status) string {
	switch s {
	case types.StatusOK:
		return "All systems operational"
	case types.StatusDegraded:
		return "Some systems experiencing degraded performance"
	case types.StatusDown:
		return "Major outage detected"
	}
	return ""
}

func (s *server) retentionLoop(ctx context.Context) {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		cutoff := time.Now().AddDate(0, 0, -35) // keep a little extra
		n, err := s.store.PurgeOlderThan(ctx, cutoff)
		if err != nil {
			log.Printf("retention: %v", err)
		} else if n > 0 {
			log.Printf("retention: purged %d rows", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *server) reactionPurgeLoop(ctx context.Context) {
	t := time.NewTicker(1 * time.Hour)
	defer t.Stop()
	for {
		if err := s.store.PurgeExpiredReactions(ctx); err != nil {
			log.Printf("reactions purge: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *server) cacheRefreshLoop(ctx context.Context) {
	// Warm the cache immediately at startup so the first /api/status request
	// after boot doesn't block on a cold computation.
	s.getOverall(ctx) //nolint:errcheck

	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		// Force a background refresh by expiring the cache.
		s.mu.Lock()
		s.cachedAt = time.Time{}
		s.mu.Unlock()
		s.getOverall(ctx) //nolint:errcheck
	}
}

func (s *server) dailyReportLoop(ctx context.Context) {
	cst := time.FixedZone("CST", 8*60*60)

	// Wait until next CST 00:00.
	now := time.Now().In(cst)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, cst)
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(next)):
	}

	for {
		now = time.Now().In(cst)
		yesterday := now.AddDate(0, 0, -1)
		dateStr := yesterday.Format("2006-01-02")
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, cst)
		end := start.Add(24 * time.Hour)

		var items []notifier.DailyReportItem
		for _, domain := range []string{"bgm.tv", "bangumi.tv", "chii.in"} {
			bucket, err := s.store.DaySummary(ctx, domain, types.KindAuth, start, end)
			if err != nil {
				log.Printf("daily report: summary error for %s: %v", domain, err)
				continue
			}
			if bucket.Total == 0 {
				continue
			}
			items = append(items, notifier.DailyReportItem{
				Domain: domain,
				Status: bucket.Status,
				Uptime: bucket.Uptime,
			})
		}

		if len(items) > 0 {
			s.notifier.SendDailyReport(dateStr, items)
		}

		now = time.Now().In(cst)
		next = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, cst)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
		}
	}
}

func withLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &statusRecorder{ResponseWriter: w, code: 200}
		h.ServeHTTP(lw, r)
		if r.URL.Path != "/api/health" {
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, lw.code, time.Since(start))
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type spaHandler struct{ fs fs.FS }

func (s spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	data, err := fs.ReadFile(s.fs, p)
	if err != nil {
		data, err = fs.ReadFile(s.fs, "index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		p = "index.html"
	}
	w.Header().Set("Content-Type", contentTypeFor(p))
	if strings.HasPrefix(p, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	_, _ = w.Write(data)
}

func contentTypeFor(p string) string {
	switch {
	case strings.HasSuffix(p, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(p, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(p, ".js"):
		return "application/javascript"
	case strings.HasSuffix(p, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(p, ".png"):
		return "image/png"
	case strings.HasSuffix(p, ".json"):
		return "application/json"
	case strings.HasSuffix(p, ".ico"):
		return "image/x-icon"
	}
	return "application/octet-stream"
}

// parseTokenPrefixes parses a third-party token allowlist of the form
//   token1:prefix1;token2:prefix2
// Each token is bound to a required probe-id prefix. The operator can
// self-pick any probe-id that starts with their prefix. Semicolons separate
// operators. Empty input yields a nil map.
//
// Prefixes must be non-empty and not prefix each other (so one operator
// can't impersonate another by reporting a probe-id whose prefix matches
// both). The empty prefix is rejected — that would let a token report any
// probe-id and impersonate the maintainer.
func parseTokenPrefixes(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	out := map[string]string{}
	for _, group := range strings.Split(raw, ";") {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		i := strings.IndexByte(group, ':')
		if i <= 0 || i == len(group)-1 {
			return nil, fmt.Errorf("invalid group %q (want token:prefix)", group)
		}
		tok := strings.TrimSpace(group[:i])
		prefix := strings.TrimSpace(group[i+1:])
		if tok == "" || prefix == "" {
			return nil, fmt.Errorf("empty token or prefix in group %q", group)
		}
		if _, dup := out[tok]; dup {
			return nil, fmt.Errorf("duplicate token entry")
		}
		out[tok] = prefix
	}
	// Reject overlapping prefixes (one is a prefix of another) — that would
	// let one operator's namespace leak into another's.
	for tokA, prefA := range out {
		for tokB, prefB := range out {
			if tokA == tokB {
				continue
			}
			if strings.HasPrefix(prefB, prefA) {
				return nil, fmt.Errorf("prefix %q overlaps %q (one operator's namespace contains another)", prefA, prefB)
			}
		}
	}
	return out, nil
}

func dirname(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return "."
}
