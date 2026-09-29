// Package status assembles the public status snapshot. Live status is cheap and
// recomputed often; the 30-day strip and incident walk are expensive and
// cached coarsely; the last snapshot is persisted so a restart serves at once.
package status

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"bangumi-status/internal/cache"
	"bangumi-status/internal/rollup"
	"bangumi-status/internal/store"
	"bangumi-status/internal/target"
	"bangumi-status/internal/types"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

// ErrWarming means no snapshot exists yet; one is being computed.
var ErrWarming = errors.New("status cache is warming up")

const (
	// staleAfter makes a read trigger a background refresh.
	staleAfter = 30 * time.Second
	// tick is the steady refresh heartbeat when no ingest arrives.
	tick = 20 * time.Second
	// debounce bounds how often ingest-driven refreshes run: probes ingest
	// every ~4s combined, and an outage makes them all report at once.
	debounce = 2 * time.Second
	// refreshTimeout bounds one snapshot computation.
	refreshTimeout = 3 * time.Minute
	// historyTTL is the most important knob for database load: the 30-day
	// strip and 14-day walk barely move between refreshes, and caching them
	// keeps load flat however many clients watch.
	historyTTL = 5 * time.Minute

	stripDays    = 30
	incidentDays = 14
	// backfillDays matches retention: everything checks can still hold.
	backfillDays = 35
	backfillKey  = "incidents_backfilled_v1"
)

type history struct {
	days      []types.DayBucket
	incidents []types.Incident
	uptime    float64
}

type snapshot struct {
	overall *types.Overall
	at      time.Time
}

type Service struct {
	store     *store.Store
	cachePath string
	// onUpdate runs after every fresh snapshot (and the restored one).
	onUpdate []func(*types.Overall)

	current atomic.Pointer[snapshot]
	sf      singleflight.Group
	refresh chan struct{}
	history *cache.Value[map[types.ComponentKey]history]

	// outageSince records when each component's live status last left ok. It
	// is the single source of truth for "ongoing for", derived from the same
	// signal as the live status so the two can never disagree, and restored
	// from the persisted snapshot so it survives restarts.
	outageMu    sync.Mutex
	outageSince map[types.ComponentKey]int64
}

func New(st *store.Store, cachePath string, onUpdate ...func(*types.Overall)) *Service {
	s := &Service{
		store:       st,
		cachePath:   cachePath,
		onUpdate:    onUpdate,
		refresh:     make(chan struct{}, 1),
		outageSince: map[types.ComponentKey]int64{},
	}
	s.history = cache.NewValue(historyTTL, s.computeHistories)
	s.load()
	return s
}

// Snapshot returns the latest snapshot without triggering a refresh.
func (s *Service) Snapshot() (*types.Overall, bool) {
	if c := s.current.Load(); c != nil {
		return c.overall, true
	}
	return nil, false
}

// Get returns the latest snapshot, kicking off a background refresh when it
// is stale. Readers never wait on the database.
func (s *Service) Get() (*types.Overall, error) {
	c := s.current.Load()
	if c == nil || time.Since(c.at) >= staleAfter {
		go s.Refresh()
	}
	if c == nil {
		return nil, ErrWarming
	}
	return c.overall, nil
}

// RequestRefresh asks Run to recompute soon. It never blocks: a pending
// request already covers this one.
func (s *Service) RequestRefresh() {
	select {
	case s.refresh <- struct{}{}:
	default:
	}
}

// Run keeps the snapshot fresh until ctx is done.
func (s *Service) Run(ctx context.Context) {
	s.Refresh()
	last := time.Now()
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-s.refresh:
			if time.Since(last) < debounce {
				continue
			}
		}
		s.Refresh()
		last = time.Now()
	}
}

// Refresh recomputes the snapshot; concurrent calls share one computation.
func (s *Service) Refresh() {
	s.sf.Do("", func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
		defer cancel()
		out, err := s.compute(ctx)
		if err != nil {
			slog.Error("status refresh failed", "err", err)
			return nil, err
		}
		s.current.Store(&snapshot{overall: out, at: time.Now()})
		s.save(out)
		for _, f := range s.onUpdate {
			f(out)
		}
		return nil, nil
	})
}

// trackOutage stamps when key's current non-ok streak began and returns it,
// or clears it and returns 0 when the component is ok. Caller holds outageMu.
func (s *Service) trackOutage(key types.ComponentKey, st types.Status, now int64) int64 {
	if st == types.StatusOK {
		delete(s.outageSince, key)
		return 0
	}
	if s.outageSince[key] == 0 {
		s.outageSince[key] = now
	}
	return s.outageSince[key]
}

// Rollups is the cheap live-only view the notifier polls: current status and
// outage start per component, without history.
func (s *Service) Rollups(ctx context.Context) ([]types.ComponentStatus, error) {
	comps := target.Components()
	views, err := s.store.LatestPerProbe(ctx, comps)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	out := make([]types.ComponentStatus, len(comps))
	s.outageMu.Lock()
	defer s.outageMu.Unlock()
	for i, c := range comps {
		st, last, _, _ := rollup.Live(views[c.Key()])
		out[i] = types.ComponentStatus{Domain: c.Domain, Kind: c.Kind, Status: st, LastCheck: last,
			Since: s.trackOutage(c.Key(), st, now)}
	}
	return out, nil
}

