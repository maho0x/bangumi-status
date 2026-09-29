package server

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"time"

	"bangumi-status/internal/types"
)

// The Atom feed lists every non-ok day of the 30-day strip, newest first.

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	Link    []atomLink  `xml:"link"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Author  atomAuthor  `xml:"author"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	ID      string   `xml:"id"`
	Title   string   `xml:"title"`
	Link    atomLink `xml:"link"`
	Updated string   `xml:"updated"`
	Summary atomText `xml:"summary"`
	Content atomText `xml:"content"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if p := r.Header.Get("X-Forwarded-Proto"); p == "http" || p == "https" {
		scheme = p
	} else if r.TLS == nil {
		scheme = "http"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); isValidHost(fh) {
		host = fh
	}
	base := scheme + "://" + host

	body, err := s.feeds.GetOrLoad(base, func() ([]byte, error) {
		overall, ok := s.status.Snapshot()
		if !ok {
			return nil, errWarmingFeed
		}
		return buildAtomFeed(base, host, overall)
	})
	if err == errWarmingFeed {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		internalError(w, r, err)
		return
	}
	writeRaw(w, "application/atom+xml; charset=utf-8", "public, max-age=300", body)
}

var errWarmingFeed = fmt.Errorf("status cache is warming up")

func buildAtomFeed(base, host string, overall *types.Overall) ([]byte, error) {
	entries := []atomEntry{}
	for _, c := range overall.Components {
		for _, b := range c.Days {
			if b.Total == 0 || b.Status == types.StatusOK {
				continue
			}
			day, err := time.Parse(time.DateOnly, b.Day)
			if err != nil {
				continue
			}
			sev := "Degraded performance"
			if b.Status == types.StatusDown {
				sev = "Service disruption"
			}
			summary := fmt.Sprintf("%s on %s. %d down, %d degraded, %d ok of %d checks (%.1f%% uptime).",
				sev, b.Day, b.Down, b.Degrade, b.Total-b.Down-b.Degrade, b.Total, b.Uptime)
			entries = append(entries, atomEntry{
				ID:    fmt.Sprintf("tag:%s,%s:%s/%s", host, b.Day, c.Domain, c.Kind),
				Title: fmt.Sprintf("%s — %s", sev, c.Label),
				Link:  atomLink{Href: base + "/history", Rel: "alternate", Type: "text/html"},
				// End of day so the latest incident day sorts first.
				Updated: day.Add(24*time.Hour - time.Second).UTC().Format(time.RFC3339),
				Summary: atomText{Type: "text", Body: summary},
				Content: atomText{Type: "text", Body: summary},
			})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Updated > entries[j].Updated })
	entries = entries[:min(len(entries), 50)]

	updated := time.Now().UTC()
	if overall.UpdatedAt > 0 {
		updated = time.Unix(overall.UpdatedAt, 0).UTC()
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(atomFeed{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: "Bangumi Status · Incidents",
		Link: []atomLink{
			{Href: base + "/api/feed.atom", Rel: "self", Type: "application/atom+xml"},
			{Href: base + "/", Rel: "alternate", Type: "text/html"},
		},
		ID:      "tag:" + host + ":feed",
		Updated: updated.Format(time.RFC3339),
		Author:  atomAuthor{Name: "Bangumi Status"},
		Entries: entries,
	}); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// isValidHost rejects header values with characters illegal in a host name,
// preventing host-header injection via X-Forwarded-Host.
func isValidHost(h string) bool {
	if h == "" || len(h) > 253 {
		return false
	}
	for _, c := range h {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == ':') {
			return false
		}
	}
	return true
}
