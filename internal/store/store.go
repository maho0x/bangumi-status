package store

import (
	"bangumi-status/internal/types"
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var cst = time.FixedZone("CST", 8*60*60)

// quorumFor returns the minimum number of probes that must agree to escalate status.
// Threshold is ceil(2/3 * n), with a floor of 2.
func quorumFor(n int) int {
	q := (n*2 + 2) / 3
	if q < 2 {
		return 2
	}
	return q
}

type Store struct {
	db *sql.DB
}

func Open(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS checks (
  id         SERIAL PRIMARY KEY,
  ts         BIGINT NOT NULL,
  probe      TEXT NOT NULL,
  region     TEXT NOT NULL,
  domain     TEXT NOT NULL,
  kind       TEXT NOT NULL,
  status     TEXT NOT NULL,
  latency_ms INTEGER NOT NULL,
  http_code  INTEGER,
  err        TEXT
);
CREATE INDEX IF NOT EXISTS idx_checks_ts ON checks(ts);
CREATE INDEX IF NOT EXISTS idx_checks_lookup ON checks(domain, kind, ts);
CREATE INDEX IF NOT EXISTS idx_checks_probe ON checks(probe, ts);

CREATE TABLE IF NOT EXISTS probes (
  name      TEXT PRIMARY KEY,
  region    TEXT NOT NULL,
  last_seen BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS config (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS online_counts (
  ts_min BIGINT PRIMARY KEY,
  count  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS reactions (
  emoji_id   SMALLINT NOT NULL,
  user_id    TEXT NOT NULL,
  ip         TEXT NOT NULL,
  count      INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (emoji_id, user_id)
);
CREATE INDEX IF NOT EXISTS reactions_created_at_idx ON reactions (created_at);
ALTER TABLE reactions ADD COLUMN IF NOT EXISTS count INTEGER NOT NULL DEFAULT 1;
`)
	return err
}

// ReactionCount is an aggregated count for one emoji over the active window.
type ReactionCount struct {
	EmojiID int  `json:"emoji_id"`
	Count   int  `json:"count"`
	Mine    bool `json:"mine"`
}

// ListReactions returns aggregated counts for all reactions added in the last
// 24h. If userID is non-empty, Mine is set on rows the user has reacted to.
func (s *Store) ListReactions(ctx context.Context, userID string) ([]ReactionCount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT emoji_id, SUM(count)::int, BOOL_OR(user_id = $1)
		FROM reactions
		WHERE created_at > NOW() - INTERVAL '24 hours'
		GROUP BY emoji_id
		ORDER BY emoji_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReactionCount{}
	for rows.Next() {
		var r ReactionCount
		if err := rows.Scan(&r.EmojiID, &r.Count, &r.Mine); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddReaction increments the reaction count for (emoji, user), refreshing the
// 24h window on each call so repeated clicks keep the entry active.
func (s *Store) AddReaction(ctx context.Context, emojiID int, userID, ip string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reactions (emoji_id, user_id, ip, count)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (emoji_id, user_id)
		DO UPDATE SET count = reactions.count + 1, created_at = NOW(), ip = EXCLUDED.ip`,
		emojiID, userID, ip)
	return err
}

// PurgeExpiredReactions removes rows older than the 24h active window.
// Reads filter by created_at, so this is purely housekeeping.
func (s *Store) PurgeExpiredReactions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM reactions WHERE created_at <= NOW() - INTERVAL '24 hours'`)
	return err
}

// InsertOnline records a bangumi.tv "online: N" sample. The key is the
// minute-aligned unix timestamp so multiple probes reporting in the same
// minute collapse to a single row (first writer wins).
func (s *Store) InsertOnline(ctx context.Context, ts int64, count int) error {
	if ts <= 0 || count <= 0 {
		return nil
	}
	tsMin := ts - (ts % 60)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO online_counts (ts_min, count) VALUES ($1, $2) ON CONFLICT (ts_min) DO NOTHING`,
		tsMin, count)
	return err
}

// OnlineSeries returns minute samples since `since`, oldest first.
func (s *Store) OnlineSeries(ctx context.Context, since time.Time) ([]types.OnlinePoint, error) {
	return s.OnlineSeriesBucketed(ctx, since, 0)
}

// OnlineSeriesBucketed returns samples since `since` grouped into buckets of
// `bucketSecs` seconds (0 = raw minute resolution), oldest first. For bucketed
// queries the value is the per-bucket MAX (peak), not the average — daily
// peaks are what users want to see for trend, and averaging smooths them away.
func (s *Store) OnlineSeriesBucketed(ctx context.Context, since time.Time, bucketSecs int64) ([]types.OnlinePoint, error) {
	var rows *sql.Rows
	var err error
	if bucketSecs <= 0 {
		rows, err = s.db.QueryContext(ctx,
			`SELECT ts_min, count FROM online_counts WHERE ts_min >= $1 ORDER BY ts_min ASC`,
			since.Unix())
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT (ts_min / $2) * $2 AS bucket, MAX(count)
			 FROM online_counts WHERE ts_min >= $1
			 GROUP BY bucket ORDER BY bucket ASC`,
			since.Unix(), bucketSecs)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.OnlinePoint
	for rows.Next() {
		var p types.OnlinePoint
		if err := rows.Scan(&p.TS, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetConfig(ctx context.Context, key string) (string, bool, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM config WHERE key=$1`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return v, err == nil, err
}

func (s *Store) SetConfig(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO config(key,value) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value)
	return err
}

func (s *Store) Insert(ctx context.Context, results []types.Result) error {
	if len(results) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO checks (ts, probe, region, domain, kind, status, latency_ms, http_code, err) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range results {
		if _, err := stmt.ExecContext(ctx, r.TS, r.Probe, r.Region, r.Domain, string(r.Kind), string(r.Status), r.Latency, r.HTTPCode, r.Err); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) UpsertProbe(ctx context.Context, name, region string, lastSeen int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO probes(name, region, last_seen) VALUES($1,$2,$3)
		ON CONFLICT(name) DO UPDATE SET region=excluded.region, last_seen=excluded.last_seen`, name, region, lastSeen)
	return err
}

func (s *Store) Probes(ctx context.Context) ([]types.Probe, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, region, last_seen FROM probes ORDER BY region, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.Probe
	now := time.Now().Unix()
	for rows.Next() {
		var p types.Probe
		if err := rows.Scan(&p.Name, &p.Region, &p.LastSeen); err != nil {
			return nil, err
		}
		p.Online = now-p.LastSeen < 180 // online if heartbeat in last 3min
		out = append(out, p)
	}
	return out, rows.Err()
}

// DaySummary returns a single-day bucket for (domain, kind) over an arbitrary
// time range. The caller is responsible for aligning start/end to a calendar day.
func (s *Store) DaySummary(ctx context.Context, domain string, kind types.Kind, start, end time.Time) (types.DayBucket, error) {
	var bucket types.DayBucket
	bucket.Day = start.Format("2006-01-02")

	row := s.db.QueryRowContext(ctx, `
SELECT
  COALESCE(SUM(CASE WHEN sub.effective_status = 'ok' THEN 1 ELSE 0 END), 0) AS ok_count,
  COALESCE(SUM(CASE WHEN sub.effective_status = 'degraded' THEN 1 ELSE 0 END), 0) AS degrade_count,
  COALESCE(SUM(CASE WHEN sub.effective_status = 'down' THEN 1 ELSE 0 END), 0) AS down_count,
  COUNT(*) AS total
FROM (
  SELECT
    CASE
      WHEN c.status = 'down' AND m.down_in_min >= GREATEST(2, CEIL(2.0/3.0 * m.checks_in_min)) THEN 'down'
      WHEN c.status IN ('down', 'degraded') AND m.bad_in_min >= GREATEST(2, CEIL(2.0/3.0 * m.checks_in_min)) THEN 'degraded'
      ELSE 'ok'
    END AS effective_status
  FROM checks c
  INNER JOIN (
    SELECT ((ts + 28800) / 60) AS minute,
      COUNT(*) AS checks_in_min,
      SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) AS down_in_min,
      SUM(CASE WHEN status IN ('down', 'degraded') THEN 1 ELSE 0 END) AS bad_in_min
    FROM checks
    WHERE domain = $1 AND kind = $2 AND ts >= $3 AND ts < $4
    GROUP BY ((ts + 28800) / 60)
  ) m ON ((c.ts + 28800) / 60) = m.minute
  WHERE c.domain = $5 AND c.kind = $6 AND c.ts >= $7 AND c.ts < $8
) sub`, domain, string(kind), start.Unix(), end.Unix(), domain, string(kind), start.Unix(), end.Unix())

	var okCount, degradeCount, downCount, total int
	if err := row.Scan(&okCount, &degradeCount, &downCount, &total); err != nil {
		return bucket, err
	}

	row2 := s.db.QueryRowContext(ctx, `
SELECT
  COALESCE(MAX(CASE WHEN sub.down_in_min >= GREATEST(2, CEIL(2.0/3.0 * sub.checks_in_min)) THEN 1 ELSE 0 END), 0) AS had_down,
  COALESCE(MAX(CASE WHEN sub.bad_in_min  >= GREATEST(2, CEIL(2.0/3.0 * sub.checks_in_min)) THEN 1 ELSE 0 END), 0) AS had_bad
FROM (
  SELECT ((ts + 28800) / 60) AS minute,
    COUNT(*) AS checks_in_min,
    SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) AS down_in_min,
    SUM(CASE WHEN status IN ('down', 'degraded') THEN 1 ELSE 0 END) AS bad_in_min
  FROM checks
  WHERE domain = $1 AND kind = $2 AND ts >= $3 AND ts < $4
  GROUP BY ((ts + 28800) / 60)
) sub`, domain, string(kind), start.Unix(), end.Unix())

	var hadDown, hadBad int
	if err := row2.Scan(&hadDown, &hadBad); err != nil {
		return bucket, err
	}

	bucket.Total = total
	bucket.Down = downCount
	bucket.Degrade = degradeCount
	if total > 0 {
		bucket.Uptime = float64(okCount) / float64(total) * 100
	}
	bucket.Status = types.StatusOK
	if hadDown > 0 {
		bucket.Status = types.StatusDown
	} else if hadBad > 0 {
		bucket.Status = types.StatusDegraded
	}
	return bucket, nil
}

// DailyBuckets returns N daily buckets (oldest first) for (domain, kind).
func (s *Store) DailyBuckets(ctx context.Context, domain string, kind types.Kind, days int) ([]types.DayBucket, error) {
	now := time.Now().In(cst)
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, cst)
	start := end.Add(-time.Duration(days) * 24 * time.Hour)

	// Single-pass CTE: aggregate per minute, then roll up per day.
	// A minute counts as down/degraded only if ≥ceil(2/3 * active_probes) agree.
	rows, err := s.db.QueryContext(ctx, `
WITH minute_stats AS (
  SELECT
    ((ts + 28800) / 86400) AS day,
    ((ts + 28800) / 60)    AS minute,
    COUNT(*)                                                          AS checks_in_min,
    SUM(CASE WHEN status = 'down'                    THEN 1 ELSE 0 END) AS down_in_min,
    SUM(CASE WHEN status IN ('down','degraded')       THEN 1 ELSE 0 END) AS bad_in_min
  FROM checks
  WHERE domain = $1 AND kind = $2 AND ts >= $3 AND ts < $4
  GROUP BY ((ts + 28800) / 86400), ((ts + 28800) / 60)
)
SELECT
  day,
  SUM(checks_in_min)                                                                                                                              AS total,
  SUM(CASE WHEN down_in_min >= GREATEST(2, CEIL(2.0/3.0 * checks_in_min)) THEN checks_in_min ELSE 0 END)                                         AS down_count,
  SUM(CASE WHEN bad_in_min  >= GREATEST(2, CEIL(2.0/3.0 * checks_in_min)) AND down_in_min < GREATEST(2, CEIL(2.0/3.0 * checks_in_min)) THEN checks_in_min ELSE 0 END) AS degrade_count,
  MAX(CASE WHEN down_in_min >= GREATEST(2, CEIL(2.0/3.0 * checks_in_min)) THEN 1 ELSE 0 END)                                                     AS had_down,
  MAX(CASE WHEN bad_in_min  >= GREATEST(2, CEIL(2.0/3.0 * checks_in_min)) THEN 1 ELSE 0 END)                                                     AS had_bad
FROM minute_stats
GROUP BY day`,
		domain, string(kind), start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type dayData struct {
		total, down, degrade, hadDown, hadBad int
	}
	m := map[int64]dayData{}
	for rows.Next() {
		var day int64
		var d dayData
		if err := rows.Scan(&day, &d.total, &d.down, &d.degrade, &d.hadDown, &d.hadBad); err != nil {
			return nil, err
		}
		m[day] = d
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]types.DayBucket, days)
	for i := 0; i < days; i++ {
		dayStart := start.Add(time.Duration(i) * 24 * time.Hour)
		dayIdx := (dayStart.Unix() + 28800) / 86400
		bucket := types.DayBucket{Day: dayStart.Format("2006-01-02")}
		if d, ok := m[dayIdx]; ok {
			bucket.Total = d.total
			bucket.Down = d.down
			bucket.Degrade = d.degrade
			if d.total > 0 {
				bucket.Uptime = float64(d.total-d.down-d.degrade) / float64(d.total) * 100
			}
			bucket.Status = types.StatusOK
			if d.hadDown > 0 {
				bucket.Status = types.StatusDown
			} else if d.hadBad > 0 {
				bucket.Status = types.StatusDegraded
			}
		}
		out[i] = bucket
	}
	return out, nil
}

// OverlayIncidentsOnBuckets marks each day touched by an incident with at least
// the incident's severity. DailyBuckets uses strict per-minute quorum and can
// miss brief outages whose bad checks straddle a minute boundary (probes are
// staggered within each minute). Incidents uses a 180s rolling state window
// and is the authoritative source of "did something bad happen on this day".
func OverlayIncidentsOnBuckets(buckets []types.DayBucket, incidents []types.Incident) {
	if len(buckets) == 0 || len(incidents) == 0 {
		return
	}
	rank := map[types.Status]int{
		types.StatusOK:       0,
		types.StatusDegraded: 1,
		types.StatusDown:     2,
	}
	idx := make(map[string]int, len(buckets))
	for i, b := range buckets {
		idx[b.Day] = i
	}
	for _, inc := range incidents {
		startDay := time.Unix(inc.StartTS, 0).In(cst)
		endDay := time.Unix(inc.EndTS, 0).In(cst)
		d := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, cst)
		stop := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, cst)
		for !d.After(stop) {
			if i, ok := idx[d.Format("2006-01-02")]; ok {
				if rank[inc.Status] > rank[buckets[i].Status] {
					buckets[i].Status = inc.Status
				}
			}
			d = d.AddDate(0, 0, 1)
		}
	}
}

// LatestPerProbe returns the most recent result per probe for (domain, kind).
func (s *Store) LatestPerProbe(ctx context.Context, domain string, kind types.Kind) ([]types.ProbeView, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.probe, c.region, c.status, c.latency_ms, COALESCE(c.http_code,0), c.ts, COALESCE(c.err,'')
FROM checks c
INNER JOIN (
  SELECT probe, MAX(ts) AS maxts FROM checks
  WHERE domain=$1 AND kind=$2 AND ts >= $3
  GROUP BY probe
) m ON c.probe = m.probe AND c.ts = m.maxts
INNER JOIN probes p ON p.name = c.probe
WHERE c.domain=$4 AND c.kind=$5
ORDER BY c.region, c.probe`,
		domain, string(kind), time.Now().Add(-30*time.Minute).Unix(), domain, string(kind))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.ProbeView
	for rows.Next() {
		var v types.ProbeView
		var st string
		if err := rows.Scan(&v.Probe, &v.Region, &st, &v.Latency, &v.HTTPCode, &v.TS, &v.Err); err != nil {
			return nil, err
		}
		v.Status = types.Status(st)
		out = append(out, v)
	}
	return out, rows.Err()
}

