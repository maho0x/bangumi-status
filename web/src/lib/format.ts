import { locale, t } from "./i18n.svelte";
import type { ComponentStatus, Kind } from "./api-types";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const WEEKDAYS_EN = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const WEEKDAYS_ZH = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];

const pad = (n: number) => String(n).padStart(2, "0");
export const nowSec = () => Math.floor(Date.now() / 1000);

/** YYYY-MM-DD in the viewer's local calendar. */
export function isoDayLocal(ts: number): string {
  const d = new Date(ts * 1000);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** YYYY-MM for a local date. */
export const ymKey = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}`;

export function fmtTimeLocal(ts: number): string {
  const d = new Date(ts * 1000);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function fmtRelative(ts: number | undefined): string {
  if (!ts) return t("never");
  const diff = Math.floor(Date.now() / 1000 - ts);
  if (diff < 0) return t("just_now");
  if (diff < 60) return diff + t("s_ago");
  if (diff < 3600) return Math.floor(diff / 60) + t("m_ago");
  if (diff < 86400) return Math.floor(diff / 3600) + t("h_ago");
  return Math.floor(diff / 86400) + t("d_ago");
}

export function fmtDayLabel(isoDay: string): string {
  const [y, m, d] = isoDay.split("-").map(Number);
  if (!y || !m || !d) return isoDay;
  return locale.lang === "zh" ? `${y}年${m}月${d}日` : `${MONTHS[m - 1]} ${d}, ${y}`;
}

/** "Today", "Yesterday", "3 days ago" within a week, otherwise "". */
export function fmtDayRelative(isoDay: string): string {
  const [y, m, d] = isoDay.split("-").map(Number);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const diff = Math.round((today - new Date(y, m - 1, d).getTime()) / 86400000);
  if (diff === 0) return t("today");
  if (diff === 1) return t("yesterday");
  if (diff > 0 && diff < 7) return diff + t("n_days_ago");
  return "";
}

export function weekdayShort(isoDay: string): string {
  const [y, m, d] = isoDay.split("-").map(Number);
  if (!y || !m || !d) return "";
  const idx = new Date(Date.UTC(y, m - 1, d)).getUTCDay();
  return (locale.lang === "zh" ? WEEKDAYS_ZH : WEEKDAYS_EN)[idx];
}

export function fmtMonthLabel(ym: string): string {
  const [y, m] = ym.split("-").map(Number);
  return locale.lang === "zh" ? `${y}年${m}月` : `${MONTHS[m - 1]} ${y}`;
}

export function fmtDuration(s: number | null | undefined): string {
  if (s == null || s <= 0) return "—";
  if (s < 60) return t("dur_s", s);
  if (s < 3600) return t("dur_m", Math.round(s / 60));
  const h = Math.floor(s / 3600);
  const m = Math.round((s % 3600) / 60);
  return m === 0 ? t("dur_h", h) : t("dur_hm", h, m);
}

/** Like fmtDuration but rolls over into days. */
export function fmtDurationLong(s: number | null | undefined): string {
  if (s == null || s <= 0) return "—";
  if (s < 86400) return fmtDuration(s);
  const d = Math.floor(s / 86400);
  const h = Math.round((s % 86400) / 3600);
  return h === 0 ? t("dur_d", d) : t("dur_dh", d, h);
}

export function fmtUptime(u: number | undefined): string {
  if (!u) return "—";
  if (u >= 99.995) return "100%";
  return u >= 99 ? u.toFixed(2) + "%" : u.toFixed(1) + "%";
}

export function kindLabel(c: { domain: string; kind: Kind }): string {
  if (c.domain === "api.bgm.tv") return t(c.kind === "guest" ? "kind_api_guest" : "kind_api_auth");
  return t(c.kind === "guest" ? "kind_guest" : "kind_auth");
}

export const componentLabel = (c: ComponentStatus) => `${c.domain} · ${kindLabel(c)}`;

export const regionFlag = (code: string) =>
  code.toUpperCase().replace(/./g, (c) => String.fromCodePoint(0x1f1e6 - 65 + c.charCodeAt(0)));

const REGION_LABEL: Record<string, string> = {
  jp: "Japan", cn: "China", hk: "Hong Kong", tw: "Taiwan", kr: "Korea", mn: "Mongolia",
  sg: "Singapore", my: "Malaysia", th: "Thailand", vn: "Vietnam", ph: "Philippines", id: "Indonesia",
  in: "India", au: "Australia", nz: "New Zealand",
  us: "US", ca: "Canada", mx: "Mexico", br: "Brazil", ar: "Argentina", cl: "Chile",
  gb: "UK", de: "Germany", fr: "France", nl: "Netherlands", be: "Belgium",
  ch: "Switzerland", at: "Austria", it: "Italy", es: "Spain", pt: "Portugal",
  se: "Sweden", no: "Norway", dk: "Denmark", fi: "Finland", ie: "Ireland",
  pl: "Poland", cz: "Czechia", ro: "Romania", hu: "Hungary", sk: "Slovakia",
  ua: "Ukraine", bg: "Bulgaria", hr: "Croatia", rs: "Serbia", lt: "Lithuania", lv: "Latvia", ee: "Estonia",
  ru: "Russia", kz: "Kazakhstan", tr: "Turkey", il: "Israel", ae: "UAE", sa: "Saudi Arabia",
  za: "South Africa", eg: "Egypt", ng: "Nigeria",
};
export const regionLabel = (code: string) => REGION_LABEL[code] ?? code;

/** Severity order for sorting: down first. */
export const severityRank = (s: string) => ({ down: 0, degraded: 1 })[s] ?? 9;

type StatusKey = "ok" | "degraded" | "down" | "none";
export const statusLabel = (s: string) => t(`status_${(s || "none") as StatusKey}`);
export const incidentLabel = (s: string) => t(s === "down" ? "incident_down" : "incident_degraded");
