-- Baseline: the schema as it stood before versioned migrations. Every
-- statement is idempotent so it applies cleanly both to an empty database and
-- to one the pre-migration aggregator (or an archived dump) already created.

-- checks is partitioned by UTC day on ts (unix seconds). Daily partitions keep
-- retention cheap (DROP partition instead of DELETE + VACUUM). The store creates
-- the day partitions on a schedule.
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

-- traffic_samples: the per-minute PEAK of concurrent status-page viewers.
CREATE TABLE IF NOT EXISTS traffic_samples (
  ts_min  BIGINT PRIMARY KEY,
  viewers INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS reactions (
  emoji_id   SMALLINT NOT NULL,
  user_id    TEXT NOT NULL,
  ip         TEXT NOT NULL,
  count      INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (emoji_id, user_id)
);
ALTER TABLE reactions ADD COLUMN IF NOT EXISTS count INTEGER NOT NULL DEFAULT 1;
CREATE INDEX IF NOT EXISTS reactions_created_at_idx ON reactions (created_at);
CREATE INDEX IF NOT EXISTS reactions_user_created_at_idx ON reactions (user_id, created_at);

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

-- incidents: the durable memory of past outages. Windows are derived from
-- checks, which drops whole day partitions after ~35 days, so the aggregator
-- mirrors every freshly walked window here. A few hundred rows a year; never
-- purged.
CREATE TABLE IF NOT EXISTS incidents (
  domain     TEXT   NOT NULL,
  kind       TEXT   NOT NULL,
  start_ts   BIGINT NOT NULL,
  end_ts     BIGINT NOT NULL,
  status     TEXT   NOT NULL,
  peak_down  INTEGER NOT NULL DEFAULT 0,
  peak_total INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (domain, kind, start_ts)
);
CREATE INDEX IF NOT EXISTS incidents_start_ts_idx ON incidents (start_ts DESC);
