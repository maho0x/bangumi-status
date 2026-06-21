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
      section_past: "Past incidents", hint_past: "Last 10 days",
      section_probes: "Probe nodes",
      section_online: "Activity",
      online_hint_24h: "last 24 hours",
      online_note: "Live “online” counter from bangumi.tv (signed-in view).",
      online_no_data: "No samples yet.",
      online_current: (n) => `${n} online now`,
      online_peak: (n) => `peak ${n}`,
      online_avg: (n) => `avg ${n}`,
      metric_bangumi: "Bangumi online",
      metric_traffic: "Site traffic",
      traffic_current: (n) => `${n} viewing now`,
      traffic_note: "Concurrent status-page viewers, sampled each minute — a proxy for whether people are flocking here because Bangumi is down.",
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
      nav_status: "Status", nav_wiki_stats: "Stats",
      wiki_recent_scrape: "Scraped", wiki_data_day: "Data day",
      wiki_empty: "No wiki stats yet.", wiki_chart_error: "Unable to load the official chart component.",
      wiki_metric_register: "Registered users", wiki_metric_collection: "Collections",
      wiki_metric_topics: "Topics", wiki_metric_replies: "Replies",
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
      section_past: "历史事故", hint_past: "最近10天",
      section_probes: "探针节点",
      section_online: "活动",
      online_hint_24h: "最近24小时",
      online_note: "Bangumi online 人数历史。",
      online_no_data: "暂无数据。",
      online_current: (n) => `当前 ${n} 人在线`,
      online_peak: (n) => `峰值 ${n}`,
      online_avg: (n) => `平均值 ${n}`,
      metric_bangumi: "班固米在线",
      metric_traffic: "本站访问",
      traffic_current: (n) => `当前 ${n} 人查看`,
      traffic_note: "Bangumi Status 访问人数历史。",
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
      nav_status: "状态", nav_wiki_stats: "透视",
      wiki_recent_scrape: "最近抓取", wiki_data_day: "数据日期",
      wiki_empty: "暂无维基透视数据。", wiki_chart_error: "无法加载官方图表组件。",
      wiki_metric_register: "注册用户", wiki_metric_collection: "收藏数",
      wiki_metric_topics: "主题数", wiki_metric_replies: "回复数",
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
  const WEEKDAYS_EN = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const WEEKDAYS_ZH = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];
  const escapeHTML = (s) => String(s).replace(/[&<>"']/g, c =>
    ({ "&":"&amp;", "<":"&lt;", ">":"&gt;", '"':"&quot;", "'":"&#39;" }[c]));

  const pageRoute = window.location.pathname.replace(/\/+$/, "") === "/stats" ? "wiki" : "status";
  document.body.dataset.page = pageRoute;

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

  // --- Diff/patch helpers --------------------------------------------------
  // Stable-DOM update primitives. Keeping nodes alive across re-renders lets
  // CSS transitions play across data changes (e.g. strip cell color shifts)
  // and preserves user state (open panels, scroll, hover) without snapshots.

  // Reconcile parent's keyed children with `items`. Children matched/reused
  // by getKey, created when missing, deleted when absent, and re-ordered to
  // match `items`. Static (non-reconciled) children — those without `_rk` —
  // are left alone so a fixed header can coexist with a reconciled body.
  function reconcile(parent, items, getKey, makeNode, updateNode) {
    const existing = new Map();
    for (const child of parent.children) {
      if (child._rk != null) existing.set(child._rk, child);
    }
    const skipStatic = (n) => { while (n && n._rk == null) n = n.nextSibling; return n; };
    let cursor = skipStatic(parent.firstChild);
    const seen = new Set();
    for (const item of items) {
      const k = String(getKey(item));
      seen.add(k);
      let node = existing.get(k);
      if (node) updateNode(node, item);
      else { node = makeNode(item); node._rk = k; }
      if (cursor === node) {
        cursor = skipStatic(node.nextSibling);
      } else {
        parent.insertBefore(node, cursor);
      }
    }
    for (const [k, n] of existing) if (!seen.has(k)) n.remove();
  }

  const setText  = (n, txt) => { const s = String(txt); if (n.textContent !== s) n.textContent = s; };
  const setClass = (n, cls) => { if (n.className !== cls) n.className = cls; };
  const setAttr  = (n, name, val) => {
    if (val == null || val === false) { if (n.hasAttribute(name)) n.removeAttribute(name); return; }
    const v = String(val);
    if (n.getAttribute(name) !== v) n.setAttribute(name, v);
  };
  const setHTML = (n, html) => { if (n._html !== html) { n.innerHTML = html; n._html = html; } };

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
  const showTip = (e, content) => {
    if (typeof content === "object" && content && content.html != null) {
      tip.innerHTML = content.html;
    } else {
      tip.textContent = String(content);
    }
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

    setClass(banner, "banner banner--" + status);
    setHTML(banner.querySelector(".banner__icon"), ICONS[status] || ICONS.loading);
    setText(banner.querySelector(".banner__title"), t("banner_" + status) || overall.message || "Status");

    const affected = nonGuest.filter(c => c.status !== "ok");
    const total = nonGuest.length;
    let subText;
    if (affected.length === 0) subText = t("banner_sub_all", total);
    else subText = t("banner_sub_affected", affected.length, total);
    setText(banner.querySelector(".banner__sub"), subText);

    const avg = nonGuest.reduce((a, c) => a + (c.uptime || 0), 0) / (nonGuest.length || 1);
    reconcile(meta, [{ key: "uptime", label: t("uptime_30d"), value: fmtUptime(avg) }],
      it => it.key,
      it => {
        const dt = el("dt", {}, it.label);
        const dd = el("dd", { class: "num" }, it.value);
        const div = el("div", {}, [dt, dd]);
        div._dt = dt; div._dd = dd;
        return div;
      },
      (div, it) => { setText(div._dt, it.label); setText(div._dd, it.value); }
    );

    setText(document.getElementById("updated"), fmtRelative(overall.updated_at));
  }

  // --- Unresolved incidents ------------------------------------------------
  function renderUnresolvedIncidents(overall) {
    const section = document.getElementById("unresolved-section");
    const list = document.getElementById("unresolved-list");
    const summary = document.getElementById("unresolved-summary");

    const affected = (overall.components || []).filter(c => c.status && c.status !== "ok" && c.kind !== "guest");
    if (affected.length === 0) {
      section.hidden = true;
      list.replaceChildren();
      return;
    }
    section.hidden = false;
    setText(summary, t("affected", affected.length));

    reconcile(list, affected,
      c => c.domain + "|" + c.kind,
      c => {
        const icon = el("div", { class: "inc-card__icon" });
        const sev = el("span", { class: "inc-title__sev" });
        const title = el("div", { class: "inc-card__title" }, [sev]);
        const meta = el("div", { class: "inc-card__meta" });
        const card = el("div", { class: "inc-card" }, [
          icon,
          el("div", { class: "inc-card__body" }, [title, meta]),
        ]);
        card._icon = icon; card._sev = sev; card._title = title; card._meta = meta;
        updateUnresolvedCard(card, c);
        return card;
      },
      (card, c) => updateUnresolvedCard(card, c)
    );
  }

  function updateUnresolvedCard(card, c) {
    setClass(card, "inc-card inc-card--" + c.status);
    setHTML(card._icon, ICONS[c.status] || ICONS.degraded);
    setText(card._sev, t("incident_" + c.status) || "Incident");
    // Keep the trailing component-name text node up to date in place.
    // Unresolved cards already filter out guest endpoints, so the kind suffix
    // ("· 认证端点") would be redundant — show domain only.
    const trailingText = " — " + c.domain;
    let trailing = card._sev.nextSibling;
    if (trailing && trailing.nodeType === 3) {
      if (trailing.nodeValue !== trailingText) trailing.nodeValue = trailingText;
    } else {
      card._title.appendChild(document.createTextNode(trailingText));
    }
    const views = c.probe_views || [];
    const badProbes = views.filter(v => v.status === "down" || v.status === "degraded");
    const regions = [...new Set(badProbes.map(v => v.region))].filter(Boolean);
    const bits = [el("span", { class: "mono" }, t("detected") + " " + fmtRelative(c.last_check))];
    if (regions.length > 0) {
      bits.push(el("span", { class: "sep" }, "·"));
      bits.push(el("span", {}, t("reported_from") + " " + regions.map(r => REGION_LABEL[r] || r).join(", ")));
    }
    if (badProbes.length > 0 && views.length > 0) {
      bits.push(el("span", { class: "sep" }, "·"));
      bits.push(el("span", {}, t("probes_failing", badProbes.length, views.length)));
    }
    card._meta.replaceChildren(...bits);
  }

  // --- Past incidents ------------------------------------------------------
  function renderPastIncidents(overall) {
    const container = document.getElementById("past-incidents");

    const comps = overall.components || [];
    if (comps.length === 0) {
      container.replaceChildren(el("div", { class: "inc-day__none" }, t("no_data")));
      return;
    }

    // Collect all incident windows across components within last 10 days,
    // indexed by UTC day of start_ts so a single event stays on one row.
    const byDay = new Map();
    // Seed last 10 days (from canonical day bucket list) so empty days still render.
    const canonical = (comps[0].days || []).slice().reverse().slice(0, 10);
    for (const b of canonical) byDay.set(b.day, []);

    const nowSec = Math.floor(Date.now() / 1000);
    const cutoff = nowSec - 10 * 86400;

    for (const c of comps) {
      for (const inc of (c.incidents || [])) {
        if (!inc || inc.start_ts < cutoff) continue;
        const iso = isoDayLocal(inc.start_ts);
        if (!byDay.has(iso)) byDay.set(iso, []);
        byDay.get(iso).push({ ...inc, component: componentLabel(c), kind: c.kind });
      }
    }

    const sortedDays = [...byDay.keys()].sort().reverse();
    const daysData = sortedDays.map(iso => {
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
      return { iso, entries, hasGuest, hasAuth };
    });

    reconcile(container, daysData, d => d.iso,
      d => createDayBlock(d),
      (block, d) => updateDayBlock(block, d)
    );
  }

  function createDayBlock(d) {
    const dateEl = el("div", { class: "inc-day__date" });
    const listEl = el("div", { class: "inc-day__list" });
    const noneEl = el("div", { class: "inc-day__none" });
    const collapsed = el("div", { class: "inc-day__collapsed", role: "button", tabindex: "0" });
    const dayBlock = el("div", { class: "inc-day", "data-day": d.iso }, [dateEl, listEl, collapsed]);

    const toggle = () => {
      dayBlock.classList.toggle("show-guest");
      const guestCount = Number(collapsed.dataset.guestCount || 0);
      setText(collapsed, dayBlock.classList.contains("show-guest")
        ? t("expanded_guest_hint", guestCount)
        : t("collapsed_guest_hint", guestCount));
    };
    collapsed.addEventListener("click", toggle);
    collapsed.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggle(); }
    });

    dayBlock._dateEl = dateEl;
    dayBlock._listEl = listEl;
    dayBlock._noneEl = noneEl;
    dayBlock._collapsed = collapsed;
    updateDayBlock(dayBlock, d);
    return dayBlock;
  }

  function updateDayBlock(dayBlock, d) {
    // Preserve the user's manual show-guest toggle across re-renders.
    const userExpanded = dayBlock.classList.contains("show-guest");
    const classes = ["inc-day"];
    if (d.hasGuest) classes.push("has-guest");
    if (d.hasAuth) classes.push("has-auth");
    if (userExpanded) classes.push("show-guest");
    setClass(dayBlock, classes.join(" "));
    setAttr(dayBlock, "data-day", d.iso);

    // Date header — rebuild children (only 1-2 nodes) for simplicity.
    const rel = fmtDayRelative(d.iso);
    const dateChildren = [document.createTextNode(fmtDayLabel(d.iso))];
    if (rel) dateChildren.push(el("span", { class: "inc-day__rel" }, rel));
    dayBlock._dateEl.replaceChildren(...dateChildren);

    if (d.entries.length === 0) {
      dayBlock._listEl.replaceChildren();
      if (!dayBlock.contains(dayBlock._noneEl)) {
        dayBlock.insertBefore(dayBlock._noneEl, dayBlock._listEl.nextSibling);
      }
      setText(dayBlock._noneEl, t("no_incidents"));
    } else {
      if (dayBlock._noneEl.parentNode) dayBlock._noneEl.remove();
      reconcile(dayBlock._listEl, d.entries,
        e => `${e.kind}|${e.start_ts}|${e.component}`,
        e => createIncEntry(e),
        (entryEl, e) => updateIncEntry(entryEl, e)
      );
    }

    if (d.hasGuest) {
      const guestCount = d.entries.filter(e => e.kind === "guest").length;
      dayBlock._collapsed.dataset.guestCount = String(guestCount);
      setText(dayBlock._collapsed, userExpanded
        ? t("expanded_guest_hint", guestCount)
        : t("collapsed_guest_hint", guestCount));
    } else {
      setText(dayBlock._collapsed, "");
    }
  }

  function createIncEntry(inc) {
    const dot = el("span", { class: "inc-day__dot" });
    const label = el("div", { class: "label" });
    const metric = el("span", { class: "metric" });
    const e = el("div", { class: "inc-day__entry" }, [dot, label, metric]);
    e._dot = dot; e._label = label; e._metric = metric;
    updateIncEntry(e, inc);
    return e;
  }

  function updateIncEntry(entryEl, inc) {
    setAttr(entryEl, "data-kind", inc.kind || "");
    setClass(entryEl._dot, "inc-day__dot status-" + inc.status);

    const nowSec = Math.floor(Date.now() / 1000);
    const endIso = isoDayLocal(inc.end_ts);
    const startIso = isoDayLocal(inc.start_ts);
    const crossDay = endIso !== startIso;
    const isOngoing = nowSec - inc.end_ts < 240;
    const endLabel = isOngoing ? t("ongoing") : (fmtTimeLocal(inc.end_ts) + (crossDay ? "⁺¹" : ""));
    const timeRange = fmtTimeLocal(inc.start_ts) + " – " + endLabel;
    const peakBits = [];
    if (inc.peak_down && inc.peak_total) peakBits.push(t("peak_failing", inc.peak_down, inc.peak_total));
    peakBits.push(fmtDuration(inc.end_ts - inc.start_ts));
    const descText = timeRange + " · " + peakBits.join(" · ");

    entryEl._label.replaceChildren(
      el("span", { class: "sev sev--" + inc.status }, t("incident_" + inc.status) || "Incident"),
      document.createTextNode(" — "),
      el("span", { class: "comp" }, inc.component),
      el("span", { class: "desc" }, descText),
    );
    setText(entryEl._metric, fmtRelative(inc.end_ts));
  }

  // --- Online users chart --------------------------------------------------
  let onlineRange = "24h";
  let onlineMetric = "bangumi"; // "bangumi" (bgm.tv online) | "traffic" (site viewers)
  let onlineIncidents = []; // incident windows from main-site auth components

  // uPlot-backed online chart. The library is vendored under /vendor/uplot and
  // lazy-loaded on first draw. Live state for the draw/cursor hooks lives in
  // onlineState so 20s refreshes can setData() the existing instance instead of
  // tearing it down (no flicker, hover preserved). The series shape is constant
  // ([x, avg]); band/area/incidents are drawn in hooks off onlineState.
  let onlineUplot = null;
  let onlineState = null;
  let onlineResizeObs = null;
  let uplotLibPromise = null;

  function loadUplotLib() {
    if (window.uPlot) return Promise.resolve();
    if (!uplotLibPromise) {
      if (!document.querySelector("link[data-uplot]")) {
        const link = document.createElement("link");
        link.rel = "stylesheet";
        link.href = "/vendor/uplot/uPlot.min.css";
        link.dataset.uplot = "1";
        document.head.appendChild(link);
      }
      uplotLibPromise = loadScriptOnce("/vendor/uplot/uPlot.iife.min.js");
    }
    return uplotLibPromise;
  }

  function cssVar(name, fallback) {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
  }
  // Append an 8-bit alpha to a #rrggbb color (a in 0..1). Falls through unchanged
  // if the value isn't a plain 6-digit hex.
  function withAlpha(hex, a) {
    hex = (hex || "").trim();
    if (!/^#[0-9a-fA-F]{6}$/.test(hex)) return hex;
    return hex + Math.round(Math.max(0, Math.min(1, a)) * 255).toString(16).padStart(2, "0");
  }

  function fmtOnlineX(range, ts) {
    const d = new Date(ts * 1000);
    return range === "24h"
      ? String(d.getHours()).padStart(2, "0") + ":" + String(d.getMinutes()).padStart(2, "0")
      : (d.getMonth() + 1) + "/" + d.getDate();
  }

  function destroyOnlineChart() {
    if (onlineResizeObs) onlineResizeObs.disconnect();
    if (onlineUplot) { onlineUplot.destroy(); onlineUplot = null; }
    onlineState = null;
  }

  function drawOnlineChart(rawPts) {
    const summaryEl = document.getElementById("online-summary");
    const pts = (rawPts || []).filter(p => p && p.count > 0);
    if (pts.length < 2) {
      destroyOnlineChart();
      const chartEl = document.getElementById("online-chart");
      if (chartEl) chartEl.innerHTML = `<div class="online-card__empty">${t("online_no_data")}</div>`;
      if (summaryEl) summaryEl.textContent = "—";
      return;
    }
    loadUplotLib()
      .then(() => renderOnlineUplot(pts))
      .catch(() => {
        destroyOnlineChart();
        const chartEl = document.getElementById("online-chart");
        if (chartEl) chartEl.innerHTML = `<div class="online-card__empty">${t("online_no_data")}</div>`;
      });
  }

  function renderOnlineUplot(pts) {
    const chartEl = document.getElementById("online-chart");
    const summaryEl = document.getElementById("online-summary");
    if (!chartEl || !summaryEl || !window.uPlot) return;

    // Bucketed series (7d/30d/all) carry a min/max band; raw 24h points don't.
    const banded = pts.some(p => (p.peak || 0) > 0);
    const xs = pts.map(p => p.ts);
    const avg = pts.map(p => p.count);
    const low = pts.map(p => banded ? (p.low || p.count) : p.count);
    const high = pts.map(p => banded ? (p.peak || p.count) : p.count);
    const peakIdx = high.reduce((bi, v, i) => (v > high[bi] ? i : bi), 0);
    const nowTs = Math.floor(Date.now() / 1000);
    const incidents = (onlineIncidents || []).filter(
      inc => (inc.end_ts || nowTs) > xs[0] && inc.start_ts < xs[xs.length - 1]);

    const vMax = Math.max(...high);
    const curKey = onlineMetric === "traffic" ? "traffic_current" : "online_current";
    summaryEl.textContent = t(curKey, avg[avg.length - 1].toLocaleString()) +
      " · " + t("online_peak", vMax.toLocaleString());

    // The series shape is always [x, avg]; the area fill, min/max band and
    // incident overlays are drawn in hooks off onlineState, so every range
    // reuses one instance (setData) — no flicker, no rebuild.
    onlineState = { xs, avg, low, high, banded, peakIdx, incidents, range: onlineRange };

    if (onlineUplot) {
      onlineUplot.setData([xs, avg]);
      return;
    }

    chartEl.innerHTML = "";
    const host = document.createElement("div");
    host.className = "online-uplot-host";
    chartEl.appendChild(host);

    const ok = cssVar("--ok", "#85c99b");
    const grid = cssVar("--border", "#272a26");
    const axis = cssVar("--text-faint", "#9a9c94");
    const degraded = cssVar("--degraded", "#D9C775");
    const down = cssVar("--down", "#D97A92");
    const accent = cssVar("--accent", "#f09199");
    const surface = cssVar("--surface", "#ffffff");

    // Incident overlays + the min/max range band, drawn behind the line. The
    // band is hand-drawn (not uPlot bands) so the fill and y-scaling are fully
    // under our control.
    const drawBands = (u) => {
      const s = onlineState;
      if (!s) return;
      const ctx = u.ctx, top = u.bbox.top, h = u.bbox.height;
      const x0 = s.xs[0], xN = s.xs[s.xs.length - 1];
      ctx.save();
      for (const inc of s.incidents) {
        const x1 = u.valToPos(Math.max(inc.start_ts, x0), "x", true);
        const x2 = u.valToPos(Math.min(inc.end_ts || Math.floor(Date.now() / 1000), xN), "x", true);
        ctx.fillStyle = inc.status === "down" ? withAlpha(down, 0.22) : withAlpha(degraded, 0.18);
        ctx.fillRect(x1, top, Math.max(1, x2 - x1), h);
      }
      if (s.banded) {
        const dpr = u.ctx.canvas.width / u.width || (window.devicePixelRatio || 1);
        // Filled area between the high and low envelopes.
        ctx.beginPath();
        for (let i = 0; i < s.xs.length; i++) {
          const x = u.valToPos(s.xs[i], "x", true), y = u.valToPos(s.high[i], "y", true);
          if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
        }
        for (let i = s.xs.length - 1; i >= 0; i--) {
          ctx.lineTo(u.valToPos(s.xs[i], "x", true), u.valToPos(s.low[i], "y", true));
        }
        ctx.closePath();
        ctx.fillStyle = withAlpha(ok, 0.13);
        ctx.fill();
        // Faint stroke along the peak (high) edge so the peak envelope reads
        // clearly instead of fading into the fill.
        ctx.beginPath();
        for (let i = 0; i < s.xs.length; i++) {
          const x = u.valToPos(s.xs[i], "x", true), y = u.valToPos(s.high[i], "y", true);
          if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
        }
        ctx.lineWidth = dpr;
        ctx.strokeStyle = withAlpha(ok, 0.5);
        ctx.stroke();
      }
      ctx.restore();
    };

    const drawPeak = (u) => {
      const s = onlineState;
      if (!s) return;
      // uPlot works in device pixels inside hooks (it scales coordinates, not
      // the context), so size everything by the pixel ratio.
      const dpr = u.ctx.canvas.width / u.width || (window.devicePixelRatio || 1);
      const ctx = u.ctx;
      const cx = u.valToPos(s.xs[s.peakIdx], "x", true);
      const cy = u.valToPos(s.high[s.peakIdx], "y", true);
      ctx.save();

      // Accent dot with a surface-colored ring so it stands out from the green
      // line/band.
      ctx.beginPath();
      ctx.arc(cx, cy, 4 * dpr, 0, Math.PI * 2);
      ctx.fillStyle = accent;
      ctx.fill();
      ctx.lineWidth = 1.5 * dpr;
      ctx.strokeStyle = surface;
      ctx.stroke();

      // Soft label (no pill) — the brighter dot already locates the peak.
      const right = cx > u.bbox.left + u.bbox.width * 0.6;
      ctx.fillStyle = axis;
      ctx.font = (11 * dpr) + 'px ui-monospace, "DejaVu Sans Mono", monospace';
      ctx.textAlign = right ? "right" : "left";
      ctx.textBaseline = "bottom";
      ctx.fillText(t("online_peak", s.high[s.peakIdx].toLocaleString()),
        cx + (right ? -6 : 6) * dpr, Math.max(u.bbox.top + 12 * dpr, cy - 5 * dpr));
      ctx.restore();
    };

    const cursorTooltip = (u) => {
      const s = onlineState;
      const idx = u.cursor.idx;
      if (s == null || idx == null) { tip.hidden = true; return; }
      const d = new Date(s.xs[idx] * 1000);
      const label = s.range === "24h"
        ? String(d.getHours()).padStart(2, "0") + ":" + String(d.getMinutes()).padStart(2, "0")
        : `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}` +
          (s.range === "7d" ? " " + String(d.getHours()).padStart(2, "0") + ":00" : "");
      const valLabel = s.banded
        ? `${t("online_avg", s.avg[idx].toLocaleString())} · ${t("online_peak", s.high[idx].toLocaleString())}`
        : s.avg[idx].toLocaleString();
      tip.textContent = `${label} · ${valLabel}`;
      tip.hidden = false;
      const rect = u.over.getBoundingClientRect();
      const tw = tip.offsetWidth;
      let tx = rect.left + u.cursor.left - tw / 2;
      tx = Math.max(6, Math.min(tx, window.innerWidth - tw - 6));
      tip.style.left = tx + "px";
      tip.style.top = (rect.top + u.valToPos(s.high[idx], "y") - tip.offsetHeight - 10) + "px";
    };

    const series = [
      {},
      { stroke: ok, width: 1.6, points: { show: false },
        // Area fill only on the raw 24h view; bucketed views use the band instead.
        fill: () => (onlineState && !onlineState.banded ? withAlpha(ok, 0.20) : null) },
    ];

    const opts = {
      width: Math.max(200, host.clientWidth),
      height: host.clientHeight || 140,
      padding: [12, 10, 2, 6],
      legend: { show: false },
      cursor: { y: false, points: { show: false }, drag: { x: false, y: false } },
      scales: {
        x: { time: false },
        // Cover the band (high/low) when bucketed; otherwise pad around avg.
        y: { range: (u, dmin, dmax) => {
          const s = onlineState;
          const lo = s && s.banded ? Math.min(...s.low) : dmin;
          const hi = s && s.banded ? Math.max(...s.high) : dmax;
          const pad = Math.max(1, (hi - lo) * 0.15);
          return [Math.max(0, lo - pad), hi + pad];
        } },
      },
      axes: [
        { stroke: axis, grid: { show: false }, ticks: { show: false }, gap: 4, size: 22,
          font: '11px ui-monospace, monospace',
          values: (u, splits) => splits.map(ts => fmtOnlineX(onlineState.range, ts)) },
        { stroke: axis, ticks: { show: false }, gap: 4, size: 34,
          font: '11px ui-monospace, monospace',
          grid: { stroke: withAlpha(grid, 0.7), width: 1, dash: [2, 3] },
          values: (u, splits) => splits.map(v => Math.round(v).toLocaleString()) },
      ],
      series,
      hooks: {
        drawClear: [drawBands],
        draw: [drawPeak],
        setCursor: [cursorTooltip],
      },
    };

    onlineUplot = new uPlot(opts, [xs, avg], host);
    onlineUplot.over.addEventListener("mouseleave", () => { tip.hidden = true; });

    if (!onlineResizeObs) {
      onlineResizeObs = new ResizeObserver(() => {
        const el = document.querySelector(".online-uplot-host");
        if (onlineUplot && el) {
          onlineUplot.setSize({ width: Math.max(200, el.clientWidth), height: el.clientHeight || 140 });
        }
      });
    }
    onlineResizeObs.observe(host);
  }

  // Wire the metric switch + range tabs once. Each just updates state and
  // reloads the series; drawOnlineChart manages the chart DOM (reusing the uPlot
  // instance), so we leave the current chart in place during the fetch.
  function wireActivityControls() {
    const tabsEl = document.getElementById("online-tabs");
    if (tabsEl && !tabsEl._wired) {
      tabsEl._wired = true;
      tabsEl.addEventListener("click", (e) => {
        const btn = e.target.closest(".online-tab");
        if (!btn || btn.dataset.range === onlineRange) return;
        onlineRange = btn.dataset.range;
        tabsEl.querySelectorAll(".online-tab").forEach(b => {
          b.classList.toggle("active", b === btn);
          b.setAttribute("aria-selected", b === btn ? "true" : "false");
        });
        loadActivitySeries();
      });
    }
    const metricsEl = document.getElementById("online-metrics");
    if (metricsEl && !metricsEl._wired) {
      metricsEl._wired = true;
      metricsEl.addEventListener("click", (e) => {
        const btn = e.target.closest(".online-metric");
        if (!btn || btn.dataset.metric === onlineMetric) return;
        onlineMetric = btn.dataset.metric;
        metricsEl.querySelectorAll(".online-metric").forEach(b => {
          b.classList.toggle("active", b === btn);
          b.setAttribute("aria-selected", b === btn ? "true" : "false");
        });
        loadActivitySeries();
      });
    }
  }

  // Fetch (or derive) the series for the current metric + range and draw it.
  // Bangumi's 24h series is embedded in /api/status; everything else is fetched.
  function loadActivitySeries(overall) {
    const noteEl = document.getElementById("online-note");
    if (noteEl) noteEl.textContent = t(onlineMetric === "traffic" ? "traffic_note" : "online_note");
    if (onlineMetric === "bangumi" && onlineRange === "24h") {
      drawOnlineChart((overall && overall.online) || (lastData && lastData.online) || []);
      return;
    }
    const path = onlineMetric === "traffic" ? "/api/traffic" : "/api/online";
    fetch(`${path}?range=${onlineRange}`)
      .then(r => r.json())
      .then(drawOnlineChart)
      .catch(() => {});
  }

  function renderOnlineChart(overall) {
    // Incident overlays from the three main sites (auth kind) — useful on both
    // metrics (shows whether a traffic spike lines up with an outage).
    const mainDomains = new Set(["bgm.tv", "bangumi.tv", "chii.in"]);
    onlineIncidents = [];
    for (const c of (overall.components || [])) {
      if (!mainDomains.has(c.domain) || c.kind !== "auth") continue;
      for (const inc of (c.incidents || [])) {
        if (inc && inc.status && inc.status !== "ok") onlineIncidents.push(inc);
      }
    }
    wireActivityControls();
    loadActivitySeries(overall);
  }

  // --- Component row -------------------------------------------------------
  // Animated open/close for the probe-detail panel inside a component row.
  // Measures scrollHeight at the moment of toggle so the height transition
  // matches the actual content size (varies with probe count).
  function toggleProbeDetail(row) {
    const detail = row._detail;
    const isOpen = row.classList.contains("open");
    clearTimeout(detail._animTimer);

    if (isOpen) {
      // Closing: freeze current open size, drop the uncapped state, then ramp to 0.
      detail.style.maxHeight = detail.scrollHeight + "px";
      detail.classList.remove("is-open");
      void detail.offsetHeight;
      row.classList.remove("open");
      detail.style.maxHeight = "0px";
      detail._animTimer = setTimeout(() => { detail.style.maxHeight = ""; }, 280);
    } else {
      // Opening: ramp height from 0 → scrollHeight, then mark uncapped so future
      // SSE-driven probe additions can grow the panel beyond the recorded value.
      row.classList.add("open");
      detail.style.maxHeight = detail.scrollHeight + "px";
      detail._animTimer = setTimeout(() => {
        if (row.classList.contains("open")) {
          detail.classList.add("is-open");
          detail.style.maxHeight = "";
        }
      }, 280);
    }
  }

  function createComponent(c) {
    const dot = el("span", { class: "status-dot" });
    const labelText = el("span", { class: "label-text" });
    const statusText = el("span", { class: "status-text" });
    const strip = el("div", { class: "strip" });
    const mid = el("span", { class: "mid" });
    const toggleCount = el("span", { class: "probe-toggle__count" });
    const detail = el("div", { class: "probe-detail" });

    const row = el("div", {
      class: "component",
      "data-key": `${c.domain}-${c.kind}`,
      "data-kind": c.kind || "",
    }, [
      el("div", { class: "component-label" }, [dot, labelText, statusText]),
      el("div", { class: "component-right" }, [
        strip,
        mid,
        el("button", {
          class: "probe-toggle",
          type: "button",
          "aria-label": "toggle probe detail",
          onclick: () => toggleProbeDetail(row),
        }, [toggleCount, el("span", { class: "probe-toggle__chevron" })]),
      ]),
      detail,
    ]);

    row._dot = dot; row._label = labelText; row._statusText = statusText;
    row._strip = strip; row._mid = mid; row._toggleCount = toggleCount; row._detail = detail;
    return row;
  }

  function updateComponent(row, c) {
    const displayStatus = c.status || "none";
    setAttr(row, "data-status", displayStatus);
    setClass(row._dot, "status-dot status-" + displayStatus);
    setText(row._label, kindLabel(c));
    setText(row._statusText, "· " + (t("status_" + displayStatus) || "unknown"));
    setText(row._mid, fmtUptime(c.uptime) + " " + t("uptime_label"));
    const views = c.probe_views || [];
    setText(row._toggleCount, String(views.length));
    reconcileStrip(row._strip, c.days || []);
    reconcileProbeDetail(row._detail, views);
  }

  function reconcileStrip(stripEl, days) {
    // Build cell list including any "padding" days (today not yet bucketed).
    const cells = days.slice();
    const todayIso = isoDayLocal(Math.floor(Date.now() / 1000));
    let lastDay = days[days.length - 1]?.day;
    while (lastDay && lastDay < todayIso) {
      const [y, m, d] = lastDay.split("-").map(Number);
      const next = new Date(y, m - 1, d + 1);
      lastDay = `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, "0")}-${String(next.getDate()).padStart(2, "0")}`;
      cells.push({ day: lastDay, total: 0, status: "none" });
    }
    reconcile(stripEl, cells, d => d.day,
      d => {
        const cell = el("div", {});
        cell._dayData = d;
        // Read fresh data on each hover so SSE updates are reflected in tooltip.
        cell.addEventListener("mouseenter", e => showTip(e, tooltipForDay(cell._dayData)));
        cell.addEventListener("mouseleave", hideTip);
        setClass(cell, "strip-cell " + (d.total === 0 ? "cell-none" : "cell-" + (d.status || "none")));
        return cell;
      },
      (cell, d) => {
        setClass(cell, "strip-cell " + (d.total === 0 ? "cell-none" : "cell-" + (d.status || "none")));
        cell._dayData = d;
      }
    );
  }

  function tooltipForDay(d) {
    const dateLabel = fmtDayLabel(d.day);
    const rel = fmtDayRelative(d.day);
    const wd = weekdayShort(d.day);
    const headRel = rel || wd;
    const headerHTML =
      `<div class="tooltip__day-head">` +
        `<span class="tooltip__day-date">${escapeHTML(dateLabel)}</span>` +
        (headRel ? `<span class="tooltip__day-rel">${escapeHTML(headRel)}</span>` : "") +
      `</div>`;

    if (!d.total) {
      return { html:
        headerHTML +
        `<div class="tooltip__day-divider"></div>` +
        `<div class="tooltip__day-empty">` +
          `<span class="tooltip__day-status-dot status-none"></span>` +
          `<span>${escapeHTML(t("no_probe_data"))}</span>` +
        `</div>`
      };
    }

    const status = d.status || "ok";
    const checksLabel = lang === "zh" ? "次检查" : (d.total === 1 ? "check" : "checks");
    const parts = [`<span>${d.total.toLocaleString()} ${checksLabel}</span>`];
    if (d.down) parts.push(`<span class="down">${escapeHTML(t("failed_count", d.down))}</span>`);
    if (d.degrade) parts.push(`<span class="degrade">${escapeHTML(t("degraded_count", d.degrade))}</span>`);
    const countsHTML = parts.join(`<span class="sep">·</span>`);

    return { html:
      headerHTML +
      `<div class="tooltip__day-divider"></div>` +
      `<div class="tooltip__day-meta">` +
        `<span class="tooltip__day-status">` +
          `<span class="tooltip__day-status-dot status-${status}"></span>` +
          `<span>${escapeHTML(t("status_" + status))}</span>` +
        `</span>` +
        `<span class="tooltip__day-uptime">${escapeHTML(fmtUptime(d.uptime))}</span>` +
      `</div>` +
      `<div class="tooltip__day-counts">${countsHTML}</div>`
    };
  }

  function weekdayShort(isoDay) {
    if (!isoDay) return "";
    const [y, m, d] = isoDay.split("-").map(Number);
    if (!y || !m || !d) return "";
    const dt = new Date(Date.UTC(y, m - 1, d));
    const idx = dt.getUTCDay();
    return (lang === "zh" ? WEEKDAYS_ZH : WEEKDAYS_EN)[idx];
  }

  function reconcileProbeDetail(detail, views) {
    if (views.length === 0) {
      if (detail._mode !== "empty") {
        detail.replaceChildren(el("div", {
          class: "probe-row",
          style: "color:var(--text-faint);grid-template-columns:1fr",
        }, t("no_probe_data_detail")));
        detail._mode = "empty";
      }
      return;
    }
    detail._mode = "list";
    reconcile(detail, views,
      v => (v.region || "") + "|" + v.probe,
      v => {
        const name = el("span", { class: "probe-name" });
        const err = el("span", { class: "probe-err" });
        const lat = el("span", { class: "probe-lat" });
        const dot = el("span", { class: "status-dot" });
        const sLabel = el("span", { class: "status-label" });
        const r = el("div", { class: "probe-row" }, [
          name, err, lat,
          el("span", { class: "probe-status" }, [dot, sLabel]),
        ]);
        r._name = name; r._err = err; r._lat = lat; r._dot = dot; r._sLabel = sLabel;
        updateProbeRow(r, v);
        return r;
      },
      (r, v) => updateProbeRow(r, v)
    );
  }

  function updateProbeRow(r, v) {
    setText(r._name, (v.region ? regionFlag(v.region) + " " : "") + v.probe);
    setText(r._err, v.err || (v.http_code ? `HTTP ${v.http_code}` : "—"));
    setText(r._lat, v.latency_ms != null ? v.latency_ms + " ms" : "—");
    setClass(r._dot, "status-dot status-" + (v.status || "none"));
    setText(r._sLabel, t("status_" + v.status) || "—");
  }

  // --- Component groups ----------------------------------------------------
  function renderComponents(overall) {
    const container = document.getElementById("components");
    const byDomain = {};
    for (const c of overall.components || []) {
      (byDomain[c.domain] = byDomain[c.domain] || []).push(c);
    }
    const orderedDomains = ["bgm.tv", "bangumi.tv", "chii.in", "next.bgm.tv", "next.bgm.tv/p1", "api.bgm.tv"]
      .filter(d => (byDomain[d] || []).length > 0);
    const groupsData = orderedDomains.map(domain => ({ domain, comps: byDomain[domain] }));

    reconcile(container, groupsData, g => g.domain,
      g => createGroup(g),
      (groupEl, g) => updateGroup(groupEl, g)
    );
  }

  function createGroup(g) {
    const hasGuest = g.comps.some(c => c.kind === "guest");
    const hasAuth = g.comps.some(c => c.kind === "auth");
    const groupEl = el("div", {
      class: "group" + (!hasAuth ? " show-guest" : ""),
      "data-domain": g.domain,
    });
    const domainDot = el("span", { class: "status-dot" });
    const hdrRight = el("div", { class: "group-hdr-right" });
    const hdr = el("div", { class: "group-hdr" }, [
      el("span", { class: "domain-name" }, [domainDot, document.createTextNode(g.domain)]),
      hdrRight,
    ]);
    groupEl.appendChild(hdr);
    groupEl._domainDot = domainDot;
    groupEl._hdrRight = hdrRight;
    groupEl._hasToggle = false;
    if (hasGuest && hasAuth) {
      const labelEl = el("span", { class: "guest-toggle__label" }, t("guest_show"));
      const btn = el("button", {
        class: "guest-toggle",
        type: "button",
        "aria-pressed": "false",
        onclick: (e) => {
          const shown = groupEl.classList.toggle("show-guest");
          e.currentTarget.setAttribute("aria-pressed", shown ? "true" : "false");
          labelEl.textContent = shown ? t("guest_hide") : t("guest_show");
        },
      }, [el("span", { class: "guest-toggle__chevron", "aria-hidden": "true" }), labelEl]);
      hdrRight.appendChild(btn);
      groupEl._hasToggle = true;
      groupEl._toggleBtn = btn;
      groupEl._toggleLabel = labelEl;
    }
    updateGroup(groupEl, g);
    return groupEl;
  }

  function updateGroup(groupEl, g) {
    setClass(groupEl._domainDot, "status-dot status-" + groupStatus(g.comps));

    // Maintain the guest-badge (count of failing public endpoints).
    const guestBad = g.comps.filter(c => c.kind === "guest" && c.status && c.status !== "ok");
    let badge = groupEl._hdrRight.querySelector(".guest-badge");
    if (guestBad.length > 0) {
      const worst = guestBad.some(c => c.status === "down") ? "down" : "degraded";
      if (!badge) {
        badge = el("span", { class: "guest-badge guest-badge--" + worst });
        groupEl._hdrRight.insertBefore(badge, groupEl._hdrRight.firstChild);
      }
      setClass(badge, "guest-badge guest-badge--" + worst);
      setText(badge, String(guestBad.length));
      badge.title = t("collapsed_guest_hint", guestBad.length);
    } else if (badge) {
      badge.remove();
    }

    // Refresh the guest-toggle label so a language switch picks it up.
    if (groupEl._hasToggle && groupEl._toggleLabel) {
      const shown = groupEl.classList.contains("show-guest");
      setText(groupEl._toggleLabel, shown ? t("guest_hide") : t("guest_show"));
    }

    reconcileComponentRows(groupEl, g.comps);
  }

  function reconcileComponentRows(groupEl, comps) {
    const existing = new Map();
    for (const child of groupEl.children) {
      if (child.classList.contains("component")) existing.set(child.getAttribute("data-key"), child);
    }
    const seen = new Set();
    let cursor = groupEl.querySelector(".group-hdr").nextSibling;
    while (cursor && !cursor.classList?.contains("component")) cursor = cursor.nextSibling;
    for (const c of comps) {
      const key = `${c.domain}-${c.kind}`;
      seen.add(key);
      let row = existing.get(key);
      if (!row) row = createComponent(c);
      updateComponent(row, c);
      if (cursor === row) {
        cursor = row.nextSibling;
        while (cursor && !cursor.classList?.contains("component")) cursor = cursor.nextSibling;
      } else {
        groupEl.insertBefore(row, cursor);
      }
    }
    for (const [k, row] of existing) {
      if (!seen.has(k)) row.remove();
    }
  }

  function groupStatus(comps) {
    const rank = { ok: 0, degraded: 1, down: 2 };
    let worst = "ok";
    for (const c of comps) {
      const effective = c.status || "ok";
      if ((rank[effective] || 0) > (rank[worst] || 0)) worst = effective;
    }
    return worst;
  }

  // --- Probes --------------------------------------------------------------
  function renderProbes(overall) {
    const list = document.getElementById("probe-list");
    const summary = document.getElementById("probes-summary");
    const probes = overall.probes || [];
    if (probes.length === 0) {
      setText(summary, "—");
      list.replaceChildren(el("div", {
        style: "color:var(--text-faint);font-size:13px;padding:12px 2px",
      }, t("no_probes")));
      return;
    }
    const online = probes.filter(p => p.online).length;
    setText(summary, t("probe_online_summary", online, probes.length));
    reconcile(list, probes,
      p => p.name,
      p => {
        const name = el("div", { class: "name" });
        const region = el("div", { class: "region" });
        const pill = el("span", { class: "pill" });
        const card = el("div", { class: "probe-card" }, [el("div", {}, [name, region]), pill]);
        card._name = name; card._region = region; card._pill = pill;
        updateProbeCard(card, p);
        return card;
      },
      (card, p) => updateProbeCard(card, p)
    );
  }

  function updateProbeCard(card, p) {
    setClass(card, "probe-card " + (p.online ? "online" : "offline"));
    setText(card._name, p.name);
    setText(card._region, (p.region ? regionFlag(p.region) + " " : "") + (REGION_LABEL[p.region] || p.region) + " · " + fmtRelative(p.last_seen));
    setText(card._pill, p.online ? t("probe_pill_online") : t("probe_pill_offline"));
  }

  // --- Wiki stats ----------------------------------------------------------
  // Distinct line colors for the multi-series wiki charts (longest set is the
  // 8-series replies breakdown).
  const WIKI_PALETTE = [
    "#f09199", "#85c99b", "#6CA0DC", "#D9C775", "#C58FD9",
    "#E0936B", "#5FC9C2", "#D97A92", "#9DC468", "#B08CC4",
  ];
  let wikiCharts = []; // [{ u, node }]
  let wikiResizeObs = null;
  let lastWikiStats = null;

  function applyRouteChrome() {
    const wikiPage = document.getElementById("wiki-stats-page");
    if (wikiPage) wikiPage.hidden = pageRoute !== "wiki";
    document.querySelectorAll(".top-nav a[data-route-link]").forEach(a => {
      const active = a.dataset.routeLink === pageRoute;
      a.setAttribute("aria-current", active ? "page" : "false");
    });
  }

  function wikiChartSets() {
    return {
      register: {
        chart_root: "chartCommunityRegister",
        series_set: { "注册用户": "register_total" },
        chart_type: "line",
        show_labels: false,
      },
      collection: {
        chart_root: "chartCommunityCollection",
        series_set: {
          "想看/读/听/玩": "collection_1",
          "看/读/听/玩过": "collection_2",
          "在看/读/听/玩": "collection_3",
          "搁置": "collection_4",
          "抛弃": "collection_5",
        },
        chart_type: "line",
        show_labels: false,
      },
      topics: {
        chart_root: "chartCommunityTopics",
        series_set: {
          "小组主题": "topic_1",
          "条目主题": "topic_2",
          "日志发布": "topic_7",
        },
        chart_type: "line",
        show_labels: false,
      },
      replies: {
        chart_root: "chartCommunityReplies",
        series_set: {
          "小组回复": "reply_1",
          "条目回复": "reply_2",
          "角色吐槽": "reply_3",
          "人物吐槽": "reply_4",
          "目录留言": "reply_5",
          "时间线回复": "reply_6",
          "日志回复": "reply_7",
          "章节讨论": "reply_8",
        },
        chart_type: "line",
        show_labels: false,
      },
    };
  }

  function loadScriptOnce(src) {
    const existing = document.querySelector(`script[data-wiki-chart-src="${src}"]`);
    if (existing) {
      return existing.dataset.loaded === "true"
        ? Promise.resolve()
        : new Promise((resolve, reject) => {
            existing.addEventListener("load", resolve, { once: true });
            existing.addEventListener("error", reject, { once: true });
          });
    }
    return new Promise((resolve, reject) => {
      const script = document.createElement("script");
      script.src = src;
      script.async = false;
      script.dataset.wikiChartSrc = src;
      script.onload = () => { script.dataset.loaded = "true"; resolve(); };
      script.onerror = () => reject(new Error("load " + src));
      document.head.appendChild(script);
    });
  }

  // Both pages share the vendored uPlot library.
  function loadWikiChartLibs() {
    return loadUplotLib();
  }

  function disposeWikiCharts() {
    for (const { u } of wikiCharts) {
      try { u.destroy(); } catch (_) {}
    }
    wikiCharts = [];
    if (wikiResizeObs) wikiResizeObs.disconnect();
  }

  // Multi-series line chart (uPlot). x is a category index → date label; the
  // built-in legend toggles series on click and shows hovered values, and the
  // cursor supports drag-to-zoom on x (double-click resets).
  function initWikiChart(rootElement, data, seriesSets) {
    const node = typeof rootElement === "string" ? document.getElementById(rootElement) : rootElement;
    if (!node || !window.uPlot) return;

    const entries = Object.entries(seriesSets); // [displayName, fieldName]
    const xs = data.map((_, i) => i);
    const labels = data.map(p => p.title);
    const cols = [xs, ...entries.map(([, field]) => data.map(p => Number(p[field]) || 0))];

    const axisColor = cssVar("--text-faint", "#9a9c94");
    const gridColor = withAlpha(cssVar("--border", "#272a26"), 0.7);
    const pointFill = cssVar("--card", "#ffffff");

    const series = [
      {},
      ...entries.map(([name], i) => {
        const color = WIKI_PALETTE[i % WIKI_PALETTE.length];
        return {
          label: name,
          stroke: color,
          width: 2,
          points: { show: true, size: 5, stroke: color, fill: pointFill, width: 1.5 },
          value: (u, v) => (v == null ? "—" : v.toLocaleString()),
        };
      }),
    ];

    const axisOpts = (extra) => Object.assign({
      stroke: axisColor,
      ticks: { show: false },
      gap: 6,
      font: '11px ui-monospace, monospace',
      grid: { stroke: gridColor, width: 1, dash: [2, 3] },
    }, extra);

    const opts = {
      width: Math.max(200, node.clientWidth),
      height: 460,
      padding: [14, 16, 0, 6],
      scales: { x: { time: false }, y: { range: (u, min, max) => [0, (max || 1) * 1.03] } },
      legend: { show: true, live: true },
      cursor: { drag: { x: true, y: false }, focus: { prox: 30 }, points: { size: 7 } },
      axes: [
        axisOpts({ space: 64, values: (u, splits) => splits.map(i => labels[Math.round(i)] || "") }),
        axisOpts({ size: 54, values: (u, splits) => splits.map(v => Math.round(v).toLocaleString()) }),
      ],
      series,
    };

    const u = new uPlot(opts, cols, node);
    wikiCharts.push({ u, node });

    if (!wikiResizeObs) {
      wikiResizeObs = new ResizeObserver(() => {
        for (const c of wikiCharts) c.u.setSize({ width: Math.max(200, c.node.clientWidth), height: 460 });
      });
    }
    wikiResizeObs.observe(node);
  }

  function wikiChartData(points) {
    return points.map(p => ({
      ...p,
      title: p.title || (p.date ? p.date.slice(5) : ""),
      none: 0,
    }));
  }

  function setWikiChartsMessage(message) {
    for (const set of Object.values(wikiChartSets())) {
      const node = document.getElementById(set.chart_root);
      if (node) node.innerHTML = `<div class="wiki-chart__empty">${escapeHTML(message)}</div>`;
    }
  }

  function drawWikiCharts(points) {
    disposeWikiCharts();
    const data = wikiChartData(points);
    for (const set of Object.values(wikiChartSets())) {
      const node = document.getElementById(set.chart_root);
      if (!node) continue;
      node.replaceChildren();
      initWikiChart(set.chart_root, data, set.series_set, set);
    }
  }

  function renderWikiSummary(points) {
    const summary = document.getElementById("wiki-stats-summary");
    if (!summary) return;
    const latest = points[points.length - 1];
    if (!latest) {
      summary.replaceChildren();
      return;
    }
    const items = [
      { key: "register", value: latest.register_total, label: t("wiki_metric_register") },
      { key: "collection", value: latest.collection_total, label: t("wiki_metric_collection") },
      { key: "topics", value: latest.topic_total, label: t("wiki_metric_topics") },
      { key: "replies", value: latest.reply_total, label: t("wiki_metric_replies") },
    ];
    reconcile(summary, items, item => item.key,
      item => {
        const value = el("span", { class: "num" }, Number(item.value || 0).toLocaleString());
        const label = el("span", { class: "desc" }, item.label);
        const card = el("div", { class: "wiki-summary__item" }, [value, label]);
        card._value = value; card._label = label;
        return card;
      },
      (card, item) => {
        setText(card._value, Number(item.value || 0).toLocaleString());
        setText(card._label, item.label);
      }
    );
  }

  async function renderWikiStats(payload) {
    lastWikiStats = payload;
    const errorEl = document.getElementById("wiki-stats-error");
    const scrapedEl = document.getElementById("wiki-scraped-at");
    const sourceDayEl = document.getElementById("wiki-source-day");
    const points = (payload?.data || []).slice().sort((a, b) => String(a.date).localeCompare(String(b.date)));
    const latest = points[points.length - 1];

    setText(scrapedEl, payload?.scraped_at ? fmtRelative(payload.scraped_at) : "—");
    setText(sourceDayEl, latest?.date ? fmtDayLabel(latest.date) : "—");
    renderWikiSummary(points);

    if (points.length === 0) {
      if (errorEl) {
        errorEl.hidden = false;
        setText(errorEl, t("wiki_empty"));
      }
      setWikiChartsMessage(t("wiki_empty"));
      return;
    }
    if (errorEl) errorEl.hidden = true;

    try {
      await loadWikiChartLibs();
      drawWikiCharts(points);
    } catch (_) {
      uplotLibPromise = null; // allow a later visit to retry the lib load
      if (errorEl) {
        errorEl.hidden = false;
        setText(errorEl, t("wiki_chart_error"));
      }
      setWikiChartsMessage(t("wiki_chart_error"));
    }
  }

  async function initWikiStatsPage() {
    applyRouteChrome();
    try {
      const resp = await fetch("/api/wiki-stats", { cache: "no-store" });
      if (!resp.ok) throw new Error("HTTP " + resp.status);
      await renderWikiStats(await resp.json());
    } catch (err) {
      const errorEl = document.getElementById("wiki-stats-error");
      if (errorEl) {
        errorEl.hidden = false;
        setText(errorEl, err.message || String(err));
      }
      setWikiChartsMessage(t("wiki_empty"));
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
    setT('.online-metric[data-metric="bangumi"]', "metric_bangumi");
    setT('.online-metric[data-metric="traffic"]', "metric_traffic");

    setT(".ftr__desc", "footer_desc");
    setT("#subscribe-title", "modal_title");
    setT(".modal__intro", "modal_intro");
    setT(".modal__foot", "modal_foot");
    setT(".ftr__link", "atom_feed");
    setT(".top-nav a[data-route-link='status']", "nav_status");
    setT(".top-nav a[data-route-link='wiki']", "nav_wiki_stats");
    setT(".wiki-hero__meta div:nth-child(1) dt", "wiki_recent_scrape");
    setT(".wiki-hero__meta div:nth-child(2) dt", "wiki_data_day");
    setT("#wiki-stats-error", "wiki_empty");

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
      if (pageRoute === "wiki" && lastWikiStats) renderWikiStats(lastWikiStats);
      else if (lastData) render(lastData);
    });
  }

  // --- Render & Refresh ----------------------------------------------------
  let lastData = null;

  function render(data) {
    // With diff/patch render, DOM nodes (and therefore user state like
    // .open / .show-guest, hover, scroll) survive across re-renders.
    // No save/restore dance needed.
    renderBanner(data);
    refreshReactions();
    renderUnresolvedIncidents(data);
    renderComponents(data);
    renderOnlineChart(data);
    renderPastIncidents(data);
    renderProbes(data);
  }

  // Probes section toggle (collapsed by default).
  // Animated open/close: measure scrollHeight on transition start so the
  // max-height transition matches the actual content size. The `hidden`
  // attribute is only re-applied after the close animation completes so
  // accessibility tooling still sees the panel as hidden when collapsed.
  const probesToggle = document.getElementById("probes-toggle");
  const probeList = document.getElementById("probe-list");
  if (probesToggle && probeList) {
    let animTimer = null;
    const open = () => {
      clearTimeout(animTimer);
      probeList.hidden = false;
      probeList.classList.add("animating", "is-collapsed");
      probeList.style.maxHeight = "0px";
      void probeList.offsetHeight;          // commit collapsed state
      probeList.classList.remove("is-collapsed");
      probeList.style.maxHeight = probeList.scrollHeight + "px";
      animTimer = setTimeout(() => {
        probeList.classList.remove("animating");
        probeList.style.maxHeight = "";     // release cap so DOM updates can grow
      }, 280);
    };
    const close = () => {
      clearTimeout(animTimer);
      probeList.classList.add("animating");
      probeList.style.maxHeight = probeList.scrollHeight + "px";
      void probeList.offsetHeight;          // commit current size
      probeList.classList.add("is-collapsed");
      probeList.style.maxHeight = "0px";
      animTimer = setTimeout(() => {
        probeList.classList.remove("animating", "is-collapsed");
        probeList.hidden = true;
        probeList.style.maxHeight = "";
      }, 280);
    };
    const toggle = () => {
      const expanded = probesToggle.getAttribute("aria-expanded") === "true";
      probesToggle.setAttribute("aria-expanded", String(!expanded));
      if (expanded) close();
      else open();
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

  function initReactionGrid() {
    const grid = document.getElementById("rx-grid");
    if (!grid) return;
    for (const id of REACTION_IDS) {
      const li = document.createElement("li");
      const a = document.createElement("a");
      a.href = "javascript:void(0)";
      a.dataset.eid = id;
      a.title = "贴贴";
      a.innerHTML = '<img class="emoji" src="/smiles/' + id + '.gif" alt="" loading="lazy" decoding="async">';
      a.addEventListener("click", (e) => { e.stopPropagation(); addReaction(id, e.currentTarget); });
      li.appendChild(a);
      grid.appendChild(li);
    }
  }

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
        '<span class="emoji" style="background-image:url(\'/smiles/' + id + '.gif\')"></span>' +
        '<span class="num">' + r.count + '</span>';
      btn.addEventListener("click", (e) => { e.stopPropagation(); addReaction(id, e.currentTarget); });
      actives.appendChild(btn);
    }

    // Picker grid: only update is-mine class; <img> nodes are created once by initReactionGrid.
    for (const id of REACTION_IDS) {
      const a = grid.querySelector("[data-eid='" + id + "']");
      if (a) a.className = (byID[id] && byID[id].mine) ? "is-mine" : "";
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
      reactionState = next;
      renderReactions();
    } catch (e) { /* network glitch — keep last render */ }
  }

  function spawnFloatEmoji(id, anchorEl) {
    const img = document.createElement("img");
    img.className = "rx-float";
    img.src = "/smiles/" + id + ".gif";
    img.alt = "";
    const rect = anchorEl.getBoundingClientRect();
    if (!rect.width && !rect.height) return; // element not yet laid out
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
  let reactionSSEReady = false; // skip delta animation on the first (baseline) push
  function connectReactionStream() {
    if (typeof EventSource === "undefined") return;
    try {
      const es = new EventSource("/api/reactions/stream?uid=" + encodeURIComponent(reactionUID));
      reactionES = es;
      es.onmessage = (ev) => {
        try {
          const next = JSON.parse(ev.data);
          const prev = reactionState;
          reactionState = next;
          renderReactions();
          if (reactionSSEReady) applyReactionDelta(prev, next);
          reactionSSEReady = true;
          reactionESBackoff = 1000;
        } catch (e) { /* ignore malformed frame */ }
      };
      es.onerror = () => {
        es.close();
        reactionES = null;
        reactionSSEReady = false;
        // Exponential backoff up to 30s.
        setTimeout(connectReactionStream, reactionESBackoff);
        reactionESBackoff = Math.min(reactionESBackoff * 2, 30000);
      };
    } catch (e) { /* SSE unavailable — poll only */ }
  }
  if (pageRoute === "status") {
    connectReactionStream();
    initReactionGrid();
    renderReactions();
  }

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

  // Live status updates via SSE for low latency. The 30s poll below always
  // runs as a safety net — a dropped or zombie SSE connection can never freeze
  // the page. Whichever delivers first wins; both just call render().
  let statusES = null;
  let statusESBackoff = 1000;
  function connectStatusStream() {
    if (typeof EventSource === "undefined") return;
    try {
      const es = new EventSource("/api/status/stream");
      statusES = es;
      es.onmessage = (ev) => {
        try {
          const data = JSON.parse(ev.data);
          lastData = data;
          render(data);
          statusESBackoff = 1000;
        } catch (e) { /* ignore malformed frame */ }
      };
      es.onerror = () => {
        es.close();
        statusES = null;
        setTimeout(connectStatusStream, statusESBackoff);
        statusESBackoff = Math.min(statusESBackoff * 2, 30000);
      };
    } catch (e) { /* SSE unavailable — poll only */ }
  }

  applyRouteChrome();
  applyI18n();
  if (pageRoute === "wiki") {
    initWikiStatsPage();
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden) initWikiStatsPage();
    });
  } else {
    refresh();
    connectStatusStream();
    setInterval(refresh, 30000);
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden) refresh();
    });
  }
})();
