package store

import (
	"bangumi-status/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

// TestStoreDBGolden runs every query against a throwaway database. It wipes
// the public schema, so it only runs when BGM_TEST_DSN points at a scratch DB.
func TestStoreDBGolden(t *testing.T) {
	dsn := os.Getenv("BGM_TEST_DSN")
	if dsn == "" {
		t.Skip("BGM_TEST_DSN not set")
	}
	ctx := context.Background()
	st := openScratch(t, dsn)
	defer st.Close()

	now := time.Now()
	nowCST := now.In(types.CST)
	today := time.Date(nowCST.Year(), nowCST.Month(), nowCST.Day(), 0, 0, 0, 0, types.CST)
	dayOff := func(s string) string {
		d, err := time.ParseInLocation("2006-01-02", s, types.CST)
		if err != nil {
			return s
		}
		return fmt.Sprintf("D%+d", int(d.Sub(today).Hours()/24+0.5*sign(d.Sub(today))))
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// Partitions for every day the fixture touches (plus some to purge).
	for i := -45; i <= 2; i++ {
		d := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, i)
		_, err := st.Pool().Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %q PARTITION OF checks FOR VALUES FROM (%d) TO (%d)`,
			"checks_"+d.Format("20060102"), d.Unix(), d.AddDate(0, 0, 1).Unix()))
		must(err)
	}

	probes := []struct{ name, region string }{{"p1", "jp"}, {"p2", "jp"}, {"p3", "sg"}, {"p4", "hk"}, {"p5", "us"}, {"pcn", "cn"}}
	rng := rand.New(rand.NewSource(7))
	var rows []types.Result
	for _, day := range []int{-20, -15, -5, -1, 0} {
		base := today.AddDate(0, 0, day)
		for m := 0; m < 180; m++ {
			for pi, p := range probes {
				ts := base.Add(time.Duration(m)*time.Minute + time.Duration(pi*7)*time.Second).Unix()
				st := types.StatusOK
				switch {
				case p.region == "cn":
					st = types.StatusDown
				case m >= 60 && m < 90 && pi < 4:
					st = types.StatusDown
				case m >= 120 && m < 130 && pi < 3:
					st = types.StatusDegraded
				case m == 150 && pi < 2:
					st = types.StatusDown // brief minute, quorum of 2 met
				}
				rows = append(rows, types.Result{TS: ts, Probe: p.name, Region: p.region, Domain: "bgm.tv", Kind: types.KindAuth, Status: st, Latency: 100 + m, HTTPCode: 200})
				gst := types.StatusOK
				if r := rng.Intn(10); r == 0 {
					gst = types.StatusDown
				} else if r == 1 {
					gst = types.StatusDegraded
				}
				rows = append(rows, types.Result{TS: ts, Probe: p.name, Region: p.region, Domain: "bangumi.tv", Kind: types.KindGuest, Status: gst, Latency: 50, Err: "e"})
			}
		}
	}
	// Live component: relative to now.
	for pi, p := range probes {
		for k := 0; k < 3; k++ {
			st := types.StatusOK
			if pi < 3 && k == 2 {
				st = types.StatusDown
			}
			rows = append(rows, types.Result{TS: now.Unix() - int64(200-k*60-pi), Probe: p.name, Region: p.region, Domain: "live.test", Kind: types.KindAuth, Status: st, Latency: 10 * k, HTTPCode: 500})
		}
	}
	for i := 0; i < len(rows); i += 5000 {
		must(st.Insert(ctx, rows[i:min(i+5000, len(rows))]))
	}
	for i, p := range probes {
		must(st.UpsertProbe(ctx, p.name, p.region, now.Unix()-int64(i*100)))
	}
	must(st.UpsertProbe(ctx, "old", "de", now.Add(-48*time.Hour).Unix()))

	out := map[string]any{}
	comps := []types.Component{{Domain: "bgm.tv", Kind: types.KindAuth}, {Domain: "bangumi.tv", Kind: types.KindGuest}}
	for _, c := range comps {
		key := c.Domain + "|" + string(c.Kind)
		b, err := st.DailyBuckets(ctx, c.Domain, c.Kind, 30)
		must(err)
		for i := range b {
			b[i].Day = dayOff(b[i].Day)
		}
		out["buckets "+key] = b
		incs, err := st.Incidents(ctx, c.Domain, c.Kind, now.Add(-14*24*time.Hour))
		must(err)
		rel := make([][]int64, 0, len(incs))
		for _, in := range incs {
			rel = append(rel, []int64{in.StartTS - today.Unix(), in.EndTS - today.Unix(), int64(in.PeakDown), int64(in.PeakTotal), int64(in.DurationS), int64(len(in.Status))})
		}
		out["incidents "+key] = rel
		for _, day := range []int{-5, -15} {
			s := today.AddDate(0, 0, day)
			ds, err := st.DaySummary(ctx, c.Domain, c.Kind, s)
			must(err)
			ds.Day = dayOff(ds.Day)
			out[fmt.Sprintf("daysummary %s %d", key, day)] = ds
		}
		must(st.SyncIncidents(ctx, c.Domain, c.Kind, now.Add(-14*24*time.Hour).Unix(), incs))
	}
	views, err := st.LatestPerProbe(ctx, []types.Component{{Domain: "live.test", Kind: types.KindAuth}, {Domain: "nothing", Kind: types.KindGuest}})
	must(err)
	for k, vs := range views {
		for i := range vs {
			vs[i].TS = now.Unix() - vs[i].TS
		}
		out["latest "+k.Domain] = vs
	}
	ps, err := st.Probes(ctx)
	must(err)
	for i := range ps {
		ps[i].LastSeen = now.Unix() - ps[i].LastSeen
	}
	out["probes"] = ps

	// Incident archive reconciliation.
	a := int64(1_600_000_000)
	must(st.SyncIncidents(ctx, "x", types.KindAuth, a, []types.Incident{{StartTS: a + 10, EndTS: a + 20, Status: types.StatusDegraded, PeakDown: 2, PeakTotal: 4}, {StartTS: a + 100, EndTS: a + 200, Status: types.StatusDown, PeakDown: 3, PeakTotal: 4}}))
	must(st.SyncIncidents(ctx, "x", types.KindAuth, a+50, []types.Incident{{StartTS: a + 100, EndTS: a + 150, Status: types.StatusDegraded, PeakDown: 1, PeakTotal: 5}, {StartTS: a + 300, EndTS: a + 400, Status: types.StatusDegraded}}))
	n, err := st.ImportIncidents(ctx, "x", types.KindGuest, []types.Incident{{StartTS: a - 1000, EndTS: a - 900, Status: types.StatusDown}})
	must(err)
	out["imported"] = n
	hist, err := st.ListIncidents(ctx, a-2000, a+1000)
	must(err)
	out["history"] = hist
	earliest, err := st.EarliestIncidentTS(ctx)
	must(err)
	out["earliest"] = earliest - a

	// Series.
	s0 := int64(1_700_000_040)
	for i, v := range []int{100, 105, 230, 110, 112, 108, 300, 290, 120, 125, 0, 130} {
		must(st.InsertOnline(ctx, s0+int64(i)*60+int64(i%3), v))
	}
	for i, v := range []int{1, 3, 2, 0, 5} {
		must(st.InsertTrafficSample(ctx, s0+int64(i/2)*60, v))
	}
	for _, b := range []int64{0, 120, 300} {
		on, err := st.Points(ctx, Online, time.Unix(s0, 0), b)
		must(err)
		tr, err := st.Points(ctx, Traffic, time.Unix(s0, 0), b)
		must(err)
		out[fmt.Sprintf("online %d", b)] = on
		out[fmt.Sprintf("traffic %d", b)] = tr
	}
	lo, err := st.LatestOnline(ctx)
	must(err)
	out["latest online"] = lo

	// Reactions.
	must(st.AddReaction(ctx, 44, "userAAAA", "1.1.1.1"))
	must(st.AddReaction(ctx, 44, "userAAAA", "1.1.1.2"))
	must(st.AddReaction(ctx, 15, "userBBBB", "1.1.1.3"))
	must(st.AddReaction(ctx, 44, "userBBBB", "1.1.1.3"))
	_, err = st.Pool().Exec(ctx, `INSERT INTO reactions (emoji_id,user_id,ip,count,created_at) VALUES (23,'old','x',9,NOW()-INTERVAL '2 days')`)
	must(err)
	rc, err := st.ReactionCounts(ctx)
	must(err)
	out["reactions"] = rc
	mine, err := st.UserReactions(ctx, "userBBBB")
	must(err)
	out["mine"] = mine
	must(st.PurgeExpiredReactions(ctx))
	var left int
	must(st.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM reactions`).Scan(&left))
	out["reactions left"] = left

	// Wiki stats.
	pts := []types.WikiStatsPoint{{Title: "3/1", Date: "2026-03-01", Timestamp: 1, RegisterTotal: 10, Reply8: 8}, {Title: "3/2", Date: "2026-03-02", Timestamp: 2, CollectionTotal: 5, Topic7: 7}}
	acc, latest, err := st.UpsertWikiStats(ctx, pts, []byte(`{"register":{"data":[]}}`))
	must(err)
	pts[0].RegisterTotal = 11
	_, _, err = st.UpsertWikiStats(ctx, pts[:1], []byte(`{}`))
	must(err)
	wp, scraped, err := st.WikiStats(ctx)
	must(err)
	out["wiki"] = map[string]any{"accepted": acc, "latest": latest, "points": wp, "scraped": scraped > 0}

	// Config.
	_, ok, err := st.GetConfig(ctx, "k")
	must(err)
	must(st.SetConfig(ctx, "k", "v1"))
	must(st.SetConfig(ctx, "k", "v2"))
	v, ok2, err := st.GetConfig(ctx, "k")
	must(err)
	out["config"] = []any{ok, v, ok2}

	dropped, err := st.DropPartitionsBefore(ctx, now.AddDate(0, 0, -35))
	must(err)
	out["dropped"] = dropped

	got, _ := json.MarshalIndent(out, "", " ")
	const path = "testdata/store_db_golden.json"
	if *update {
		must(os.WriteFile(path, got, 0o644))
		return
	}
	want, err := os.ReadFile(path)
	must(err)
	if string(want) != string(got) {
		t.Fatalf("golden mismatch against %s; got:\n%s", path, got)
	}
}

func sign(d time.Duration) float64 {
	if d < 0 {
		return -1
	}
	return 1
}
