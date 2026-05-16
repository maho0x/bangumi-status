(() => {
  "use strict";

  // --- i18n ----------------------------------------------------------------
  const I18N = {
    en: {
      status_ok: "Operational", status_degraded: "Degraded", status_down: "Outage",
      incident_down: "Service disruption", incident_degraded: "Degraded performance",
      banner_ok: "All systems operational", banner_degraded: "Partial degradation",
      banner_down: "Service disruption",
      kind_guest: "Guest access", kind_auth: "Logged-in access",
      kind_api_guest: "Public endpoint", kind_api_auth: "Authenticated",
      never: "never", just_now: "just now",
      s_ago: "s ago", m_ago: "m ago", h_ago: "h ago", d_ago: "d ago",
      today: "Today", yesterday: "Yesterday", n_days_ago: " days ago",
      uptime_30d: "Uptime 30d", probes_online: "Probes online",
      banner_sub_all: (n) => `All ${n} components reporting normally.`,
      banner_sub_affected: (a, n) => `${a} of ${n} components affected.`,
      days_ago_label: (n) => `${n}d ago`,
      uptime_label: "uptime", today_label: "today",
      no_incidents_label: "no incidents", fail_days_label: (n) => `${n} day${n === 1 ? "" : "s"} with incidents`,
      probe_count: (n) => `${n} probe${n === 1 ? "" : "s"}`,
      no_probe_data: "no probe data", no_probe_data_detail: "No recent probe data.",
      guest_show: "Show guest access", guest_hide: "Hide guest access",
      collapsed_guest_hint: (n) => `${n} public endpoint incident${n === 1 ? "" : "s"} hidden`,
      expanded_guest_hint: (n) => `${n} public endpoint incident${n === 1 ? "" : "s"} shown`,
      affected: (n) => `${n} affected`,
      no_incidents: "No incidents reported.", no_data: "No data yet.",
      failed_count: (n) => `${n} failed`, degraded_count: (n) => `${n} degraded`,
      of_checks: (n) => `of ${n} checks`,
      uptime_suffix: " uptime",
      probe_online_summary: (on, tot) => `${on} / ${tot} online`,
      probe_pill_online: "online", probe_pill_offline: "offline",
      no_probes: "No probes registered yet.",
      detected: "Detected", reported_from: "Reported from",
      probes_failing: (b, tot) => `${b} of ${tot} probes failing`,
      peak_failing: (b, tot) => `peak ${b}/${tot} probes failing`,
      inc_down_peak: (b, tot, d) => `Outage · peak ${b}/${tot} probes failing · lasted ~${d}`,
      inc_deg_peak: (b, tot, d) => `Degraded · peak ${b}/${tot} probes failing · lasted ~${d}`,
      inc_down: (d) => `Outage · lasted ~${d}`,
      inc_deg: (d) => `Degraded · lasted ~${d}`,
      dur_s: (n) => `${n}s`,
      dur_m: (n) => `${n}m`,
      dur_hm: (h, m) => `${h}h ${m}m`,
      dur_h: (n) => `${n}h`,
      ongoing: "ongoing",
      last_updated: "Last updated", auto_refresh: "auto-refresh 30s",
      atom_feed: "Atom feed", error_load: "Unable to load status",
      copy: "Copy", copied: "Copied", subscribe_btn: "Subscribe",
      section_current: "Current status by service", hint_30d: "30-day uptime",
      legend_ok: "Operational", legend_degraded: "Degraded",
      legend_down: "Outage", legend_none: "No data",

      section_unresolved: "Unresolved incidents",
      section_past: "Past incidents", hint_past: "Last 14 days",
      section_probes: "Probe nodes",
      section_online: "User activity",
      online_hint_24h: "last 24 hours",
      online_note: "Live “online” counter from bangumi.tv (signed-in view).",
      online_no_data: "No online samples yet.",
      online_current: (n) => `${n} online now`,
      online_peak: (n) => `peak ${n}`,
      now_label: "now",
      ago_label: (rel) => rel,
      footer_desc: "Independent, community-run availability monitor. Not affiliated with Bangumi.",
      modal_title: "Subscribe to updates",
      modal_intro: "Get notified when bgm.tv, bangumi.tv, chii.in or the Bangumi API experiences availability issues.",
      sub_atom_title: "Atom / RSS feed",
      sub_atom_desc: "Add this feed to any RSS reader for incident notifications.",
      sub_tg_title: "Telegram channel",
      sub_tg_desc: "Follow the channel to receive push notifications for outages and recoveries.",
      sub_live_title: "Live page",
      sub_live_desc: "This page auto-refreshes every 30 seconds — bookmark it for a quick at-a-glance check.",
      modal_foot: "No personal data is collected. This monitor is community-run and open source.",
    },
    zh: {
      status_ok: "正常", status_degraded: "降级", status_down: "中断",
      incident_down: "服务中断", incident_degraded: "性能降级",
      banner_ok: "完全正常", banner_degraded: "部分服务降级",
      banner_down: "服务中断",
      kind_guest: "公共端点", kind_auth: "认证端点",
      kind_api_guest: "公共端点", kind_api_auth: "认证端点",
      never: "从未", just_now: "刚刚",
      s_ago: " 秒前", m_ago: " 分钟前", h_ago: " 小时前", d_ago: " 天前",
      today: "今天", yesterday: "昨天", n_days_ago: " 天前",
      uptime_30d: "30天可用率", probes_online: "在线探针",
      banner_sub_all: (n) => `全部 ${n} 个服务运行正常。`,
      banner_sub_affected: (a, n) => `${n} 个服务中 ${a} 个受影响。`,
      days_ago_label: (n) => `${n}天前`,
      uptime_label: "可用率", today_label: "今日",
      no_incidents_label: "无故障", fail_days_label: (n) => `${n} 天有故障`,
      probe_count: (n) => `${n} 个探针`,
      no_probe_data: "暂无探针数据", no_probe_data_detail: "暂无近期探针数据。",
      guest_show: "显示公共端点", guest_hide: "隐藏公共端点",
      collapsed_guest_hint: (n) => `${n} 个公共端点事件已折叠`,
      expanded_guest_hint: (n) => `${n} 个公共端点事件已展开`,
      affected: (n) => `${n} 个受影响`,
      no_incidents: "无故障记录。", no_data: "暂无数据。",
      failed_count: (n) => `${n} 次失败`, degraded_count: (n) => `${n} 次降级`,
      of_checks: (n) => `/ ${n} 次检查`,
      uptime_suffix: " 可用率",
      probe_online_summary: (on, tot) => `${on} / ${tot} 在线`,
      probe_pill_online: "在线", probe_pill_offline: "离线",
      no_probes: "暂无探针注册。",
      detected: "检测于", reported_from: "来自",
      probes_failing: (b, tot) => `${b} / ${tot} 个探针异常`,
      peak_failing: (b, tot) => `峰值 ${b}/${tot} 个探针异常`,
      inc_down_peak: (b, tot, d) => `服务中断 · 峰值 ${b}/${tot} 探针异常 · 持续约 ${d}`,
      inc_deg_peak: (b, tot, d) => `性能降级 · 峰值 ${b}/${tot} 探针异常 · 持续约 ${d}`,
      inc_down: (d) => `服务中断 · 持续约 ${d}`,
      inc_deg: (d) => `性能降级 · 持续约 ${d}`,
      dur_s: (n) => `${n} 秒`,
      dur_m: (n) => `${n} 分钟`,
      dur_hm: (h, m) => `${h} 小时 ${m} 分钟`,
      dur_h: (n) => `${n} 小时`,
      ongoing: "进行中",
      last_updated: "最后更新", auto_refresh: "30秒自动刷新",
      atom_feed: "Atom 订阅", error_load: "无法加载状态",
      copy: "复制", copied: "已复制", subscribe_btn: "订阅",
      section_current: "当前服务状态", hint_30d: "30天可用率",
      legend_ok: "正常", legend_degraded: "降级",
      legend_down: "中断", legend_none: "无数据",

      section_unresolved: "未解决的事故",
      section_past: "历史事故", hint_past: "最近14天",
      section_probes: "探针节点",
      section_online: "在线人数",
      online_hint_24h: "最近24小时",
      online_note: "Bangumi online 人数历史。",
      online_no_data: "暂无在线数据。",
      online_current: (n) => `当前 ${n} 人在线`,
      online_peak: (n) => `峰值 ${n}`,
      now_label: "现在",
      ago_label: (rel) => rel,
      footer_desc: "社区运营的Bangumi可用性监测。",
      modal_title: "订阅更新",
      modal_intro: "当 bgm.tv、bangumi.tv、chii.in 或 Bangumi API 出现可用性问题时获得通知。",
      sub_atom_title: "Atom / RSS 订阅",
      sub_atom_desc: "将此订阅源添加到任何 RSS 阅读器以获取故障通知。",
      sub_tg_title: "Telegram 频道",
      sub_tg_desc: "关注频道以接收故障和恢复推送通知。",
      sub_live_title: "实时页面",
      sub_live_desc: "本页面每30秒自动刷新，收藏以便快速查看。",
      modal_foot: "本监测由社区运营并开源。",
    },
  };

  let lang = localStorage.getItem("lang") || "zh";
  const t = (key, ...args) => {
    const val = (I18N[lang] || I18N.zh)[key];
    return typeof val === "function" ? val(...args) : (val ?? key);
  };

  const regionFlag = code => code.toUpperCase().replace(/./g, c => String.fromCodePoint(0x1F1E6 - 65 + c.charCodeAt(0)));

  const REGION_LABEL = {
    // East Asia
    jp: "Japan", cn: "China", hk: "Hong Kong", tw: "Taiwan", kr: "Korea", mn: "Mongolia",
    // Southeast Asia
    sg: "Singapore", my: "Malaysia", th: "Thailand", vn: "Vietnam", ph: "Philippines", id: "Indonesia",
    // South Asia
    in: "India",
    // Oceania
    au: "Australia", nz: "New Zealand",
    // North America
    us: "US", ca: "Canada", mx: "Mexico",
    // South America
    br: "Brazil", ar: "Argentina", cl: "Chile",
    // Western Europe
    gb: "UK", de: "Germany", fr: "France", nl: "Netherlands", be: "Belgium",
    ch: "Switzerland", at: "Austria", it: "Italy", es: "Spain", pt: "Portugal",
    se: "Sweden", no: "Norway", dk: "Denmark", fi: "Finland", ie: "Ireland",
    // Central & Eastern Europe
    pl: "Poland", cz: "Czechia", ro: "Romania", hu: "Hungary", sk: "Slovakia",
    ua: "Ukraine", bg: "Bulgaria", hr: "Croatia", rs: "Serbia",
    lt: "Lithuania", lv: "Latvia", ee: "Estonia",
    // Russia & Central Asia
    ru: "Russia", kz: "Kazakhstan",
    // Middle East
    tr: "Turkey", il: "Israel", ae: "UAE", sa: "Saudi Arabia",
    // Africa
    za: "South Africa", eg: "Egypt", ng: "Nigeria",
  };
  const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  const ZH_MONTHS = ["一", "二", "三", "四", "五", "六", "七", "八", "九", "十", "十一", "十二"];

  const ICONS = {
    ok: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="5 12.5 10 17.5 19 7.5"/></svg>',
    degraded: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="6" x2="12" y2="13"/><circle cx="12" cy="17.5" r="0.2" stroke-width="3.2"/></svg>',
    down: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="6.5" y1="6.5" x2="17.5" y2="17.5"/><line x1="17.5" y1="6.5" x2="6.5" y2="17.5"/></svg>',
    loading: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><circle cx="6" cy="12" r="1.2"/><circle cx="12" cy="12" r="1.2"/><circle cx="18" cy="12" r="1.2"/></svg>',
  };

  const el = (tag, attrs = {}, children = []) => {
    const n = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs)) {
      if (k === "class") n.className = v;
      else if (k === "html") n.innerHTML = v;
      else if (k.startsWith("on") && typeof v === "function") n.addEventListener(k.slice(2), v);
      else if (v !== undefined && v !== null) n.setAttribute(k, v);
    }
    for (const c of [].concat(children)) {
      if (c == null) continue;
      if (typeof c === "string") n.appendChild(document.createTextNode(c));
      else n.appendChild(c);
    }
    return n;
  };

  const fmtRelative = (ts) => {
    if (!ts) return t("never");
    const diff = Math.floor(Date.now() / 1000 - ts);
    if (diff < 0) return t("just_now");
    if (diff < 60) return diff + t("s_ago");
    if (diff < 3600) return Math.floor(diff / 60) + t("m_ago");
    if (diff < 86400) return Math.floor(diff / 3600) + t("h_ago");
    return Math.floor(diff / 86400) + t("d_ago");
  };

  const fmtDayLabel = (isoDay) => {
    if (!isoDay) return "";
    const [y, m, d] = isoDay.split("-").map(Number);
    if (!y || !m || !d) return isoDay;
    if (lang === "zh") return `${y}年${m}月${d}日`;
    return `${MONTHS[m - 1]} ${d}, ${y}`;
  };

  const fmtDayRelative = (isoDay) => {
    if (!isoDay) return "";
    const [y, m, d] = isoDay.split("-").map(Number);
    const target = Date.UTC(y, m - 1, d);
    const now = new Date();
    const todayLocal = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const diff = Math.round((todayLocal - target) / 86400000);
    if (diff === 0) return t("today");
    if (diff === 1) return t("yesterday");
    if (diff > 0 && diff < 7) return diff + t("n_days_ago");
    return "";
  };

  const fmtTimeLocal = (ts) => {
    const d = new Date(ts * 1000);
    return String(d.getHours()).padStart(2, "0") + ":" + String(d.getMinutes()).padStart(2, "0");
  };

  const fmtDuration = (s) => {
    if (s == null || s <= 0) return "—";
    if (s < 60) return t("dur_s", s);
    if (s < 3600) return t("dur_m", Math.round(s / 60));
    const h = Math.floor(s / 3600);
    const m = Math.round((s % 3600) / 60);
    if (m === 0) return t("dur_h", h);
    return t("dur_hm", h, m);
  };

  const isoDayLocal = (ts) => {
    const d = new Date(ts * 1000);
    const y = d.getFullYear();
    const mo = String(d.getMonth() + 1).padStart(2, "0");
    const da = String(d.getDate()).padStart(2, "0");
    return `${y}-${mo}-${da}`;
  };

  const fmtUptime = (u) => {
    if (u === 0 || u == null) return "—";
    if (u >= 99.995) return "100%";
    if (u >= 99) return u.toFixed(2) + "%";
    return u.toFixed(1) + "%";
  };

  const kindLabel = (c) => {
    if (c.domain === "api.bgm.tv") return t(c.kind === "guest" ? "kind_api_guest" : "kind_api_auth");
    return t(c.kind === "guest" ? "kind_guest" : "kind_auth");
  };

  const componentLabel = (c) => `${c.domain} · ${kindLabel(c)}`;

  // --- Tooltip -------------------------------------------------------------
  const tip = document.getElementById("tooltip");
  const showTip = (e, text) => {
    tip.textContent = text;
    tip.hidden = false;
    const r = e.currentTarget.getBoundingClientRect();
    const tw = tip.offsetWidth;
    let x = r.left + r.width / 2 - tw / 2;
    x = Math.max(6, Math.min(x, window.innerWidth - tw - 6));
    const y = r.top - tip.offsetHeight - 8;
    tip.style.left = x + "px";
    tip.style.top = (y < 6 ? r.bottom + 6 : y) + "px";
  };
  const hideTip = () => { tip.hidden = true; };

  // --- Banner --------------------------------------------------------------
  function renderBanner(overall) {
    const banner = document.getElementById("banner");
    const meta = document.getElementById("banner-meta");

    // Guest/public endpoint failures should not escalate the top banner;
    // they surface as a small badge on the affected group instead.
    const rank = { ok: 0, degraded: 1, down: 2 };
    const nonGuest = overall.components.filter(c => c.kind !== "guest");
    let status = "ok";
    for (const c of nonGuest) {
      if ((rank[c.status] || 0) > (rank[status] || 0)) status = c.status;
    }

    banner.className = "banner banner--" + status;
    banner.querySelector(".banner__icon").innerHTML = ICONS[status] || ICONS.loading;
    banner.querySelector(".banner__title").textContent = t("banner_" + status) || overall.message || "Status";

    const affected = nonGuest.filter(c => c.status !== "ok");
    const total = nonGuest.length;
    banner.querySelector(".banner__sub").textContent = affected.length === 0
      ? t("banner_sub_all", total)
      : t("banner_sub_affected", affected.length, total);

    const avg = nonGuest.reduce((a, c) => a + (c.uptime || 0), 0) / (nonGuest.length || 1);
    const online = overall.probes.filter(p => p.online).length;

    meta.innerHTML = "";
    meta.appendChild(el("div", {}, [el("dt", {}, t("uptime_30d")), el("dd", { class: "num" }, fmtUptime(avg))]));

    document.getElementById("updated").textContent = fmtRelative(overall.updated_at);
  }

  // --- Unresolved incidents ------------------------------------------------
  function renderUnresolvedIncidents(overall) {
    const section = document.getElementById("unresolved-section");
    const list = document.getElementById("unresolved-list");
    const summary = document.getElementById("unresolved-summary");
    list.innerHTML = "";

    const affected = (overall.components || []).filter(c => c.status && c.status !== "ok");
    if (affected.length === 0) { section.hidden = true; return; }
    section.hidden = false;
    summary.textContent = t("affected", affected.length);

    for (const c of affected) {
      const views = c.probe_views || [];
      const badProbes = views.filter(v => v.status === "down" || v.status === "degraded");
      const regions = [...new Set(badProbes.map(v => v.region))].filter(Boolean);
      const metaBits = [];
      metaBits.push(el("span", { class: "mono" }, t("detected") + " " + fmtRelative(c.last_check)));
      if (regions.length > 0) {
        metaBits.push(el("span", { class: "sep" }, "·"));
        metaBits.push(el("span", {}, t("reported_from") + " " + regions.map(r => REGION_LABEL[r] || r).join(", ")));
      }
      if (badProbes.length > 0 && views.length > 0) {
        metaBits.push(el("span", { class: "sep" }, "·"));
        metaBits.push(el("span", {}, t("probes_failing", badProbes.length, views.length)));
      }

      list.appendChild(el("div", { class: "inc-card inc-card--" + c.status }, [
        el("div", { class: "inc-card__icon", html: ICONS[c.status] || ICONS.degraded }),
        el("div", { class: "inc-card__body" }, [
          el("div", { class: "inc-card__title" }, [
            el("span", { class: "inc-title__sev" }, t("incident_" + c.status) || "Incident"),
            document.createTextNode(" — " + componentLabel(c)),
          ]),
          el("div", { class: "inc-card__meta" }, metaBits),
        ]),
      ]));
    }
  }

  // --- Past incidents ------------------------------------------------------
  function renderPastIncidents(overall) {
    const container = document.getElementById("past-incidents");
    container.innerHTML = "";

    const comps = overall.components || [];
    if (comps.length === 0) {
      container.appendChild(el("div", { class: "inc-day__none" }, t("no_data")));
      return;
    }

    // Collect all incident windows across components within last 14 days,
    // indexed by UTC day of start_ts so a single event stays on one row.
    const byDay = new Map();
    // Seed last 14 days (from canonical day bucket list) so empty days still render.
    const canonical = (comps[0].days || []).slice().reverse().slice(0, 14);
    for (const b of canonical) byDay.set(b.day, []);

    const nowSec = Math.floor(Date.now() / 1000);
    const cutoff = nowSec - 14 * 86400;

    for (const c of comps) {
      for (const inc of (c.incidents || [])) {
        if (!inc || inc.start_ts < cutoff) continue;
        const iso = isoDayLocal(inc.start_ts);
        if (!byDay.has(iso)) byDay.set(iso, []);
        byDay.get(iso).push({ ...inc, component: componentLabel(c), kind: c.kind });
      }
    }

    // Skip any unresolved window whose end is still "now" — that's already
    // shown in the unresolved-incidents section above.
    const unresolvedStarts = new Set();
    for (const c of comps) {
      if (c.status && c.status !== "ok" && (c.incidents || []).length) {
        const last = c.incidents[c.incidents.length - 1];
        if (last && nowSec - last.end_ts < 240) unresolvedStarts.add(c.domain + "|" + c.kind + "|" + last.start_ts);
      }
    }

    const sortedDays = [...byDay.keys()].sort().reverse();

    for (const iso of sortedDays) {
      const entries = byDay.get(iso).slice().sort((a, b) => {
        if (b.start_ts !== a.start_ts) return b.start_ts - a.start_ts;
        const rank = { down: 0, degraded: 1 };
        return (rank[a.status] ?? 9) - (rank[b.status] ?? 9);
      });

      let hasGuest = false, hasAuth = false;
      for (const inc of entries) {
        if (inc.kind === "guest") hasGuest = true;
        else if (inc.kind === "auth") hasAuth = true;
      }

      const dayBlock = el("div", { class: "inc-day" + (hasGuest ? " has-guest" : "") + (hasAuth ? " has-auth" : ""), "data-day": iso });
      const rel = fmtDayRelative(iso);
      dayBlock.appendChild(el("div", { class: "inc-day__date" }, [
        document.createTextNode(fmtDayLabel(iso)),
        rel ? el("span", { class: "inc-day__rel" }, rel) : null,
      ]));

      if (entries.length === 0) {
        dayBlock.appendChild(el("div", { class: "inc-day__none" }, t("no_incidents")));
      } else {
        const listEl = el("div", { class: "inc-day__list" });
        for (const inc of entries) {
          const endIso = isoDayLocal(inc.end_ts);
          const crossDay = endIso !== iso;
          const isOngoing = nowSec - inc.end_ts < 240;
          const endLabel = isOngoing ? t("ongoing") : (fmtTimeLocal(inc.end_ts) + (crossDay ? "⁺¹" : ""));
          const timeRange = fmtTimeLocal(inc.start_ts) + " – " + endLabel;
          const peakBits = [];
          if (inc.peak_down && inc.peak_total) {
            peakBits.push(t("peak_failing", inc.peak_down, inc.peak_total));
          }
          peakBits.push(fmtDuration(inc.end_ts - inc.start_ts));
          const descText = timeRange + " · " + peakBits.join(" · ");

          listEl.appendChild(el("div", { class: "inc-day__entry", "data-kind": inc.kind || "" }, [
            el("span", { class: "inc-day__dot status-" + inc.status }),
            el("div", { class: "label" }, [
              el("span", { class: "sev sev--" + inc.status }, t("incident_" + inc.status) || "Incident"),
              document.createTextNode(" — "),
              el("span", { class: "comp" }, inc.component),
              el("span", { class: "desc" }, descText),
            ]),
            el("span", { class: "metric" }, fmtRelative(inc.end_ts)),
          ]));
        }
        dayBlock.appendChild(listEl);
        if (hasGuest) {
          const guestCount = entries.filter(e => e.kind === "guest").length;
          const updateLabel = () => {
            collapsed.textContent = dayBlock.classList.contains("show-guest")
              ? t("expanded_guest_hint", guestCount)
              : t("collapsed_guest_hint", guestCount);
          };
          const toggle = () => {
            dayBlock.classList.toggle("show-guest");
            updateLabel();
          };
          const collapsed = el("div", {
            class: "inc-day__collapsed",
            role: "button",
            tabindex: "0",
            "data-guest-count": String(guestCount),
            onclick: toggle,
            onkeydown: (e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                toggle();
              }
            },
          }, t("collapsed_guest_hint", guestCount));
          dayBlock.appendChild(collapsed);
        }
      }

      container.appendChild(dayBlock);
    }
  }

  // --- Online users chart --------------------------------------------------
  let onlineRange = "24h";

  function drawOnlineChart(rawPts) {
    const chartEl = document.getElementById("online-chart");
    const summaryEl = document.getElementById("online-summary");
    if (!chartEl || !summaryEl) return;

    const pts = (rawPts || []).filter(p => p && p.count > 0);
    if (pts.length < 2) {
      chartEl.innerHTML = `<div class="online-card__empty">${t("online_no_data")}</div>`;
      summaryEl.textContent = "—";
      return;
    }

    const last = pts[pts.length - 1];
    const values = pts.map(p => p.count);
    const vMin = Math.min(...values);
    const vMax = Math.max(...values);
    const peakPt = pts.reduce((a, b) => (b.count >= a.count ? b : a), pts[0]);
    summaryEl.textContent = t("online_current", last.count.toLocaleString()) +
      " · " + t("online_peak", vMax.toLocaleString());

    const W = Math.max(300, chartEl.clientWidth || 800), H = 140, padL = 42, padR = 10, padT = 12, padB = 20;
    const iw = W - padL - padR, ih = H - padT - padB;
    const t0 = pts[0].ts, tN = pts[pts.length - 1].ts;
    const tSpan = Math.max(60, tN - t0);
    const pad = Math.max(1, (vMax - vMin) * 0.15);
    const yLo = Math.max(0, vMin - pad);
    const yHi = vMax + pad;
    const ySpan = Math.max(1, yHi - yLo);

    const xFor = ts => padL + ((ts - t0) / tSpan) * iw;
    const yFor = v => padT + (1 - (v - yLo) / ySpan) * ih;

    let line = "";
    pts.forEach((p, i) => {
      line += (i === 0 ? "M" : "L") + xFor(p.ts).toFixed(1) + "," + yFor(p.count).toFixed(1);
    });
    const area = line + "L" + xFor(tN).toFixed(1) + "," + (padT + ih) + "L" + xFor(t0).toFixed(1) + "," + (padT + ih) + "Z";

    const yTicks = [];
    for (let i = 0; i <= 3; i++) {
      const v = yLo + (ySpan * i) / 3;
      yTicks.push({ v: Math.round(v), y: yFor(v) });
    }

    // X-axis labels: format depends on range
    function fmtXLabel(ts) {
      const d = new Date(ts * 1000);
      if (onlineRange === "24h") {
        return String(d.getHours()).padStart(2, "0") + ":" + String(d.getMinutes()).padStart(2, "0");
      }
      return (d.getMonth() + 1) + "/" + d.getDate();
    }
    const xLabelStart = fmtXLabel(t0);
    const xLabelEnd = onlineRange === "24h" ? t("now_label") : fmtXLabel(tN);

    // Peak marker (always visible). Anchor label to opposite side from chart edge.
    const peakX = xFor(peakPt.ts);
    const peakY = yFor(peakPt.count);
    const peakD = new Date(peakPt.ts * 1000);
    const peakTimeLabel = onlineRange === "24h"
      ? String(peakD.getHours()).padStart(2, "0") + ":" + String(peakD.getMinutes()).padStart(2, "0")
      : `${peakD.getMonth() + 1}/${peakD.getDate()}`;
    const peakLabel = `${t("online_peak", peakPt.count.toLocaleString())} · ${peakTimeLabel}`;
    const peakLabelAnchor = peakX > padL + iw * 0.6 ? "end" : "start";
    const peakLabelDx = peakLabelAnchor === "end" ? -8 : 8;
    const peakLabelY = peakY < padT + 14 ? peakY + 14 : peakY - 6;

    chartEl.innerHTML = `<svg viewBox="0 0 ${W} ${H}" preserveAspectRatio="none" role="img" aria-label="online users over time">
      ${yTicks.map(tk => `<line class="grid" x1="${padL}" y1="${tk.y.toFixed(1)}" x2="${W - padR}" y2="${tk.y.toFixed(1)}"/>`).join("")}
      ${yTicks.map(tk => `<text class="axis" x="${padL - 6}" y="${(tk.y + 3).toFixed(1)}" text-anchor="end">${tk.v.toLocaleString()}</text>`).join("")}
      <text class="axis" x="${padL}" y="${H - 5}">${xLabelStart}</text>
      <text class="axis" x="${W - padR}" y="${H - 5}" text-anchor="end">${xLabelEnd}</text>
      <path class="area" d="${area}"/>
      <path class="line" d="${line}"/>
      <circle class="peak-dot" cx="${peakX.toFixed(1)}" cy="${peakY.toFixed(1)}" r="3.5"/>
      <text class="peak-label" x="${(peakX + peakLabelDx).toFixed(1)}" y="${peakLabelY.toFixed(1)}" text-anchor="${peakLabelAnchor}">${peakLabel}</text>
      <line class="cursor" x1="0" y1="${padT}" x2="0" y2="${padT + ih}"/>
      <circle class="marker" r="3.5" cx="0" cy="0"/>
      <rect class="hit" x="${padL}" y="${padT}" width="${iw}" height="${ih}"/>
    </svg>`;

    const svgEl = chartEl.querySelector("svg");
    const cursor = svgEl.querySelector(".cursor");
    const marker = svgEl.querySelector(".marker");
    const hit = svgEl.querySelector(".hit");

    const onMove = (e) => {
      const rect = svgEl.getBoundingClientRect();
      const pctX = (e.clientX - rect.left) / rect.width;
      const xViewport = pctX * W;
      const tTarget = t0 + ((xViewport - padL) / iw) * tSpan;
      let best = pts[0], bestDiff = Infinity;
      for (const p of pts) {
        const d = Math.abs(p.ts - tTarget);
        if (d < bestDiff) { bestDiff = d; best = p; }
      }
      const x2 = xFor(best.ts), y2 = yFor(best.count);
      cursor.setAttribute("x1", x2);
      cursor.setAttribute("x2", x2);
      marker.setAttribute("cx", x2);
      marker.setAttribute("cy", y2);

      const d = new Date(best.ts * 1000);
      const label = onlineRange === "24h"
        ? String(d.getHours()).padStart(2, "0") + ":" + String(d.getMinutes()).padStart(2, "0")
        : `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,"0")}-${String(d.getDate()).padStart(2,"0")} ${String(d.getHours()).padStart(2,"0")}:00`;
      tip.textContent = `${label} · ${best.count.toLocaleString()}`;
      tip.hidden = false;
      const absX = rect.left + (x2 / W) * rect.width;
      const absY = rect.top + (y2 / H) * rect.height;
      const tw = tip.offsetWidth;
      let tx = absX - tw / 2;
      tx = Math.max(6, Math.min(tx, window.innerWidth - tw - 6));
      tip.style.left = tx + "px";
      tip.style.top = (absY - tip.offsetHeight - 10) + "px";
    };
    hit.addEventListener("mousemove", onMove);
    hit.addEventListener("mouseleave", hideTip);
  }

  function renderOnlineChart(overall) {
    const noteEl = document.getElementById("online-note");
    if (noteEl) noteEl.textContent = t("online_note");
    // Wire up range tabs once
    const tabsEl = document.getElementById("online-tabs");
    if (tabsEl && !tabsEl._wired) {
      tabsEl._wired = true;
      tabsEl.addEventListener("click", async (e) => {
        const btn = e.target.closest(".online-tab");
        if (!btn) return;
        const range = btn.dataset.range;
        if (range === onlineRange) return;
        onlineRange = range;
        tabsEl.querySelectorAll(".online-tab").forEach(b => {
          b.classList.toggle("active", b === btn);
          b.setAttribute("aria-selected", b === btn ? "true" : "false");
        });
        const chartEl = document.getElementById("online-chart");
        if (chartEl) chartEl.innerHTML = `<div class="online-card__empty" style="opacity:.5">${t("online_no_data")}</div>`;
        try {
          const res = await fetch(`/api/online?range=${range}`);
          const pts = await res.json();
          drawOnlineChart(pts);
        } catch (_) {}
      });
    }
    if (onlineRange === "24h") {
      drawOnlineChart(overall.online);
    } else {
      fetch(`/api/online?range=${onlineRange}`)
        .then(r => r.json())
        .then(pts => drawOnlineChart(pts))
        .catch(() => {});
    }
  }

  // --- Component row -------------------------------------------------------
  function renderComponent(c) {
    const row = el("div", {
      class: "component",
      "data-key": `${c.domain}-${c.kind}`,
      "data-kind": c.kind || "",
      "data-status": c.status || "none",
    });

    const label = el("div", { class: "component-label" }, [
      el("span", { class: "status-dot status-" + (c.status || "none") }),
      el("span", { class: "label-text" }, kindLabel(c)),
      el("span", { class: "status-text" }, "· " + (t("status_" + c.status) || "unknown")),
    ]);
    row.appendChild(label);

    const right = el("div", { class: "component-right" });
    const strip = el("div", { class: "strip" });

    const days = c.days || [];
    for (const d of days) {
      const cls = d.total === 0 ? "cell-none" : "cell-" + (d.status || "none");
      strip.appendChild(el("div", {
        class: "strip-cell " + cls,
        onmouseenter: (e) => showTip(e, tooltipForDay(d)),
        onmouseleave: hideTip,
      }));
    }
    // Pad empty cells if local "today" is ahead of the last UTC bucket.
    const todayIso = isoDayLocal(Math.floor(Date.now() / 1000));
    let lastDay = days[days.length - 1]?.day;
    while (lastDay && lastDay < todayIso) {
      const [y, m, d] = lastDay.split("-").map(Number);
      const next = new Date(y, m - 1, d + 1);
      lastDay = `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, "0")}-${String(next.getDate()).padStart(2, "0")}`;
      strip.appendChild(el("div", {
        class: "strip-cell cell-none",
        onmouseenter: (e) => showTip(e, `${lastDay} · no data`),
        onmouseleave: hideTip,
      }));
    }

    const views = c.probe_views || [];
    const toggle = el("button", {
      class: "probe-toggle",
      type: "button",
      "aria-label": "toggle probe detail",
      onclick: () => row.classList.toggle("open"),
    }, [
      el("span", { class: "probe-toggle__count" }, String(views.length)),
      el("span", { class: "probe-toggle__chevron" }),
    ]);

    right.appendChild(strip);
    right.appendChild(el("span", { class: "mid" }, fmtUptime(c.uptime) + " " + t("uptime_label")));
    right.appendChild(toggle);
    row.appendChild(right);

    const detail = el("div", { class: "probe-detail" });
    if (views.length > 0) {
      const listEl = el("div");
      for (const v of views) {
        listEl.appendChild(el("div", { class: "probe-row" }, [
          el("span", { class: "probe-name" }, (v.region ? regionFlag(v.region) + " " : "") + v.probe),
          el("span", { class: "probe-err" }, v.err || (v.http_code ? `HTTP ${v.http_code}` : "—")),
          el("span", { class: "probe-lat" }, v.latency_ms != null ? v.latency_ms + " ms" : "—"),
          el("span", { class: "probe-status" }, [
            el("span", { class: "status-dot status-" + (v.status || "none") }),
            el("span", {}, t("status_" + v.status) || "—"),
          ]),
        ]));
      }
      detail.appendChild(listEl);
    } else {
      detail.appendChild(el("div", {
        class: "probe-row",
        style: "color:var(--text-faint);grid-template-columns:1fr",
      }, t("no_probe_data_detail")));
    }
    row.appendChild(detail);

    return row;
  }

  function tooltipForDay(d) {
    if (!d.total) return `${d.day} · no data`;
    const bits = [`${d.total} checks`];
    if (d.down) bits.push(`${d.down} down`);
    if (d.degrade) bits.push(`${d.degrade} degraded`);
    return `${d.day} · ${fmtUptime(d.uptime)} · ${bits.join(", ")}`;
  }

  // --- Component groups ----------------------------------------------------
  function renderComponents(overall) {
    const container = document.getElementById("components");
    container.innerHTML = "";

    const byDomain = {};
    for (const c of overall.components) {
      (byDomain[c.domain] = byDomain[c.domain] || []).push(c);
    }

    for (const domain of ["bgm.tv", "bangumi.tv", "chii.in", "next.bgm.tv", "next.bgm.tv/p1", "api.bgm.tv"]) {
      const comps = byDomain[domain] || [];
      if (comps.length === 0) continue;
      const hasGuest = comps.some(c => c.kind === "guest");
      const hasAuth  = comps.some(c => c.kind === "auth");
      const guestIssue = comps.some(c => c.kind === "guest" && c.status && c.status !== "ok");
      const group = el("div", { class: "group" + (guestIssue || !hasAuth ? " show-guest" : ""), "data-domain": domain });

      const hdrRight = el("div", { class: "group-hdr-right" });
      const guestBad = comps.filter(c => c.kind === "guest" && c.status && c.status !== "ok");
      if (guestBad.length > 0) {
        const worst = guestBad.some(c => c.status === "down") ? "down" : "degraded";
        hdrRight.appendChild(el("span", {
          class: "guest-badge guest-badge--" + worst,
          title: t("collapsed_guest_hint", guestBad.length),
        }, String(guestBad.length)));
      }
      if (hasGuest && hasAuth) {
        const btn = el("button", {
          class: "guest-toggle",
          type: "button",
          "aria-pressed": guestIssue ? "true" : "false",
          onclick: (e) => {
            const shown = group.classList.toggle("show-guest");
            e.currentTarget.setAttribute("aria-pressed", shown ? "true" : "false");
            e.currentTarget.querySelector(".guest-toggle__label").textContent =
              shown ? t("guest_hide") : t("guest_show");
          },
        }, [
          el("span", { class: "guest-toggle__chevron", "aria-hidden": "true" }),
          el("span", { class: "guest-toggle__label" }, guestIssue ? t("guest_hide") : t("guest_show")),
        ]);
        hdrRight.appendChild(btn);
      }

      group.appendChild(el("div", { class: "group-hdr" }, [
        el("span", { class: "domain-name" }, [
          el("span", { class: "status-dot status-" + groupStatus(comps) }),
          domain,
        ]),
        hdrRight,
      ]));
      for (const c of comps) group.appendChild(renderComponent(c));
      container.appendChild(group);
    }
  }

  function groupStatus(comps) {
    const rank = { ok: 0, degraded: 1, down: 2 };
    let worst = "ok";
    for (const c of comps) {
      if ((rank[c.status] || 0) > (rank[worst] || 0)) worst = c.status;
    }
    return worst;
  }

  // --- Probes --------------------------------------------------------------
  function renderProbes(overall) {
    const list = document.getElementById("probe-list");
    const summary = document.getElementById("probes-summary");
    list.innerHTML = "";

    const probes = overall.probes || [];
    if (probes.length === 0) {
      summary.textContent = "—";
      list.appendChild(el("div", {
        style: "color:var(--text-faint);font-size:13px;padding:12px 2px",
      }, t("no_probes")));
      return;
    }

    const online = probes.filter(p => p.online).length;
    summary.textContent = t("probe_online_summary", online, probes.length);

    for (const p of probes) {
      list.appendChild(el("div", { class: "probe-card " + (p.online ? "online" : "offline") }, [
        el("div", {}, [
          el("div", { class: "name" }, p.name),
          el("div", { class: "region" }, (p.region ? regionFlag(p.region) + " " : "") + (REGION_LABEL[p.region] || p.region) + " · " + fmtRelative(p.last_seen)),
        ]),
        el("span", { class: "pill" }, p.online ? t("probe_pill_online") : t("probe_pill_offline")),
      ]));
    }
  }

  // --- Static i18n ---------------------------------------------------------
  function applyI18n() {
    const setT = (sel, key) => { const n = document.querySelector(sel); if (n) n.textContent = t(key); };

    setT(".components-section .section-head h2", "section_current");
    setT(".components-section .section-head .section-head__hint", "hint_30d");
    setT("#unresolved-section h2", "section_unresolved");
    setT(".inc-section:not(#unresolved-section) .section-head h2", "section_past");
    setT(".inc-section:not(#unresolved-section) .section-head__hint", "hint_past");
    setT(".probes-section h2", "section_probes");
    setT(".online-section h2", "section_online");

    setT(".ftr__desc", "footer_desc");
    setT("#subscribe-title", "modal_title");
    setT(".modal__intro", "modal_intro");
    setT(".modal__foot", "modal_foot");
    setT(".ftr__link", "atom_feed");

    // Subscribe button label
    const subBtnLabel = document.querySelector("#subscribe-btn span");
    if (subBtnLabel) subBtnLabel.textContent = t("subscribe_btn");

    // Lang button
    const langBtn = document.getElementById("lang-btn");
    if (langBtn) langBtn.textContent = lang === "zh" ? "EN" : "中文";

    // Legend items (text node after dot span)
    const legendKeys = ["legend_ok", "legend_degraded", "legend_down", "legend_none"];
    document.querySelectorAll(".legend__item").forEach((item, i) => {
      if (!legendKeys[i]) return;
      for (const node of item.childNodes) {
        if (node.nodeType === Node.TEXT_NODE) { node.textContent = t(legendKeys[i]); break; }
      }
    });

    // Modal sub-options
    const subTitles = document.querySelectorAll(".sub-option__title");
    const subDescs  = document.querySelectorAll(".sub-option__desc");
    if (subTitles[0]) subTitles[0].textContent = t("sub_atom_title");
    if (subTitles[1]) subTitles[1].textContent = t("sub_tg_title");
    if (subTitles[2]) subTitles[2].textContent = t("sub_live_title");
    if (subDescs[0])  subDescs[0].textContent  = t("sub_atom_desc");
    if (subDescs[1])  subDescs[1].textContent  = t("sub_tg_desc");
    if (subDescs[2])  subDescs[2].textContent  = t("sub_live_desc");

    // Footer meta: "Last updated <span#updated> · auto-refresh ..."
    const ftrSpans = document.querySelectorAll(".ftr__meta > span");
    if (ftrSpans[0]) {
      for (const node of ftrSpans[0].childNodes) {
        if (node.nodeType === Node.TEXT_NODE) { node.textContent = t("last_updated") + " "; break; }
      }
    }
    if (ftrSpans[2]) ftrSpans[2].textContent = t("auto_refresh");

    // Copy button (only if not mid-copy)
    const copyBtn = document.getElementById("copy-feed");
    if (copyBtn && !copyBtn.classList.contains("copied")) copyBtn.textContent = t("copy");

    document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  }

  // --- Subscribe modal -----------------------------------------------------
  const modal = document.getElementById("subscribe-modal");
  const feedInput = document.getElementById("feed-url");
  const feedURL = window.location.origin + "/api/feed.atom";
  if (feedInput) feedInput.value = feedURL;

  function openModal() {
    modal.hidden = false;
    document.body.style.overflow = "hidden";
    setTimeout(() => {
      const closer = modal.querySelector("[data-close]");
      closer && closer.focus({ preventScroll: true });
    }, 20);
  }
  function closeModal() {
    modal.hidden = true;
    document.body.style.overflow = "";
  }

  document.getElementById("subscribe-btn").addEventListener("click", openModal);
  modal.addEventListener("click", (e) => {
    if (e.target.matches("[data-close]") || e.target.closest("[data-close]")) closeModal();
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !modal.hidden) closeModal();
  });

  const copyBtn = document.getElementById("copy-feed");
  if (copyBtn) {
    copyBtn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(feedURL);
      } catch (_) {
        feedInput.select();
        try { document.execCommand("copy"); } catch (_) {}
      }
      copyBtn.textContent = t("copied");
      copyBtn.classList.add("copied");
      setTimeout(() => {
        copyBtn.textContent = t("copy");
        copyBtn.classList.remove("copied");
      }, 1600);
    });
  }

  // --- Language switch -----------------------------------------------------
  const langBtn = document.getElementById("lang-btn");
  if (langBtn) {
    langBtn.addEventListener("click", () => {
      lang = lang === "zh" ? "en" : "zh";
      localStorage.setItem("lang", lang);
      applyI18n();
      if (lastData) render(lastData);
    });
  }

  // --- Render & Refresh ----------------------------------------------------
  let lastData = null;

  function render(data) {
    // Save user expansion states before re-rendering
    const openComponents = new Set();
    document.querySelectorAll('.component.open').forEach(el => openComponents.add(el.getAttribute('data-key')));
    const showGuestGroups = new Set();
    document.querySelectorAll('.group.show-guest').forEach(el => showGuestGroups.add(el.getAttribute('data-domain')));
    const hiddenGuestGroups = new Set();
    document.querySelectorAll('.group:not(.show-guest)').forEach(el => hiddenGuestGroups.add(el.getAttribute('data-domain')));
    const expandedDays = new Set();
    document.querySelectorAll('.inc-day.show-guest').forEach(el => {
      const day = el.getAttribute('data-day');
      if (day) expandedDays.add(day);
    });
    const probesExpanded = document.getElementById('probes-toggle')?.getAttribute('aria-expanded') === 'true';

    renderBanner(data);
    refreshReactions();
    renderUnresolvedIncidents(data);
    renderComponents(data);
    renderOnlineChart(data);
    renderPastIncidents(data);
    renderProbes(data);

    // Restore expansion states
    document.querySelectorAll('.component').forEach(el => {
      if (openComponents.has(el.getAttribute('data-key'))) el.classList.add('open');
    });
    document.querySelectorAll('.group').forEach(el => {
      const domain = el.getAttribute('data-domain');
      if (hiddenGuestGroups.has(domain)) {
        el.classList.remove('show-guest');
        const btn = el.querySelector('.guest-toggle');
        if (btn) {
          btn.setAttribute('aria-pressed', 'false');
          btn.querySelector('.guest-toggle__label').textContent = t('guest_show');
        }
      } else if (showGuestGroups.has(domain)) {
        el.classList.add('show-guest');
        const btn = el.querySelector('.guest-toggle');
        if (btn) {
          btn.setAttribute('aria-pressed', 'true');
          btn.querySelector('.guest-toggle__label').textContent = t('guest_hide');
        }
      }
    });
    document.querySelectorAll('.inc-day').forEach(el => {
      const day = el.getAttribute('data-day');
      if (day && expandedDays.has(day)) {
        el.classList.add('show-guest');
        const collapsed = el.querySelector('.inc-day__collapsed');
        if (collapsed && collapsed.dataset.guestCount) {
          const n = Number(collapsed.dataset.guestCount);
          collapsed.textContent = t('expanded_guest_hint', n);
        }
      }
    });
    if (probesExpanded) {
      const pt = document.getElementById('probes-toggle');
      const pl = document.getElementById('probe-list');
      if (pt) pt.setAttribute('aria-expanded', 'true');
      if (pl) pl.hidden = false;
    }
  }

  // Probes section toggle (collapsed by default)
  const probesToggle = document.getElementById("probes-toggle");
  const probeList = document.getElementById("probe-list");
  if (probesToggle && probeList) {
    const toggle = () => {
      const expanded = probesToggle.getAttribute("aria-expanded") === "true";
      probesToggle.setAttribute("aria-expanded", String(!expanded));
      probeList.hidden = expanded;
    };
    probesToggle.addEventListener("click", toggle);
    probesToggle.addEventListener("keydown", e => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggle(); } });
  }

  // --- Reactions -----------------------------------------------------------
  const REACTION_IDS = [44, 40, 15, 23, 83, 65, 41, 102, 49, 46, 51, 101];
  function getReactionUserID() {
    let id = localStorage.getItem("rx_uid");
    if (!id || id.length < 8) {
      id = (crypto.randomUUID ? crypto.randomUUID() : (Date.now().toString(36) + Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2))).replace(/-/g, "");
      localStorage.setItem("rx_uid", id);
    }
    return id;
  }
  const reactionUID = getReactionUserID();
  let reactionState = REACTION_IDS.map(id => ({ emoji_id: id, count: 0, mine: false }));
  const reactionCooldown = new Map(); // emoji_id -> last-click ms

  function renderReactions() {
    const actives = document.getElementById("rx-actives");
    const grid = document.getElementById("rx-grid");
    if (!actives || !grid) return;
    const byID = {};
    for (const r of reactionState) byID[r.emoji_id] = r;

    // Active chips: only count > 0, use bangumi .item markup with bg-image emoji.
    actives.innerHTML = "";
    for (const id of REACTION_IDS) {
      const r = byID[id];
      if (!r || (r.count || 0) <= 0) continue;
      const btn = document.createElement("button");
      btn.type = "button";
      btn.dataset.eid = id;
      btn.className = "item" + (r.mine ? " selected" : "");
      btn.title = "再贴一个";
      btn.innerHTML =
        '<span class="emoji" style="background-image:url(\'https://chii.in/img/smiles/tv/' + id + '.gif\')"></span>' +
        '<span class="num">' + r.count + '</span>';
      btn.addEventListener("click", (e) => { e.stopPropagation(); addReaction(id, e.currentTarget); });
      actives.appendChild(btn);
    }

    // Picker grid: all 12 emoji.
    grid.innerHTML = "";
    for (const id of REACTION_IDS) {
      const r = byID[id] || { emoji_id: id, count: 0, mine: false };
      const li = document.createElement("li");
      const a = document.createElement("a");
      a.href = "javascript:void(0)";
      a.className = r.mine ? "is-mine" : "";
      a.title = "贴贴";
      a.innerHTML = '<img class="emoji" src="https://chii.in/img/smiles/tv/' + id + '.gif" alt="" loading="lazy" decoding="async">';
      a.addEventListener("click", (e) => { e.stopPropagation(); addReaction(id, e.currentTarget); });
      li.appendChild(a);
      grid.appendChild(li);
    }
  }

  (function setupReactionDropdown() {
    const dd = document.getElementById("rx-dd");
    const trigger = document.getElementById("rx-trigger");
    if (!dd || !trigger) return;
    const setOpen = (open) => {
      dd.classList.toggle("open", open);
      trigger.setAttribute("aria-expanded", open ? "true" : "false");
    };
    trigger.addEventListener("click", (e) => {
      e.stopPropagation();
      setOpen(!dd.classList.contains("open"));
    });
    document.addEventListener("click", (e) => {
      if (!dd.classList.contains("open")) return;
      if (dd.contains(e.target)) return;
      setOpen(false);
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && dd.classList.contains("open")) setOpen(false);
    });
  })();

  async function refreshReactions() {
    try {
      const resp = await fetch("/api/reactions", {
        cache: "no-store",
        headers: { "X-User-ID": reactionUID },
      });
      if (!resp.ok) return;
      const next = await resp.json();
      applyReactionDelta(reactionState, next);
      reactionState = next;
      renderReactions();
    } catch (e) { /* network glitch — keep last render */ }
  }

  function spawnFloatEmoji(id, anchorEl) {
    const img = document.createElement("img");
    img.className = "rx-float";
    img.src = "https://chii.in/img/smiles/tv/" + id + ".gif";
    img.alt = "";
    const rect = anchorEl.getBoundingClientRect();
    const dx = (Math.random() - 0.5) * 28; // ±14px horizontal drift
    const rot = (Math.random() - 0.5) * 40; // ±20deg rotation
    img.style.left = (rect.left + rect.width / 2 - 15 + dx) + "px";
    img.style.top  = (rect.top  - 20) + "px";
    img.style.setProperty("--rx-rot", rot + "deg");
    document.body.appendChild(img);
    img.addEventListener("animationend", () => img.remove(), { once: true });
  }

  // Find the best anchor element for a given emoji id.
  function reactionAnchor(id) {
    return document.querySelector("#rx-actives [data-eid='" + id + "']")
        || document.getElementById("rx-trigger");
  }

  // Called after SSE delivers a new state; fires float animations for the delta.
  function applyReactionDelta(oldState, newState) {
    const oldMap = {};
    for (const r of oldState) oldMap[r.emoji_id] = r.count || 0;
    for (const r of newState) {
      const delta = (r.count || 0) - (oldMap[r.emoji_id] || 0);
      if (delta <= 0) continue;
      const anchor = reactionAnchor(r.emoji_id);
      if (!anchor) continue;
      const bursts = Math.min(delta, 4); // cap at 4 floats per update
      for (let i = 0; i < bursts; i++) {
        setTimeout(() => spawnFloatEmoji(r.emoji_id, anchor), i * 80);
      }
    }
  }

  async function addReaction(id, anchorEl) {
    const now = Date.now();
    if ((reactionCooldown.get(id) || 0) + 100 > now) return;
    reactionCooldown.set(id, now);

    spawnFloatEmoji(id, anchorEl);

    // Optimistic update.
    let cur = reactionState.find(r => r.emoji_id === id);
    if (!cur) { cur = { emoji_id: id, count: 0, mine: false }; reactionState.push(cur); }
    cur.count = (cur.count || 0) + 1;
    cur.mine = true;
    renderReactions();

    try {
      const resp = await fetch("/api/reactions", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-User-ID": reactionUID },
        body: JSON.stringify({ emoji_id: id }),
      });
      if (resp.status === 429) {
        // Rate-limited: roll back optimistic count and shake the button.
        cur.count = Math.max(0, (cur.count || 1) - 1);
        renderReactions();
        anchorEl.classList.add("rx-shake");
        anchorEl.addEventListener("animationend", () => anchorEl.classList.remove("rx-shake"), { once: true });
        return;
      }
    } catch (e) { /* network glitch — SSE/poll will reconcile */ }
  }

  // Live updates via SSE. Falls back silently to the 30s polling refresh.
  let reactionES = null;
  let reactionESBackoff = 1000;
  function connectReactionStream() {
    if (typeof EventSource === "undefined") return;
    try {
      const es = new EventSource("/api/reactions/stream?uid=" + encodeURIComponent(reactionUID));
      reactionES = es;
      es.onmessage = (ev) => {
        try {
          const next = JSON.parse(ev.data);
          applyReactionDelta(reactionState, next);
          reactionState = next;
          renderReactions();
          reactionESBackoff = 1000;
        } catch (e) { /* ignore malformed frame */ }
      };
      es.onerror = () => {
        es.close();
        reactionES = null;
        // Exponential backoff up to 30s.
        setTimeout(connectReactionStream, reactionESBackoff);
        reactionESBackoff = Math.min(reactionESBackoff * 2, 30000);
      };
    } catch (e) { /* SSE unavailable — poll only */ }
  }
  connectReactionStream();

  renderReactions();

  async function refresh() {
    try {
      const resp = await fetch("/api/status", { cache: "no-store" });
      if (!resp.ok) throw new Error("HTTP " + resp.status);
      const data = await resp.json();
      lastData = data;
      render(data);
    } catch (err) {
      const banner = document.getElementById("banner");
      banner.className = "banner banner--down";
      banner.querySelector(".banner__icon").innerHTML = ICONS.down;
      banner.querySelector(".banner__title").textContent = t("error_load");
      banner.querySelector(".banner__sub").textContent = err.message || String(err);
    }
  }

  applyI18n();
  refresh();
  setInterval(refresh, 30000);
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) refresh();
  });
})();
