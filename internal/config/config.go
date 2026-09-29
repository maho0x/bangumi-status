// Package config reads every environment variable the binaries accept, so
// the rest of the code receives plain values instead of calling os.Getenv.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"bangumi-status/internal/region"
)

// Aggregator is the aggregator's configuration.
type Aggregator struct {
	Addr  string
	DBDSN string

	// IngestSecret is the legacy single shared token: it accepts any probe-id.
	IngestSecret string
	// TokenPrefixes maps a third-party token to the probe-id prefix it may
	// report under, so operators self-pick ids inside their own namespace.
	TokenPrefixes map[string]string

	TelegramToken  string
	TelegramChatID string

	WikiStatsURL      string
	WikiStatsInterval time.Duration
	BangumiUserAgent  string
	BangumiCookie     string

	StatusCachePath string
}

// LoadAggregator reads the aggregator configuration. addr and dsn come from
// flags (dsn defaults to DB_DSN).
func LoadAggregator(addr, dsn string) (Aggregator, error) {
	prefixes, err := ParseTokenPrefixes(os.Getenv("INGEST_SECRETS"))
	if err != nil {
		return Aggregator{}, fmt.Errorf("parse INGEST_SECRETS: %w", err)
	}
	c := Aggregator{
		Addr:              addr,
		DBDSN:             dsn,
		IngestSecret:      os.Getenv("INGEST_SECRET"),
		TokenPrefixes:     prefixes,
		TelegramToken:     os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:    os.Getenv("TELEGRAM_CHAT_ID"),
		WikiStatsURL:      env("WIKI_STATS_URL", "https://chii.in/wiki/stats"),
		WikiStatsInterval: duration("WIKI_STATS_INTERVAL", time.Hour, 5*time.Minute),
		BangumiUserAgent:  env("BGM_USER_AGENT", "BangumiStatus/1.0 (+https://bgm-status.ry.mk)"),
		BangumiCookie:     cookie(),
		StatusCachePath:   env("STATUS_CACHE_PATH", "/var/lib/bangumi-status/status-cache.json"),
	}
	if c.IngestSecret == "" && len(c.TokenPrefixes) == 0 {
		return c, errors.New("INGEST_SECRET or INGEST_SECRETS env var is required")
	}
	if c.DBDSN == "" {
		return c, errors.New("DB_DSN env var or -db-dsn flag is required")
	}
	return c, nil
}

// Probe is the probe's configuration.
type Probe struct {
	ID           string
	Region       string
	AggURL       string
	IngestSecret string
	UserAgent    string
	Cookie       string
	APIToken     string
}

// LoadProbe reads the probe configuration and validates REGION.
func LoadProbe() (Probe, error) {
	c := Probe{
		ID:           os.Getenv("PROBE_ID"),
		AggURL:       os.Getenv("AGG_URL"),
		IngestSecret: os.Getenv("INGEST_SECRET"),
		UserAgent:    env("BGM_USER_AGENT", "bangumi-status-probe/1.0"),
		Cookie:       os.Getenv("BGM_COOKIE"),
		APIToken:     os.Getenv("BGM_API_TOKEN"),
	}
	for k, v := range map[string]string{"PROBE_ID": c.ID, "REGION": os.Getenv("REGION"), "AGG_URL": c.AggURL, "INGEST_SECRET": c.IngestSecret} {
		if v == "" {
			return c, fmt.Errorf("env %s is required", k)
		}
	}
	raw := os.Getenv("REGION")
	code, ok := region.Normalize(raw)
	if !ok {
		return c, fmt.Errorf("REGION %q is not a valid ISO 3166-1 alpha-2 country code (e.g. jp, cn, us, de, sg)", raw)
	}
	c.Region = code
	return c, nil
}

// ParseTokenPrefixes parses a third-party token allowlist of the form
//
//	token1:prefix1;token2:prefix2
//
// Each token is bound to a required probe-id prefix. Prefixes must be
// non-empty (an empty one could impersonate the maintainer) and must not
// prefix each other (one operator's namespace would contain another's).
func ParseTokenPrefixes(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	out := map[string]string{}
	for _, group := range strings.Split(raw, ";") {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		tok, prefix, ok := strings.Cut(group, ":")
		tok, prefix = strings.TrimSpace(tok), strings.TrimSpace(prefix)
		if !ok || tok == "" || prefix == "" {
			return nil, fmt.Errorf("invalid group %q (want token:prefix)", group)
		}
		if _, dup := out[tok]; dup {
			return nil, errors.New("duplicate token entry")
		}
		out[tok] = prefix
	}
	for tokA, prefA := range out {
		for tokB, prefB := range out {
			if tokA != tokB && strings.HasPrefix(prefB, prefA) {
				return nil, fmt.Errorf("prefix %q overlaps %q (one operator's namespace contains another)", prefA, prefB)
			}
		}
	}
	return out, nil
}

func env(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func duration(name string, fallback, floor time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < floor {
		slog.Warn("invalid duration, using default", "env", name, "value", raw, "default", fallback)
		return fallback
	}
	return d
}

// cookie reads BGM_COOKIE, or builds the header from the browser-export JSON
// in BGM_COOKIE_JSON.
func cookie() string {
	if c := strings.TrimSpace(os.Getenv("BGM_COOKIE")); c != "" {
		return c
	}
	raw := strings.TrimSpace(os.Getenv("BGM_COOKIE_JSON"))
	if raw == "" {
		return ""
	}
	var cookies []struct{ Name, Value string }
	if err := json.Unmarshal([]byte(raw), &cookies); err != nil {
		slog.Warn("ignoring invalid BGM_COOKIE_JSON", "err", err)
		return ""
	}
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		if c.Name != "" {
			parts = append(parts, c.Name+"="+c.Value)
		}
	}
	return strings.Join(parts, "; ")
}
