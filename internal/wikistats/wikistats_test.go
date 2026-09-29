package wikistats

import "testing"

func TestParse(t *testing.T) {
	page := `<script>var CHART_SETS = {"register":{"label":"a \"}{\" b","data":[` +
		`{"title":"3/1","date":"2026-03-01","register_total":5,"reply_8":2}]},"replies":{"data":[]}};` +
		`var other = {"x":1};</script>`
	raw, points, err := Parse(page)
	if err != nil {
		t.Fatal(err)
	}
	if want := page[len(`<script>var CHART_SETS = `) : len(page)-len(`;var other = {"x":1};</script>`)]; string(raw) != want {
		t.Fatalf("raw = %s", raw)
	}
	if len(points) != 1 || points[0].Date != "2026-03-01" || points[0].RegisterTotal != 5 || points[0].Reply8 != 2 {
		t.Fatalf("points = %+v", points)
	}

	for _, bad := range []string{`nothing here`, `var CHART_SETS = {"register":{"data":[]}}`, `var CHART_SETS = {"a":`} {
		if _, _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) succeeded", bad)
		}
	}
}
