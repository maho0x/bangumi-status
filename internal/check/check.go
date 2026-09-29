// Package check runs one round of health checks against every target in the
// registry and classifies each response.
package check

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"bangumi-status/internal/target"
	"bangumi-status/internal/types"
)

// onlineRe matches the site-wide "online: 12345" badge.
var onlineRe = regexp.MustCompile(`online:\s*(\d+)`)

// slowAfter turns an otherwise healthy but slow response into a degradation.
const slowAfter = 5 * time.Second

// Config holds everything a probe needs to run its checks.
type Config struct {
	UserAgent string
	Cookie    string // full Cookie header value
	APIToken  string // Bearer token for the JSON APIs
	Timeout   time.Duration
}

// Runner executes checks for every target.
type Runner struct {
	cfg    Config
	client *http.Client
}

func NewRunner(cfg Config) *Runner {
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          20,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ResponseHeaderTimeout: cfg.Timeout,
		ForceAttemptHTTP2:     true,
	}
	return &Runner{cfg: cfg, client: &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		// Never follow redirects: an expired cookie shows up as a 302 to login.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Outcome bundles per-component results with the online badge observed
// during the same round-trips.
type Outcome struct {
	Results     []types.Result
	OnlineCount int   // 0 if not captured this tick
	OnlineTS    int64 // unix seconds at which OnlineCount was observed
}

// guestOnly reports whether no Bangumi credential is configured, in which case
// authenticated targets are skipped entirely.
func (r *Runner) guestOnly() bool { return r.cfg.Cookie == "" && r.cfg.APIToken == "" }

// RunAll checks every target concurrently.
func (r *Runner) RunAll(ctx context.Context) Outcome {
	var targets []target.Target
	for _, t := range target.All {
		if t.Kind == types.KindAuth && r.guestOnly() {
			continue
		}
		targets = append(targets, t)
	}
	results := make([]types.Result, len(targets))
	online := make([]int, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Go(func() { results[i], online[i] = r.run(ctx, t) })
	}
	wg.Wait()

	out := Outcome{Results: results}
	for i, n := range online {
		if n > 0 {
			out.OnlineCount, out.OnlineTS = n, results[i].TS
			break
		}
	}
	return out
}

// run checks one target. The second return value is the online badge, or 0.
func (r *Runner) run(ctx context.Context, t target.Target) (types.Result, int) {
	start := time.Now()
	res := types.Result{TS: start.Unix(), Domain: t.Domain, Kind: t.Kind}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		res.Status, res.Err = types.StatusDown, trimErr(err.Error())
		return res, 0
	}
	req.Header.Set("User-Agent", r.cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "identity")
	if t.JSON {
		req.Header.Set("Accept", "application/json")
	}
	r.authorize(req, t.Credential)

	resp, err := r.client.Do(req)
	res.Latency = int(time.Since(start).Milliseconds())
	if err != nil {
		res.Status, res.Err = types.StatusDown, trimErr(err.Error())
		return res, 0
	}
	defer resp.Body.Close()
	res.HTTPCode = resp.StatusCode
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	_, _ = io.Copy(io.Discard, resp.Body)

	res.Status, res.Err = classify(t, resp, body)
	if res.Status == types.StatusOK && res.Latency > int(slowAfter.Milliseconds()) {
		res.Status, res.Err = types.StatusDegraded, "slow"
	}

	online := 0
	if t.Online && resp.StatusCode == http.StatusOK {
		if m := onlineRe.FindSubmatch(body); m != nil {
			online, _ = strconv.Atoi(string(m[1]))
		}
	}
	return res, online
}

func (r *Runner) authorize(req *http.Request, c target.Credential) {
	bearer := func() bool {
		if r.cfg.APIToken == "" {
			return false
		}
		req.Header.Set("Authorization", "Bearer "+r.cfg.APIToken)
		return true
	}
	cookie := func() {
		if r.cfg.Cookie != "" {
			req.Header.Set("Cookie", r.cfg.Cookie)
		}
	}
	switch c {
	case target.Cookie:
		cookie()
	case target.Bearer:
		bearer()
	case target.BearerOrCookie:
		if !bearer() {
			cookie()
		}
	}
}

// classify maps a response to a status and an optional error note.
func classify(t target.Target, resp *http.Response, body []byte) (types.Status, string) {
	code := resp.StatusCode
	switch {
	case code >= 500:
		return types.StatusDown, ""
	case t.AuthErrors && (code == http.StatusUnauthorized || code == http.StatusForbidden):
		return types.StatusDegraded, fmt.Sprintf("auth %d", code)
	case t.Redirects && (code == http.StatusMovedPermanently || code == http.StatusFound):
		return types.StatusDegraded, "redirect: " + resp.Header.Get("Location")
	case code >= 400:
		return types.StatusDegraded, ""
	case t.Marker != "" && !strings.Contains(strings.ToLower(string(body)), t.Marker):
		return types.StatusDegraded, "marker missing"
	}
	return types.StatusOK, ""
}

func trimErr(s string) string {
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
