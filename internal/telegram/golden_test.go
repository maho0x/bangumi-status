package telegram

import (
	"bangumi-status/internal/types"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite golden files")

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type memStore map[string]string

func (m memStore) GetConfig(_ context.Context, k string) (string, bool, error) {
	v, ok := m[k]
	return v, ok, nil
}
func (m memStore) SetConfig(_ context.Context, k, v string) error { m[k] = v; return nil }

var (
	reDur   = regexp.MustCompile(`\d+ (小时|分|秒)( \d+ (分|秒))*`)
	reClock = regexp.MustCompile(`(\d{2}-\d{2} )?\d{2}:\d{2}(:\d{2})?`)
	reMonth = regexp.MustCompile(`[A-Z][a-z]{2} \d{2} T`)
)

func norm(s string) string {
	s = reDur.ReplaceAllString(s, "DUR")
	s = reClock.ReplaceAllString(s, "T")
	return reMonth.ReplaceAllString(s, "DATE T")
}

// harness wires a Telegram to a fake Bot API and records every call.
type harness struct {
	tg       *Telegram
	log      []string
	nextID   int
	editGone bool
}

func newHarness() *harness {
	h := &harness{nextID: 100}
	h.tg = New("TOKEN", "CHAT", memStore{})
	h.tg.api.http.Transport = rtFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		line := method
		for _, k := range []string{"message_id", "reply_to_message_id"} {
			if v := form.Get(k); v != "" {
				line += " " + k + "=" + v
			}
		}
		if txt := form.Get("text"); txt != "" {
			line += "\n" + norm(txt)
		}
		h.log = append(h.log, line)
		resp := `{"ok":true,"result":{"message_id":0}}`
		switch method {
		case "sendMessage":
			h.nextID++
			resp = fmt.Sprintf(`{"ok":true,"result":{"message_id":%d}}`, h.nextID)
		case "editMessageText":
			if h.editGone {
				resp = `{"ok":false,"error_code":400,"description":"Bad Request: message to edit not found"}`
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(resp)), Header: http.Header{}}, nil
	})
	return h
}

// tick feeds one Process call. spec is "domain|kind=status[@sinceAgo]" items.
func (h *harness) tick(spec ...string) {
	var comps []types.ComponentStatus
	for _, s := range spec {
		kv := strings.SplitN(s, "=", 2)
		dk := strings.SplitN(kv[0], "|", 2)
		st, since := kv[1], int64(0)
		if i := strings.IndexByte(st, '@'); i >= 0 {
			var ago int64
			fmt.Sscan(st[i+1:], &ago)
			since = time.Now().Unix() - ago
			st = st[:i]
		}
		comps = append(comps, types.ComponentStatus{Domain: dk[0], Kind: types.Kind(dk[1]), Status: types.Status(st), Since: since})
	}
	h.tg.Process(comps)
}

const (
	bgm   = "bgm.tv|auth"
	bgt   = "bangumi.tv|auth"
	guest = "bgm.tv|guest"
	api   = "api.bgm.tv|auth"
	next  = "next.bgm.tv/p1|auth"
)

