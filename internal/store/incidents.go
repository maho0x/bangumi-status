package store

import (
	"context"

	"bangumi-status/internal/rollup"
	"bangumi-status/internal/target"
	"bangumi-status/internal/types"

	"github.com/jackc/pgx/v5"
)

// upsertIncidentSQL merges a window into the archive. Every field can only
// grow — end_ts, the peaks, and severity (never let history get quieter, even
// if a shrinking probe fleet re-rolls a window as merely degraded) — so
// overlapping sources converge in any order as long as they agree on start_ts.
const upsertIncidentSQL = `
INSERT INTO incidents (domain, kind, start_ts, end_ts, status, peak_down, peak_total)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (domain, kind, start_ts) DO UPDATE SET
  end_ts     = GREATEST(incidents.end_ts, excluded.end_ts),
  status     = CASE WHEN 'down' IN (incidents.status, excluded.status) THEN 'down' ELSE excluded.status END,
  peak_down  = GREATEST(incidents.peak_down, excluded.peak_down),
  peak_total = GREATEST(incidents.peak_total, excluded.peak_total)`

func queueUpserts(b *pgx.Batch, domain string, kind types.Kind, incs []types.Incident) {
	for _, inc := range incs {
		b.Queue(upsertIncidentSQL, domain, string(kind), inc.StartTS, inc.EndTS, string(inc.Status), inc.PeakDown, inc.PeakTotal)
	}
}

// SyncIncidents mirrors one component's freshly walked windows into the
// archive, reconciling the whole [since, ∞) range in one transaction: windows
// are upserted (an ongoing outage keeps its start_ts while end_ts advances, so
// repeated syncs converge instead of piling up rows) and any stored window in
// the range the walk no longer produces is deleted. Callers pass the `since`
// they walked with, so anything older is never touched.
func (s *Store) SyncIncidents(ctx context.Context, domain string, kind types.Kind, since int64, incs []types.Incident) error {
	b := &pgx.Batch{}
	starts := make([]int64, 0, len(incs))
	var kept []types.Incident
	for _, inc := range incs {
		starts = append(starts, inc.StartTS)
		if inc.StartTS >= since {
			kept = append(kept, inc)
		}
	}
	queueUpserts(b, domain, kind, kept)
	b.Queue(`DELETE FROM incidents WHERE domain=$1 AND kind=$2 AND start_ts>=$3 AND start_ts <> ALL($4::bigint[])`,
		domain, string(kind), since, starts)
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error { return tx.SendBatch(ctx, b).Close() })
}

// ImportIncidents upserts windows recovered from an archived dump, without
// SyncIncidents' delete-missing step: a dump only covers a bounded slice of
// raw history, so a window it cannot see is not a window that no longer exists.
func (s *Store) ImportIncidents(ctx context.Context, domain string, kind types.Kind, incs []types.Incident) (int, error) {
	b := &pgx.Batch{}
	queueUpserts(b, domain, kind, incs)
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error { return tx.SendBatch(ctx, b).Close() })
	if err != nil {
		return 0, err
	}
	return len(incs), nil
}

// ListIncidents returns every archived window starting in [from, to), newest
// first, across all components.
func (s *Store) ListIncidents(ctx context.Context, from, to int64) ([]types.HistoryIncident, error) {
	rows, err := s.db.Query(ctx, `
SELECT domain, kind, start_ts, end_ts, status, peak_down, peak_total
FROM incidents WHERE start_ts >= $1 AND start_ts < $2
ORDER BY start_ts DESC`, from, to)
	if err != nil {
		return nil, err
	}
	out, err := collect(rows, func(r pgx.Rows, h *types.HistoryIncident) error {
		if err := r.Scan(&h.Domain, &h.Kind, &h.StartTS, &h.EndTS, &h.Status, &h.PeakDown, &h.PeakTotal); err != nil {
			return err
		}
		h.DurationS = rollup.Duration(h.StartTS, h.EndTS)
		h.Label = target.Label(h.Domain, h.Kind)
		return nil
	})
	if out == nil && err == nil {
		out = []types.HistoryIncident{}
	}
	return out, err
}

// EarliestIncidentTS returns the start of the oldest archived window, or 0.
func (s *Store) EarliestIncidentTS(ctx context.Context) (int64, error) {
	var ts *int64
	if err := s.db.QueryRow(ctx, `SELECT MIN(start_ts) FROM incidents`).Scan(&ts); err != nil || ts == nil {
		return 0, err
	}
	return *ts, nil
}
