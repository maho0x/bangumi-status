// Package notifier delivers status notifications to a Telegram channel.
package notifier

import (
	"bangumi-status/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ConfigStore persists small key/value pairs (backed by SQLite config table).
type ConfigStore interface {
	GetConfig(ctx context.Context, key string) (string, bool, error)
	SetConfig(ctx context.Context, key, value string) error
}

// defaultMergeWindow groups outage/recovery notifications that occur within
// this duration of an existing group into a single edited Telegram message.
const defaultMergeWindow = 10 * time.Minute

type groupEntry struct {
	Domain    string
	Kind      types.Kind
	Status    types.Status
	StartedAt time.Time
	GroupID   int // message_id of containing outageGroup
}

type outageGroup struct {
	MessageID int
	OpenedAt  time.Time
	Entries   []*groupEntry
}

// Telegram posts transition events and maintains a pinned summary message.
// Zero value is a no-op; use NewTelegram.
type Telegram struct {
	token   string
	chatID  string
	pageURL string
	db      ConfigStore
	client  *http.Client

	mergeWindow time.Duration

	mu           sync.Mutex
	last         map[string]types.Status // last-notified status per component
	pending      map[string]pendingTxn   // transitions awaiting confirmation
	outages      map[string]*groupEntry  // active (non-recovered) entries by key
	groups       map[int]*outageGroup    // known groups by message_id
	activeGroup  *outageGroup            // most-recent group that may absorb new events
	pinnedMsgID  int
	pinnedLoaded bool
}

type pendingTxn struct {
	Status types.Status
	Count  int
}

func NewTelegram(token, chatID, pageURL string, db ConfigStore) *Telegram {
	return &Telegram{
		token:       token,
		chatID:      chatID,
		pageURL:     pageURL,
		db:          db,
		client:      &http.Client{Timeout: 10 * time.Second},
		mergeWindow: defaultMergeWindow,
		last:        map[string]types.Status{},
		pending:     map[string]pendingTxn{},
		outages:     map[string]*groupEntry{},
		groups:      map[int]*outageGroup{},
	}
}

func (t *Telegram) Enabled() bool { return t != nil && t.token != "" && t.chatID != "" }

// Process detects component status transitions and fires notifications.
// Only auth-kind components trigger messages. A transition must be observed
// in two consecutive calls (≈20–40 s) before a message is sent.
func (t *Telegram) Process(comps []types.ComponentStatus) {
	if !t.Enabled() {
		return
	}

	type action struct {
		kind string // "outage" | "recovery"
		comp types.ComponentStatus
	}
	var actions []action

	t.mu.Lock()
	for _, c := range comps {
		key := c.Domain + "|" + string(c.Kind)
		current := c.Status
		notified, seen := t.last[key]
		if !seen {
			t.last[key] = current
			delete(t.pending, key)
			continue
		}
		if current == notified {
			delete(t.pending, key)
			continue
		}
		p := t.pending[key]
		if p.Status != current {
			p = pendingTxn{Status: current, Count: 1}
		} else {
			p.Count++
		}
		if p.Count >= 2 {
			// Only auth kind triggers messages; api/next subdomains are silent.
			silent := c.Domain == "api.bgm.tv" || c.Domain == "next.bgm.tv"
			if c.Kind != types.KindGuest && !silent {
				_, activeOutage := t.outages[key]
				switch {
				case current != types.StatusOK && !activeOutage:
					actions = append(actions, action{kind: "outage", comp: c})
				case current == types.StatusOK && activeOutage:
					actions = append(actions, action{kind: "recovery", comp: c})
				}
			}
			t.last[key] = current
			delete(t.pending, key)
		} else {
			t.pending[key] = p
		}
	}
	t.mu.Unlock()

	for _, a := range actions {
		switch a.kind {
		case "outage":
			t.handleOutage(a.comp)
		case "recovery":
			t.handleRecovery(a.comp)
		}
	}
}

// handleOutage either edits the active group's message (if still within the
// merge window) or opens a new group with a fresh Telegram message.
func (t *Telegram) handleOutage(c types.ComponentStatus) {
	key := c.Domain + "|" + string(c.Kind)
	now := time.Now()

	t.mu.Lock()
	entry := &groupEntry{
		Domain:    c.Domain,
		Kind:      c.Kind,
		Status:    c.Status,
		StartedAt: now,
	}
	t.outages[key] = entry

	var (
		merge bool
		group *outageGroup
		msgID int
	)
	if t.activeGroup != nil && now.Sub(t.activeGroup.OpenedAt) < t.mergeWindow {
		merge = true
		group = t.activeGroup
		msgID = group.MessageID
		entry.GroupID = msgID
		group.Entries = append(group.Entries, entry)
	} else {
		group = &outageGroup{OpenedAt: now, Entries: []*groupEntry{entry}}
	}
	text := renderGroupText(group)
	t.mu.Unlock()

	if merge {
		switch t.editMessage(msgID, text) {
		case editOK:
			log.Printf("telegram: outage merged into msg %d (%s)", msgID, key)
			return
		case editFailed:
			// Transient — leave entry attached; a later edit will pick it up.
			log.Printf("telegram: edit failed for msg %d, leaving entry attached", msgID)
			return
		case editGone:
			// Message vanished — detach and fall through to send a fresh one.
			log.Printf("telegram: active group msg %d gone, opening new group", msgID)
			t.mu.Lock()
			group.Entries = removeEntry(group.Entries, entry)
			if t.activeGroup == group {
				t.activeGroup = nil
			}
			group = &outageGroup{OpenedAt: now, Entries: []*groupEntry{entry}}
			entry.GroupID = 0
			text = renderGroupText(group)
			t.mu.Unlock()
		}
	}

	id := t.sendMessage(text, 0)
	if id == 0 {
		return
	}
	t.mu.Lock()
	group.MessageID = id
	entry.GroupID = id
	t.groups[id] = group
	t.activeGroup = group
	t.mu.Unlock()
	log.Printf("telegram: outage group %d opened for %s", id, key)
}

// handleRecovery sends a new recovery message as a reply to the original outage
// message, leaving the outage message itself unchanged.
func (t *Telegram) handleRecovery(c types.ComponentStatus) {
	key := c.Domain + "|" + string(c.Kind)
	now := time.Now()

	t.mu.Lock()
	entry, ok := t.outages[key]
	if !ok {
		t.mu.Unlock()
		return
	}
	dur := fmtDuration(now.Sub(entry.StartedAt))
	delete(t.outages, key)

	replyTo := 0
	if group := t.groups[entry.GroupID]; group != nil {
		replyTo = group.MessageID
	}
	t.mu.Unlock()

	text := fmt.Sprintf("Bangumi 活了！这次炸了 <b>%s</b>\n#活了", dur)
	t.sendMessage(text, replyTo)
	log.Printf("telegram: recovery for %s (duration %s), reply to msg %d", key, dur, replyTo)
}

func removeEntry(s []*groupEntry, target *groupEntry) []*groupEntry {
	out := s[:0]
	for _, e := range s {
		if e != target {
			out = append(out, e)
		}
	}
	return out
}

func renderGroupText(g *outageGroup) string {
	if len(g.Entries) == 1 {
		e := g.Entries[0]
		label := html.EscapeString(serviceLabel(e.Domain))
		if e.Status == types.StatusDown {
			return fmt.Sprintf("Bangumi 可能Boom了，这次炸的是 <b>%s</b>\n#炸了", label)
		}
		return fmt.Sprintf("Bangumi 有点不对劲，<b>%s</b> 响应异常\n#降级", label)
	}

	anyDown := false
	for _, e := range g.Entries {
		if e.Status == types.StatusDown {
			anyDown = true
			break
		}
	}

	var b strings.Builder
	if anyDown {
		b.WriteString("Bangumi 可能Boom了\n\n")
	} else {
		b.WriteString("Bangumi 有点不对劲\n\n")
	}
	for _, e := range g.Entries {
		label := html.EscapeString(serviceLabel(e.Domain))
		if e.Status == types.StatusDown {
			fmt.Fprintf(&b, "🔴 <b>%s</b> 中断\n", label)
		} else {
			fmt.Fprintf(&b, "🟡 <b>%s</b> 响应异常\n", label)
		}
	}
	b.WriteString("\n#炸了")
	return b.String()
}

// UpdateSummary edits the pinned channel summary message with the current
// overall status. On the first call (or after the pinned message disappears),
// a new message is sent and pinned.
func (t *Telegram) UpdateSummary(overall *types.Overall) {
	if !t.Enabled() || overall == nil {
		return
	}
	text := formatSummary(overall)

	t.mu.Lock()
	if !t.pinnedLoaded {
		t.pinnedLoaded = true
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		v, ok, err := t.db.GetConfig(ctx, "telegram_pinned_msg_id")
		cancel()
		if err != nil {
			log.Printf("telegram: load pinned id: %v", err)
		} else if ok {
			t.pinnedMsgID, _ = strconv.Atoi(v)
		}
	}
	msgID := t.pinnedMsgID
	t.mu.Unlock()

	if msgID != 0 {
		switch t.editMessage(msgID, text) {
		case editOK:
			return
		case editFailed:
			return
		case editGone:
			log.Printf("telegram: pinned msg %d gone, sending new", msgID)
		}
	}

	newID := t.sendMessage(text, 0)
	if newID == 0 {
		return
	}
	t.pinMessage(newID)

	t.mu.Lock()
	t.pinnedMsgID = newID
	t.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := t.db.SetConfig(ctx, "telegram_pinned_msg_id", strconv.Itoa(newID)); err != nil {
		log.Printf("telegram: save pinned id: %v", err)
	}
}

// editResult is the outcome of an editMessage call.
type editResult int

const (
	editOK     editResult = iota
	editGone              // message deleted — safe to send a replacement
	editFailed            // transient error — retry next cycle
)

func (t *Telegram) editMessage(msgID int, text string) editResult {
	api := "https://api.telegram.org/bot" + t.token + "/editMessageText"
	form := url.Values{}
	form.Set("chat_id", t.chatID)
	form.Set("message_id", strconv.Itoa(msgID))
	form.Set("text", text)
	form.Set("parse_mode", "HTML")
	form.Set("disable_web_page_preview", "true")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", api, strings.NewReader(form.Encode()))
	if err != nil {
		return editFailed
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("telegram edit: %v", err)
		return editFailed
	}
	defer resp.Body.Close()
	var body struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.OK {
		return editOK
	}
	desc := body.Description
	if body.ErrorCode == 400 && strings.Contains(desc, "not modified") {
		return editOK
	}
	if body.ErrorCode == 400 && (strings.Contains(desc, "message to edit not found") ||
		strings.Contains(desc, "MESSAGE_ID_INVALID")) {
		return editGone
	}
	log.Printf("telegram edit %d: %d %s", msgID, body.ErrorCode, desc)
	return editFailed
}

// sendMessage sends a message, optionally as a reply, and returns its message_id.
func (t *Telegram) sendMessage(text string, replyToID int) int {
	api := "https://api.telegram.org/bot" + t.token + "/sendMessage"
	form := url.Values{}
	form.Set("chat_id", t.chatID)
	form.Set("text", text)
	form.Set("parse_mode", "HTML")
	form.Set("disable_web_page_preview", "true")
	if replyToID != 0 {
		form.Set("reply_to_message_id", strconv.Itoa(replyToID))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", api, strings.NewReader(form.Encode()))
	if err != nil {
		log.Printf("telegram send req: %v", err)
		return 0
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("telegram send: %v", err)
		return 0
	}
	defer resp.Body.Close()
	var body struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
		Result      struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		log.Printf("telegram send: decode: %v", err)
		return 0
	}
	if !body.OK {
		log.Printf("telegram send: %d %s", body.ErrorCode, body.Description)
		return 0
	}
	return body.Result.MessageID
}

func (t *Telegram) pinMessage(msgID int) {
	api := "https://api.telegram.org/bot" + t.token + "/pinChatMessage"
	form := url.Values{}
	form.Set("chat_id", t.chatID)
	form.Set("message_id", strconv.Itoa(msgID))
	form.Set("disable_notification", "true")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", api, strings.NewReader(form.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("telegram pin: %v", err)
		return
	}
	resp.Body.Close()
}