func TestTelegramGolden(t *testing.T) {
	out := map[string][]string{}
	run := func(name string, f func(h *harness)) {
		h := newHarness()
		f(h)
		out[name] = h.log
	}
	all := func(st map[string]string) []string {
		var s []string
		for _, k := range []string{bgm, bgt, guest, api, next} {
			v := "ok"
			if x, ok := st[k]; ok {
				v = x
			}
			s = append(s, k+"="+v)
		}
		return s
	}

	run("outage-merge-recover", func(h *harness) {
		h.tick(all(nil)...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(map[string]string{bgm: "down", bgt: "down"})...)
		h.tick(all(map[string]string{bgm: "down", bgt: "down"})...)
		h.tick(all(nil)...)
		h.tick(all(nil)...)
	})
	run("degraded-then-down-then-recover", func(h *harness) {
		h.tick(all(nil)...)
		h.tick(all(map[string]string{bgm: "degraded"})...)
		h.tick(all(map[string]string{bgm: "degraded"})...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(nil)...)
		h.tick(all(nil)...)
	})
	run("degraded-silent-recovery", func(h *harness) {
		h.tick(all(nil)...)
		h.tick(all(map[string]string{bgt: "degraded"})...)
		h.tick(all(map[string]string{bgt: "degraded"})...)
		h.tick(all(nil)...)
		h.tick(all(nil)...)
	})
	run("restart-restore", func(h *harness) {
		h.tick(all(map[string]string{bgm: "down@600", bgt: "degraded@300"})...)
		h.tick(all(map[string]string{bgm: "down@600", bgt: "degraded@300"})...)
		h.tick(all(nil)...)
		h.tick(all(nil)...)
	})
	run("non-alertable", func(h *harness) {
		h.tick(all(nil)...)
		for i := 0; i < 3; i++ {
			h.tick(all(map[string]string{guest: "down", api: "down", next: "down"})...)
		}
		h.tick(all(nil)...)
		h.tick(all(nil)...)
	})
	run("flapping", func(h *harness) {
		h.tick(all(nil)...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(nil)...)
		h.tick(all(map[string]string{bgm: "degraded"})...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(nil)...)
	})
	run("no-merge-window", func(h *harness) {
		h.tg.mergeWindow = 0
		h.tick(all(nil)...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(map[string]string{bgm: "down", bgt: "down"})...)
		h.tick(all(map[string]string{bgm: "down", bgt: "down"})...)
		h.tick(all(map[string]string{bgt: "down"})...)
		h.tick(all(map[string]string{bgt: "down"})...)
		h.tick(all(nil)...)
		h.tick(all(nil)...)
	})
	run("edit-gone", func(h *harness) {
		h.tick(all(nil)...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.tick(all(map[string]string{bgm: "down"})...)
		h.editGone = true
		h.tick(all(map[string]string{bgm: "down", bgt: "down"})...)
		h.tick(all(map[string]string{bgm: "down", bgt: "down"})...)
		h.tick(all(nil)...)
		h.tick(all(nil)...)
	})

	overall := &types.Overall{
		Components: []types.ComponentStatus{
			{Domain: "bgm.tv", Kind: types.KindAuth, Status: types.StatusDown, Uptime: 97.1234},
			{Domain: "bgm.tv", Kind: types.KindGuest, Status: types.StatusOK, Uptime: 99.5},
			{Domain: "bangumi.tv", Kind: types.KindAuth, Status: types.StatusOK, Uptime: 99.999},
			{Domain: "next.bgm.tv/p1", Kind: types.KindAuth, Status: types.StatusDegraded, Uptime: 88},
			{Domain: "api.bgm.tv", Kind: types.KindAuth, Status: types.StatusDegraded, Uptime: 0},
		},
		Probes: []types.Probe{{Name: "a", Online: true}, {Name: "b"}, {Name: "c", Online: true}},
	}
	run("summary", func(h *harness) {
		h.tg.UpdateSummary(overall)
		h.tg.UpdateSummary(&types.Overall{})
		h.editGone = true
		h.tg.UpdateSummary(overall)
	})
	run("daily", func(h *harness) {
		h.tg.SendDailyReport("2026-03-01", []DailyReportItem{{Domain: "bgm.tv", Status: types.StatusOK, Uptime: 100}, {Domain: "bangumi.tv", Status: types.StatusOK, Uptime: 99.9}})
		h.tg.SendDailyReport("2026-03-02", []DailyReportItem{{Domain: "bgm.tv", Status: types.StatusDown, Uptime: 91.2345}, {Domain: "bangumi.tv", Status: types.StatusDegraded, Uptime: 99.5}})
	})

	got, _ := json.MarshalIndent(out, "", " ")
	const path = "testdata/telegram_golden.json"
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
