package check

import (
	"bangumi-status/internal/types"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// onlineRe matches the "online: 12345" badge rendered for signed-in users on
// bangumi.tv. It is a site-wide counter so a single reading per tick is enough.
var onlineRe = regexp.MustCompile(`online:\s*(\d+)`)

// Config holds everything a probe needs to run its checks.
type Config struct {
	UserAgent string
	Cookie    string // full Cookie header value
	APIToken  string // Bearer token for api.bgm.tv
	Timeout   time.Duration
}

// Runner executes checks for all (domain, kind) pairs.
type Runner struct {
	cfg    Config
	client *http.Client
}

func NewRunner(cfg Config) *Runner {
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          20,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: cfg.Timeout,
		ForceAttemptHTTP2:     true,
	}
	// Don't follow redirects silently — we want to detect cookie-expired 302s.
	client := &http.Client{
		Transport: t,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Runner{cfg: cfg, client: client}
}

// Outcome bundles per-component results with any side-channel observations
// gathered during the same HTTP round-trips (e.g. the bangumi.tv online badge).
type Outcome struct {
	Results     []types.Result
	OnlineCount int   // 0 if not captured this tick
	OnlineTS    int64 // unix seconds at which OnlineCount was observed
}

// RunAll executes every component check concurrently and returns results.
// Auth-kind checks are skipped entirely when no Bangumi credentials are
// configured, so a probe with no cookie/token contributes only Guest data.
func (r *Runner) RunAll(ctx context.Context) Outcome {
	all := types.AllComponents()
	comps := all
	if r.cfg.Cookie == "" && r.cfg.APIToken == "" {
		comps = make([]types.Component, 0, len(all))
		for _, c := range all {
			if c.Kind == types.KindAuth {
				continue
			}
			comps = append(comps, c)
		}
	}
	type item struct {
		res    types.Result
		online int
	}
	ch := make(chan item, len(comps))
	for _, c := range comps {
		c := c
		go func() {
			res, online := r.run(ctx, c.Domain, c.Kind)
			ch <- item{res: res, online: online}
		}()
	}
	out := Outcome{Results: make([]types.Result, 0, len(comps))}
	for range comps {
		it := <-ch
		out.Results = append(out.Results, it.res)
		if it.online > 0 && out.OnlineCount == 0 {
			out.OnlineCount = it.online
			out.OnlineTS = it.res.TS
		}
	}
	return out
}

// run executes one check. `site` is the logical site (e.g. "bgm.tv",
// "api.bgm.tv"). The guest/auth semantics differ between the web sites and
// the JSON API. The second return is the bangumi.tv "online: N" badge value
// if it was parseable from the response body, else 0.
func (r *Runner) run(ctx context.Context, site string, kind types.Kind) (types.Result, int) {
	start := time.Now()
	res := types.Result{
		TS:     start.Unix(),
		Domain: site,
		Kind:   kind,
	}

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.invalid/", nil)
	httpReq.Header.Set("User-Agent", r.cfg.UserAgent)
	httpReq.Header.Set("Accept-Encoding", "identity")

	var expectMarker string
	isAPI := site == "api.bgm.tv"
	isNext := site == "next.bgm.tv"
	isNextAPI := site == "next.bgm.tv/p1"

	switch {
	case isNextAPI:
		u := "https://next.bgm.tv/p1/subjects/1"
		httpReq.URL, _ = httpReq.URL.Parse(u)
		httpReq.Host = ""
		httpReq.Header.Set("Accept", "application/json")
	case isNext:
		u := "https://next.bgm.tv/"
		httpReq.URL, _ = httpReq.URL.Parse(u)
		httpReq.Host = ""
		expectMarker = "bangumi"
	case isAPI && kind == types.KindGuest:
		// Unauthenticated public endpoint.
		u := "https://api.bgm.tv/v0/subjects/1"
		httpReq.URL, _ = httpReq.URL.Parse(u)
		httpReq.Host = ""
		httpReq.Header.Set("Accept", "application/json")
	case isAPI && kind == types.KindAuth:
		// Same public endpoint as guest but with Bearer token — /v0/me was
		// too volatile (frequent 401s unrelated to actual availability).
		u := "https://api.bgm.tv/v0/subjects/1"
		httpReq.URL, _ = httpReq.URL.Parse(u)
		httpReq.Host = ""
		httpReq.Header.Set("Accept", "application/json")
		if r.cfg.APIToken != "" {
			httpReq.Header.Set("Authorization", "Bearer "+r.cfg.APIToken)
		}
	case kind == types.KindGuest:
		u := "https://" + site + "/"
		httpReq.URL, _ = httpReq.URL.Parse(u)
		httpReq.Host = ""
	case kind == types.KindAuth:
		u := "https://" + site + "/"
		httpReq.URL, _ = httpReq.URL.Parse(u)
		httpReq.Host = ""
		if r.cfg.Cookie != "" {
			httpReq.Header.Set("Cookie", r.cfg.Cookie)
		}
		// Bangumi templates show a "logout" link once signed in; missing
		// marker means the cookie is invalid even if we got a 200.
		expectMarker = "logout"
	}

	resp, err := r.client.Do(httpReq)
	res.Latency = int(time.Since(start).Milliseconds())
	if err != nil {
		res.Status = types.StatusDown
		res.Err = trimErr(err.Error())
		return res, 0
	}
	defer resp.Body.Close()
	res.HTTPCode = resp.StatusCode

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	_, _ = io.Copy(io.Discard, resp.Body)

	// Piggy-back the site-wide "online: N" counter off the authenticated
	// bangumi.tv homepage (it only renders for signed-in users, and the cookie
	// domain is .bangumi.tv so bgm.tv/chii.in responses don't carry it).
	online := 0
	if site == "bangumi.tv" && kind == types.KindAuth && resp.StatusCode == 200 {
		if m := onlineRe.FindSubmatch(body); m != nil {
			if n, perr := strconv.Atoi(string(m[1])); perr == nil {
				online = n
			}
		}
	}

	switch {
	case isAPI:
		if resp.StatusCode >= 500 {
			res.Status = types.StatusDown
		} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
			res.Status = types.StatusDegraded
			res.Err = fmt.Sprintf("auth %d", resp.StatusCode)
		} else if resp.StatusCode >= 400 {
			res.Status = types.StatusDegraded
		} else {
			res.Status = types.StatusOK
		}
	case isNextAPI:
		if resp.StatusCode >= 500 {
			res.Status = types.StatusDown
		} else if resp.StatusCode >= 400 {
			res.Status = types.StatusDegraded
		} else {
			res.Status = types.StatusOK
		}
	case isNext:
		if resp.StatusCode >= 500 {
			res.Status = types.StatusDown
		} else if resp.StatusCode >= 400 {
			res.Status = types.StatusDegraded
		} else if expectMarker != "" && !strings.Contains(strings.ToLower(string(body)), expectMarker) {
			res.Status = types.StatusDegraded
			res.Err = "marker missing"
		} else {
			res.Status = types.StatusOK
		}
	case kind == types.KindGuest:
		if resp.StatusCode >= 500 {
			res.Status = types.StatusDown
		} else if resp.StatusCode >= 400 {
			res.Status = types.StatusDegraded
		} else {
			res.Status = types.StatusOK
		}
	case kind == types.KindAuth:
		if resp.StatusCode >= 500 {
			res.Status = types.StatusDown
		} else if resp.StatusCode == 301 || resp.StatusCode == 302 {
			res.Status = types.StatusDegraded
			res.Err = "redirect: " + resp.Header.Get("Location")
		} else if resp.StatusCode >= 400 {
			res.Status = types.StatusDegraded
		} else if expectMarker != "" && !strings.Contains(strings.ToLower(string(body)), expectMarker) {
			res.Status = types.StatusDegraded
			res.Err = "marker missing"
		} else {
			res.Status = types.StatusOK
		}
	}

	if res.Status == types.StatusOK && res.Latency > 5000 {
		res.Status = types.StatusDegraded
		res.Err = "slow"
	}
	return res, online
}

func trimErr(s string) string {
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
