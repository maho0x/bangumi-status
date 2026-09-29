// Package telegram announces outages and recoveries on a Telegram channel and
// keeps a pinned status summary there.
package telegram

import (
	"context"
	"html"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"bangumi-status/internal/target"
	"bangumi-status/internal/types"
)

// ConfigStore persists small key/value pairs.
type ConfigStore interface {
	GetConfig(ctx context.Context, key string) (string, bool, error)
	SetConfig(ctx context.Context, key, value string) error
}

const pinnedKey = "telegram_pinned_msg_id"

// defaultMergeWindow groups transitions within this long of an open message
// into that message instead of posting a new one.
const defaultMergeWindow = time.Minute

// confirmations is how many consecutive observations a transition needs
// before it is announced.
const confirmations = 2

type entry struct {
	Domain    string
	Status    types.Status
	StartedAt time.Time
	GroupID   int // message id of the containing outageGroup; 0 when restored after a restart
}

type outageGroup struct {
	MessageID int
	OpenedAt  time.Time
	Entries   []*entry
}

func (g *outageGroup) hasDown() bool {
	for _, e := range g.Entries {
		if e.Status == types.StatusDown {
			return true
		}
	}
	return false
}

func (g *outageGroup) remove(target *entry) {
	out := g.Entries[:0]
	for _, e := range g.Entries {
		if e != target {
			out = append(out, e)
		}
	}
	g.Entries = out
}

type recoveryGroup struct {
	MessageID   int
	OpenedAt    time.Time
	StartedAt   time.Time
	RecoveredAt time.Time
	Domains     []string
	MaxDur      time.Duration
}

type recovery struct {
	domain    string
	dur       time.Duration
	replyTo   int
	startedAt time.Time
	at        time.Time
}

type pending struct {
	Status types.Status
	Count  int
}

// Telegram posts transition events and maintains a pinned summary. A nil or
// unconfigured Telegram is a no-op.
type Telegram struct {
	api         client
	db          ConfigStore
	mergeWindow time.Duration

	mu             sync.Mutex
	last           map[string]types.Status // last-announced status per component
	pending        map[string]pending      // transitions awaiting confirmation
	outages        map[string]*entry       // active (non-recovered) outages
	groups         map[int]*outageGroup    // outage messages by id
	activeGroup    *outageGroup            // newest outage message, if still mergeable
	activeRecovery *recoveryGroup          // newest recovery message, if still mergeable
	pinnedMsgID    int
	pinnedLoaded   bool
}

func New(token, chatID string, db ConfigStore) *Telegram {
	return &Telegram{
		api:         client{http: &http.Client{Timeout: 10 * time.Second}, token: token, chatID: chatID},
		db:          db,
		mergeWindow: defaultMergeWindow,
		last:        map[string]types.Status{},
		pending:     map[string]pending{},
		outages:     map[string]*entry{},
		groups:      map[int]*outageGroup{},
	}
}

func (t *Telegram) Enabled() bool { return t != nil && t.api.token != "" && t.api.chatID != "" }

type outageAction struct {
	comp types.ComponentStatus
	// separate: a degraded outage escalated to down; announce the down state
	// on its own instead of editing it into a degraded-only message.
	separate bool
}

