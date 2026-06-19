package store

import (
	"bangumi-status/internal/types"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var cst = time.FixedZone("CST", 8*60*60)

const probeListMaxOffline = 24 * time.Hour
const statusExcludedRegion = "cn"

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
	db.SetMaxOpenConns(6)
	db.SetMaxIdleConns(6)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func migrate(db *sql.DB) error {
	// checks is partitioned by UTC day on ts (unix seconds). Daily partitions
	// keep retention cheap (DROP partition instead of DELETE + VACUUM) and
	// prevent the index bloat that single-table DELETE accumulates.
	// EnsureChecksPartitions creates the day partitions on a schedule.
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS checks (
  ts         BIGINT NOT NULL,
  probe      TEXT NOT NULL,
  region     TEXT NOT NULL,
  domain     TEXT NOT NULL,
  kind       TEXT NOT NULL,
  status     TEXT NOT NULL,
  latency_ms INTEGER NOT NULL,
  http_code  INTEGER,
  err        TEXT
) PARTITION BY RANGE (ts);
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
-- is_canonical: TRUE if the sample came from ONLINE_SOURCE_PROBE. Non-canonical
-- samples are used as fallback coverage when the canonical probe can't reach
-- bangumi.tv (e.g. during an outage); canonical samples take precedence.
-- Existing rows pre-date this column and were all written by the canonical
-- probe under the prior single-source rule, so default to TRUE.
ALTER TABLE online_counts ADD COLUMN IF NOT EXISTS is_canonical BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS reactions (
  emoji_id   SMALLINT NOT NULL,
  user_id    TEXT NOT NULL,
  ip         TEXT NOT NULL,
  count      INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (emoji_id, user_id)
);
CREATE INDEX IF NOT EXISTS reactions_created_at_idx ON reactions (created_at);
CREATE INDEX IF NOT EXISTS reactions_user_created_at_idx ON reactions (user_id, created_at);
ALTER TABLE reactions ADD COLUMN IF NOT EXISTS count INTEGER NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS wiki_stats_daily (
  day              DATE PRIMARY KEY,
  title            TEXT NOT NULL,
  ts               BIGINT NOT NULL,
  register_total   INTEGER NOT NULL DEFAULT 0,
  collection_total INTEGER NOT NULL DEFAULT 0,
  topic_total      INTEGER NOT NULL DEFAULT 0,
  reply_total      INTEGER NOT NULL DEFAULT 0,
  collection_1     INTEGER NOT NULL DEFAULT 0,
  collection_2     INTEGER NOT NULL DEFAULT 0,
  collection_3     INTEGER NOT NULL DEFAULT 0,
  collection_4     INTEGER NOT NULL DEFAULT 0,
  collection_5     INTEGER NOT NULL DEFAULT 0,
  topic_1          INTEGER NOT NULL DEFAULT 0,
  topic_2          INTEGER NOT NULL DEFAULT 0,
  topic_7          INTEGER NOT NULL DEFAULT 0,
  reply_1          INTEGER NOT NULL DEFAULT 0,
  reply_2          INTEGER NOT NULL DEFAULT 0,
  reply_3          INTEGER NOT NULL DEFAULT 0,
  reply_4          INTEGER NOT NULL DEFAULT 0,
  reply_5          INTEGER NOT NULL DEFAULT 0,
  reply_6          INTEGER NOT NULL DEFAULT 0,
  reply_7          INTEGER NOT NULL DEFAULT 0,
  reply_8          INTEGER NOT NULL DEFAULT 0,
  raw              JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at       BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS wiki_stats_daily_ts_idx ON wiki_stats_daily (ts);

CREATE TABLE IF NOT EXISTS wiki_stats_snapshots (
  id          BIGSERIAL PRIMARY KEY,
  scraped_at  BIGINT NOT NULL,
  source_date DATE,
  row_count   INTEGER NOT NULL,
  chart_sets  JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS wiki_stats_snapshots_scraped_at_idx ON wiki_stats_snapshots (scraped_at);
`)
	return err
}

// ReactionCount is an aggregated count for one emoji over the active window.
type ReactionCount struct {
	EmojiID int  `json:"emoji_id"`
	Count   int  `json:"count"`
	Mine    bool `json:"mine"`
}

// ListReactionCounts returns aggregated counts for all reactions added in the
// last 24h. The caller can cache this because it is not user-specific.
func (s *Store) ListReactionCounts(ctx context.Context) ([]ReactionCount, error) {
	rows, err := s.db.QueryContext(ctx, `
			SELECT emoji_id, SUM(count)::int
			FROM reactions
			WHERE created_at > NOW() - INTERVAL '24 hours'
			GROUP BY emoji_id
			ORDER BY emoji_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReactionCount{}
	for rows.Next() {
		var r ReactionCount
		if err := rows.Scan(&r.EmojiID, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UserReactionEmojiIDs returns the active emoji IDs a user has reacted to.
func (s *Store) UserReactionEmojiIDs(ctx context.Context, userID string) (map[int]bool, error) {
	out := map[int]bool{}
	if userID == "" {
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `
			SELECT emoji_id
			FROM reactions
			WHERE user_id = $1 AND created_at > NOW() - INTERVAL '24 hours'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var emojiID int
		if err := rows.Scan(&emojiID); err != nil {
			return nil, err
		}
		out[emojiID] = true
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
// minute collapse to a single row.
//
// Conflict policy: canonical samples beat non-canonical samples; within the
// same priority, first writer wins. This lets a non-canonical probe fill in
// coverage when the canonical probe can't reach bangumi.tv, while still
// preferring canonical readings (which avoid per-probe systematic offsets)
// whenever they exist for that minute.
func (s *Store) InsertOnline(ctx context.Context, ts int64, count int, isCanonical bool) error {
	if ts <= 0 || count <= 0 {
		return nil
	}
	tsMin := ts - (ts % 60)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO online_counts (ts_min, count, is_canonical) VALUES ($1, $2, $3)
		 ON CONFLICT (ts_min) DO UPDATE
		    SET count = EXCLUDED.count, is_canonical = TRUE
		  WHERE online_counts.is_canonical = FALSE AND EXCLUDED.is_canonical = TRUE`,
		tsMin, count, isCanonical)
	return err
}

// OnlineSeries returns minute samples since `since`, oldest first.
func (s *Store) OnlineSeries(ctx context.Context, since time.Time) ([]types.OnlinePoint, error) {
	return s.OnlineSeriesBucketed(ctx, since, 0)
}

// OnlineSeriesBucketed returns samples since `since` grouped into buckets of
// `bucketSecs` seconds (0 = raw minute resolution), oldest first.
//
// For raw queries each point is a single minute sample. For bucketed queries
// Count is the per-bucket average — the trend — while Low/Peak carry the
// bucket's min/max so the chart can draw a range band. Averaging (rather than
// the old per-bucket MAX) keeps a single spiky minute from lifting the whole
// bucket; the band still surfaces that the spike happened.
func (s *Store) OnlineSeriesBucketed(ctx context.Context, since time.Time, bucketSecs int64) ([]types.OnlinePoint, error) {
	if bucketSecs <= 0 {
		rows, err := s.db.QueryContext(ctx,
			`SELECT ts_min, count FROM online_counts WHERE ts_min >= $1 ORDER BY ts_min ASC`,
			since.Unix())
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

	rows, err := s.db.QueryContext(ctx,
		`SELECT (ts_min / $2) * $2 AS bucket,
		        CAST(ROUND(AVG(count)) AS INTEGER) AS avg_count,
		        MIN(count) AS low,
		        MAX(count) AS peak
		 FROM online_counts WHERE ts_min >= $1
		 GROUP BY bucket ORDER BY bucket ASC`,
		since.Unix(), bucketSecs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.OnlinePoint
	for rows.Next() {
		var p types.OnlinePoint
		if err := rows.Scan(&p.TS, &p.Count, &p.Low, &p.Peak); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertWikiStats stores every daily row exposed by chii.in/wiki/stats and
// keeps the raw CHART_SETS payload as an immutable scrape snapshot.
func (s *Store) UpsertWikiStats(ctx context.Context, points []types.WikiStatsPoint, chartSetsJSON []byte) (int, string, error) {
	if len(points) == 0 {
		return 0, "", nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO wiki_stats_daily (
  day, title, ts, register_total, collection_total, topic_total, reply_total,
  collection_1, collection_2, collection_3, collection_4, collection_5,
  topic_1, topic_2, topic_7,
  reply_1, reply_2, reply_3, reply_4, reply_5, reply_6, reply_7, reply_8,
  raw, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11, $12,
  $13, $14, $15,
  $16, $17, $18, $19, $20, $21, $22, $23,
  $24, $25
)
ON CONFLICT (day) DO UPDATE SET
  title = EXCLUDED.title,
  ts = EXCLUDED.ts,
  register_total = EXCLUDED.register_total,
  collection_total = EXCLUDED.collection_total,
  topic_total = EXCLUDED.topic_total,
  reply_total = EXCLUDED.reply_total,
  collection_1 = EXCLUDED.collection_1,
  collection_2 = EXCLUDED.collection_2,
  collection_3 = EXCLUDED.collection_3,
  collection_4 = EXCLUDED.collection_4,
  collection_5 = EXCLUDED.collection_5,
  topic_1 = EXCLUDED.topic_1,
  topic_2 = EXCLUDED.topic_2,
  topic_7 = EXCLUDED.topic_7,
  reply_1 = EXCLUDED.reply_1,
  reply_2 = EXCLUDED.reply_2,
  reply_3 = EXCLUDED.reply_3,
  reply_4 = EXCLUDED.reply_4,
  reply_5 = EXCLUDED.reply_5,
  reply_6 = EXCLUDED.reply_6,
  reply_7 = EXCLUDED.reply_7,
  reply_8 = EXCLUDED.reply_8,
  raw = EXCLUDED.raw,
  updated_at = EXCLUDED.updated_at`)
	if err != nil {
		return 0, "", err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	accepted := 0
	latest := ""
	for _, p := range points {
		if _, err := time.Parse("2006-01-02", p.Date); err != nil {
			return accepted, latest, fmt.Errorf("invalid wiki stats date %q: %w", p.Date, err)
		}
		raw, err := json.Marshal(p)
		if err != nil {
			return accepted, latest, err
		}
		if _, err := stmt.ExecContext(ctx,
			p.Date, p.Title, p.Timestamp, p.RegisterTotal, p.CollectionTotal, p.TopicTotal, p.ReplyTotal,
			p.Collection1, p.Collection2, p.Collection3, p.Collection4, p.Collection5,
			p.Topic1, p.Topic2, p.Topic7,
			p.Reply1, p.Reply2, p.Reply3, p.Reply4, p.Reply5, p.Reply6, p.Reply7, p.Reply8,
			string(raw), now,
		); err != nil {
			return accepted, latest, err
		}
		accepted++
		if p.Date > latest {
			latest = p.Date
		}
	}

	if accepted > 0 && len(chartSetsJSON) > 0 {
		var sourceDate any
		if latest != "" {
			sourceDate = latest
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO wiki_stats_snapshots (scraped_at, source_date, row_count, chart_sets)
			 VALUES ($1, $2, $3, $4)`,
			now, sourceDate, accepted, string(chartSetsJSON)); err != nil {
			return accepted, latest, err
		}
	}

	if err := tx.Commit(); err != nil {
		return accepted, latest, err
	}
	return accepted, latest, nil
}

func (s *Store) WikiStats(ctx context.Context) ([]types.WikiStatsPoint, int64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT day, title, ts, register_total, collection_total, topic_total, reply_total,
       collection_1, collection_2, collection_3, collection_4, collection_5,
       topic_1, topic_2, topic_7,
       reply_1, reply_2, reply_3, reply_4, reply_5, reply_6, reply_7, reply_8
FROM wiki_stats_daily
ORDER BY day ASC`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []types.WikiStatsPoint
	for rows.Next() {
		var day time.Time
		var p types.WikiStatsPoint
		if err := rows.Scan(
			&day, &p.Title, &p.Timestamp, &p.RegisterTotal, &p.CollectionTotal, &p.TopicTotal, &p.ReplyTotal,
			&p.Collection1, &p.Collection2, &p.Collection3, &p.Collection4, &p.Collection5,
			&p.Topic1, &p.Topic2, &p.Topic7,
			&p.Reply1, &p.Reply2, &p.Reply3, &p.Reply4, &p.Reply5, &p.Reply6, &p.Reply7, &p.Reply8,
		); err != nil {
			return nil, 0, err
		}
		p.Date = day.Format("2006-01-02")
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var scrapedAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(scraped_at) FROM wiki_stats_snapshots`).Scan(&scrapedAt); err != nil {
		return nil, 0, err
	}
	if scrapedAt.Valid {
		return out, scrapedAt.Int64, nil
	}
	return out, 0, nil
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
	raw, err := json.Marshal(results)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO checks (ts, probe, region, domain, kind, status, latency_ms, http_code, err)
SELECT ts, probe, region, domain, kind, status, latency_ms, COALESCE(http_code, 0), COALESCE(err, '')
FROM jsonb_to_recordset($1::jsonb) AS r(
	ts bigint,
	probe text,
	region text,
	domain text,
	kind text,
	status text,
	latency_ms integer,
	http_code integer,
	err text
)`, string(raw))
	return err
}

func (s *Store) UpsertProbe(ctx context.Context, name, region string, lastSeen int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO probes(name, region, last_seen) VALUES($1,$2,$3)
		ON CONFLICT(name) DO UPDATE SET region=excluded.region, last_seen=excluded.last_seen`, name, region, lastSeen)
	return err
}

func (s *Store) Probes(ctx context.Context) ([]types.Probe, error) {
	cutoff := time.Now().Add(-probeListMaxOffline).Unix()
	rows, err := s.db.QueryContext(ctx, `
SELECT name, region, last_seen
FROM probes
WHERE last_seen >= $1
ORDER BY region, name`, cutoff)
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

// ComponentKey identifies one monitored (domain, kind) pair in batch query maps.
type ComponentKey struct {
	Domain string
	Kind   types.Kind
}

type componentRequest struct {
	Domain string `json:"domain"`
	Kind   string `json:"kind"`
}

func componentRequestJSON(components []types.Component) ([]byte, []ComponentKey, error) {
	seen := make(map[ComponentKey]bool, len(components))
	reqs := make([]componentRequest, 0, len(components))
	keys := make([]ComponentKey, 0, len(components))
	for _, c := range components {
		if c.Domain == "" || c.Kind == "" {
			continue
		}
		key := ComponentKey{Domain: c.Domain, Kind: c.Kind}
		if seen[key] {
			continue
		}
		seen[key] = true
		reqs = append(reqs, componentRequest{Domain: c.Domain, Kind: string(c.Kind)})
		keys = append(keys, key)
	}
	raw, err := json.Marshal(reqs)
	return raw, keys, err
}

func dailyBucketWindow(days int) (time.Time, time.Time) {
	now := time.Now().In(cst)
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, cst)
	start := end.Add(-time.Duration(days) * 24 * time.Hour)
	return start, end
}

type dayBucketData struct {
	total, down, degrade, hadDown, hadBad int
}

func fillDailyBuckets(start time.Time, days int, data map[int64]dayBucketData) []types.DayBucket {
	out := make([]types.DayBucket, days)
	for i := 0; i < days; i++ {
		dayStart := start.Add(time.Duration(i) * 24 * time.Hour)
		dayIdx := (dayStart.Unix() + 28800) / 86400
		bucket := types.DayBucket{Day: dayStart.Format("2006-01-02")}
		if d, ok := data[dayIdx]; ok {
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
	return out
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
    WHERE domain = $1 AND kind = $2 AND ts >= $3 AND ts < $4 AND region <> 'cn'
    GROUP BY ((ts + 28800) / 60)
  ) m ON ((c.ts + 28800) / 60) = m.minute
  WHERE c.domain = $5 AND c.kind = $6 AND c.ts >= $7 AND c.ts < $8 AND c.region <> 'cn'
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
  WHERE domain = $1 AND kind = $2 AND ts >= $3 AND ts < $4 AND region <> 'cn'
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
	start, end := dailyBucketWindow(days)
	rows, err := s.db.QueryContext(ctx, `
WITH minute_stats AS (
  SELECT
    ((ts + 28800) / 86400) AS day,
    ((ts + 28800) / 60)    AS minute,
    COUNT(*)                                                            AS checks_in_min,
    SUM(CASE WHEN status = 'down'                    THEN 1 ELSE 0 END) AS down_in_min,
    SUM(CASE WHEN status IN ('down','degraded')       THEN 1 ELSE 0 END) AS bad_in_min,
    GREATEST(2, CEIL(2.0/3.0 * COUNT(*)))                               AS quorum
  FROM checks
  WHERE domain = $1 AND kind = $2 AND ts >= $3 AND ts < $4 AND region <> 'cn'
  GROUP BY ((ts + 28800) / 86400), ((ts + 28800) / 60)
)
SELECT
  day,
  SUM(checks_in_min)                                                                         AS total,
  SUM(CASE WHEN down_in_min >= quorum THEN checks_in_min ELSE 0 END)                          AS down_count,
  SUM(CASE WHEN bad_in_min  >= quorum AND down_in_min < quorum THEN checks_in_min ELSE 0 END) AS degrade_count,
  MAX(CASE WHEN down_in_min >= quorum THEN 1 ELSE 0 END)                                      AS had_down,
  MAX(CASE WHEN bad_in_min  >= quorum THEN 1 ELSE 0 END)                                      AS had_bad
FROM minute_stats
GROUP BY day`,
		domain, string(kind), start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := map[int64]dayBucketData{}
	for rows.Next() {
		var day int64
		var d dayBucketData
		if err := rows.Scan(&day, &d.total, &d.down, &d.degrade, &d.hadDown, &d.hadBad); err != nil {
			return nil, err
		}
		data[day] = d
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return fillDailyBuckets(start, days, data), nil
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

// LatestPerProbeBatch returns the most recent result per probe for every
// requested component using one grouped lookup over the recent checks window.
func (s *Store) LatestPerProbeBatch(ctx context.Context, components []types.Component) (map[ComponentKey][]types.ProbeView, error) {
	raw, keys, err := componentRequestJSON(components)
	if err != nil {
		return nil, err
	}
	out := make(map[ComponentKey][]types.ProbeView, len(keys))
	for _, key := range keys {
		out[key] = nil
	}
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `
WITH requested AS (
  SELECT domain, kind
  FROM jsonb_to_recordset($1::jsonb) AS r(domain text, kind text)
), latest AS (
  SELECT c.domain, c.kind, c.probe, MAX(c.ts) AS maxts
  FROM checks c
  INNER JOIN requested r ON r.domain = c.domain AND r.kind = c.kind
  WHERE c.ts >= $2
  GROUP BY c.domain, c.kind, c.probe
)
SELECT c.domain, c.kind, c.probe, c.region, c.status, c.latency_ms, COALESCE(c.http_code,0), c.ts, COALESCE(c.err,'')
FROM checks c
INNER JOIN latest m ON c.domain = m.domain AND c.kind = m.kind AND c.probe = m.probe AND c.ts = m.maxts
INNER JOIN probes p ON p.name = c.probe
ORDER BY c.domain, c.kind, c.region, c.probe`,
		string(raw), time.Now().Add(-30*time.Minute).Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var domain, kind string
		var v types.ProbeView
		var st string
		if err := rows.Scan(&domain, &kind, &v.Probe, &v.Region, &st, &v.Latency, &v.HTTPCode, &v.TS, &v.Err); err != nil {
			return nil, err
		}
		v.Status = types.Status(st)
		key := ComponentKey{Domain: domain, Kind: types.Kind(kind)}
		out[key] = append(out[key], v)
	}
	return out, rows.Err()
}

// RollupStatus computes the current status for (domain, kind) from already-fetched views.
// Call LatestPerProbeBatch first, then pass the per-component views here to avoid a redundant DB round-trip.
func RollupStatus(views []types.ProbeView) (types.Status, int64) {
	if len(views) == 0 {
		return types.StatusOK, 0
	}
	var ok, deg, down int
	var latest int64
	for _, v := range views {
		if v.Region == statusExcludedRegion {
			continue
		}
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
WHERE domain=$1 AND kind=$2 AND ts>=$3 AND region <> 'cn'
ORDER BY ts ASC`, domain, string(kind), since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	acc := newIncidentAccumulator()
	for rows.Next() {
		var ts int64
		var probe, status string
		if err := rows.Scan(&ts, &probe, &status); err != nil {
			return nil, err
		}
		acc.add(ts, probe, types.Status(status))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return acc.finish(), nil
}

type incidentProbeState struct {
	status types.Status
	ts     int64
}

type incidentAccumulator struct {
	probes map[string]incidentProbeState
	cur    *types.Incident
	out    []types.Incident
}

func newIncidentAccumulator() *incidentAccumulator {
	return &incidentAccumulator{probes: map[string]incidentProbeState{}}
}

func (a *incidentAccumulator) add(ts int64, probe string, status types.Status) {
	a.probes[probe] = incidentProbeState{status: status, ts: ts}
	rolled, active, bad := a.rollup(ts)
	if rolled != types.StatusOK {
		if a.cur == nil {
			a.cur = &types.Incident{
				StartTS:   ts,
				EndTS:     ts,
				Status:    rolled,
				PeakDown:  bad,
				PeakTotal: active,
			}
			return
		}
		a.cur.EndTS = ts
		if incidentStatusRank(rolled) > incidentStatusRank(a.cur.Status) {
			a.cur.Status = rolled
		}
		if bad > a.cur.PeakDown {
			a.cur.PeakDown = bad
			a.cur.PeakTotal = active
		}
		return
	}
	a.closeWindow()
}

func (a *incidentAccumulator) finish() []types.Incident {
	a.closeWindow()
	return a.out
}

func (a *incidentAccumulator) closeWindow() {
	if a.cur == nil {
		return
	}
	a.cur.DurationS = int(a.cur.EndTS - a.cur.StartTS)
	if a.cur.DurationS < 60 {
		a.cur.DurationS = 60
	}
	a.out = append(a.out, *a.cur)
	a.cur = nil
}

func (a *incidentAccumulator) rollup(now int64) (status types.Status, active, bad int) {
	const staleSec = int64(180)
	var deg, down int
	for _, p := range a.probes {
		if now-p.ts > staleSec {
			continue
		}
		active++
		switch p.status {
		case types.StatusOK:
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

func incidentStatusRank(s types.Status) int {
	switch s {
	case types.StatusDown:
		return 2
	case types.StatusDegraded:
		return 1
	default:
		return 0
	}
}

// PurgeOlderThan drops daily partitions of `checks` whose entire range is
// before `cutoff`. Returns the number of partitions dropped. With daily
// partitioning, retention is O(1) per day and reclaims disk immediately —
// no DELETE + VACUUM bloat cycle.
// online_counts is kept indefinitely for historical charting.
func (s *Store) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoffUnix := cutoff.Unix()
	rows, err := s.db.QueryContext(ctx, `
SELECT child.relname
FROM pg_inherits
JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
JOIN pg_class child  ON pg_inherits.inhrelid  = child.oid
WHERE parent.relname = 'checks'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var toDrop []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return 0, err
		}
		// Partition names are checks_YYYYMMDD (UTC). A partition covers
		// [day, day+1) — drop only when day+1 <= cutoff so we never drop a
		// partition that still has live data.
		const prefix = "checks_"
		if len(name) != len(prefix)+8 || name[:len(prefix)] != prefix {
			continue
		}
		day, err := time.Parse("20060102", name[len(prefix):])
		if err != nil {
			continue
		}
		if day.AddDate(0, 0, 1).Unix() <= cutoffUnix {
			toDrop = append(toDrop, name)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var dropped int64
	for _, name := range toDrop {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %q`, name)); err != nil {
			return dropped, err
		}
		dropped++
	}
	return dropped, nil
}

// EnsureChecksPartitions creates daily partitions of `checks` from one day
// before today through `daysAhead` days after today (UTC). Idempotent.
// Call this before each retention sweep so today's writes always land in an
// existing partition.
func (s *Store) EnsureChecksPartitions(ctx context.Context, daysAhead int) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := -1; i <= daysAhead; i++ {
		d := today.AddDate(0, 0, i)
		name := fmt.Sprintf("checks_%s", d.Format("20060102"))
		start := d.Unix()
		end := d.AddDate(0, 0, 1).Unix()
		stmt := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %q PARTITION OF checks FOR VALUES FROM (%d) TO (%d)`,
			name, start, end)
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// Stats returns simple store stats for debugging.
func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	out := map[string]any{}
	var n int64
	// Health checks should not scan the full partitioned checks table. PostgreSQL
	// reltuples is approximate, but good enough for this debugging endpoint.
	if err := s.db.QueryRowContext(ctx, `
WITH child_rels AS (
	SELECT inhrelid AS oid
	FROM pg_inherits
	WHERE inhparent = 'checks'::regclass
), rels AS (
	SELECT oid FROM child_rels
	UNION ALL
	SELECT 'checks'::regclass
	WHERE NOT EXISTS (SELECT 1 FROM child_rels)
)
SELECT COALESCE(SUM(GREATEST(c.reltuples, 0))::bigint, 0)
FROM rels r
JOIN pg_class c ON c.oid = r.oid`).Scan(&n); err != nil {
		return nil, err
	}
	out["check_rows"] = n
	var firstPartition sql.NullString
	_ = s.db.QueryRowContext(ctx, `
SELECT MIN(c.relname)
FROM pg_inherits i
JOIN pg_class c ON c.oid = i.inhrelid
WHERE i.inhparent = 'checks'::regclass
  AND c.relname ~ '^checks_[0-9]{8}$'`).Scan(&firstPartition)
	if firstPartition.Valid {
		if day, err := time.Parse("checks_20060102", firstPartition.String); err == nil {
			out["earliest"] = day.Unix()
		}
	}
	var latest sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `SELECT MAX(last_seen) FROM probes`).Scan(&latest)
	if latest.Valid {
		out["latest"] = latest.Int64
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wiki_stats_daily`).Scan(&n); err == nil {
		out["wiki_stats_days"] = n
	}
	return out, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func must(err error) {
	if err != nil {
		panic(fmt.Errorf("store: %w", err))
	}
}
