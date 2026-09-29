// Bilingual strings. Reading `locale.lang` inside t() makes every template
// that calls t() re-render when the language switches.

type Entry = string | ((...args: any[]) => string);

const en = {
  status_ok: "Operational", status_degraded: "Degraded", status_down: "Outage", status_none: "No data",
  incident_down: "Service disruption", incident_degraded: "Degraded performance",
  banner_ok: "All systems operational", banner_degraded: "Partial degradation", banner_down: "Service disruption",
  banner_loading: "Loading", banner_loading_sub: "Fetching probe data…",
  kind_guest: "Guest access", kind_auth: "Logged-in access",
  kind_api_guest: "Public endpoint", kind_api_auth: "Authenticated",
  never: "never", just_now: "just now",
  s_ago: "s ago", m_ago: "m ago", h_ago: "h ago", d_ago: "d ago",
  today: "Today", yesterday: "Yesterday", n_days_ago: " days ago",
  uptime_30d: "Uptime 30d",
  banner_sub_all: (n: number) => `All ${n} components reporting normally.`,
  banner_sub_affected: (a: number, n: number) => `${a} of ${n} components affected.`,
  uptime_label: "uptime",
  no_probe_data: "no probe data", no_probe_data_detail: "No recent probe data.",
  guest_show: "Show guest access", guest_hide: "Hide guest access",
  collapsed_guest_hint: (n: number) => `${n} public endpoint incident${n === 1 ? "" : "s"} hidden`,
  expanded_guest_hint: (n: number) => `${n} public endpoint incident${n === 1 ? "" : "s"} shown`,
  affected: (n: number) => `${n} affected`,
  no_incidents: "No incidents reported.", no_data: "No data yet.",
  checks: (n: number) => `${n.toLocaleString()} check${n === 1 ? "" : "s"}`,
  failed_count: (n: number) => `${n} failed`, degraded_count: (n: number) => `${n} degraded`,
  probe_online_summary: (on: number, tot: number) => `${on} / ${tot} online`,
  probe_pill_online: "online", probe_pill_offline: "offline",
  no_probes: "No probes registered yet.",
  probes_failing: (b: number, tot: number) => `${b} of ${tot} probes failing`,
  peak_failing: (b: number, tot: number) => `peak ${b}/${tot} probes failing`,
  dur_s: (n: number) => `${n}s`, dur_m: (n: number) => `${n}m`,
  dur_hm: (h: number, m: number) => `${h}h ${m}m`, dur_h: (n: number) => `${n}h`,
  dur_d: (n: number) => `${n}d`, dur_dh: (d: number, h: number) => `${d}d ${h}h`,
  held_label: "Ongoing for",
  ongoing: "ongoing",
  auto_refresh: "auto-refresh 30s",
  atom_feed: "Atom feed", error_load: "Unable to load status",
  copy: "Copy", copied: "Copied", subscribe_btn: "Subscribe", close: "Close",
  section_current: "Current status by service", hint_30d: "30-day uptime",
  section_unresolved: "Unresolved incidents",
  section_past: "Past incidents", hint_past: "Last 10 days",
  section_probes: "Probe nodes",
  section_online: "Activity",
  online_note: "Live “online” counter from bangumi.tv (signed-in view).",
  online_no_data: "No samples yet.",
  online_current: (n: string) => `${n} online now`,
  online_peak: (n: string) => `peak ${n}`,
  online_avg: (n: string) => `avg ${n}`,
  metric_bangumi: "Bangumi online", metric_traffic: "Site traffic",
  range_all: "All",
  traffic_current: (n: string) => `${n} viewing now`,
  traffic_note: "Status-page viewers: live current count, history as per-minute peak.",
  footer_desc: "Independent, community-run availability monitor. Not affiliated with Bangumi.",
  modal_title: "Subscribe to updates",
  modal_intro: "Get notified when bgm.tv, bangumi.tv or the Bangumi API experiences availability issues.",
  sub_atom_title: "Atom / RSS feed",
  sub_atom_desc: "Add this feed to any RSS reader for incident notifications.",
  sub_tg_title: "Telegram channel",
  sub_tg_desc: "Follow the channel to receive push notifications for outages and recoveries.",
  sub_tg_go: "Open",
  sub_live_title: "Live page",
  sub_live_desc: "This page auto-refreshes every 30 seconds — bookmark it for a quick at-a-glance check.",
  modal_foot: "No personal data is collected. This monitor is community-run and open source.",
  nav_status: "Status", nav_wiki_stats: "Stats", nav_label: "Pages",
  react: "React", react_again: "React again",
  past_incidents_more: "Full history →",
  history_title: "Incident history",
  history_intro: "Availability incidents, archived by month and derived automatically from probe data.",
  history_earliest_label: "Records since",
  history_prev: "Earlier months", history_next: "Later months",
  history_empty: "No incidents recorded yet.",
  history_month_none: "No incidents this month.",
  history_month_summary: (n: number, dur: string) => `${n} incident${n === 1 ? "" : "s"} · ${dur} total downtime`,
  history_month_clear: "No incidents",
  wiki_title: "Stats",
  wiki_intro: "Daily new registrations, collections, topics and replies across the site.",
  wiki_recent_scrape: "Scraped", wiki_data_day: "Data day",
  wiki_empty: "No wiki stats yet.",
  wiki_metric_register: "Registered users", wiki_metric_collection: "Collections",
  wiki_metric_topics: "Topics", wiki_metric_replies: "Replies",
  wiki_hint_daily: "daily new", wiki_hint_by_type: "by collection type", wiki_hint_by_source: "by source",
} satisfies Record<string, Entry>;

