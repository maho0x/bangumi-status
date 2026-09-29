package store

import (
	"context"
	"fmt"
	"time"

	"bangumi-status/internal/rollup"
	"bangumi-status/internal/types"

	"github.com/jackc/pgx/v5"
)

// Insert appends probe observations to the partitioned checks table.
func (s *Store) Insert(ctx context.Context, results []types.Result) error {
	if len(results) == 0 {
		return nil
	}
	_, err := s.db.CopyFrom(ctx, pgx.Identifier{"checks"},
		[]string{"ts", "probe", "region", "domain", "kind", "status", "latency_ms", "http_code", "err"},
		pgx.CopyFromSlice(len(results), func(i int) ([]any, error) {
			r := results[i]
			return []any{r.TS, r.Probe, r.Region, r.Domain, string(r.Kind), string(r.Status), r.Latency, r.HTTPCode, r.Err}, nil
		}))
	return err
}

// LatestPerProbe returns every probe's most recent observation of each
// component within the last 30 minutes, for probes that are still registered.
func (s *Store) LatestPerProbe(ctx context.Context, components []types.Component) (map[types.ComponentKey][]types.ProbeView, error) {
	out := make(map[types.ComponentKey][]types.ProbeView, len(components))
	domains := make([]string, 0, len(components))
	kinds := make([]string, 0, len(components))
	for _, c := range components {
		if _, seen := out[c.Key()]; seen || c.Domain == "" || c.Kind == "" {
			continue
		}
		out[c.Key()] = nil
		domains = append(domains, c.Domain)
		kinds = append(kinds, string(c.Kind))
	}
	if len(domains) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
WITH requested AS (
  SELECT * FROM unnest($1::text[], $2::text[]) AS r(domain, kind)
), latest AS (
  SELECT c.domain, c.kind, c.probe, MAX(c.ts) AS maxts
  FROM checks c
  JOIN requested r ON r.domain = c.domain AND r.kind = c.kind
  WHERE c.ts >= $3
  GROUP BY c.domain, c.kind, c.probe
)
SELECT c.domain, c.kind, c.probe, c.region, c.status, c.latency_ms, COALESCE(c.http_code,0), c.ts, COALESCE(c.err,'')
FROM checks c
JOIN latest m ON c.domain = m.domain AND c.kind = m.kind AND c.probe = m.probe AND c.ts = m.maxts
JOIN probes p ON p.name = c.probe
ORDER BY c.domain, c.kind, c.region, c.probe`,
		domains, kinds, time.Now().Add(-30*time.Minute).Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key types.ComponentKey
		var v types.ProbeView
		if err := rows.Scan(&key.Domain, &key.Kind, &v.Probe, &v.Region, &v.Status, &v.Latency, &v.HTTPCode, &v.TS, &v.Err); err != nil {
			return nil, err
		}
		out[key] = append(out[key], v)
	}
	return out, rows.Err()
}

// dailyBucketsSQL aggregates checks into CST days using the same per-minute
// quorum as rollup.Quorum: a minute counts as down (or degraded) when at least
// GREATEST(2, CEIL(2/3 × checks in that minute)) of its checks agree, and then
// every check in that minute counts against uptime.
const dailyBucketsSQL = `
WITH minute_stats AS (
  SELECT
    ((ts + 28800) / 86400) AS day,
    ((ts + 28800) / 60)    AS minute,
    COUNT(*)                                                    AS checks_in_min,
    COUNT(*) FILTER (WHERE status = 'down')                     AS down_in_min,
    COUNT(*) FILTER (WHERE status IN ('down','degraded'))       AS bad_in_min,
    GREATEST(2, CEIL(2.0/3.0 * COUNT(*)))                       AS quorum
  FROM checks
  WHERE domain = $1 AND kind = $2 AND ts >= $3 AND ts < $4 AND region <> '` + rollup.ExcludedRegion + `'
  GROUP BY 1, 2
)
SELECT
  day,
  SUM(checks_in_min)::int,
  COALESCE(SUM(checks_in_min) FILTER (WHERE down_in_min >= quorum), 0)::int,
  COALESCE(SUM(checks_in_min) FILTER (WHERE bad_in_min >= quorum AND down_in_min < quorum), 0)::int,
  BOOL_OR(down_in_min >= quorum),
  BOOL_OR(bad_in_min >= quorum)
FROM minute_stats
GROUP BY day`

// DailyBuckets returns the last `days` CST days (oldest first, today included).
func (s *Store) DailyBuckets(ctx context.Context, domain string, kind types.Kind, days int) ([]types.DayBucket, error) {
	now := time.Now().In(types.CST)
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, types.CST)
	return s.dailyBuckets(ctx, domain, kind, end.AddDate(0, 0, -days), days)
}

// DaySummary returns the bucket for the single CST day starting at start. It
// is the same aggregation as the 30-day strip, so the daily report and the
// status page always agree on a day's uptime.
func (s *Store) DaySummary(ctx context.Context, domain string, kind types.Kind, start time.Time) (types.DayBucket, error) {
	b, err := s.dailyBuckets(ctx, domain, kind, start, 1)
	if err != nil {
		return types.DayBucket{}, err
	}
	return b[0], nil
}

func (s *Store) dailyBuckets(ctx context.Context, domain string, kind types.Kind, start time.Time, days int) ([]types.DayBucket, error) {
	end := start.AddDate(0, 0, days)
	type row struct {
		total, down, degrade int
		hadDown, hadBad      bool
	}
	data := map[int64]row{}
	rows, err := s.db.Query(ctx, dailyBucketsSQL, domain, string(kind), start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var day int64
		var r row
		if err := rows.Scan(&day, &r.total, &r.down, &r.degrade, &r.hadDown, &r.hadBad); err != nil {
			return nil, err
		}
		data[day] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]types.DayBucket, days)
	for i := range out {
		dayStart := start.AddDate(0, 0, i)
		b := types.DayBucket{Day: dayStart.Format(time.DateOnly)}
		if r, ok := data[(dayStart.Unix()+28800)/86400]; ok {
			b.Total, b.Down, b.Degrade = r.total, r.down, r.degrade
			if r.total > 0 {
				b.Uptime = float64(r.total-r.down-r.degrade) / float64(r.total) * 100
			}
			switch {
			case r.hadDown:
				b.Status = types.StatusDown
			case r.hadBad:
				b.Status = types.StatusDegraded
			default:
				b.Status = types.StatusOK
			}
		}
		out[i] = b
	}
	return out, nil
}

// Incidents walks raw observations for (domain, kind) since `since` and
// returns the non-ok windows, oldest first. See rollup.Walker.
func (s *Store) Incidents(ctx context.Context, domain string, kind types.Kind, since time.Time) ([]types.Incident, error) {
	rows, err := s.db.Query(ctx, `
SELECT ts, probe, status FROM checks
WHERE domain=$1 AND kind=$2 AND ts>=$3 AND region <> '`+rollup.ExcludedRegion+`'
ORDER BY ts ASC`, domain, string(kind), since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	w := rollup.NewWalker()
	for rows.Next() {
		var ts int64
		var probe string
		var status types.Status
		if err := rows.Scan(&ts, &probe, &status); err != nil {
			return nil, err
		}
		w.Add(ts, probe, status)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return w.Finish(), nil
}

// EnsurePartitions creates the daily checks partitions from yesterday through
// daysAhead days from now (UTC days). Idempotent.
func (s *Store) EnsurePartitions(ctx context.Context, daysAhead int) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := -1; i <= daysAhead; i++ {
		d := today.AddDate(0, 0, i)
		if _, err := s.db.Exec(ctx, fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %q PARTITION OF checks FOR VALUES FROM (%d) TO (%d)`,
			"checks_"+d.Format("20060102"), d.Unix(), d.AddDate(0, 0, 1).Unix())); err != nil {
			return err
		}
	}
	return nil
}

// DropPartitionsBefore drops every daily checks partition that ends at or
// before cutoff and returns how many it dropped. Retention is O(1) per day and
// reclaims disk immediately, with no DELETE + VACUUM cycle.
func (s *Store) DropPartitionsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	rows, err := s.db.Query(ctx, `
SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
WHERE i.inhparent = 'checks'::regclass`)
	if err != nil {
		return 0, err
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, err
	}
	dropped := 0
	for _, name := range names {
		day, err := time.Parse("checks_20060102", name)
		if err != nil || day.AddDate(0, 0, 1).After(cutoff) {
			continue
		}
		if _, err := s.db.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %q`, name)); err != nil {
			return dropped, err
		}
		dropped++
	}
	return dropped, nil
}