func (s *Service) compute(ctx context.Context) (*types.Overall, error) {
	comps := target.Components()
	views, err := s.store.LatestPerProbe(ctx, comps)
	if err != nil {
		return nil, err
	}
	hist, err := s.history.Get(ctx)
	if err != nil {
		return nil, err
	}
	probes, err := s.store.Probes(ctx)
	if err != nil {
		return nil, err
	}
	online, err := s.store.Points(ctx, store.Online, time.Now().Add(-24*time.Hour), 0)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	out := &types.Overall{UpdatedAt: now, Probes: probes, Online: online, Status: types.StatusOK}
	s.outageMu.Lock()
	for _, c := range comps {
		v := views[c.Key()]
		st, last, active, bad := rollup.Live(v)
		since := s.trackOutage(c.Key(), st, now)
		h := hist[c.Key()]
		// The walk is cached and lags; fold the live outage in so the list
		// shows one ongoing incident anchored at `since`.
		incidents := h.incidents
		if since != 0 {
			incidents = rollup.MergeOngoing(h.incidents, since, now, st, bad, active)
		}
		out.Components = append(out.Components, types.ComponentStatus{
			Domain: c.Domain, Kind: c.Kind, Label: c.Label, Status: st, Uptime: h.uptime, Since: since,
			Days: h.days, LastCheck: last, ProbeViews: v, Incidents: incidents,
		})
		out.Status = types.Worst(out.Status, st)
	}
	s.outageMu.Unlock()
	out.Message = out.Status.Message()
	return out, nil
}

// computeHistories builds every component's strip and incident walk, and
// mirrors the walk into the durable incident archive. Each query is scoped to
// one component so it rides the (domain, kind, ts) index; concurrency is
// capped well under the pool size so a refresh never starves ingest.
func (s *Service) computeHistories(ctx context.Context) (map[types.ComponentKey]history, error) {
	since := time.Now().AddDate(0, 0, -incidentDays)
	out := map[types.ComponentKey]history{}
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(2)
	for _, c := range target.Components() {
		g.Go(func() error {
			days, err := s.store.DailyBuckets(ctx, c.Domain, c.Kind, stripDays)
			if err != nil {
				return err
			}
			incidents, err := s.store.Incidents(ctx, c.Domain, c.Kind, since)
			if err != nil {
				return err
			}
			rollup.Overlay(days, incidents)
			// The archive rides the walk we already paid for. Its failure must
			// not fail the status page.
			if err := s.store.SyncIncidents(ctx, c.Domain, c.Kind, since.Unix(), incidents); err != nil {
				slog.Error("sync incidents", "component", c.Label, "err", err)
			}
			var ok, total int
			for _, b := range days {
				total += b.Total
				ok += b.Total - b.Down - b.Degrade
			}
			h := history{days: days, incidents: incidents}
			if total > 0 {
				h.uptime = float64(ok) / float64(total) * 100
			}
			mu.Lock()
			out[c.Key()] = h
			mu.Unlock()
			return nil
		})
	}
	return out, g.Wait()
}

// Backfill seeds the incident archive, once ever, from whatever raw checks
// retention still holds. It walks components serially so it never competes
// with ingest; if interrupted it leaves the marker unset and retries next boot.
func (s *Service) Backfill(ctx context.Context) {
	if _, done, err := s.store.GetConfig(ctx, backfillKey); err != nil || done {
		if err != nil {
			slog.Error("incident backfill: read marker", "err", err)
		}
		return
	}
	since := time.Now().AddDate(0, 0, -backfillDays)
	total := 0
	for _, c := range target.Components() {
		incs, err := s.store.Incidents(ctx, c.Domain, c.Kind, since)
		if err == nil {
			err = s.store.SyncIncidents(ctx, c.Domain, c.Kind, since.Unix(), incs)
		}
		if err != nil {
			slog.Error("incident backfill", "component", c.Label, "err", err)
			return
		}
		total += len(incs)
	}
	if err := s.store.SetConfig(ctx, backfillKey, time.Now().UTC().Format(time.RFC3339)); err != nil {
		slog.Error("incident backfill: write marker", "err", err)
		return
	}
	slog.Info("incident backfill done", "incidents", total, "days", backfillDays)
}

// load restores the persisted snapshot, marked stale so the first read
// refreshes it, together with each component's outage start.
func (s *Service) load() {
	if s.cachePath == "" {
		return
	}
	body, err := os.ReadFile(s.cachePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("status cache load", "err", err)
		}
		return
	}
	var o types.Overall
	if err := json.Unmarshal(body, &o); err != nil {
		slog.Warn("status cache load", "err", err)
		return
	}
	// Drop components that are no longer monitored.
	kept := o.Components[:0]
	o.Status = types.StatusOK
	for _, c := range o.Components {
		if target.Monitored(c.Domain, c.Kind) {
			kept = append(kept, c)
			o.Status = types.Worst(o.Status, c.Status)
			if c.Status != types.StatusOK && c.Since != 0 {
				s.outageSince[types.ComponentKey{Domain: c.Domain, Kind: c.Kind}] = c.Since
			}
		}
	}
	o.Components = kept
	o.Message = o.Status.Message()
	s.current.Store(&snapshot{overall: &o, at: time.Now().Add(-staleAfter - time.Second)})
	for _, f := range s.onUpdate {
		f(&o)
	}
	slog.Info("loaded status cache", "path", s.cachePath)
}

// save persists the snapshot atomically (write + rename).
func (s *Service) save(o *types.Overall) {
	if s.cachePath == "" {
		return
	}
	err := func() error {
		body, err := json.Marshal(o)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(s.cachePath), 0o755); err != nil {
			return err
		}
		tmp := s.cachePath + ".tmp"
		if err := os.WriteFile(tmp, body, 0o644); err != nil {
			return err
		}
		return os.Rename(tmp, s.cachePath)
	}()
	if err != nil {
		slog.Warn("status cache save", "err", err)
	}
}
