package rollup

import (
	"bangumi-status/internal/types"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestRollupGolden(t *testing.T) {
	out := map[string]any{}

	var quorums []int
	for n := 0; n <= 12; n++ {
		quorums = append(quorums, Quorum(n))
	}
	out["quorum"] = quorums

	// Incident walks over deterministic pseudo-random event streams.
	statuses := []types.Status{types.StatusOK, types.StatusOK, types.StatusOK, types.StatusDegraded, types.StatusDown}
	var walks [][]types.Incident
	for seed := int64(1); seed <= 12; seed++ {
		rng := rand.New(rand.NewSource(seed))
		nProbes := 2 + rng.Intn(6)
		acc := NewWalker()
		ts := int64(1_700_000_000)
		// Regime switches make long outages likely instead of pure noise.
		bias := 0
		for i := 0; i < 600; i++ {
			if rng.Intn(40) == 0 {
				bias = rng.Intn(3)
			}
			ts += int64(rng.Intn(25))
			if rng.Intn(80) == 0 {
				ts += 400 // gap longer than the staleness window
			}
			probe := fmt.Sprintf("p%d", rng.Intn(nProbes))
			st := statuses[rng.Intn(len(statuses))]
			if bias == 1 && rng.Intn(3) > 0 {
				st = types.StatusDown
			} else if bias == 2 && rng.Intn(3) > 0 {
				st = types.StatusDegraded
			}
			acc.Add(ts, probe, st)
		}
		walks = append(walks, acc.Finish())
	}
	out["walks"] = walks

	// Live rollup: relative timestamps, cn excluded, stale probes ignored.
	now := time.Now().Unix()
	views := func(spec ...any) []types.ProbeView {
		var vs []types.ProbeView
		for i := 0; i < len(spec); i += 3 {
			vs = append(vs, types.ProbeView{Probe: fmt.Sprint("p", i), Region: spec[i].(string), Status: spec[i+1].(types.Status), TS: now - int64(spec[i+2].(int))})
		}
		return vs
	}
	ok, deg, down := types.StatusOK, types.StatusDegraded, types.StatusDown
	type liveOut struct {
		Status     types.Status
		LastAgo    int64
		Active, Bd int
	}
	var live []liveOut
	for _, vs := range [][]types.ProbeView{
		nil,
		views("jp", down, 5, "jp", down, 10, "sg", ok, 3),
		views("jp", down, 5, "cn", down, 1, "sg", ok, 3),
		views("jp", down, 5, "hk", deg, 10, "sg", ok, 3),
		views("jp", down, 5, "hk", down, 500, "sg", ok, 3),
		views("jp", down, 5, "hk", down, 5, "sg", down, 3, "us", ok, 1),
		views("jp", deg, 5, "hk", deg, 5, "sg", ok, 3, "us", ok, 1),
		views("cn", down, 1),
	} {
		st, last, a, b := Live(vs)
		ago := int64(-1)
		if last > 0 {
			ago = now - last
		}
		live = append(live, liveOut{st, ago, a, b})
	}
	out["live"] = live

	walk := []types.Incident{
		{StartTS: 100, EndTS: 200, Status: deg, PeakDown: 2, PeakTotal: 5},
		{StartTS: 900, EndTS: 1000, Status: down, PeakDown: 4, PeakTotal: 5},
		{StartTS: 1100, EndTS: 1150, Status: deg, PeakDown: 3, PeakTotal: 5},
	}
	out["merge"] = [][]types.Incident{
		MergeOngoing(walk, 950, 1300, deg, 2, 5),
		MergeOngoing(walk, 1200, 1210, down, 5, 6),
		MergeOngoing(nil, 10, 20, deg, 1, 3),
	}

	start := time.Date(2026, 3, 1, 0, 0, 0, 0, types.CST)
	buckets := []types.DayBucket{
		{Day: "2026-03-01", Uptime: 95, Total: 100, Down: 3, Degrade: 2, Status: types.StatusDown},
		{Day: "2026-03-02", Uptime: 95, Total: 100, Degrade: 5, Status: types.StatusDegraded},
		{Day: "2026-03-03"},
		{Day: "2026-03-04", Uptime: 100, Total: 50, Status: types.StatusOK},
		{Day: "2026-03-05"},
	}
	Overlay(buckets, []types.Incident{
		{StartTS: start.AddDate(0, 0, 1).Add(23 * time.Hour).Unix(), EndTS: start.AddDate(0, 0, 3).Add(time.Hour).Unix(), Status: down},
		{StartTS: start.AddDate(0, 0, 4).Unix(), EndTS: start.AddDate(0, 0, 4).Unix() + 60, Status: deg},
		{StartTS: start.AddDate(0, 0, -3).Unix(), EndTS: start.AddDate(0, 0, -2).Unix(), Status: down},
	})
	out["buckets"] = buckets

	got, _ := json.MarshalIndent(out, "", " ")
	const path = "testdata/rollup_golden.json"
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Fatalf("golden mismatch against %s; got:\n%s", path, got)
	}
}