type Key = keyof typeof en;

const zh: Record<Key, Entry> = {
  status_ok: "正常", status_degraded: "降级", status_down: "中断", status_none: "无数据",
  incident_down: "服务中断", incident_degraded: "性能降级",
  banner_ok: "完全正常", banner_degraded: "部分服务降级", banner_down: "服务中断",
  banner_loading: "加载中", banner_loading_sub: "正在获取探针数据……",
  kind_guest: "公共端点", kind_auth: "认证端点",
  kind_api_guest: "公共端点", kind_api_auth: "认证端点",
  never: "从未", just_now: "刚刚",
  s_ago: " 秒前", m_ago: " 分钟前", h_ago: " 小时前", d_ago: " 天前",
  today: "今天", yesterday: "昨天", n_days_ago: " 天前",
  uptime_30d: "30天可用率",
  banner_sub_all: (n: number) => `全部 ${n} 个服务运行正常。`,
  banner_sub_affected: (a: number, n: number) => `${n} 个服务中 ${a} 个受影响。`,
  uptime_label: "可用率",
  no_probe_data: "暂无探针数据", no_probe_data_detail: "暂无近期探针数据。",
  guest_show: "显示公共端点", guest_hide: "隐藏公共端点",
  collapsed_guest_hint: (n: number) => `${n} 个公共端点事件已折叠`,
  expanded_guest_hint: (n: number) => `${n} 个公共端点事件已展开`,
  affected: (n: number) => `${n} 个受影响`,
  no_incidents: "无故障记录。", no_data: "暂无数据。",
  checks: (n: number) => `${n.toLocaleString()} 次检查`,
  failed_count: (n: number) => `${n} 次失败`, degraded_count: (n: number) => `${n} 次降级`,
  probe_online_summary: (on: number, tot: number) => `${on} / ${tot} 在线`,
  probe_pill_online: "在线", probe_pill_offline: "离线",
  no_probes: "暂无探针注册。",
  probes_failing: (b: number, tot: number) => `${b} / ${tot} 个探针异常`,
  peak_failing: (b: number, tot: number) => `峰值 ${b}/${tot} 个探针异常`,
  dur_s: (n: number) => `${n} 秒`, dur_m: (n: number) => `${n} 分钟`,
  dur_hm: (h: number, m: number) => `${h} 小时 ${m} 分钟`, dur_h: (n: number) => `${n} 小时`,
  dur_d: (n: number) => `${n} 天`, dur_dh: (d: number, h: number) => `${d} 天 ${h} 小时`,
  held_label: "已持续",
  ongoing: "进行中",
  auto_refresh: "30秒自动刷新",
  atom_feed: "Atom 订阅", error_load: "无法加载状态",
  copy: "复制", copied: "已复制", subscribe_btn: "订阅", close: "关闭",
  section_current: "当前服务状态", hint_30d: "30天可用率",
  section_unresolved: "未解决的事故",
  section_past: "历史事故", hint_past: "最近10天",
  section_probes: "探针节点",
  section_online: "活动",
  online_note: "Bangumi online 人数历史。",
  online_no_data: "暂无数据。",
  online_current: (n: string) => `当前 ${n} 人在线`,
  online_peak: (n: string) => `峰值 ${n}`,
  online_avg: (n: string) => `平均值 ${n}`,
  metric_bangumi: "班固米在线", metric_traffic: "本站访问",
  range_all: "全部",
  traffic_current: (n: string) => `当前 ${n} 人查看`,
  traffic_note: "Bangumi Status 当前访问人数实时更新，历史按每分钟峰值记录。",
  footer_desc: "社区运营的Bangumi可用性监测。",
  modal_title: "订阅更新",
  modal_intro: "当 bgm.tv、bangumi.tv 或 Bangumi API 出现可用性问题时获得通知。",
  sub_atom_title: "Atom / RSS 订阅",
  sub_atom_desc: "将此订阅源添加到任何 RSS 阅读器以获取故障通知。",
  sub_tg_title: "Telegram 频道",
  sub_tg_desc: "关注频道以接收故障和恢复推送通知。",
  sub_tg_go: "前往订阅",
  sub_live_title: "实时页面",
  sub_live_desc: "本页面每30秒自动刷新，收藏以便快速查看。",
  modal_foot: "本监测由社区运营并开源。",
  nav_status: "状态", nav_wiki_stats: "透视", nav_label: "页面",
  react: "贴贴", react_again: "再贴一个",
  past_incidents_more: "全部历史 →",
  history_title: "历史事故",
  history_intro: "按月归档的可用性事故记录，从探针数据自动生成。",
  history_earliest_label: "记录起始",
  history_prev: "更早的月份", history_next: "更近的月份",
  history_empty: "暂无事故记录。",
  history_month_none: "本月无事故。",
  history_month_summary: (n: number, dur: string) => `${n} 起事故 · 累计中断 ${dur}`,
  history_month_clear: "无事故",
  wiki_title: "透视",
  wiki_intro: "全站每日新增的注册用户、收藏、主题与回复数据。",
  wiki_recent_scrape: "最近抓取", wiki_data_day: "数据日期",
  wiki_empty: "暂无维基透视数据。",
  wiki_metric_register: "注册用户", wiki_metric_collection: "收藏数",
  wiki_metric_topics: "主题数", wiki_metric_replies: "回复数",
  wiki_hint_daily: "每日新增", wiki_hint_by_type: "按收藏类型", wiki_hint_by_source: "按来源",
};

export type Lang = "zh" | "en";

function storedLang(): Lang {
  try {
    return localStorage.getItem("lang") === "en" ? "en" : "zh";
  } catch {
    return "zh";
  }
}

export const locale = $state({ lang: storedLang() });

$effect.root(() => {
  $effect(() => {
    document.documentElement.lang = locale.lang === "zh" ? "zh-CN" : "en";
    try {
      localStorage.setItem("lang", locale.lang);
    } catch {
      /* private mode: the choice just isn't remembered */
    }
  });
});

export function toggleLang() {
  locale.lang = locale.lang === "zh" ? "en" : "zh";
}

export function t(key: Key, ...args: any[]): string {
  const v = (locale.lang === "zh" ? zh : en)[key];
  return typeof v === "function" ? (v as (...a: any[]) => string)(...args) : v;
}
