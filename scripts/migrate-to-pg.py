#!/usr/bin/env python3
"""Migrate bangumi-status data from SQLite to PostgreSQL."""
import sqlite3
import psycopg2
import sys
from contextlib import closing

SQLITE_PATH = sys.argv[1] if len(sys.argv) > 1 else "/var/lib/bangumi-status/status.db"
PG_DSN = sys.argv[2] if len(sys.argv) > 2 else "dbname=bangumi_status user=postgres host=localhost port=5432"

def migrate():
    print(f"Source: {SQLITE_PATH}")
    print(f"Target: {PG_DSN}")

    with closing(sqlite3.connect(SQLITE_PATH)) as src, closing(psycopg2.connect(PG_DSN)) as dst:
        src_cur = src.cursor()
        dst_cur = dst.cursor()

        # Clear target tables (idempotent)
        for tbl in ["checks", "probes", "config", "online_counts"]:
            dst_cur.execute(f"TRUNCATE TABLE {tbl} CASCADE")
        dst.commit()

        # Migrate checks
        print("Migrating checks...")
        src_cur.execute("SELECT ts, probe, region, domain, kind, status, latency_ms, http_code, err FROM checks ORDER BY id")
        batch = []
        total = 0
        for row in src_cur:
            batch.append(row)
            if len(batch) >= 5000:
                dst_cur.executemany(
                    "INSERT INTO checks (ts, probe, region, domain, kind, status, latency_ms, http_code, err) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)",
                    batch
                )
                dst.commit()
                total += len(batch)
                print(f"  ... {total} rows")
                batch = []
        if batch:
            dst_cur.executemany(
                "INSERT INTO checks (ts, probe, region, domain, kind, status, latency_ms, http_code, err) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)",
                batch
            )
            dst.commit()
            total += len(batch)
        print(f"  checks: {total} rows")

        # Migrate probes
        print("Migrating probes...")
        src_cur.execute("SELECT name, region, last_seen FROM probes")
        rows = src_cur.fetchall()
        if rows:
            dst_cur.executemany(
                "INSERT INTO probes (name, region, last_seen) VALUES (%s,%s,%s) ON CONFLICT (name) DO UPDATE SET region=EXCLUDED.region, last_seen=EXCLUDED.last_seen",
                rows
            )
            dst.commit()
        print(f"  probes: {len(rows)} rows")

        # Migrate config
        print("Migrating config...")
        src_cur.execute("SELECT key, value FROM config")
        rows = src_cur.fetchall()
        if rows:
            dst_cur.executemany(
                "INSERT INTO config (key, value) VALUES (%s,%s) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value",
                rows
            )
            dst.commit()
        print(f"  config: {len(rows)} rows")

        # Migrate online_counts
        print("Migrating online_counts...")
        src_cur.execute("SELECT ts_min, count FROM online_counts")
        rows = src_cur.fetchall()
        if rows:
            dst_cur.executemany(
                "INSERT INTO online_counts (ts_min, count) VALUES (%s,%s) ON CONFLICT (ts_min) DO NOTHING",
                rows
            )
            dst.commit()
        print(f"  online_counts: {len(rows)} rows")

        # Verify
        dst_cur.execute("SELECT COUNT(*) FROM checks")
        check_count = dst_cur.fetchone()[0]
        print(f"\nVerification: {check_count} checks in PostgreSQL")

    print("Done.")

if __name__ == "__main__":
    migrate()
