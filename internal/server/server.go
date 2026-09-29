// Package server is the aggregator's HTTP surface: probe ingest, the public
// JSON/Atom API, the live event stream and the embedded frontend.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bangumi-status/internal/cache"
	"bangumi-status/internal/config"
	"bangumi-status/internal/sse"
	"bangumi-status/internal/status"
	"bangumi-status/internal/store"
	"bangumi-status/internal/types"
)

type Server struct {
	cfg    config.Aggregator
	store  *store.Store
	status *status.Service
	site   fs.FS

	// Hubs behind /api/events.
	statusHub    *sse.Hub
	viewersHub   *sse.Hub
	reactionsHub *sse.Hub
	// viewers counts open status-page event streams: "people watching now".
	viewers atomic.Int64
	peak    trafficPeak

	// lastOnline is the last persisted online count. Every probe reports the
	// counter but a row is written only when it changes.
	onlineMu   sync.Mutex
	lastOnline int

	reactionCounts *cache.Value[[]types.ReactionCount]
	reactionLimit  *rateLimiter
	feeds          *cache.Map[string, []byte]
	incidentPages  *cache.Map[[2]int64, []byte]
}

// New builds the server. site is the built frontend.
func New(ctx context.Context, cfg config.Aggregator, st *store.Store, site fs.FS) *Server {
	s := &Server{
		cfg:           cfg,
		store:         st,
		site:          site,
		statusHub:     sse.NewHub(),
		viewersHub:    sse.NewHub(),
		reactionsHub:  sse.NewHub(),
		reactionLimit: newRateLimiter(300, time.Minute),
		feeds:         cache.NewMap[string, []byte](5*time.Minute, 8),
		incidentPages: cache.NewMap[[2]int64, []byte](5*time.Minute, 64),
	}
	s.reactionCounts = cache.NewValue(2*time.Second, st.ReactionCounts)
	if n, err := st.LatestOnline(ctx); err == nil {
		s.lastOnline = n
	}
	return s
}

// SetStatus wires the status service in; the two reference each other.
func (s *Server) SetStatus(svc *status.Service) { s.status = svc }

// StatusChanged pushes a fresh snapshot to connected pages.
func (s *Server) StatusChanged(*types.Overall) { s.statusHub.Notify() }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/ingest", s.handleIngest)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/mini", s.handleMini)
	mux.HandleFunc("GET /api/online", s.handleSeries(store.Online))
	mux.HandleFunc("GET /api/traffic", s.handleSeries(store.Traffic))
	mux.HandleFunc("GET /api/wiki-stats", s.handleWikiStats)
	mux.HandleFunc("GET /api/probes", s.handleProbes)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/feed.atom", s.handleFeed)
	mux.HandleFunc("GET /api/incidents", s.handleIncidents)
	mux.HandleFunc("GET /api/reactions", s.handleReactions)
	mux.HandleFunc("POST /api/reactions", s.handleReact)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	mux.Handle("/", s.spa())
	return securityHeaders(logRequests(mux))
}

// spa serves the built frontend. Unknown paths get index.html, which picks
// its page from location.pathname (/, /stats, /history).
func (s *Server) spa() http.Handler {
	files := http.FileServerFS(s.site)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if st, err := fs.Stat(s.site, p); p == "" || err != nil || st.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, s.site, "index.html")
			return
		}
		if strings.HasPrefix(p, "assets/") { // content-hashed by Vite
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, cacheControl string, v any) {
	w.Header().Set("Content-Type", "application/json")
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeRaw(w http.ResponseWriter, contentType, cacheControl string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", cacheControl)
	_, _ = w.Write(body)
}

// internalError logs err and answers 500 without leaking details.
func internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "path", r.URL.Path, "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func securityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hd := w.Header()
		hd.Set("X-Content-Type-Options", "nosniff")
		hd.Set("X-Frame-Options", "DENY")
		hd.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		hd.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.ServeHTTP(w, r)
	})
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		h.ServeHTTP(rec, r)
		if r.URL.Path != "/api/health" {
			slog.Info("http", "method", r.Method, "path", r.URL.Path, "code", rec.code, "dur", time.Since(start).Round(time.Microsecond))
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

// Unwrap lets http.ResponseController reach the connection (flush, deadlines).
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// clientIP is the caller's address as reported by the reverse proxy.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
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