// --- Formatting -----------------------------------------------------------

type DailyReportItem struct {
	Domain string
	Status types.Status
	Uptime float64
}

// SendDailyReport pushes a once-a-day summary to Telegram.
// Items are expected to be the three main sites (bgm.tv, bangumi.tv, chii.in).
func (t *Telegram) SendDailyReport(date string, items []DailyReportItem) {
	if !t.Enabled() || len(items) == 0 {
		return
	}
	var hasIssue bool
	for _, it := range items {
		if it.Status != types.StatusOK {
			hasIssue = true
			break
		}
	}
	var b strings.Builder
	if !hasIssue {
		fmt.Fprintf(&b, "🎉 班固米昨天没炸！ (%s) 🎉\n", date)
		fmt.Fprintf(&b, "太棒了！班固米 (bangumi.tv, bgm.tv, chii.in) 在昨天服务一切正常，いちご100%%！\n")
	} else {
		fmt.Fprintf(&b, "💥 班固米昨天炸了！ (%s) 💥\n", date)
		for _, it := range items {
			if it.Status == types.StatusOK {
				continue
			}
			label := serviceLabel(it.Domain)
			statusStr := ""
			switch it.Status {
			case types.StatusDegraded:
				statusStr = "降级"
			case types.StatusDown:
				statusStr = "中断"
			}
			fmt.Fprintf(&b, "• %s：%s（%.2f%%）\n", label, statusStr, it.Uptime)
		}
	}
	b.WriteString("#日报")
	t.sendMessage(b.String(), 0)
	log.Printf("telegram: daily report sent for %s", date)
}

