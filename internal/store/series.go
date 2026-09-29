package store

import (
	"context"
	"time"

	"bangumi-status/internal/types"

	"github.com/jackc/pgx/v5"
)

// InsertOnline records a bangumi.tv "online: N" sample at minute resolution.
// The aggregator only calls this when the value changes — the site-wide
// counter is server-cached for ~10 min, so each row is a genuine step and the
// chart carries the last value forward in between.
func (s *Store) InsertOnline(ctx context.Context, ts int64, count int) error {
	if ts <= 0 || count <= 0 {
		return nil
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO online_counts (ts_min, count) VALUES ($1, $2)
		 ON CONFLICT (ts_min) DO UPDATE SET count = EXCLUDED.count`, ts-ts%60, count)
	return err
}

// LatestOnline returns the most recent stored online count (0 if none).
func (s *Store) LatestOnline(ctx context.Context) (int, error) {
	var c int
	err := s.db.QueryRow(ctx, `SELECT count FROM online_counts ORDER BY ts_min DESC LIMIT 1`).Scan(&c)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	return c, err
}

// InsertTrafficSample folds a concurrent-viewer count into its minute's peak.
// Zero is a valid sample (nobody watching).
func (s *Store) InsertTrafficSample(ctx context.Context, ts int64, viewers int) error {
	if ts <= 0 || viewers < 0 {
		return nil
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO traffic_samples (ts_min, viewers) VALUES ($1, $2)
		 ON CONFLICT (ts_min) DO UPDATE SET viewers = GREATEST(traffic_samples.viewers, EXCLUDED.viewers)`,
		ts-ts%60, viewers)
	return err
}

// Series names a (ts_min, value) table.
type Series struct{ table, column string }

var (
	// Online is bangumi.tv's online counter. Its cache-rebuild artifacts (a
	// reading ≈ old+new ≈ 2× the real count) are filtered at read time.
	Online = Series{"online_counts", "count"}
	// Traffic is status-page concurrent viewers.
	Traffic = Series{"traffic_samples", "viewers"}
)

// spikeFiltered wraps the online table in a subquery dropping points greater
// than 1.5× BOTH temporal neighbours. The both-sided test only fires on
// isolated peaks, so genuine ramps and oscillation are untouched, and filtering
// at read time keeps the raw data intact.
const spikeFiltered = `(SELECT ts_min, count FROM (
  SELECT ts_min, count,
         LAG(count)  OVER (ORDER BY ts_min) AS prev,
         LEAD(count) OVER (ORDER BY ts_min) AS next
  FROM online_counts WHERE ts_min >= $1) z
WHERE prev IS NULL OR next IS NULL
   OR NOT (count::numeric > prev * 1.5 AND count::numeric > next * 1.5)) f`

// Points returns the series since `since`, oldest first: raw minute samples
// when bucketSecs is 0, otherwise one point per bucket whose Count is the
// average (the trend) and Low/Peak the bucket's range, so one spiky minute
// never lifts a whole bucket but still shows in the band.
func (s *Store) Points(ctx context.Context, series Series, since time.Time, bucketSecs int64) ([]types.OnlinePoint, error) {
	// table and column are package constants, never user input.
	src, col := series.table, series.column
	if series == Online {
		src = spikeFiltered
	}
	var rows pgx.Rows
	var err error
	if bucketSecs <= 0 {
		rows, err = s.db.Query(ctx,
			`SELECT ts_min, `+col+`, 0, 0 FROM `+src+` WHERE ts_min >= $1 ORDER BY ts_min`, since.Unix())
	} else {
		rows, err = s.db.Query(ctx,
			`SELECT (ts_min / $2) * $2 AS bucket, ROUND(AVG(`+col+`))::int, MIN(`+col+`), MAX(`+col+`)
			 FROM `+src+` WHERE ts_min >= $1 GROUP BY bucket ORDER BY bucket`, since.Unix(), bucketSecs)
	}
	if err != nil {
		return nil, err
	}
	return collect(rows, func(r pgx.Rows, p *types.OnlinePoint) error {
		return r.Scan(&p.TS, &p.Count, &p.Low, &p.Peak)
	})
}
