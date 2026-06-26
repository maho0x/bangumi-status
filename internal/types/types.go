package types

import "time"

type Status string

const (
	StatusOK       Status = "ok"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
)

type Kind string

const (
	KindGuest Kind = "guest"
	KindAuth  Kind = "auth"
)

// SiteConfig describes which kinds are monitored for a domain.
type SiteConfig struct {
	Domain string
	Kinds  []Kind
}

// SiteConfigs is the authoritative list of monitored sites and their kinds.
var SiteConfigs = []SiteConfig{
	{"bgm.tv", []Kind{KindGuest, KindAuth}},
	{"bangumi.tv", []Kind{KindGuest, KindAuth}},
	{"chii.in", []Kind{KindGuest, KindAuth}},
	{"next.bgm.tv/p1", []Kind{KindGuest, KindAuth}},
	{"api.bgm.tv", []Kind{KindGuest, KindAuth}},
}

// Sites returns the ordered domain list (UI grouping order).
var Sites = func() []string {
	s := make([]string, len(SiteConfigs))
	for i, c := range SiteConfigs {
		s[i] = c.Domain
	}
	return s
}()

// Kinds is the union of all kinds across sites (kept for compatibility).
var Kinds = []Kind{KindGuest, KindAuth}

// Result is one probe observation.
type Result struct {
	TS       int64  `json:"ts"`
	Probe    string `json:"probe"`
	Region   string `json:"region"`
	Domain   string `json:"domain"`
	Kind     Kind   `json:"kind"`
	Status   Status `json:"status"`
	Latency  int    `json:"latency_ms"`
	HTTPCode int    `json:"http_code,omitempty"`
	Err      string `json:"err,omitempty"`
}

type IngestPayload struct {
	Probe       string   `json:"probe"`
	Region      string   `json:"region"`
	Results     []Result `json:"results"`
	OnlineCount int      `json:"online_count,omitempty"` // bangumi.tv signed-in "online: N" badge
	OnlineTS    int64    `json:"online_ts,omitempty"`
}

// OnlinePoint is one sample of the bangumi.tv "online: N" counter.
//
// For raw (per-minute) series Count is the sampled value and Low/Peak are
// omitted. For bucketed series Count is the per-bucket average (the trend),
// while Low/Peak are the bucket's min/max so the chart can draw a range band
// around the trend line — a single spiky minute lifts Peak without distorting
// the average.
type OnlinePoint struct {
	TS    int64 `json:"ts"`
	Count int   `json:"count"`
	Low   int   `json:"low,omitempty"`
	Peak  int   `json:"peak,omitempty"`
}

// WikiStatsPoint is one daily row scraped from chii.in/wiki/stats.
type WikiStatsPoint struct {
	Title           string `json:"title"`
	Date            string `json:"date"`
	Timestamp       int64  `json:"timestamp"`
	RegisterTotal   int    `json:"register_total"`
	CollectionTotal int    `json:"collection_total"`
	TopicTotal      int    `json:"topic_total"`
	ReplyTotal      int    `json:"reply_total"`
	Collection1     int    `json:"collection_1"`
	Collection2     int    `json:"collection_2"`
	Collection3     int    `json:"collection_3"`
	Collection4     int    `json:"collection_4"`
	Collection5     int    `json:"collection_5"`
	Topic1          int    `json:"topic_1"`
	Topic2          int    `json:"topic_2"`
	Topic7          int    `json:"topic_7"`
	Reply1          int    `json:"reply_1"`
	Reply2          int    `json:"reply_2"`
	Reply3          int    `json:"reply_3"`
	Reply4          int    `json:"reply_4"`
	Reply5          int    `json:"reply_5"`
	Reply6          int    `json:"reply_6"`
	Reply7          int    `json:"reply_7"`
	Reply8          int    `json:"reply_8"`
}

// Component is a (domain, kind) pair shown as one row in the UI.
type Component struct {
	Domain string `json:"domain"`
	Kind   Kind   `json:"kind"`
	Label  string `json:"label"`
}

func AllComponents() []Component {
	var out []Component
	for _, sc := range SiteConfigs {
		for _, k := range sc.Kinds {
			out = append(out, Component{Domain: sc.Domain, Kind: k, Label: LabelFor(sc.Domain, k)})
		}
	}
	return out
}

func LabelFor(site string, kind Kind) string {
	if site == "next.bgm.tv/p1" {
		if kind == KindGuest {
			return "next.bgm.tv · API (/p1)"
		}
		return "next.bgm.tv · Authenticated (/p1/me)"
	}
	if site == "api.bgm.tv" {
		if kind == KindGuest {
			return site + " · Public endpoint"
		}
		return site + " · Authenticated"
	}
	switch kind {
	case KindGuest:
		return site + " · Guest"
	case KindAuth:
		return site + " · Authenticated"
	}
	return site
}

// DayBucket is a single daily uptime cell (used in 30-day strip).
type DayBucket struct {
	Day     string  `json:"day"` // YYYY-MM-DD (UTC)
	Uptime  float64 `json:"uptime"`
	Total   int     `json:"total"`
	Down    int     `json:"down"`
	Degrade int     `json:"degrade"`
	Status  Status  `json:"status"` // summary status for the day
}

type ComponentStatus struct {
	Domain     string      `json:"domain"`
	Kind       Kind        `json:"kind"`
	Label      string      `json:"label"`
	Status     Status      `json:"status"`              // current rolled-up status
	Uptime     float64     `json:"uptime"`              // 0-100
	Since      int64        `json:"since,omitempty"`     // unix seconds the current non-ok state began (0 when ok)
	Days       []DayBucket `json:"days"`                // 30 entries, oldest first
	LastCheck  int64       `json:"last_check"`          // unix seconds
	ProbeViews []ProbeView `json:"probe_views"`         // per-probe latest status
	Incidents  []Incident  `json:"incidents,omitempty"` // recent incident windows (≤14d)
}

// Incident is a contiguous window where the rolled-up status was not ok.
type Incident struct {
	StartTS   int64  `json:"start_ts"`
	EndTS     int64  `json:"end_ts"`
	Status    Status `json:"status"` // worst status during the window
	DurationS int    `json:"duration_s"`
	PeakDown  int    `json:"peak_down"`  // max probes reporting down
	PeakTotal int    `json:"peak_total"` // active probes at peak
}

type ProbeView struct {
	Probe    string `json:"probe"`
	Region   string `json:"region"`
	Status   Status `json:"status"`
	Latency  int    `json:"latency_ms"`
	HTTPCode int    `json:"http_code,omitempty"`
	TS       int64  `json:"ts"`
	Err      string `json:"err,omitempty"`
}

type Probe struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	LastSeen int64  `json:"last_seen"`
	Online   bool   `json:"online"`
}

type Overall struct {
	Status     Status            `json:"status"`
	Message    string            `json:"message"`
	UpdatedAt  int64             `json:"updated_at"`
	Components []ComponentStatus `json:"components"`
	Probes     []Probe           `json:"probes"`
	Online     []OnlinePoint     `json:"online,omitempty"`
}

func (r Result) Valid() bool {
	if r.Probe == "" || r.Domain == "" || r.Kind == "" {
		return false
	}
	if r.Status != StatusOK && r.Status != StatusDegraded && r.Status != StatusDown {
		return false
	}
	// reject results older than 1h or more than 5min in the future
	now := time.Now().Unix()
	if r.TS < now-3600 || r.TS > now+300 {
		return false
	}
	return true
}