func serviceLabel(domain string) string {
	return domain
}

func fmtDuration(d time.Duration) string {
	if d < time.Minute {
		return "不到1分钟"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%d分钟", m)
	}
	if m == 0 {
		return fmt.Sprintf("%d小时", h)
	}
	return fmt.Sprintf("%d小时%d分钟", h, m)
}

var statusEmoji = map[types.Status]string{
	types.StatusOK:       "✅",
	types.StatusDegraded: "🟡",
	types.StatusDown:     "🔴",
}

func formatSummary(overall *types.Overall) string {
	rank := map[types.Status]int{types.StatusOK: 0, types.StatusDegraded: 1, types.StatusDown: 2}

	// Aggregate per-domain: worst status.
	// For the three main sites, uptime is the authenticated endpoint only;
	// for others (next/api) it is the worst across monitored kinds.
	type domainInfo struct {
		status types.Status
		uptime float64
		initd  bool
	}
	domains := map[string]*domainInfo{}
	order := []string{"bgm.tv", "bangumi.tv", "chii.in", "next.bgm.tv", "api.bgm.tv"}
	for _, d := range order {
		domains[d] = &domainInfo{status: types.StatusOK, uptime: 100}
	}
	authUptimeDomains := map[string]bool{"bgm.tv": true, "bangumi.tv": true, "chii.in": true}
	for _, c := range overall.Components {
		d, ok := domains[c.Domain]
		if !ok {
			continue
		}
		if rank[c.Status] > rank[d.status] {
			d.status = c.Status
		}
		if authUptimeDomains[c.Domain] {
			if c.Kind == types.KindAuth {
				d.uptime = c.Uptime
				d.initd = true
			}
		} else {
			if !d.initd || c.Uptime < d.uptime {
				d.uptime = c.Uptime
				d.initd = true
			}
		}
	}

	// Overall headline.
	headEmoji := statusEmoji[overall.Status]
	if headEmoji == "" {
		headEmoji = "⚪️"
	}
	headline := "全部系统正常"
	affected := 0
	for _, d := range domains {
		if d.status != types.StatusOK {
			affected++
		}
	}
	switch overall.Status {
	case types.StatusDegraded:
		headline = fmt.Sprintf("部分服务异常（%d 个）", affected)
	case types.StatusDown:
		headline = fmt.Sprintf("服务中断（%d 个）", affected)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "📊 <b>Bangumi Status</b>\n")
	fmt.Fprintf(&b, "%s <b>%s</b>\n", headEmoji, headline)

	// Service table inside a code block for monospace alignment.
	// Longest domain: "next.bgm.tv" = 11 chars → pad to 13.
	b.WriteString("\n<code>")
	statusChar := map[types.Status]string{
		types.StatusOK:       " OK ",
		types.StatusDegraded: "WARN",
		types.StatusDown:     "DOWN",
	}
	for _, domain := range order {
		d := domains[domain]
		sc := statusChar[d.status]
		if sc == "" {
			sc = "   ?"
		}
		up := fmtUptimeShort(d.uptime)
		fmt.Fprintf(&b, "%-13s [%s] %s\n", domain, sc, up)
	}
	b.WriteString("</code>")

	// Probes + timestamp.
	online := 0
	for _, p := range overall.Probes {
		if p.Online {
			online++
		}
	}
	fmt.Fprintf(&b, "\n📡 %d/%d 探针  ·  🕐 %s UTC",
		online, len(overall.Probes),
		time.Now().UTC().Format("Jan 02 15:04"))

	return b.String()
}

func fmtUptimeShort(u float64) string {
	if u <= 0 {
		return "  —   "
	}
	if u >= 99.995 {
		return "100%  "
	}
	return fmt.Sprintf("%5.2f%%", u)
}

func kindLabel(domain string, kind types.Kind) string {
	if domain == "api.bgm.tv" {
		if kind == types.KindGuest {
			return "Public endpoint"
		}
		return "Authenticated"
	}
	switch kind {
	case types.KindGuest:
		return "Guest access"
	case types.KindAuth:
		return "Logged-in access"
	}
	return string(kind)
}
