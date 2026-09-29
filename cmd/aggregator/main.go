// Command aggregator receives probe results, serves the status page and API,
// and announces outages on Telegram.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bangumi-status/internal/config"
	"bangumi-status/internal/jobs"
	"bangumi-status/internal/server"
	"bangumi-status/internal/status"
	"bangumi-status/internal/store"
	"bangumi-status/internal/telegram"
	"bangumi-status/internal/types"
	"bangumi-status/internal/wikistats"
	"bangumi-status/web"
)

// retention is how long raw checks are kept; incident history outlives it in
// the incidents table.
const retention = 35 * 24 * time.Hour

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dsn := flag.String("db-dsn", os.Getenv("DB_DSN"), "database DSN (postgres://...)")
	flag.Parse()
	if err := run(*addr, *dsn); err != nil {
		slog.Error("aggregator failed", "err", err)
		os.Exit(1)
	}
}

func run(addr, dsn string) error {
	cfg, err := config.LoadAggregator(addr, dsn)
	if err != nil {
		return err
	}
	if n := len(cfg.TokenPrefixes); n > 0 {
		slog.Info("loaded third-party tokens (prefix-namespaced)", "count", n)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	st, err := store.Open(ctx, cfg.DBDSN)
	if err != nil {
		return err
	}
	defer st.Close()
	// Inserts fail without a partition for the current day.
	if err := st.EnsurePartitions(ctx, 2); err != nil {
		return err
	}

	tg := telegram.New(cfg.TelegramToken, cfg.TelegramChatID, st)
	if tg.Enabled() {
		slog.Info("telegram notifications enabled")
	}
	srv := server.New(ctx, cfg, st, web.FS())
	svc := status.New(st, cfg.StatusCachePath, tg.UpdateSummary, srv.StatusChanged)
	srv.SetStatus(svc)

	go svc.Run(ctx)
	go svc.Backfill(ctx)
	go jobs.Every(ctx, 6*time.Hour, func(ctx context.Context) {
		if err := st.EnsurePartitions(ctx, 2); err != nil {
			slog.Error("ensure partitions", "err", err)
		}
		if n, err := st.DropPartitionsBefore(ctx, time.Now().Add(-retention)); err != nil {
			slog.Error("retention", "err", err)
		} else if n > 0 {
			slog.Info("retention: dropped partitions", "count", n)
		}
	})
	go jobs.Every(ctx, time.Hour, srv.PurgeReactions)
	go jobs.Every(ctx, time.Minute, srv.RecordTraffic)
	if tg.Enabled() {
		// A fixed cadence, independent of how long snapshot refreshes take,
		// keeps the notifier's two-observation debounce meaningful.
		go jobs.Every(ctx, 20*time.Second, func(ctx context.Context) {
			comps, err := svc.Rollups(ctx)
			if err != nil {
				slog.Error("notifier rollups", "err", err)
				return
			}
			tg.Process(comps)
		})
		go jobs.Daily(ctx, types.CST, func(ctx context.Context, day time.Time) { dailyReport(ctx, st, tg, day) })
	}
	if cfg.BangumiCookie == "" {
		slog.Info("wiki stats scraper disabled: BGM_COOKIE or BGM_COOKIE_JSON is required")
	} else {
		scraper := &wikistats.Scraper{URL: cfg.WikiStatsURL, UserAgent: cfg.BangumiUserAgent, Cookie: cfg.BangumiCookie, Store: st}
		go jobs.Every(ctx, cfg.WikiStatsInterval, func(ctx context.Context) {
			if err := scraper.Scrape(ctx); err != nil {
				slog.Warn("wiki stats scrape", "err", err)
			}
		})
	}

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second, // lifted per request for event streams
	}
	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Addr, "db", redactDSN(cfg.DBDSN))
		errc <- httpSrv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	return httpSrv.Shutdown(shutdown)
}

// dailyReport posts yesterday's authenticated uptime for the main sites.
func dailyReport(ctx context.Context, st *store.Store, tg *telegram.Telegram, day time.Time) {
	var items []telegram.DailyReportItem
	for _, domain := range []string{"bgm.tv", "bangumi.tv"} {
		b, err := st.DaySummary(ctx, domain, types.KindAuth, day)
		if err != nil {
			slog.Error("daily report", "domain", domain, "err", err)
			continue
		}
		if b.Total > 0 {
			items = append(items, telegram.DailyReportItem{Domain: domain, Status: b.Status, Uptime: b.Uptime})
		}
	}
	tg.SendDailyReport(day.Format(time.DateOnly), items)
}

// redactDSN masks the password so the DSN can be logged.
func redactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "***"
	}
	if _, ok := u.User.Password(); ok {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.String()
}
