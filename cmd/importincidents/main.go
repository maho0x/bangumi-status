// Command importincidents rebuilds durable incident history from an archived
// database dump.
//
// The aggregator derives incidents from the raw `checks` table, which is dropped
// partition-by-partition after ~35 days, so the live database can never look
// further back than its retention window. Restoring an old dump into a scratch
// database and replaying the same walk over it recovers history that would
// otherwise be gone for good; this command does the replay and upserts the
// result into the target database's `incidents` table.
//
// Dumps may overlap. Because the walk starts with no per-probe state, the first
// minutes of any dump under-report (quorum needs several probes reporting before
// it escalates), so windows starting inside -warmup of a dump's first sample are
// skipped: whichever dump was already warm at that instant contributes them.
//
//	importincidents -src 'postgres:///bgmold?...' -dst 'postgres:///archive?...'
//	importincidents -src 'postgres:///bgmold?...' -out incidents.sql
//
// Components are discovered from the dump itself rather than from
// types.SiteConfigs, so history for a decommissioned target is preserved rather
// than silently dropped.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"bangumi-status/internal/store"
	"bangumi-status/internal/types"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var cst = time.FixedZone("CST", 8*60*60)

type component struct {
	domain string
	kind   types.Kind
}

func main() {
	var (
		src    = flag.String("src", "", "DSN of the restored dump to read `checks` from (required)")
		dst    = flag.String("dst", "", "DSN to upsert incidents into (optional)")
		out    = flag.String("out", "", "write the incidents as a replayable SQL upsert to this file (optional)")
		warmup = flag.Duration("warmup", 30*time.Minute, "ignore windows starting within this long after the dump's first sample")
	)
	flag.Parse()
	if *src == "" {
		log.Fatal("-src is required")
	}

	ctx := context.Background()
	srcStore, err := store.Open(*src)
	if err != nil {
		log.Fatalf("open src: %v", err)
	}
	defer srcStore.Close()

	meta, err := sql.Open("pgx", *src)
	if err != nil {
		log.Fatalf("open src meta: %v", err)
	}
	defer meta.Close()

	first, last, err := checkRange(ctx, meta)
	if err != nil {
		log.Fatalf("check range: %v", err)
	}
	comps, err := components(ctx, meta)
	if err != nil {
		log.Fatalf("components: %v", err)
	}
	floor := first + int64(warmup.Seconds())
	log.Printf("dump covers %s → %s CST (%d components), skipping windows before %s",
		fmtTS(first), fmtTS(last), len(comps), fmtTS(floor))

	found := map[component][]types.Incident{}
	total, skipped := 0, 0
	var totalDur float64
	fmt.Printf("%-18s %-6s %8s %10s\n", "component", "kind", "windows", "hours")
	for _, c := range comps {
		incs, err := srcStore.Incidents(ctx, c.domain, c.kind, time.Unix(first, 0))
		if err != nil {
			log.Fatalf("walk %s/%s: %v", c.domain, c.kind, err)
		}
		kept := make([]types.Incident, 0, len(incs))
		var hours float64
		for _, inc := range incs {
			if inc.StartTS < floor {
				skipped++
				continue
			}
			kept = append(kept, inc)
			hours += float64(inc.EndTS-inc.StartTS) / 3600
		}
		if len(kept) == 0 {
			continue
		}
		found[c] = kept
		total += len(kept)
		totalDur += hours
		fmt.Printf("%-18s %-6s %8d %10.1f\n", c.domain, c.kind, len(kept), hours)
	}
	fmt.Printf("\n%d windows / %.1f h (skipped %d inside the warm-up window)\n", total, totalDur, skipped)

	if *out != "" {
		if err := writeSQL(*out, found); err != nil {
			log.Fatalf("write %s: %v", *out, err)
		}
		log.Printf("wrote %s", *out)
	}

	if *dst == "" {
		return
	}
	dstStore, err := store.Open(*dst)
	if err != nil {
		log.Fatalf("open dst: %v", err)
	}
	defer dstStore.Close()
	written := 0
	for c, incs := range found {
		n, err := dstStore.ImportIncidents(ctx, c.domain, c.kind, incs)
		if err != nil {
			log.Fatalf("import %s/%s: %v", c.domain, c.kind, err)
		}
		written += n
	}
	log.Printf("upserted %d windows into dst", written)
}

func checkRange(ctx context.Context, db *sql.DB) (int64, int64, error) {
	var first, last sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT MIN(ts), MAX(ts) FROM checks`).Scan(&first, &last)
	if err != nil {
		return 0, 0, err
	}
	if !first.Valid {
		return 0, 0, fmt.Errorf("no rows in checks")
	}
	return first.Int64, last.Int64, nil
}

func components(ctx context.Context, db *sql.DB) ([]component, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT domain, kind FROM checks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []component
	for rows.Next() {
		var d, k string
		if err := rows.Scan(&d, &k); err != nil {
			return nil, err
		}
		out = append(out, component{domain: d, kind: types.Kind(k)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].domain != out[j].domain {
			return out[i].domain < out[j].domain
		}
		return out[i].kind < out[j].kind
	})
	return out, rows.Err()
}

func fmtTS(ts int64) string { return time.Unix(ts, 0).In(cst).Format("2006-01-02 15:04") }

// lit quotes a value read out of the dump for the emitted SQL. The set of
// domains and kinds is closed in practice, but nothing here should depend on
// that.
func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// writeSQL emits the whole import as one idempotent statement so it can be
// reviewed before it is applied to a database this command has no business
// connecting to.
func writeSQL(path string, found map[component][]types.Incident) error {
	type row struct {
		c   component
		inc types.Incident
	}
	var rows []row
	for c, incs := range found {
		for _, inc := range incs {
			rows = append(rows, row{c: c, inc: inc})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].inc.StartTS < rows[j].inc.StartTS })
	if len(rows) == 0 {
		return fmt.Errorf("nothing to write")
	}

	var b strings.Builder
	b.WriteString("-- Incident history recovered from an archived checks dump by cmd/importincidents.\n")
	b.WriteString("-- Idempotent: re-running only extends windows, never duplicates them.\n")
	b.WriteString("BEGIN;\n")
	b.WriteString("INSERT INTO incidents (domain, kind, start_ts, end_ts, status, peak_down, peak_total) VALUES\n")
	for i, r := range rows {
		sep := ","
		if i == len(rows)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "  (%s,%s,%d,%d,%s,%d,%d)%s\n",
			lit(r.c.domain), lit(string(r.c.kind)), r.inc.StartTS, r.inc.EndTS,
			lit(string(r.inc.Status)), r.inc.PeakDown, r.inc.PeakTotal, sep)
	}
	b.WriteString(`ON CONFLICT (domain, kind, start_ts) DO UPDATE SET
  end_ts     = GREATEST(incidents.end_ts, excluded.end_ts),
  status     = CASE WHEN excluded.status = 'down' THEN 'down'
                    WHEN incidents.status = 'down' THEN 'down'
                    ELSE excluded.status END,
  peak_down  = GREATEST(incidents.peak_down, excluded.peak_down),
  peak_total = GREATEST(incidents.peak_total, excluded.peak_total);
COMMIT;
`)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
