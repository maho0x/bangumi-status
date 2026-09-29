package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"bangumi-status/internal/sse"
	"bangumi-status/internal/types"
)

// handleEvents is the status page's single live connection. It multiplexes
//
//	event: status     the full status snapshot, on every refresh
//	event: viewers    {viewers, updated_at}, whenever a page opens or closes
//	event: reactions  reaction counts, flagged with the caller's own (?uid=)
//
// Each open stream counts as one viewer of the status page.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	uid := userID(r.URL.Query().Get("uid"))
	s.viewers.Add(1)
	s.viewersChanged()
	defer func() {
		s.viewers.Add(-1)
		s.viewersChanged()
	}()
	sse.Serve(w, r,
		sse.Stream{Event: "status", Hub: s.statusHub, Data: func(context.Context) (any, error) { return s.status.Get() }},
		sse.Stream{Event: "viewers", Hub: s.viewersHub, Data: func(context.Context) (any, error) {
			return map[string]int64{"viewers": s.viewers.Load(), "updated_at": time.Now().Unix()}, nil
		}},
		sse.Stream{Event: "reactions", Hub: s.reactionsHub, Data: func(ctx context.Context) (any, error) {
			return s.reactionsFor(ctx, uid)
		}},
	)
}

// viewersChanged broadcasts the new viewer count and folds it into this
// minute's peak. The write is detached: on disconnect the request context is
// already cancelled.
func (s *Server) viewersChanged() {
	s.viewersHub.Notify()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.RecordTraffic(ctx)
	}()
}

// RecordTraffic folds the current viewer count into the per-minute peak and
// persists the minute when its peak moves. It also runs once a minute so a
// minute with a steady audience still gets a row.
func (s *Server) RecordTraffic(ctx context.Context) {
	ts, peak, ok := s.peak.observe(time.Now(), int(s.viewers.Load()))
	if !ok {
		return
	}
	if err := s.store.InsertTrafficSample(ctx, ts, peak); err != nil {
		slog.Warn("traffic sample", "err", err)
	}
}

// trafficPeak tracks the highest concurrent viewer count within the current
// minute. A once-a-minute snapshot would miss a visitor who reads for 20s
// between two ticks; the peak is the honest "how many had the page open".
type trafficPeak struct {
	mu     sync.Mutex
	bucket int64
	peak   int
}

// observe folds n into the current minute and returns the minute to persist,
// or ok=false when n does not raise its peak.
func (p *trafficPeak) observe(now time.Time, n int) (tsMin int64, peak int, ok bool) {
	b := now.Unix() - now.Unix()%60
	p.mu.Lock()
	defer p.mu.Unlock()
	if b != p.bucket {
		p.bucket, p.peak = b, n
		return b, n, true
	}
	if n <= p.peak {
		return 0, 0, false
	}
	p.peak = n
	return b, n, true
}

// Reactions: the chii.in TV smiles a visitor can throw at the page. Counts
// cover the last 24h; the user id is an anonymous id the page generates.

var reactionIDs = []int{15, 23, 40, 41, 44, 46, 49, 51, 65, 83, 101, 102}

func userID(raw string) string {
	id := strings.TrimSpace(raw)
	if len(id) > 128 {
		return ""
	}
	return id
}

func (s *Server) reactionsFor(ctx context.Context, uid string) ([]types.ReactionCount, error) {
	counts, err := s.reactionCounts.Get(ctx)
	if err != nil {
		return nil, err
	}
	mine, err := s.store.UserReactions(ctx, uid)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]int, len(counts))
	for _, c := range counts {
		byID[c.EmojiID] = c.Count
	}
	out := make([]types.ReactionCount, len(reactionIDs))
	for i, id := range reactionIDs {
		out[i] = types.ReactionCount{EmojiID: id, Count: byID[id], Mine: mine[id]}
	}
	return out, nil
}

func (s *Server) handleReactions(w http.ResponseWriter, r *http.Request) {
	out, err := s.reactionsFor(r.Context(), userID(r.Header.Get("X-User-ID")))
	if err != nil {
		internalError(w, r, err)
		return
	}
	writeJSON(w, "no-store", out)
}

func (s *Server) handleReact(w http.ResponseWriter, r *http.Request) {
	uid := userID(r.Header.Get("X-User-ID"))
	if len(uid) < 8 {
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
	if !isReaction(body.EmojiID) {
		http.Error(w, "unknown emoji_id", http.StatusBadRequest)
		return
	}
	ip := clientIP(r)
	if !s.reactionLimit.allow(ip) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	if err := s.store.AddReaction(r.Context(), body.EmojiID, uid, ip); err != nil {
		internalError(w, r, err)
		return
	}
	s.reactionCounts.Invalidate()
	s.reactionsHub.Notify()
	w.WriteHeader(http.StatusNoContent)
}

func isReaction(id int) bool {
	for _, r := range reactionIDs {
		if r == id {
			return true
		}
	}
	return false
}

// PurgeReactions drops expired reactions (housekeeping; reads already ignore them).
func (s *Server) PurgeReactions(ctx context.Context) {
	if err := s.store.PurgeExpiredReactions(ctx); err != nil {
		slog.Warn("purge reactions", "err", err)
		return
	}
	s.reactionCounts.Invalidate()
}

// rateLimiter is a per-key sliding-window limiter, adequate for low-rate UI
// actions.
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
	recent := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	if len(l.hits) > 4096 { // keep the map bounded
		for k, v := range l.hits {
			if len(v) == 0 || v[len(v)-1].Before(cutoff) {
				delete(l.hits, k)
			}
		}
	}
	return true
}
