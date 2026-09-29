// Package wikistats scrapes the daily site statistics chii.in publishes on
// /wiki/stats (embedded in the page as a CHART_SETS JavaScript literal).
package wikistats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bangumi-status/internal/types"
)

// Store is where scraped days are kept.
type Store interface {
	UpsertWikiStats(ctx context.Context, points []types.WikiStatsPoint, chartSets []byte) (int, string, error)
}

type Scraper struct {
	URL       string
	UserAgent string
	Cookie    string // the page requires a signed-in session
	Store     Store
}

// Scrape fetches the page once and stores every day it contains.
func (s *Scraper) Scrape(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", s.UserAgent)
	req.Header.Set("Cookie", s.Cookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	raw, points, err := Parse(string(body))
	if err != nil {
		return err
	}
	n, latest, err := s.Store.UpsertWikiStats(ctx, points, raw)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	slog.Info("wiki stats scraped", "days", n, "latest", latest)
	return nil
}

// Parse extracts the CHART_SETS object from the page and returns it verbatim
// together with its daily rows.
func Parse(page string) ([]byte, []types.WikiStatsPoint, error) {
	const marker = "var CHART_SETS = "
	i := strings.Index(page, marker)
	if i < 0 {
		return nil, nil, errors.New("CHART_SETS not found")
	}
	rest := page[i+len(marker):]
	end, err := objectEnd(rest)
	if err != nil {
		return nil, nil, err
	}
	raw := []byte(rest[:end])
	var sets map[string]struct {
		Data []types.WikiStatsPoint `json:"data"`
	}
	if err := json.Unmarshal(raw, &sets); err != nil {
		return nil, nil, fmt.Errorf("parse CHART_SETS: %w", err)
	}
	// Every set carries the same daily rows; take the first non-empty one.
	for _, key := range []string{"register", "collection", "topics", "replies"} {
		if set := sets[key]; len(set.Data) > 0 {
			return raw, set.Data, nil
		}
	}
	return nil, nil, errors.New("CHART_SETS contains no daily data")
}

// objectEnd returns the index just past the JSON object that starts at the
// first '{' in s, honouring strings and escapes.
func objectEnd(s string) (int, error) {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return 0, errors.New("CHART_SETS JSON object not found")
	}
	depth, inString, escaped := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case escaped:
			escaped = false
		case inString:
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
		case c == '"':
			inString = true
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, errors.New("unterminated CHART_SETS JSON object")
}