// RollupStatus computes the current status for (domain, kind) from already-fetched views.
// Call LatestPerProbe first, then pass the result here to avoid a redundant DB round-trip.
func RollupStatus(views []types.ProbeView) (types.Status, int64) {
	if len(views) == 0 {
		return types.StatusOK, 0
	}
	var ok, deg, down int
	var latest int64
	for _, v := range views {
		if v.TS > latest {
			latest = v.TS
		}
		switch v.Status {
		case types.StatusOK:
			ok++
		case types.StatusDegraded:
			deg++
		case types.StatusDown:
			down++
		}
	}
	quorum := quorumFor(ok + deg + down)
	if down >= quorum {
		return types.StatusDown, latest
	}
	if down+deg >= quorum {
		return types.StatusDegraded, latest
	}
	return types.StatusOK, latest
}

// Incidents returns contiguous non-ok windows for (domain, kind) within [since, now].
// It walks raw check events, maintains per-probe (status, ts) state, and recomputes
// the rolled-up status after each event using the same quorum rule as RollupStatus.
// Transitions ok→non-ok open a window; non-ok→ok close it. Oldest first.
func (s *Store) Incidents(ctx context.Context, domain string, kind types.Kind, since time.Time) ([]types.Incident, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT ts, probe, status FROM checks
WHERE domain=$1 AND kind=$2 AND ts>=$3
ORDER BY ts ASC`, domain, string(kind), since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pState struct {
		status types.Status
		ts     int64
	}
	probes := map[string]pState{}

	const staleSec = int64(180)

	statusRank := map[types.Status]int{types.StatusOK: 0, types.StatusDegraded: 1, types.StatusDown: 2}

	rollup := func(now int64) (status types.Status, active, bad int) {
		var ok, deg, down int
		for _, p := range probes {
			if now-p.ts > staleSec {
				continue
			}
			active++
			switch p.status {
			case types.StatusOK:
				ok++
			case types.StatusDegraded:
				deg++
			case types.StatusDown:
				down++
			}
		}
		bad = down + deg
		quorum := quorumFor(active)
		if down >= quorum {
			return types.StatusDown, active, bad
		}
		if down+deg >= quorum {
			return types.StatusDegraded, active, bad
		}
		return types.StatusOK, active, bad
	}

	var out []types.Incident
	var cur *types.Incident

	closeWindow := func() {
		if cur == nil {
			return
		}
		cur.DurationS = int(cur.EndTS - cur.StartTS)
		if cur.DurationS < 60 {
			cur.DurationS = 60
		}
		out = append(out, *cur)
		cur = nil
	}

	for rows.Next() {
		var ts int64
		var probe, status string
		if err := rows.Scan(&ts, &probe, &status); err != nil {
			return nil, err
		}
		probes[probe] = pState{status: types.Status(status), ts: ts}
		rolled, active, bad := rollup(ts)

		if rolled != types.StatusOK {
			if cur == nil {
				cur = &types.Incident{
					StartTS:   ts,
					EndTS:     ts,
					Status:    rolled,
					PeakDown:  bad,
					PeakTotal: active,
				}
			} else {
				cur.EndTS = ts
				if statusRank[rolled] > statusRank[cur.Status] {
					cur.Status = rolled
				}
				if bad > cur.PeakDown {
					cur.PeakDown = bad
					cur.PeakTotal = active
				}
			}
		} else {
			closeWindow()
		}
	}
	closeWindow()
	return out, rows.Err()
}

// PurgeOlderThan deletes check rows older than the cutoff.
// online_counts is kept indefinitely for historical charting.
func (s *Store) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM checks WHERE ts < $1`, cutoff.Unix())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// Stats returns simple store stats for debugging.
func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	out := map[string]any{}
	var n int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM checks`).Scan(&n); err != nil {
		return nil, err
	}
	out["check_rows"] = n
	var earliest, latest sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `SELECT MIN(ts), MAX(ts) FROM checks`).Scan(&earliest, &latest)
	if earliest.Valid {
		out["earliest"] = earliest.Int64
		out["latest"] = latest.Int64
	}
	return out, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func must(err error) {
	if err != nil {
		panic(fmt.Errorf("store: %w", err))
	}
}
