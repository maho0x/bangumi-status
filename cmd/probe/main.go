package main

import (
	"bangumi-status/internal/check"
	"bangumi-status/internal/types"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	interval := flag.Duration("interval", 60*time.Second, "check interval")
	flag.Parse()

	probeID := mustEnv("PROBE_ID")
	region := mustEnv("REGION")
	aggURL := mustEnv("AGG_URL")
	secret := mustEnv("INGEST_SECRET")

	cfg := check.Config{
		UserAgent: getenv("BGM_USER_AGENT", "bangumi-status-probe/1.0"),
		Cookie:    os.Getenv("BGM_COOKIE"),
		APIToken:  os.Getenv("BGM_API_TOKEN"),
		Timeout:   15 * time.Second,
	}
	r := check.NewRunner(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("probe %s region=%s interval=%s agg=%s", probeID, region, *interval, aggURL)

	// Run first check immediately, then on ticker.
	runOnce := func() {
		cctx, cc := context.WithTimeout(ctx, 45*time.Second)
		defer cc()
		outcome := r.RunAll(cctx)
		for i := range outcome.Results {
			outcome.Results[i].Probe = probeID
			outcome.Results[i].Region = region
		}
		if err := send(ctx, aggURL, secret, types.IngestPayload{
			Probe:       probeID,
			Region:      region,
			Results:     outcome.Results,
			OnlineCount: outcome.OnlineCount,
			OnlineTS:    outcome.OnlineTS,
		}); err != nil {
			log.Printf("send: %v", err)
			return
		}
		var ok, deg, down int
		for _, res := range outcome.Results {
			switch res.Status {
			case types.StatusOK:
				ok++
			case types.StatusDegraded:
				deg++
			case types.StatusDown:
				down++
			}
		}
		log.Printf("tick: ok=%d degraded=%d down=%d online=%d", ok, deg, down, outcome.OnlineCount)
	}

	runOnce()
	t := time.NewTicker(*interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runOnce()
		}
	}
}

func send(ctx context.Context, url, secret string, p types.IngestPayload) error {
	body, _ := json.Marshal(p)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &httpErr{Code: resp.StatusCode, Body: string(b)}
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

type httpErr struct {
	Code int
	Body string
}

func (e *httpErr) Error() string {
	return "ingest returned " + http.StatusText(e.Code) + ": " + e.Body
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("env %s is required", k)
	}
	return v
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
