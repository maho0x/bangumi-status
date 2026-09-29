package store

import (
	"context"
	"flag"
	"testing"

	"github.com/jackc/pgx/v5"
)

var update = flag.Bool("update", false, "rewrite golden files")

// openScratch wipes the scratch database's public schema and opens a Store on it.
func openScratch(t *testing.T, dsn string) *Store {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	conn.Close(ctx)
	st, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	return st
}
