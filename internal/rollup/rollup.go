// Package rollup turns individual probe observations into a component status.
// It is pure: the live status (Live), the historical incident walk (Walker)
// and the daily strip overlay all share one quorum rule, so they can never
// disagree about whether a component was affected.
package rollup

import (
	"time"

	"bangumi-status/internal/types"
)

// ExcludedRegion never counts toward status: probes there see the GFW, not
// Bangumi.
const ExcludedRegion = "cn"

// StaleAfter is how long a probe's last observation still counts. Probes idle
// longer are treated as offline.
const StaleAfter = int64(180)

// Quorum is the minimum number of agreeing probes needed to escalate status
// among n active ones: ceil(2/3 × n), with a floor of 2.
func Quorum(n int) int {
	return max((n*2+2)/3, 2)
}

type vote struct {
	status types.Status
	ts     int64
}

// decide applies the staleness gate and the quorum rule. It returns the
// escalated status, the active probe count and how many of those are non-ok.
func decide(votes []vote, now int64) (status types.Status, active, bad int) {
	var down int
	for _, v := range votes {
		if now-v.ts > StaleAfter {
			continue
		}
		active++
		switch v.status {
		case types.StatusDegraded:
			bad++
		case types.StatusDown:
			bad++
			down++
		}
	}
	q := Quorum(active)
	switch {
	case down >= q:
		return types.StatusDown, active, bad
	case bad >= q:
		return types.StatusDegraded, active, bad
	}
	return types.StatusOK, active, bad
}

// Live computes the current status from each probe's latest view, plus the
// newest observation time and the active / non-ok probe counts.
func Live(views []types.ProbeView) (status types.Status, last int64, active, bad int) {
	votes := make([]vote, 0, len(views))
	for _, v := range views {
		if v.Region == ExcludedRegion {
			continue
		}
		last = max(last, v.TS)
		votes = append(votes, vote{v.Status, v.TS})
	}
	status, active, bad = decide(votes, time.Now().Unix())
	return status, last, active, bad
}

// Walker replays raw observations in time order and emits contiguous non-ok
// windows. After every event it re-decides the status from each probe's most
// recent observation, exactly as Live would have at that instant.
type Walker struct {
	probes map[string]vote
	cur    *types.Incident
	out    []types.Incident
}

func NewWalker() *Walker { return &Walker{probes: map[string]vote{}} }

// Add feeds one observation. Callers must feed them oldest first and must have
// dropped ExcludedRegion already.
func (w *Walker) Add(ts int64, probe string, status types.Status) {
	w.probes[probe] = vote{status, ts}
	votes := make([]vote, 0, len(w.probes))
	for _, v := range w.probes {
		votes = append(votes, v)
	}
	rolled, active, bad := decide(votes, ts)
	if rolled == types.StatusOK {
		w.close()
		return
	}
	if w.cur == nil {
		w.cur = &types.Incident{StartTS: ts, EndTS: ts, Status: rolled, PeakDown: bad, PeakTotal: active}
		return
	}
	w.cur.EndTS = ts
	w.cur.Status = types.Worst(w.cur.Status, rolled)
	if bad > w.cur.PeakDown {
		w.cur.PeakDown, w.cur.PeakTotal = bad, active
	}
}

// Finish closes any open window and returns every window, oldest first.
func (w *Walker) Finish() []types.Incident {
	w.close()
	return w.out
}

func (w *Walker) close() {
	if w.cur == nil {
		return
	}
	w.cur.DurationS = Duration(w.cur.StartTS, w.cur.EndTS)
	w.out = append(w.out, *w.cur)
	w.cur = nil
}

// Duration is an incident's displayed length: never shorter than a minute,
// because a window seen by a single sample still lasted at least one tick.
func Duration(start, end int64) int {
	return max(int(end-start), 60)
}

// MergeOngoing folds the currently-active outage into the (coarsely cached,
// possibly lagging) incident walk so the list and chart always show one
// ongoing incident that starts at `since` and stays open to `now` — consistent
// with the banner's "已持续" duration. Resolved windows are kept untouched; any
// walk window overlapping the current outage is absorbed into the synthetic
// ongoing one, carrying its peak counts and worst severity. liveBad/liveTotal
// seed the peak from the current instant so a just-started outage still shows
// sensible figures before the walk catches up. Returns oldest first.
func MergeOngoing(walk []types.Incident, since, now int64, status types.Status, liveBad, liveTotal int) []types.Incident {
	ongoing := types.Incident{StartTS: since, EndTS: now, Status: status, PeakDown: liveBad, PeakTotal: liveTotal}
	out := make([]types.Incident, 0, len(walk)+1)
	for _, inc := range walk {
		if inc.EndTS < since {
			out = append(out, inc) // resolved before the current outage began
			continue
		}
		ongoing.Status = types.Worst(ongoing.Status, inc.Status)
		if inc.PeakDown > ongoing.PeakDown {
			ongoing.PeakDown, ongoing.PeakTotal = inc.PeakDown, inc.PeakTotal
		}
	}
	ongoing.DurationS = Duration(since, now)
	return append(out, ongoing)
}

// Overlay marks each day touched by an incident with at least the incident's
// severity. Daily buckets use strict per-minute quorum and can miss brief
// outages whose bad checks straddle a minute boundary (probes are staggered
// within each minute); the walk uses the rolling staleness window and is the
// authoritative answer to "did something bad happen on this day".
func Overlay(buckets []types.DayBucket, incidents []types.Incident) {
	if len(buckets) == 0 || len(incidents) == 0 {
		return
	}
	idx := make(map[string]int, len(buckets))
	for i, b := range buckets {
		idx[b.Day] = i
	}
	for _, inc := range incidents {
		d := startOfDay(time.Unix(inc.StartTS, 0))
		stop := startOfDay(time.Unix(inc.EndTS, 0))
		for ; !d.After(stop); d = d.AddDate(0, 0, 1) {
			if i, ok := idx[d.Format(time.DateOnly)]; ok {
				buckets[i].Status = types.Worst(buckets[i].Status, inc.Status)
			}
		}
	}
}

func startOfDay(t time.Time) time.Time {
	t = t.In(types.CST)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, types.CST)
}