// Process detects status transitions and announces confirmed ones. Only
// alertable components (see target.Alertable) produce messages, and a
// transition must be observed on `confirmations` consecutive calls first.
func (t *Telegram) Process(comps []types.ComponentStatus) {
	if !t.Enabled() {
		return
	}
	var outages []outageAction
	var recoveries []recovery
	now := time.Now()

	t.mu.Lock()
	for _, c := range comps {
		key := c.Domain + "|" + string(c.Kind)
		alertable := target.Alertable(c.Domain, c.Kind)
		announced, seen := t.last[key]
		if !seen {
			t.last[key] = c.Status
			// State is in memory, so a restart during an outage would forget it.
			// Restore it silently — no duplicate alert — but keep enough to
			// announce the recovery.
			if c.Status != types.StatusOK && alertable {
				started := now
				if c.Since > 0 && c.Since <= now.Unix() {
					started = time.Unix(c.Since, 0)
				}
				t.outages[key] = &entry{Domain: c.Domain, Status: c.Status, StartedAt: started}
				slog.Info("telegram: restored active outage", "component", key)
			}
			continue
		}
		if c.Status == announced {
			delete(t.pending, key)
			continue
		}
		p := t.pending[key]
		if p.Status != c.Status {
			p = pending{Status: c.Status}
		}
		p.Count++
		if p.Count < confirmations {
			t.pending[key] = p
			continue
		}
		delete(t.pending, key)
		t.last[key] = c.Status
		if !alertable {
			continue
		}

		e, active := t.outages[key]
		switch {
		case c.Status != types.StatusOK && !active:
			outages = append(outages, outageAction{comp: c})
		case c.Status == types.StatusDown && active && e.Status == types.StatusDegraded:
			t.detach(e)
			outages = append(outages, outageAction{comp: c, separate: true})
		case c.Status == types.StatusOK && active:
			// Recoveries of degraded-only outages stay silent, matching the
			// fact that "有点不对劲" posts never get a "活了" follow-up.
			degradedOnly := e.Status == types.StatusDegraded
			replyTo := 0
			if g := t.groups[e.GroupID]; g != nil {
				replyTo = g.MessageID
				degradedOnly = !g.hasDown()
			}
			t.detach(e)
			delete(t.outages, key)
			if e.Status == types.StatusDegraded && degradedOnly {
				slog.Info("telegram: degraded recovery (silent)", "domain", c.Domain)
				continue
			}
			recoveries = append(recoveries, recovery{
				domain: c.Domain, dur: now.Sub(e.StartedAt), replyTo: replyTo, startedAt: e.StartedAt, at: now,
			})
		}
	}
	t.mu.Unlock()

	for _, a := range outages {
		t.announceOutage(a)
	}
	if len(recoveries) > 0 {
		t.announceRecoveries(recoveries)
	}
}

// detach removes e from its outage message. Caller holds t.mu.
func (t *Telegram) detach(e *entry) {
	g := t.groups[e.GroupID]
	if g == nil {
		return
	}
	g.remove(e)
	if len(g.Entries) == 0 && t.activeGroup == g {
		t.activeGroup = nil
	}
}

// announceOutage edits the entry into the open outage message when one is
// still inside the merge window, otherwise posts a new one.
func (t *Telegram) announceOutage(a outageAction) {
	key := a.comp.Domain + "|" + string(a.comp.Kind)
	now := time.Now()

	t.mu.Lock()
	e := &entry{Domain: a.comp.Domain, Status: a.comp.Status, StartedAt: now}
	t.outages[key] = e
	g := t.activeGroup
	merge := g != nil && now.Sub(g.OpenedAt) < t.mergeWindow && !(a.separate && !g.hasDown())
	if merge {
		e.GroupID = g.MessageID
		g.Entries = append(g.Entries, e)
	} else {
		g = &outageGroup{OpenedAt: now, Entries: []*entry{e}}
	}
	text := renderOutage(g)
	t.mu.Unlock()

	if merge {
		switch t.api.edit(g.MessageID, text) {
		case editOK:
			slog.Info("telegram: outage merged", "msg", g.MessageID, "component", key)
			return
		case editFailed:
			return // leave the entry attached; a later edit picks it up
		case editGone:
			slog.Info("telegram: outage message gone, opening new", "msg", g.MessageID)
			t.mu.Lock()
			g.remove(e)
			if t.activeGroup == g {
				t.activeGroup = nil
			}
			g = &outageGroup{OpenedAt: now, Entries: []*entry{e}}
			e.GroupID = 0
			text = renderOutage(g)
			t.mu.Unlock()
		}
	}

	id := t.api.send(text, 0)
	if id == 0 {
		return
	}
	t.mu.Lock()
	g.MessageID, e.GroupID = id, id
	t.groups[id] = g
	t.activeGroup = g
	t.mu.Unlock()
	slog.Info("telegram: outage message opened", "msg", id, "component", key)
}

