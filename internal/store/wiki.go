package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bangumi-status/internal/types"

	"github.com/jackc/pgx/v5"
)

// wikiColumns are the numeric wiki_stats_daily columns in WikiStatsPoint field
// order; wikiFields binds them to a point.
var wikiColumns = []string{
	"register_total", "collection_total", "topic_total", "reply_total",
	"collection_1", "collection_2", "collection_3", "collection_4", "collection_5",
	"topic_1", "topic_2", "topic_7",
	"reply_1", "reply_2", "reply_3", "reply_4", "reply_5", "reply_6", "reply_7", "reply_8",
}

func wikiFields(p *types.WikiStatsPoint) []*int {
	return []*int{
		&p.RegisterTotal, &p.CollectionTotal, &p.TopicTotal, &p.ReplyTotal,
		&p.Collection1, &p.Collection2, &p.Collection3, &p.Collection4, &p.Collection5,
		&p.Topic1, &p.Topic2, &p.Topic7,
		&p.Reply1, &p.Reply2, &p.Reply3, &p.Reply4, &p.Reply5, &p.Reply6, &p.Reply7, &p.Reply8,
	}
}

var wikiUpsertSQL = func() string {
	cols := append([]string{"day", "title", "ts"}, wikiColumns...)
	cols = append(cols, "raw", "updated_at")
	ph := make([]string, len(cols))
	set := make([]string, 0, len(cols)-1)
	for i, c := range cols {
		ph[i] = fmt.Sprintf("$%d", i+1)
		if c != "day" {
			set = append(set, c+" = EXCLUDED."+c)
		}
	}
	return "INSERT INTO wiki_stats_daily (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(ph, ", ") +
		") ON CONFLICT (day) DO UPDATE SET " + strings.Join(set, ", ")
}()

// UpsertWikiStats stores every daily row scraped from chii.in/wiki/stats and
// keeps the raw CHART_SETS payload as an immutable snapshot. It returns how
// many rows it stored and the newest date among them.
func (s *Store) UpsertWikiStats(ctx context.Context, points []types.WikiStatsPoint, chartSets []byte) (int, string, error) {
	if len(points) == 0 {
		return 0, "", nil
	}
	now := time.Now().Unix()
	latest := ""
	b := &pgx.Batch{}
	for i := range points {
		p := &points[i]
		if _, err := time.Parse(time.DateOnly, p.Date); err != nil {
			return 0, "", fmt.Errorf("invalid wiki stats date %q: %w", p.Date, err)
		}
		raw, err := json.Marshal(p)
		if err != nil {
			return 0, "", err
		}
		args := []any{p.Date, p.Title, p.Timestamp}
		for _, f := range wikiFields(p) {
			args = append(args, *f)
		}
		b.Queue(wikiUpsertSQL, append(args, string(raw), now)...)
		latest = max(latest, p.Date)
	}
	if len(chartSets) > 0 {
		b.Queue(`INSERT INTO wiki_stats_snapshots (scraped_at, source_date, row_count, chart_sets) VALUES ($1, $2, $3, $4)`,
			now, latest, len(points), string(chartSets))
	}
	if err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error { return tx.SendBatch(ctx, b).Close() }); err != nil {
		return 0, "", err
	}
	return len(points), latest, nil
}

// WikiStats returns every stored day (oldest first) and when it was last
// scraped (0 if never).
func (s *Store) WikiStats(ctx context.Context) ([]types.WikiStatsPoint, int64, error) {
	rows, err := s.db.Query(ctx, `SELECT day::text, title, ts, `+strings.Join(wikiColumns, ", ")+
		` FROM wiki_stats_daily ORDER BY day`)
	if err != nil {
		return nil, 0, err
	}
	points, err := collect(rows, func(r pgx.Rows, p *types.WikiStatsPoint) error {
		dest := []any{&p.Date, &p.Title, &p.Timestamp}
		for _, f := range wikiFields(p) {
			dest = append(dest, f)
		}
		return r.Scan(dest...)
	})
	if err != nil {
		return nil, 0, err
	}
	var scraped *int64
	if err := s.db.QueryRow(ctx, `SELECT MAX(scraped_at) FROM wiki_stats_snapshots`).Scan(&scraped); err != nil {
		return nil, 0, err
	}
	if scraped == nil {
		return points, 0, nil
	}
	return points, *scraped, nil
}
