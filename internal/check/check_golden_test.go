package check

import (
	"bangumi-status/internal/target"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type goldenRow struct {
	Scenario string `json:"scenario"`
	Mode     string `json:"mode"`
	Domain   string `json:"domain"`
	Kind     string `json:"kind"`
	URL      string `json:"url"`
	Cookie   bool   `json:"cookie"`
	Auth     string `json:"auth,omitempty"`
	Status   string `json:"status"`
	Code     int    `json:"code,omitempty"`
	Err      string `json:"err,omitempty"`
}

type goldenOut struct {
	Rows   []goldenRow    `json:"rows"`
	Online map[string]int `json:"online"`
}

func TestGolden(t *testing.T) {
	scenarios := []struct {
		name string
		code int
		body string
		fail bool
	}{
		{"ok-marker", 200, `<a href="/logout">Logout</a> online: 1234`, false},
		{"ok-nomarker", 200, `hello`, false},
		{"redirect", 302, ``, false},
		{"unauth", 401, ``, false},
		{"forbidden", 403, ``, false},
		{"notfound", 404, ``, false},
		{"server", 503, ``, false},
		{"neterr", 0, ``, true},
	}
	modes := []struct {
		name string
		cfg  Config
	}{
		{"full", Config{UserAgent: "ua", Cookie: "c=1", APIToken: "tok"}},
		{"cookie-only", Config{UserAgent: "ua", Cookie: "c=1"}},
		{"guest", Config{UserAgent: "ua"}},
	}
	out := goldenOut{Online: map[string]int{}}
	for _, sc := range scenarios {
		for _, m := range modes {
			var mu sync.Mutex
			seen := map[string]goldenRow{}
			r := NewRunner(m.cfg)
			r.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
				mu.Lock()
				seen[req.URL.String()] = goldenRow{
					URL:    req.URL.String(),
					Cookie: req.Header.Get("Cookie") != "",
					Auth:   req.Header.Get("Authorization"),
				}
				mu.Unlock()
				if sc.fail {
					return nil, errors.New("boom")
				}
				h := http.Header{}
				if sc.code == 302 {
					h.Set("Location", "/login")
				}
				return &http.Response{StatusCode: sc.code, Header: h, Body: io.NopCloser(strings.NewReader(sc.body)), Request: req}, nil
			})
			o := r.RunAll(context.Background())
			out.Online[sc.name+"/"+m.name] = o.OnlineCount
			for _, res := range o.Results {
				mu.Lock()
				seen = map[string]goldenRow{}
				mu.Unlock()
				tg, _ := target.Lookup(res.Domain, res.Kind)
				single, _ := r.run(context.Background(), tg)
				var row goldenRow
				for _, v := range seen {
					row = v
				}
				row.Scenario, row.Mode = sc.name, m.name
				row.Domain, row.Kind = single.Domain, string(single.Kind)
				row.Status, row.Code = string(single.Status), single.HTTPCode
				row.Err = single.Err
				if sc.fail {
					row.Err = "neterr"
				}
				if res.Status != single.Status || res.Err != single.Err {
					t.Fatalf("RunAll and run disagree for %s/%s", res.Domain, res.Kind)
				}
				out.Rows = append(out.Rows, row)
			}
		}
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		a, b := out.Rows[i], out.Rows[j]
		return fmt.Sprint(a.Scenario, a.Mode, a.Domain, a.Kind) < fmt.Sprint(b.Scenario, b.Mode, b.Domain, b.Kind)
	})
	got, _ := json.MarshalIndent(out, "", " ")
	const path = "testdata/golden.json"
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
		t.Fatalf("golden mismatch; diff %s against output:\n%s", path, got)
	}
}
