package server

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bangumi-status/internal/region"
	"bangumi-status/internal/target"
	"bangumi-status/internal/types"
)

// authorizeProbe reports whether token may report as probeID: the legacy
// shared secret accepts any id, a third-party token only ids strictly inside
// its prefix namespace. tokenOK says whether the token was recognised at all.
func (s *Server) authorizeProbe(token, probeID string) (ok, tokenOK bool) {
	eq := func(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
	if s.cfg.IngestSecret != "" && eq(token, s.cfg.IngestSecret) {
		return true, true
	}
	for tok, prefix := range s.cfg.TokenPrefixes {
		if eq(token, tok) {
			return strings.HasPrefix(probeID, prefix) && len(probeID) > len(prefix), true
		}
	}
	return false, false
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if _, known := s.authorizeProbe(token, ""); !found || !known {
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
	// Reject unknown regions so a misconfigured probe surfaces immediately
	// instead of polluting the data set.
	code, ok := region.Normalize(p.Region)
	if !ok {
		http.Error(w, "invalid region (must be ISO 3166-1 alpha-2 country code)", http.StatusBadRequest)
		return
	}
	p.Region = code
	if ok, _ := s.authorizeProbe(token, p.Probe); !ok {
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
		switch {
		case !res.Valid():
		// Stale reports from old probe binaries or retired targets.
		case !target.Monitored(res.Domain, res.Kind):
		// A missing marker means this probe's cookie expired, not an outage.
		case res.Kind == types.KindAuth && res.Err == "marker missing":
		default:
			kept = append(kept, res)
		}
	}
	if err := s.store.Insert(r.Context(), kept); err != nil {
		internalError(w, r, err)
		return
	}
	if err := s.store.UpsertProbe(r.Context(), p.Probe, p.Region, time.Now().Unix()); err != nil {
		slog.Warn("upsert probe", "probe", p.Probe, "err", err)
	}
	if p.OnlineCount > 0 {
		s.recordOnline(r, p)
	}
	// Ingest-driven refresh gets a status change to open pages within seconds.
	// The status service debounces and never recomputes history here.
	if len(kept) > 0 {
		s.status.RequestRefresh()
	}
	writeJSON(w, "", map[string]any{"ok": true, "accepted": len(kept)})
}

// recordOnline persists the online counter when it changes. Every probe feeds
// it and the site-wide value is cached ~10 min upstream, so storing only
// transitions sheds ~10× duplicates. Values are stored faithfully, including
// the ~2× cache-rebuild spikes, which reads filter out (see store.Points).
func (s *Server) recordOnline(r *http.Request, p types.IngestPayload) {
	ts := p.OnlineTS
	if ts == 0 {
		ts = time.Now().Unix()
	}
	s.onlineMu.Lock()
	changed := p.OnlineCount != s.lastOnline
	s.lastOnline = p.OnlineCount
	s.onlineMu.Unlock()
	if !changed {
		return
	}
	if err := s.store.InsertOnline(r.Context(), ts, p.OnlineCount); err != nil {
		slog.Warn("insert online count", "err", err)
	}
}
