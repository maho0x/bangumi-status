// Command probe checks every Bangumi target on an interval and reports the
// results to the aggregator.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bangumi-status/internal/check"
	"bangumi-status/internal/config"
	"bangumi-status/internal/jobs"
	"bangumi-status/internal/types"
)

func main() {
	interval := flag.Duration("interval", 60*time.Second, "check interval")
	flag.Parse()

	cfg, err := config.LoadProbe()
	if err != nil {
		slog.Error("probe config", "err", err)
		os.Exit(1)
	}
	runner := check.NewRunner(check.Config{UserAgent: cfg.UserAgent, Cookie: cfg.Cookie, APIToken: cfg.APIToken, Timeout: 15 * time.Second})
	mode := "auth+guest"
	if cfg.Cookie == "" && cfg.APIToken == "" {
		mode = "guest-only (no BGM_COOKIE / BGM_API_TOKEN)"
	}
	slog.Info("probe starting", "id", cfg.ID, "region", cfg.Region, "interval", *interval, "agg", cfg.AggURL, "mode", mode)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	client := &http.Client{Timeout: 20 * time.Second}
	jobs.Every(ctx, *interval, func(ctx context.Context) {
		checkCtx, done := context.WithTimeout(ctx, 45*time.Second)
		defer done()
		out := runner.RunAll(checkCtx)
		counts := map[types.Status]int{}
		for i := range out.Results {
			out.Results[i].Probe, out.Results[i].Region = cfg.ID, cfg.Region
			counts[out.Results[i].Status]++
		}
		payload := types.IngestPayload{Probe: cfg.ID, Region: cfg.Region, Results: out.Results, OnlineCount: out.OnlineCount, OnlineTS: out.OnlineTS}
		if err := send(ctx, client, cfg, payload); err != nil {
			slog.Warn("send failed", "err", err)
			return
		}
		slog.Info("tick", "ok", counts[types.StatusOK], "degraded", counts[types.StatusDegraded], "down", counts[types.StatusDown], "online", out.OnlineCount)
	})
}

func send(ctx context.Context, client *http.Client, cfg config.Probe, p types.IngestPayload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.AggURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.IngestSecret)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("ingest returned %s: %s", http.StatusText(resp.StatusCode), msg)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
