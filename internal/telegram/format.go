package telegram

import (
	"fmt"
	"html"
	"strings"
	"time"

	"bangumi-status/internal/types"
)

func renderOutage(g *outageGroup) string {
	down := g.hasDown()
	var b strings.Builder
	if down {
		b.WriteString("Bangumi 可能Boom了\n")
	} else {
		b.WriteString("Bangumi 有点不对劲\n")
	}
	fmt.Fprintf(&b, "在 %s\n", fmtMoment(g.OpenedAt, g.OpenedAt))
	for _, e := range g.Entries {
		if e.Status == types.StatusDown {
			fmt.Fprintf(&b, "🔴 <b>%s</b> 中断\n", html.EscapeString(e.Domain))
		} else {
			fmt.Fprintf(&b, "🟡 <b>%s</b> 响应异常\n", html.EscapeString(e.Domain))
		}
	}
	if down {
		b.WriteString("#炸了")
	} else {
		b.WriteString("#降级")
	}
	return b.String()
}

func renderRecovery(rg *recoveryGroup) string {
	return fmt.Sprintf("Bangumi 活了！\n开始于 %s\n恢复于 %s\n<b>%s</b> 恢复正常，这次大概炸了 <b>%s</b>。\n#活了",
		fmtMoment(rg.StartedAt, rg.RecoveredAt), fmtMoment(rg.RecoveredAt, rg.StartedAt),
		strings.Join(rg.Domains, "、"), fmtApproxDuration(rg.MaxDur))
}

// DailyReportItem is one site's line in the daily report.
type DailyReportItem struct {
	Domain string
	Status types.Status
	Uptime float64
}

func renderDailyReport(date string, items []DailyReportItem) string {
	var b strings.Builder
	healthy := true
	for _, it := range items {
		healthy = healthy && it.Status == types.StatusOK
	}
	if healthy {
		fmt.Fprintf(&b, "🎉 班固米昨天没炸！ (%s) 🎉\n", date)
		b.WriteString("太棒了！班固米 (bangumi.tv, bgm.tv) 在昨天服务一切正常，いちご100%！\n")
	} else {
		fmt.Fprintf(&b, "💥 班固米昨天炸了！ (%s) 💥\n", date)
		for _, it := range items {
			switch it.Status {
			case types.StatusDegraded:
				fmt.Fprintf(&b, "• %s：降级（%.2f%%）\n", it.Domain, it.Uptime)
			case types.StatusDown:
				fmt.Fprintf(&b, "• %s：中断（%.2f%%）\n", it.Domain, it.Uptime)
			}
		}
	}
	b.WriteString("#日报")
	return b.String()
}

// summaryDomains are the pinned summary's rows. The summary tracks
// authenticated access only. NOTE: "next.bgm.tv" matches no component (that
// target's domain is "next.bgm.tv/p1"), so its row always reads OK; kept as-is
// to leave the pinned message unchanged.
var summaryDomains = []string{"bgm.tv", "bangumi.tv", "next.bgm.tv", "api.bgm.tv"}

func renderSummary(overall *types.Overall) string {
	type row struct {
		status types.Status
		uptime float64
	}
	rows := map[string]*row{}
	for _, d := range summaryDomains {
		rows[d] = &row{status: types.StatusOK, uptime: 100}
	}
	for _, c := range overall.Components {
		if r, ok := rows[c.Domain]; ok && c.Kind != types.KindGuest {
			r.status = types.Worst(r.status, c.Status)
			r.uptime = c.Uptime
		}
	}

	worst, affected := types.StatusOK, 0
	for _, r := range rows {
		if r.status != types.StatusOK {
			affected++
		}
		worst = types.Worst(worst, r.status)
	}
	var b strings.Builder
	b.WriteString("📊 <b>Bangumi Status</b>\n")
	switch worst {
	case types.StatusDegraded:
		fmt.Fprintf(&b, "🟡 <b>部分服务异常（%d 个）</b>\n", affected)
	case types.StatusDown:
		fmt.Fprintf(&b, "🔴 <b>服务中断（%d 个）</b>\n", affected)
	default:
		b.WriteString("✅ <b>全部系统正常</b>\n")
	}

	// Monospace table; the longest name, "next.bgm.tv", pads to 13.
	b.WriteString("\n<code>")
	label := map[types.Status]string{types.StatusOK: " OK ", types.StatusDegraded: "WARN", types.StatusDown: "DOWN"}
	for _, d := range summaryDomains {
		fmt.Fprintf(&b, "%-13s [%s] %s\n", d, label[rows[d].status], fmtUptimeShort(rows[d].uptime))
	}
	b.WriteString("</code>")

	online := 0
	for _, p := range overall.Probes {
		if p.Online {
			online++
		}
	}
	fmt.Fprintf(&b, "\n📡 %d/%d 探针  ·  🕐 %s UTC", online, len(overall.Probes), time.Now().UTC().Format("Jan 02 15:04"))
	return b.String()
}

func fmtUptimeShort(u float64) string {
	switch {
	case u <= 0:
		return "  —   "
	case u >= 99.995:
		return "100%  "
	}
	return fmt.Sprintf("%5.2f%%", u)
}

// fmtMoment formats t as a Beijing wall-clock time, adding the date only when
// it falls on a different day than ref.
func fmtMoment(t, ref time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	if ref.IsZero() {
		ref = time.Now()
	}
	bt, br := t.In(types.CST), ref.In(types.CST)
	if bt.Year() == br.Year() && bt.YearDay() == br.YearDay() {
		return bt.Format(time.TimeOnly)
	}
	return bt.Format("01-02 15:04:05")
}

func fmtApproxDuration(d time.Duration) string {
	total := max(int(d.Round(time.Second)/time.Second), 1)
	h, m, s := total/3600, total%3600/60, total%60
	var parts []string
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%d 小时", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%d 分", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d 秒", s))
	}
	return strings.Join(parts, " ")
}
