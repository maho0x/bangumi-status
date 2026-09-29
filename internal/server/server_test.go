package server

import (
	"net/url"
	"testing"
	"time"

	"bangumi-status/internal/config"
	"bangumi-status/internal/types"
)

func TestAuthorizeProbe(t *testing.T) {
	s := &Server{cfg: config.Aggregator{IngestSecret: "admin", TokenPrefixes: map[string]string{"tokA": "alice-"}}}
	cases := []struct {
		token, probe string
		ok, known    bool
	}{
		{"admin", "anything", true, true},
		{"tokA", "alice-tokyo", true, true},
		{"tokA", "alice-", false, true}, // must name a probe inside the namespace
		{"tokA", "bob-tokyo", false, true},
		{"nope", "alice-tokyo", false, false},
		{"", "x", false, false},
	}
	for _, c := range cases {
		ok, known := s.authorizeProbe(c.token, c.probe)
		if ok != c.ok || known != c.known {
			t.Errorf("authorizeProbe(%q, %q) = %v, %v; want %v, %v", c.token, c.probe, ok, known, c.ok, c.known)
		}
	}

	// Without a legacy secret, the empty token must not match it.
	s = &Server{cfg: config.Aggregator{TokenPrefixes: map[string]string{"tokA": "alice-"}}}
	if _, known := s.authorizeProbe("", ""); known {
		t.Error("empty token accepted with no legacy secret configured")
	}
}

func TestIncidentWindow(t *testing.T) {
	from, to, err := incidentWindow(url.Values{"from": {"100"}, "to": {"200"}})
	if err != nil || from != 100 || to != 200 {
		t.Fatalf("explicit: %d %d %v", from, to, err)
	}
	from, to, err = incidentWindow(url.Values{"months": {"2"}})
	if err != nil || to <= from || time.Unix(from, 0).In(types.CST).AddDate(0, 2, 0).Unix() != to {
		t.Fatalf("months: %d %d %v", from, to, err)
	}
	for _, q := range []url.Values{
		{"from": {"x"}, "to": {"1"}},
		{"from": {"5"}, "to": {"5"}},
		{"from": {"0"}, "to": {"99999999999"}},
		{"months": {"0"}},
	} {
		if _, _, err := incidentWindow(q); err == nil {
			t.Errorf("%v: expected error", q)
		}
	}
}

func TestTrafficPeak(t *testing.T) {
	var p trafficPeak
	base := time.Unix(1_700_000_040, 0)
	steps := []struct {
		at      time.Duration
		n       int
		wantOK  bool
		wantMax int
	}{
		{0, 3, true, 3},
		{10 * time.Second, 2, false, 0},
		{20 * time.Second, 5, true, 5},
		{61 * time.Second, 1, true, 1}, // next minute starts over
	}
	for i, st := range steps {
		ts, peak, ok := p.observe(base.Add(st.at), st.n)
		if ok != st.wantOK || (ok && (peak != st.wantMax || ts%60 != 0)) {
			t.Errorf("step %d: got %d %d %v", i, ts, peak, ok)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(2, time.Minute)
	if !l.allow("a") || !l.allow("a") || l.allow("a") {
		t.Fatal("limit of 2 not enforced")
	}
	if !l.allow("b") {
		t.Fatal("keys must be independent")
	}
}

func TestIsValidHost(t *testing.T) {
	for h, want := range map[string]bool{"bgm-status.ry.mk": true, "localhost:8080": true, "": false, "a b": false, "evil/x": false} {
		if isValidHost(h) != want {
			t.Errorf("isValidHost(%q) != %v", h, want)
		}
	}
}