// announceRecoveries reports every recovery from one Process cycle together,
// editing them into the previous recovery message while it is mergeable.
func (t *Telegram) announceRecoveries(rs []recovery) {
	now := time.Now()
	add := recoveryGroup{}
	replyTo := 0
	for _, r := range rs {
		add.Domains = append(add.Domains, html.EscapeString(r.domain))
		add.MaxDur = max(add.MaxDur, r.dur)
		if add.StartedAt.IsZero() || r.startedAt.Before(add.StartedAt) {
			add.StartedAt = r.startedAt
		}
		if r.at.After(add.RecoveredAt) {
			add.RecoveredAt = r.at
		}
		if replyTo == 0 {
			replyTo = r.replyTo
		}
	}
	if add.StartedAt.IsZero() {
		add.StartedAt = now
	}
	if add.RecoveredAt.IsZero() {
		add.RecoveredAt = now
	}
	add.OpenedAt = add.RecoveredAt

	t.mu.Lock()
	prev := t.activeRecovery
	merge := prev != nil && now.Sub(prev.OpenedAt) < t.mergeWindow
	next := add
	if merge {
		next = recoveryGroup{
			MessageID:   prev.MessageID,
			OpenedAt:    prev.OpenedAt,
			StartedAt:   prev.StartedAt,
			RecoveredAt: prev.RecoveredAt,
			Domains:     append(append([]string{}, prev.Domains...), add.Domains...),
			MaxDur:      max(prev.MaxDur, add.MaxDur),
		}
		if next.StartedAt.IsZero() || add.StartedAt.Before(next.StartedAt) {
			next.StartedAt = add.StartedAt
		}
		if add.RecoveredAt.After(next.RecoveredAt) {
			next.RecoveredAt = add.RecoveredAt
		}
	}
	t.mu.Unlock()

	if merge {
		switch t.api.edit(next.MessageID, renderRecovery(&next)) {
		case editOK:
			t.setRecovery(&next)
			slog.Info("telegram: recovery merged", "msg", next.MessageID)
			return
		case editFailed:
			return
		case editGone:
			slog.Info("telegram: recovery message gone, sending new", "msg", next.MessageID)
			next = add
		}
	}
	id := t.api.send(renderRecovery(&next), replyTo)
	if id == 0 {
		return
	}
	next.MessageID = id
	t.setRecovery(&next)
	slog.Info("telegram: recovery message opened", "msg", id)
}

func (t *Telegram) setRecovery(rg *recoveryGroup) {
	t.mu.Lock()
	t.activeRecovery = rg
	t.mu.Unlock()
}

// UpdateSummary edits the pinned summary message, sending and pinning a new
// one the first time or after the old one disappears.
func (t *Telegram) UpdateSummary(overall *types.Overall) {
	if !t.Enabled() || overall == nil {
		return
	}
	text := renderSummary(overall)

	t.mu.Lock()
	if !t.pinnedLoaded {
		t.pinnedLoaded = true
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if v, ok, err := t.db.GetConfig(ctx, pinnedKey); err != nil {
			slog.Warn("telegram: load pinned id", "err", err)
		} else if ok {
			t.pinnedMsgID, _ = strconv.Atoi(v)
		}
		cancel()
	}
	msgID := t.pinnedMsgID
	t.mu.Unlock()

	if msgID != 0 {
		if r := t.api.edit(msgID, text); r != editGone {
			return
		}
		slog.Info("telegram: pinned message gone, sending new", "msg", msgID)
	}
	id := t.api.send(text, 0)
	if id == 0 {
		return
	}
	t.api.pin(id)
	t.mu.Lock()
	t.pinnedMsgID = id
	t.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := t.db.SetConfig(ctx, pinnedKey, strconv.Itoa(id)); err != nil {
		slog.Warn("telegram: save pinned id", "err", err)
	}
}

// SendDailyReport posts the once-a-day summary for the main sites.
func (t *Telegram) SendDailyReport(date string, items []DailyReportItem) {
	if !t.Enabled() || len(items) == 0 {
		return
	}
	t.api.send(renderDailyReport(date, items), 0)
	slog.Info("telegram: daily report sent", "date", date)
}
