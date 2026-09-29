package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"bangumi-status/internal/status"
	"bangumi-status/internal/store"
	"bangumi-status/internal/types"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	overall, err := s.status.Get()
	if errors.Is(err, status.ErrWarming) {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	// no-cache, not no-store: edge caches may keep a copy but must revalidate,
	// so a reload never shows a stale snapshot. Answered from memory.
	writeJSON(w, "no-cache", overall)
}

// handleMini is a tiny badge-style summary of the two main sites.
func (s *Server) handleMini(w http.ResponseWriter, r *http.Request) {
	comps, updatedAt := []types.ComponentStatus(nil), int64(0)
	if o, ok := s.status.Snapshot(); ok {
		comps, updatedAt = o.Components, o.UpdatedAt
	} else {
		var err error
		if comps, err = s.status.Rollups(r.Context()); err != nil {
			internalError(w, r, err)
			return
		}
		for _, c := range comps {
			updatedAt = max(updatedAt, c.LastCheck)
		}
	}
	worst := types.StatusOK
	for _, c := range comps {
		if c.Domain == "bgm.tv" || c.Domain == "bangumi.tv" {
			worst = types.Worst(worst, c.Status)
		}
	}
	writeJSON(w, "public, max-age=20", map[string]any{
		"status": worst, "message": worst.Message(), "updated_at": updatedAt,
	})
}

// handleSeries serves a time series; ?range= is 24h (default), 7d, 30d or all.
func (s *Server) handleSeries(series store.Series) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		since, bucket := time.Now().Add(-24*time.Hour), int64(0)
		switch r.URL.Query().Get("range") {
		case "7d":
			since, bucket = time.Now().AddDate(0, 0, -7), 600
		case "30d":
			since, bucket = time.Now().AddDate(0, 0, -30), 3600
		case "all":
			since, bucket = time.Unix(0, 0), 21600
		}
		pts, err := s.store.Points(r.Context(), series, since, bucket)
		if err != nil {
			internalError(w, r, err)
			return
		}
		if pts == nil {
			pts = []types.OnlinePoint{}
		}
		writeJSON(w, "public, max-age=60", pts)
	}
}

func (s *Server) handleWikiStats(w http.ResponseWriter, r *http.Request) {
	points, scrapedAt, err := s.store.WikiStats(r.Context())
	if err != nil {
		internalError(w, r, err)
		return
	}
	if points == nil {
		points = []types.WikiStatsPoint{}
	}
	writeJSON(w, "public, max-age=300", map[string]any{
		"updated_at": time.Now().Unix(),
		"scraped_at": scrapedAt,
		"source_url": s.cfg.WikiStatsURL,
		"data":       points,
	})
}

func (s *Server) handleProbes(w http.ResponseWriter, r *http.Request) {
	ps, err := s.store.Probes(r.Context())
	if err != nil {
		internalError(w, r, err)
		return
	}
	writeJSON(w, "", ps)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.Stats(r.Context())
	if err != nil {
		internalError(w, r, err)
		return
	}
	stats["now"] = time.Now().Unix()
	writeJSON(w, "", stats)
}

const incidentMonthsMax = 24

// incidentWindow resolves the [from, to) an /api/incidents caller asked for.
//
// The page passes explicit unix bounds because it groups incidents by the
// viewer's calendar; server-chosen CST month boundaries would land mid-month
// for a viewer elsewhere. A bare call gets `months` (default 3) whole CST
// months up to the end of the current one, the project's calendar elsewhere.
func incidentWindow(q url.Values) (int64, int64, error) {
	fromRaw, toRaw := q.Get("from"), q.Get("to")
	if fromRaw == "" && toRaw == "" {
		months := 3
		if v := q.Get("months"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return 0, 0, errors.New("bad months")
			}
			months = min(n, incidentMonthsMax)
		}
		now := time.Now().In(types.CST)
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, types.CST).AddDate(0, 1, 0)
		return end.AddDate(0, -months, 0).Unix(), end.Unix(), nil
	}
	from, err := strconv.ParseInt(fromRaw, 10, 64)
	if err != nil {
		return 0, 0, errors.New("bad from")
	}
	to, err := strconv.ParseInt(toRaw, 10, 64)
	if err != nil {
		return 0, 0, errors.New("bad to")
	}
	if to <= from {
		return 0, 0, errors.New("to must be after from")
	}
	if to-from > incidentMonthsMax*31*86400 {
		return 0, 0, errors.New("window too large")
	}
	return from, to, nil
}

// handleIncidents serves one window of the incident archive. The archive only
// changes when a walk lands, so pages are cached.
func (s *Server) handleIncidents(w http.ResponseWriter, r *http.Request) {
	from, to, err := incidentWindow(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body, err := s.incidentPages.GetOrLoad([2]int64{from, to}, func() ([]byte, error) {
		incidents, err := s.store.ListIncidents(r.Context(), from, to)
		if err != nil {
			return nil, err
		}
		earliest, err := s.store.EarliestIncidentTS(r.Context())
		if err != nil {
			return nil, err
		}
		return json.Marshal(types.IncidentHistory{Incidents: incidents, From: from, To: to, EarliestTS: earliest})
	})
	if err != nil {
		internalError(w, r, err)
		return
	}
	writeRaw(w, "application/json; charset=utf-8", "public, max-age=300", body)
}
