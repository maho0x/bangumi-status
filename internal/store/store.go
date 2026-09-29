// Package store is the PostgreSQL persistence layer.
package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *pgxpool.Pool
}

// Open connects and applies any pending migrations.
func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Small on purpose: history refreshes cap their own concurrency and must
	// never starve ingest.
	cfg.MaxConns = 6
	cfg.MaxConnLifetime = 30 * time.Minute
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() { s.db.Close() }

// Pool exposes the connection pool for one-off tools and tests.
func (s *Store) Pool() *pgxpool.Pool { return s.db }

// migrate applies migrations/NNN_*.sql in order, each in its own transaction,
// recording applied versions in schema_migrations. An advisory lock keeps two
// starting processes from racing.
func (s *Store) migrate(ctx context.Context) error {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7340125);
CREATE TABLE IF NOT EXISTS schema_migrations (
  version    INTEGER PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
			return err
		}
		for _, f := range files {
			name := strings.TrimPrefix(f, "migrations/")
			num, _, _ := strings.Cut(name, "_")
			version, err := strconv.Atoi(num)
			if err != nil {
				return fmt.Errorf("migration %s: bad version prefix", name)
			}
			var done bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&done); err != nil {
				return err
			}
			if done {
				continue
			}
			body, err := migrations.ReadFile(f)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, string(body)); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, version); err != nil {
				return err
			}
			slog.Info("applied migration", "name", name)
		}
		return nil
	})
}

// collect scans every row with scan, appending the results.
func collect[T any](rows pgx.Rows, scan func(pgx.Rows, *T) error) ([]T, error) {
	defer rows.Close()
	var out []T
	for rows.Next() {
		var v T
		if err := scan(rows, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetConfig reads a small persisted key/value setting.
func (s *Store) GetConfig(ctx context.Context, key string) (string, bool, error) {
	var v string
	err := s.db.QueryRow(ctx, `SELECT value FROM config WHERE key=$1`, key).Scan(&v)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	return v, err == nil, err
}

// SetConfig upserts a key/value setting.
func (s *Store) SetConfig(ctx context.Context, key, value string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO config(key,value) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

// Stats returns cheap store figures for the debug health endpoint.
func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	out := map[string]any{}
	// reltuples is approximate but never scans the partitioned table.
	var rows int64
	if err := s.db.QueryRow(ctx, `
WITH child_rels AS (
  SELECT inhrelid AS oid FROM pg_inherits WHERE inhparent = 'checks'::regclass
), rels AS (
  SELECT oid FROM child_rels
  UNION ALL
  SELECT 'checks'::regclass WHERE NOT EXISTS (SELECT 1 FROM child_rels)
)
SELECT COALESCE(SUM(GREATEST(c.reltuples, 0))::bigint, 0)
FROM rels r JOIN pg_class c ON c.oid = r.oid`).Scan(&rows); err != nil {
		return nil, err
	}
	out["check_rows"] = rows
	var first *string
	_ = s.db.QueryRow(ctx, `
SELECT MIN(c.relname) FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
WHERE i.inhparent = 'checks'::regclass AND c.relname ~ '^checks_[0-9]{8}$'`).Scan(&first)
	if first != nil {
		if day, err := time.Parse("checks_20060102", *first); err == nil {
			out["earliest"] = day.Unix()
		}
	}
	var latest *int64
	_ = s.db.QueryRow(ctx, `SELECT MAX(last_seen) FROM probes`).Scan(&latest)
	if latest != nil {
		out["latest"] = *latest
	}
	var days int64
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wiki_stats_daily`).Scan(&days); err == nil {
		out["wiki_stats_days"] = days
	}
	return out, nil
}
