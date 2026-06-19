package main

import (
	"bangumi-status/internal/notifier"
	"bangumi-status/internal/region"
	"bangumi-status/internal/store"
	"bangumi-status/internal/types"
	"bytes"
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
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
	wikiStatsURL      string
	statusCachePath   string
	notifier          *notifier.Telegram

	mu       sync.RWMutex
	cached   *types.Overall
	cachedAt time.Time
	sfGroup  singleflight.Group

	// refreshCh coalesces "data changed, refresh soon" pings from ingest into
	// the cacheRefreshLoop. Buffered size 1 so a burst of ingests collapses to
	// at most one pending refresh; the loop applies a short debounce on top.
	refreshCh chan struct{}

	// history caches the expensive, slowly-changing parts of the status payload
	// (30-day daily buckets + 14-day incident walk). It is refreshed at most
	// once per historyRefreshInterval so the 20s status-refresh loop only pays
	// for the cheap live-status queries. See componentHistories.
	histMu    sync.RWMutex
	history   map[store.ComponentKey]componentHistory
	historyAt time.Time
	histSF    singleflight.Group

	feedMu    sync.Mutex
	feedCache cachedAtomFeed

	reactionMu       sync.RWMutex
	reactionCounts   []store.ReactionCount
	reactionCountsAt time.Time
	reactionSF       singleflight.Group

	rxHub     *reactionHub
	statusHub *reactionHub
}

type cachedAtomFeed struct {
	base     string
	body     []byte
	cachedAt time.Time
}

var errStatusCacheWarming = errors.New("status cache is warming up")

const statusRefreshTimeout = 3 * time.Minute

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

	wikiStatsURL := os.Getenv("WIKI_STATS_URL")
	if wikiStatsURL == "" {
		wikiStatsURL = "https://chii.in/wiki/stats"
	}
	wikiStatsCfg := wikiStatsConfig{
		url:       wikiStatsURL,
		userAgent: bangumiUserAgent(),
		cookie:    bangumiCookieFromEnv(),
		interval:  durationFromEnv("WIKI_STATS_INTERVAL", time.Hour),
	}

	statusCachePath := stringFromEnv("STATUS_CACHE_PATH", "/var/lib/bangumi-status/status-cache.json")
	s := &server{store: st, secret: secret, tokenPrefixes: tokenPrefixes, onlineSourceProbe: onlineSourceProbe, wikiStatsURL: wikiStatsURL, statusCachePath: statusCachePath, notifier: tg, rxHub: newReactionHub(), statusHub: newReactionHub(), refreshCh: make(chan struct{}, 1)}
	s.loadStatusCache()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/ingest", s.handleIngest)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/mini", s.handleMini)
	mux.HandleFunc("GET /api/online", s.handleOnline)
	mux.HandleFunc("GET /api/wiki-stats", s.handleWikiStats)
	mux.HandleFunc("GET /api/probes", s.handleProbes)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/feed.atom", s.handleFeed)
	mux.HandleFunc("GET /api/reactions", s.handleReactionsList)
	mux.HandleFunc("POST /api/reactions", s.handleReactionsToggle)
	mux.HandleFunc("GET /api/reactions/stream", s.handleReactionsStream)
	mux.HandleFunc("GET /api/status/stream", s.handleStatusStream)

	// Serve embedded static site at /.
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("embed sub: %v", err)
	}
	mux.Handle("/", spaHandler{fs: sub})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withSecurityHeaders(withLogging(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create today's + nearby partitions before any ingest hits — inserts into
	// the partitioned `checks` will fail if no matching range partition exists.
	if err := s.store.EnsureChecksPartitions(ctx, 2); err != nil {
		log.Fatalf("ensure checks partitions: %v", err)
	}

	go s.retentionLoop(ctx)
	go s.cacheRefreshLoop(ctx)
	go s.notifierLoop(ctx)
	go s.dailyReportLoop(ctx)
	go s.reactionPurgeLoop(ctx)
	go s.wikiStatsLoop(ctx, wikiStatsCfg)

	go func() {
		log.Printf("listening on %s db=%s", *addr, redactDSN(*dbDSN))
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
	legacyAuthorized := false
	matchedPrefix := ""
	matchedThirdParty := false
	if s.secret != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.secret)) == 1 {
		legacyAuthorized = true
	}
	for tok, prefix := range s.tokenPrefixes {
		if subtle.ConstantTimeCompare([]byte(token), []byte(tok)) == 1 {
			matchedPrefix = prefix
			matchedThirdParty = true
		}
	}
	if !legacyAuthorized && !matchedThirdParty {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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
	// token whose required prefix matches the payload's probe-id.
	authorized := legacyAuthorized ||
		(matchedThirdParty && strings.HasPrefix(p.Probe, matchedPrefix) && len(p.Probe) > len(matchedPrefix))
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
	if p.OnlineCount > 0 {
		ts := p.OnlineTS
		if ts == 0 {
			ts = time.Now().Unix()
		}
		// Accept any probe's reading. The store keeps canonical samples when
		// available and uses non-canonical ones as outage-time fallback.
		_ = s.store.InsertOnline(r.Context(), ts, p.OnlineCount, p.Probe == s.onlineSourceProbe)
	}

	// Nudge the refresh loop so a status change reaches connected clients (via
	// the SSE stream) within a couple seconds instead of waiting out the 20s
	// background tick. This is cheap now that the expensive 30-day/incident
	// aggregation lives behind its own coarse cache (componentHistories): a
	// refresh only reruns the narrow live-status queries. The loop debounces, so
	// the ~4s combined ingest rate can never trigger a refresh storm. We never
	// invalidate the cache inline — callers always read the last good snapshot.
	if len(kept) > 0 {
		s.requestRefresh()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "accepted": len(kept)})
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	overall, err := s.getOverall(r.Context())
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, errStatusCacheWarming) {
			code = http.StatusServiceUnavailable
		}
		http.Error(w, err.Error(), code)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=20")
	_ = json.NewEncoder(w).Encode(overall)
}

