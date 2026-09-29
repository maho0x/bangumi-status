// Package target is the single registry of what Bangumi Status monitors: every
// (domain, kind) component, how a probe checks it, how the result is
// classified, and how the rest of the system treats it. Adding or retiring a
// target is an edit to All and nothing else.
package target

import "bangumi-status/internal/types"

// Credential says which Bangumi credential a check sends.
type Credential int

const (
	NoCredential   Credential = iota
	Cookie                    // session cookie
	Bearer                    // API token
	BearerOrCookie            // API token, falling back to the cookie
)

// Target describes one monitored component.
type Target struct {
	Domain string
	Kind   types.Kind
	Label  string

	// Probe side.
	URL        string
	JSON       bool       // send Accept: application/json
	Credential Credential // credential attached to the request
	// Marker must appear (case-insensitively) in a 2xx body, otherwise the
	// session is not actually signed in.
	Marker string
	// AuthErrors tags 401/403 as "auth <code>" instead of a bare degradation.
	AuthErrors bool
	// Redirects treats 301/302 as degraded: an expired cookie bounces to login.
	Redirects bool
	// Online scrapes the site-wide "online: N" badge from this response.
	Online bool

	// Aggregator side.
	Alert bool // announce transitions on Telegram
}

// All is the authoritative, UI-ordered list of monitored components.
var All = []Target{
	{Domain: "bgm.tv", Kind: types.KindGuest, Label: "bgm.tv · Guest", URL: "https://bgm.tv/"},
	{Domain: "bgm.tv", Kind: types.KindAuth, Label: "bgm.tv · Authenticated", URL: "https://bgm.tv/",
		Credential: Cookie, Marker: "logout", Redirects: true, Alert: true},

	{Domain: "bangumi.tv", Kind: types.KindGuest, Label: "bangumi.tv · Guest", URL: "https://bangumi.tv/"},
	// The session cookie is scoped to .bangumi.tv, so only this response
	// carries the signed-in "online: N" badge.
	{Domain: "bangumi.tv", Kind: types.KindAuth, Label: "bangumi.tv · Authenticated", URL: "https://bangumi.tv/",
		Credential: Cookie, Marker: "logout", Redirects: true, Online: true, Alert: true},

	{Domain: "next.bgm.tv/p1", Kind: types.KindGuest, Label: "next.bgm.tv · API (/p1)",
		URL: "https://next.bgm.tv/p1/subjects/1", JSON: true},
	{Domain: "next.bgm.tv/p1", Kind: types.KindAuth, Label: "next.bgm.tv · Authenticated (/p1/me)",
		URL: "https://next.bgm.tv/p1/me", JSON: true, Credential: BearerOrCookie, AuthErrors: true},

	{Domain: "api.bgm.tv", Kind: types.KindGuest, Label: "api.bgm.tv · Public endpoint",
		URL: "https://api.bgm.tv/v0/subjects/1", JSON: true, AuthErrors: true},
	// /v0/me actually validates the token (401 when it is bad). Single-tick
	// 401s are absorbed by the notifier debounce + quorum.
	{Domain: "api.bgm.tv", Kind: types.KindAuth, Label: "api.bgm.tv · Authenticated",
		URL: "https://api.bgm.tv/v0/me", JSON: true, Credential: Bearer, AuthErrors: true},
}

var byKey = func() map[types.ComponentKey]Target {
	m := make(map[types.ComponentKey]Target, len(All))
	for _, t := range All {
		m[t.Key()] = t
	}
	return m
}()

func (t Target) Key() types.ComponentKey { return types.ComponentKey{Domain: t.Domain, Kind: t.Kind} }

func (t Target) Component() types.Component {
	return types.Component{Domain: t.Domain, Kind: t.Kind, Label: t.Label}
}

// Components lists every monitored component in UI order.
func Components() []types.Component {
	out := make([]types.Component, len(All))
	for i, t := range All {
		out[i] = t.Component()
	}
	return out
}

// Lookup returns the target for (domain, kind).
func Lookup(domain string, kind types.Kind) (Target, bool) {
	t, ok := byKey[types.ComponentKey{Domain: domain, Kind: kind}]
	return t, ok
}

// Monitored reports whether results for (domain, kind) are still accepted.
func Monitored(domain string, kind types.Kind) bool {
	_, ok := Lookup(domain, kind)
	return ok
}

// Label names a component, including decommissioned ones that only survive in
// the incident archive.
func Label(domain string, kind types.Kind) string {
	if t, ok := Lookup(domain, kind); ok {
		return t.Label
	}
	switch kind {
	case types.KindGuest:
		return domain + " · Guest"
	case types.KindAuth:
		return domain + " · Authenticated"
	}
	return domain
}

// Alertable reports whether transitions of (domain, kind) go to Telegram.
func Alertable(domain string, kind types.Kind) bool {
	t, ok := Lookup(domain, kind)
	return ok && t.Alert
}