func (s *server) handleMini(w http.ResponseWriter, r *http.Request) {
	// Core sites: include both guest and auth endpoints.
	var worstStatus = types.StatusOK
	updatedAt := int64(0)
	overall, err := s.getCachedOverall()
	var components []types.ComponentStatus
	if err == nil {
		components = overall.Components
		updatedAt = overall.UpdatedAt
	} else {
		components, err = s.computeRollups(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, c := range components {
			if c.LastCheck > updatedAt {
				updatedAt = c.LastCheck
			}
		}
	}
	for _, c := range components {
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
		"updated_at": updatedAt,
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

func (s *server) handleWikiStats(w http.ResponseWriter, r *http.Request) {
	points, scrapedAt, err := s.store.WikiStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if points == nil {
		points = []types.WikiStatsPoint{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"updated_at": time.Now().Unix(),
		"scraped_at": scrapedAt,
		"source_url": s.wikiStatsURL,
		"data":       points,
	})
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
	ID      string   `xml:"id"`
	Title   string   `xml:"title"`
	Link    atomLink `xml:"link"`
	Updated string   `xml:"updated"`
	Summary atomText `xml:"summary"`
	Content atomText `xml:"content"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

func (s *server) handleFeed(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "http" || proto == "https" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" && isValidHost(fh) {
		host = fh
	}
	base := scheme + "://" + host

	body, ok := s.cachedFeed(base)
	if !ok {
		overall, err := s.getCachedOverall()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		body, err = buildAtomFeed(base, host, overall)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.setCachedFeed(base, body)
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(body)
}

func (s *server) getCachedOverall() (*types.Overall, error) {
	s.mu.RLock()
	cached := s.cached
	s.mu.RUnlock()
	if cached == nil {
		return nil, errors.New("status cache is warming up")
	}
	return cached, nil
}

func (s *server) loadStatusCache() {
	if s.statusCachePath == "" {
		return
	}
	body, err := os.ReadFile(s.statusCachePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("status cache load: %v", err)
		}
		return
	}
	var cached types.Overall
	if err := json.Unmarshal(body, &cached); err != nil {
		log.Printf("status cache load: %v", err)
		return
	}
	s.mu.Lock()
	s.cached = &cached
	s.cachedAt = time.Now().Add(-31 * time.Second)
	s.mu.Unlock()
	s.notifier.UpdateSummary(&cached)
	log.Printf("loaded status cache from %s", s.statusCachePath)
}

func (s *server) saveStatusCache(overall *types.Overall) {
	if s.statusCachePath == "" || overall == nil {
		return
	}
	body, err := json.Marshal(overall)
	if err != nil {
		log.Printf("status cache save: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.statusCachePath), 0755); err != nil {
		log.Printf("status cache save: %v", err)
		return
	}
	tmp := s.statusCachePath + ".tmp"
	if err := os.WriteFile(tmp, body, 0644); err != nil {
		log.Printf("status cache save: %v", err)
		return
	}
	if err := os.Rename(tmp, s.statusCachePath); err != nil {
		log.Printf("status cache save: %v", err)
		return
	}
}

func (s *server) cachedFeed(base string) ([]byte, bool) {
	s.feedMu.Lock()
	defer s.feedMu.Unlock()
	if s.feedCache.base != base || len(s.feedCache.body) == 0 {
		return nil, false
	}
	if time.Since(s.feedCache.cachedAt) >= 5*time.Minute {
		return nil, false
	}
	body := append([]byte(nil), s.feedCache.body...)
	return body, true
}

func (s *server) setCachedFeed(base string, body []byte) {
	s.feedMu.Lock()
	defer s.feedMu.Unlock()
	s.feedCache = cachedAtomFeed{
		base:     base,
		body:     append([]byte(nil), body...),
		cachedAt: time.Now(),
	}
}

func buildAtomFeed(base, host string, overall *types.Overall) ([]byte, error) {
	entries := make([]atomEntry, 0)
	for _, c := range overall.Components {
		for _, b := range c.Days {
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

	updated := time.Now().UTC()
	if overall.UpdatedAt > 0 {
		updated = time.Unix(overall.UpdatedAt, 0).UTC()
	}
	feed := atomFeed{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: "Bangumi Status · Incidents",
		Link: []atomLink{
			{Href: base + "/api/feed.atom", Rel: "self", Type: "application/atom+xml"},
			{Href: base + "/", Rel: "alternate", Type: "text/html"},
		},
		ID:      "tag:" + host + ":feed",
		Updated: updated.Format(time.RFC3339),
		Author:  atomAuthor{Name: "Bangumi Status"},
		Entries: entries,
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// allowedReactions is the canonical set of reaction emoji IDs (chii.in TV smiles).
var allowedReactions = map[int]bool{
	44: true, 40: true, 15: true, 23: true, 83: true, 65: true,
	41: true, 102: true, 49: true, 46: true, 51: true, 101: true,
}

var allowedReactionIDs = []int{15, 23, 40, 41, 44, 46, 49, 51, 65, 83, 101, 102}

const reactionCacheTTL = 2 * time.Second

// reactionRate throttles reaction calls per source IP (token bucket: 300/min).
var reactionRate = newRateLimiter(300, time.Minute)

func (s *server) handleReactionsList(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if len(userID) > 128 {
		userID = ""
	}
	out, err := s.reactionSnapshot(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	s.invalidateReactionCache()
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

	// Clear the server's WriteTimeout for this long-lived stream; otherwise
	// every write past the 30s deadline fails and the connection churns.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	ch, cleanup := s.rxHub.subscribe()
	defer cleanup()

	send := func() bool {
		out, err := s.reactionSnapshot(r.Context(), userID)
		if err != nil {
			return false
		}
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

func (s *server) reactionSnapshot(ctx context.Context, userID string) ([]store.ReactionCount, error) {
	counts, err := s.cachedReactionCounts(ctx)
	if err != nil {
		return nil, err
	}
	mine, err := s.store.UserReactionEmojiIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]store.ReactionCount, len(counts))
	for _, c := range counts {
		c.Mine = mine[c.EmojiID]
		byID[c.EmojiID] = c
	}
	out := make([]store.ReactionCount, 0, len(allowedReactionIDs))
	for _, id := range allowedReactionIDs {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		} else {
			out = append(out, store.ReactionCount{EmojiID: id, Mine: mine[id]})
		}
	}
	return out, nil
}

func (s *server) cachedReactionCounts(ctx context.Context) ([]store.ReactionCount, error) {
	s.reactionMu.RLock()
	if s.reactionCounts != nil && time.Since(s.reactionCountsAt) < reactionCacheTTL {
		out := cloneReactionCounts(s.reactionCounts)
		s.reactionMu.RUnlock()
		return out, nil
	}
	s.reactionMu.RUnlock()

	v, err, _ := s.reactionSF.Do("counts", func() (any, error) {
		counts, err := s.store.ListReactionCounts(ctx)
		if err != nil {
			return nil, err
		}
		counts = cloneReactionCounts(counts)
		s.reactionMu.Lock()
		s.reactionCounts = counts
		s.reactionCountsAt = time.Now()
		s.reactionMu.Unlock()
		return cloneReactionCounts(counts), nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]store.ReactionCount), nil
}

func (s *server) invalidateReactionCache() {
	s.reactionMu.Lock()
	s.reactionCounts = nil
	s.reactionCountsAt = time.Time{}
	s.reactionMu.Unlock()
}

func cloneReactionCounts(in []store.ReactionCount) []store.ReactionCount {
	if in == nil {
		return nil
	}
	out := make([]store.ReactionCount, len(in))
	copy(out, in)
	return out
}

func (s *server) handleStatusStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	// Clear the server's WriteTimeout for this long-lived stream; otherwise
	// every write past the 30s deadline fails and the connection churns.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	ch, cleanup := s.statusHub.subscribe()
	defer cleanup()

	send := func() bool {
		overall, err := s.getOverall(r.Context())
		if err != nil {
			return false
		}
		buf, err := json.Marshal(overall)
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
			s.triggerStatusRefresh()
		}
		return cached, nil
	}

	s.triggerStatusRefresh()
	return nil, errStatusCacheWarming
}

func (s *server) triggerStatusRefresh() {
	go s.sfGroup.Do("refresh", func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), statusRefreshTimeout)
		defer cancel()
		out, err := s.computeOverall(ctx)
		if err != nil {
			log.Printf("status cache refresh: %v", err)
			return nil, err
		}
		s.mu.Lock()
		s.cached = out
		s.cachedAt = time.Now()
		s.mu.Unlock()
		s.saveStatusCache(out)
		s.notifier.UpdateSummary(out)
		s.statusHub.notify()
		return nil, nil
	})
}

// computeOverall assembles the full status payload. The live, fast-changing
// parts (current rolled-up status, per-probe views, probe list, online series)
// are recomputed on every call from cheap, narrowly-scoped queries. The
// expensive parts (the 30-day daily buckets and the 14-day incident walk, which
// each scan a large slice of the partitioned checks table) are served from a
// cache refreshed at most once per historyRefreshInterval — they change far too
// slowly to justify rescanning the table on every 20s refresh tick.
func (s *server) computeOverall(ctx context.Context) (*types.Overall, error) {
	allComponents := types.AllComponents()

	viewsByComponent, err := s.store.LatestPerProbeBatch(ctx, allComponents)
	if err != nil {
		return nil, err
	}
	histories, err := s.componentHistories(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]types.ComponentStatus, len(allComponents))
	for i, c := range allComponents {
		key := store.ComponentKey{Domain: c.Domain, Kind: c.Kind}
		views := viewsByComponent[key]
		currentStatus, last := store.RollupStatus(views)
		h := histories[key]
		results[i] = types.ComponentStatus{
			Domain:     c.Domain,
			Kind:       c.Kind,
			Label:      c.Label,
			Status:     currentStatus,
			Uptime:     h.uptime,
			Days:       h.days,
			LastCheck:  last,
			ProbeViews: views,
			Incidents:  h.incidents,
		}
	}

	probes, err := s.store.Probes(ctx)
	if err != nil {
		return nil, err
	}
	online, err := s.store.OnlineSeries(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
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

// componentHistory holds the slowly-changing, expensive-to-compute parts of a
// component's status: its 30-day daily buckets, recent incident windows, and
// overall uptime. These are cached and refreshed on a coarse cadence.
type componentHistory struct {
	days      []types.DayBucket
	incidents []types.Incident
	uptime    float64
}

// historyRefreshInterval bounds how often the per-component daily-bucket +
// incident aggregation runs. The 30-day uptime strip and 14-day incident list
// barely move between refreshes, so a coarse cadence here is what keeps
// database load flat regardless of how often /api/status is hit or how many SSE
// clients are connected. Live status is unaffected — it is recomputed on every
// call in computeOverall from the cheap LatestPerProbeBatch query, and outage
// alerting runs on its own loop (computeRollups). This is the single most
// important knob for database load; lower it only if the history feels stale.
const historyRefreshInterval = 5 * time.Minute

// componentHistories returns the cached per-component history, refreshing it
// when older than historyRefreshInterval. Concurrent callers coalesce onto one
// refresh; on refresh failure the last good snapshot is reused so a transient
// DB hiccup never blanks the 30-day chart.
func (s *server) componentHistories(ctx context.Context) (map[store.ComponentKey]componentHistory, error) {
	s.histMu.RLock()
	cached, at := s.history, s.historyAt
	s.histMu.RUnlock()
	if cached != nil && time.Since(at) < historyRefreshInterval {
		return cached, nil
	}

	v, err, _ := s.histSF.Do("history", func() (any, error) {
		// Re-check: another goroutine may have refreshed while we waited on
		// the singleflight.
		s.histMu.RLock()
		cached2, at2 := s.history, s.historyAt
		s.histMu.RUnlock()
		if cached2 != nil && time.Since(at2) < historyRefreshInterval {
			return cached2, nil
		}
		fresh, err := s.computeHistories(ctx)
		if err != nil {
			return nil, err
		}
		s.histMu.Lock()
		s.history = fresh
		s.historyAt = time.Now()
		s.histMu.Unlock()
		return fresh, nil
	})
	if err != nil {
		if cached != nil {
			return cached, nil // serve stale history rather than fail the whole page
		}
		return nil, err
	}
	return v.(map[store.ComponentKey]componentHistory), nil
}

// computeHistories aggregates the 30-day daily buckets and the 14-day incident
// windows for every component. Each query is scoped to one (domain, kind) so it
// rides the idx_checks_lookup(domain, kind, ts) index: the daily buckets are a
// bounded index-range aggregation and the incident walk gets its rows already
// ts-ordered, avoiding the multi-million-row external sort that a single
// cross-component ORDER BY ts would force. Concurrency is capped (well under the
// pool size) so a refresh never monopolizes the database.
func (s *server) computeHistories(ctx context.Context) (map[store.ComponentKey]componentHistory, error) {
	allComponents := types.AllComponents()
	since := time.Now().Add(-14 * 24 * time.Hour)

	out := make(map[store.ComponentKey]componentHistory, len(allComponents))
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(2)
	for _, c := range allComponents {
		c := c
		g.Go(func() error {
			buckets, err := s.store.DailyBuckets(gctx, c.Domain, c.Kind, 30)
			if err != nil {
				return err
			}
			incidents, err := s.store.Incidents(gctx, c.Domain, c.Kind, since)
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
			key := store.ComponentKey{Domain: c.Domain, Kind: c.Kind}
			mu.Lock()
			out[key] = componentHistory{days: buckets, incidents: incidents, uptime: up}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// computeRollups is the cheap subset of computeOverall the notifier needs:
// per-component rolled-up status from LatestPerProbeBatch, no Incidents walk or
// daily-bucket aggregation.
func (s *server) computeRollups(ctx context.Context) ([]types.ComponentStatus, error) {
	allComponents := types.AllComponents()
	results := make([]types.ComponentStatus, len(allComponents))
	viewsByComponent, err := s.store.LatestPerProbeBatch(ctx, allComponents)
	if err != nil {
		return nil, err
	}
	for i, c := range allComponents {
		key := store.ComponentKey{Domain: c.Domain, Kind: c.Kind}
		status, last := store.RollupStatus(viewsByComponent[key])
		results[i] = types.ComponentStatus{
			Domain:    c.Domain,
			Kind:      c.Kind,
			Status:    status,
			LastCheck: last,
		}
	}
	return results, nil
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

type wikiStatsConfig struct {
	url       string
	userAgent string
	cookie    string
	interval  time.Duration
}

type wikiStatsChartSet struct {
	Data []types.WikiStatsPoint `json:"data"`
}

func bangumiUserAgent() string {
	if ua := strings.TrimSpace(os.Getenv("BGM_USER_AGENT")); ua != "" {
		return ua
	}
	return "BangumiStatus/1.0 (+https://bgm-status.ry.mk)"
}

func bangumiCookieFromEnv() string {
	if cookie := strings.TrimSpace(os.Getenv("BGM_COOKIE")); cookie != "" {
		return cookie
	}
	raw := strings.TrimSpace(os.Getenv("BGM_COOKIE_JSON"))
	if raw == "" {
		return ""
	}
	var cookies []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(raw), &cookies); err != nil {
		log.Printf("wiki stats scraper disabled: invalid BGM_COOKIE_JSON: %v", err)
		return ""
	}
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		if c.Name == "" {
			continue
		}
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 5*time.Minute {
		log.Printf("invalid %s=%q, using %s", name, raw, fallback)
		return fallback
	}
	return d
}

func stringFromEnv(name, fallback string) string {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		return raw
	}
	return fallback
}

func (s *server) wikiStatsLoop(ctx context.Context, cfg wikiStatsConfig) {
	if strings.TrimSpace(cfg.cookie) == "" {
		log.Printf("wiki stats scraper disabled: BGM_COOKIE or BGM_COOKIE_JSON is required")
		return
	}
	log.Printf("wiki stats scraper enabled: %s every %s", cfg.url, cfg.interval)

	s.scrapeWikiStatsOnce(ctx, cfg)

	t := time.NewTicker(cfg.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.scrapeWikiStatsOnce(ctx, cfg)
		}
	}
}

func (s *server) scrapeWikiStatsOnce(parent context.Context, cfg wikiStatsConfig) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.url, nil)
	if err != nil {
		log.Printf("wiki stats scrape: %v", err)
		return
	}
	req.Header.Set("User-Agent", cfg.userAgent)
	req.Header.Set("Cookie", cfg.cookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("wiki stats scrape: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("wiki stats scrape: HTTP %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		log.Printf("wiki stats scrape: read: %v", err)
		return
	}
	raw, sets, err := extractWikiStatsChartSets(string(body))
	if err != nil {
		log.Printf("wiki stats scrape: %v", err)
		return
	}
	points, err := wikiStatsPointsFromSets(sets)
	if err != nil {
		log.Printf("wiki stats scrape: %v", err)
		return
	}
	accepted, latest, err := s.store.UpsertWikiStats(ctx, points, raw)
	if err != nil {
		log.Printf("wiki stats scrape: store: %v", err)
		return
	}
	log.Printf("wiki stats scrape: stored %d day(s), latest=%s", accepted, latest)
}

func extractWikiStatsChartSets(html string) ([]byte, map[string]wikiStatsChartSet, error) {
	const marker = "var CHART_SETS = "
	i := strings.Index(html, marker)
	if i < 0 {
		return nil, nil, errors.New("CHART_SETS not found")
	}
	rest := html[i+len(marker):]
	end, err := jsonObjectEnd(rest)
	if err != nil {
		return nil, nil, err
	}
	raw := []byte(rest[:end])
	var sets map[string]wikiStatsChartSet
	if err := json.Unmarshal(raw, &sets); err != nil {
		return nil, nil, fmt.Errorf("parse CHART_SETS: %w", err)
	}
	return raw, sets, nil
}

func jsonObjectEnd(s string) (int, error) {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return 0, errors.New("CHART_SETS JSON object not found")
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, errors.New("unterminated CHART_SETS JSON object")
}

func wikiStatsPointsFromSets(sets map[string]wikiStatsChartSet) ([]types.WikiStatsPoint, error) {
	for _, key := range []string{"register", "collection", "topics", "replies"} {
		if set, ok := sets[key]; ok && len(set.Data) > 0 {
			return set.Data, nil
		}
	}
	return nil, errors.New("CHART_SETS contains no daily data")
}

func (s *server) retentionLoop(ctx context.Context) {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		if err := s.store.EnsureChecksPartitions(ctx, 2); err != nil {
			log.Printf("retention: ensure partitions: %v", err)
		}
		cutoff := time.Now().AddDate(0, 0, -35) // keep a little extra
		n, err := s.store.PurgeOlderThan(ctx, cutoff)
		if err != nil {
			log.Printf("retention: %v", err)
		} else if n > 0 {
			log.Printf("retention: dropped %d partitions", n)
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
		} else {
			s.invalidateReactionCache()
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// requestRefresh signals cacheRefreshLoop to recompute the status snapshot
// soon. It never blocks: if a refresh is already pending, the ping is dropped
// (the pending one will pick up this ingest's data too).
func (s *server) requestRefresh() {
	select {
	case s.refreshCh <- struct{}{}:
	default:
	}
}

// refreshDebounce bounds how often ingest-driven refreshes run. Probes ingest
// every ~4s combined; without this an active outage (every probe reporting at
// once) could fire many refreshes back to back. 2s keeps the page near-live
// while collapsing bursts onto a single recompute.
const refreshDebounce = 2 * time.Second

func (s *server) cacheRefreshLoop(ctx context.Context) {
	// Warm the cache immediately at startup so the first /api/status request
	// after boot can use either the persisted cache or a bounded background refresh.
	s.triggerStatusRefresh()
	lastRefresh := time.Now()

	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// Steady heartbeat: a safety net if no ingest arrives and so SSE
			// clients get a periodic UpdatedAt bump.
		case <-s.refreshCh:
			// Ingest-driven: skip if we just refreshed for an earlier ping.
			if time.Since(lastRefresh) < refreshDebounce {
				continue
			}
		}
		// Force a background refresh by expiring the cache.
		s.mu.Lock()
		s.cachedAt = time.Time{}
		s.mu.Unlock()
		s.triggerStatusRefresh()
		lastRefresh = time.Now()
	}
}

// notifierLoop runs outage/recovery detection on a steady 20s tick. The
// notifier debounce needs a fixed cadence; running Process inside the
// computeOverall refresh coupled it to that refresh's duration, which grows
// with the checks table and had stretched the effective tick to ~60s.
func (s *server) notifierLoop(ctx context.Context) {
	if !s.notifier.Enabled() {
		return
	}
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		comps, err := s.computeRollups(ctx)
		if err != nil {
			log.Printf("notifier loop: %v", err)
			continue
		}
		s.notifier.Process(comps)
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

func withSecurityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.ServeHTTP(w, r)
	})
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

// Unwrap lets http.ResponseController reach the underlying connection (e.g. to
// clear the write deadline for long-lived SSE streams).
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

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
	case strings.HasSuffix(p, ".xml"):
		return "application/xml; charset=utf-8"
	case strings.HasSuffix(p, ".txt"):
		return "text/plain; charset=utf-8"
	}
	return "application/octet-stream"
}

// parseTokenPrefixes parses a third-party token allowlist of the form
//
//	token1:prefix1;token2:prefix2
//
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

// isValidHost rejects header values that contain characters illegal in a
// hostname (prevents host header injection via X-Forwarded-Host).
func isValidHost(h string) bool {
	if h == "" || len(h) > 253 {
		return false
	}
	for _, c := range h {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == ':') {
			return false
		}
	}
	return true
}

// redactDSN masks the password in a database connection string so it can be
// logged safely.
func redactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "***"
	}
	if _, hasPwd := u.User.Password(); hasPwd {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.String()
}
