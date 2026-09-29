<script lang="ts">
  import uPlot from "uplot";
  import "uplot/dist/uPlot.min.css";
  import { live } from "../lib/live.svelte";
  import { t } from "../lib/i18n.svelte";
  import { nowSec } from "../lib/format";
  import { autosize, cssVar, withAlpha } from "../lib/chart";
  import { hideTip, showTipAt } from "../lib/tooltip.svelte";
  import type { Incident, OnlinePoint, Overall } from "../lib/api-types";

  let { overall }: { overall: Overall | null } = $props();

  type Metric = "bangumi" | "traffic";
  type Range = "24h" | "7d" | "30d" | "all";
  let metric = $state<Metric>("bangumi");
  let range = $state<Range>("24h");
  let fetched = $state<{ key: string; points: OnlinePoint[] } | null>(null);

  // Bangumi's 24h series rides along in /api/status; everything else is
  // fetched, and re-fetched whenever a new snapshot lands.
  $effect(() => {
    void overall?.updated_at;
    const key = metric + "|" + range;
    if (metric === "bangumi" && range === "24h") return;
    const path = metric === "traffic" ? "/api/traffic" : "/api/online";
    fetch(`${path}?range=${range}`)
      .then((r) => r.json())
      .then((points: OnlinePoint[]) => {
        if (key === metric + "|" + range) fetched = { key, points: Array.isArray(points) ? points : [] };
      })
      .catch(() => {});
  });

  const raw = $derived.by((): OnlinePoint[] | null => {
    if (metric === "bangumi" && range === "24h") return overall?.online ?? [];
    return fetched?.key === metric + "|" + range ? fetched.points : null;
  });

  // Traffic history is a per-minute peak; the live viewer count extends it to
  // now, never pulling the current minute below its recorded peak.
  const points = $derived.by(() => {
    if (!raw) return null;
    const pts = raw.filter((p) => Number.isFinite(p.count) && (metric === "traffic" ? p.count >= 0 : p.count > 0));
    if (metric === "traffic" && live.viewers != null) {
      const ts = live.viewersAt || nowSec();
      const last = pts.at(-1);
      const count = last && Math.floor(last.ts / 60) === Math.floor(ts / 60) ? Math.max(live.viewers, last.count) : live.viewers;
      if (last && last.ts >= ts) pts[pts.length - 1] = { ts, count };
      else pts.push({ ts, count });
    }
    return pts;
  });

  // Main-site outages shaded behind the series.
  const incidents = $derived(
    (overall?.components ?? [])
      .filter((c) => (c.domain === "bgm.tv" || c.domain === "bangumi.tv") && c.kind === "auth")
      .flatMap((c) => c.incidents ?? []),
  );

  type Model = {
    xs: number[]; avg: number[]; low: number[]; high: number[];
    banded: boolean; peakIdx: number; incidents: Incident[]; range: Range;
  };

  // While another range loads (points === null), keep showing the last one.
  let shown: Model | null = null;
  const model = $derived.by((): Model | null => {
    if (!points) return shown;
    if (points.length < 2) return (shown = null);
    // Bucketed ranges carry a min/max band; raw 24h points don't.
    const banded = points.some((p) => (p.peak ?? 0) > 0);
    const xs = points.map((p) => p.ts);
    const avg = points.map((p) => p.count);
    const low = points.map((p) => (banded ? p.low || p.count : p.count));
    const high = points.map((p) => (banded ? p.peak || p.count : p.count));
    const peakIdx = high.reduce((bi, v, i) => (v > high[bi] ? i : bi), 0);
    // The series stores transitions only: carry the last value to now so the
    // stepped line reaches the right edge.
    const now = nowSec();
    if (xs.at(-1)! < now) {
      xs.push(now);
      avg.push(avg.at(-1)!);
      low.push(low.at(-1)!);
      high.push(high.at(-1)!);
    }
    const visible = incidents.filter((i) => (i.end_ts || now) > xs[0] && i.start_ts < xs.at(-1)!);
    return (shown = { xs, avg, low, high, banded, peakIdx, incidents: visible, range });
  });

  const summary = $derived.by(() => {
    if (metric === "traffic" && live.viewers != null && !model) return t("traffic_current", live.viewers.toLocaleString());
    if (!model) return "—";
    const cur = model.avg.at(-1)!.toLocaleString();
    const peak = Math.max(...model.high).toLocaleString();
    return t(metric === "traffic" ? "traffic_current" : "online_current", cur) + " · " + t("online_peak", peak);
  });

  let host = $state<HTMLDivElement>();
  let chart: uPlot | null = null;
  // Hooks read the latest model through this, so updates only setData().
  let current: Model | null = null;

  const pad2 = (n: number) => String(n).padStart(2, "0");
  function xLabel(r: Range, ts: number) {
    const d = new Date(ts * 1000);
    return r === "24h" ? `${pad2(d.getHours())}:${pad2(d.getMinutes())}` : `${d.getMonth() + 1}/${d.getDate()}`;
  }

  function create(el: HTMLDivElement, m: Model): uPlot {
    const ok = cssVar("--ok", "#85c99b");
    const grid = cssVar("--border", "#272a26");
    const axis = cssVar("--text-faint", "#9a9c94");
    const degraded = cssVar("--degraded", "#D9C775");
    const down = cssVar("--down", "#D97A92");
    const accent = cssVar("--accent", "#f09199");
    const surface = cssVar("--surface", "#ffffff");
    const dprOf = (u: uPlot) => u.ctx.canvas.width / u.width || window.devicePixelRatio || 1;

    // Incident shading and the min/max band, behind the line.
    const drawBands = (u: uPlot) => {
      const s = current;
      if (!s) return;
      const ctx = u.ctx, top = u.bbox.top, h = u.bbox.height;
      const x0 = s.xs[0], xN = s.xs.at(-1)!;
      ctx.save();
      for (const inc of s.incidents) {
        const x1 = u.valToPos(Math.max(inc.start_ts, x0), "x", true);
        const x2 = u.valToPos(Math.min(inc.end_ts || nowSec(), xN), "x", true);
        ctx.fillStyle = inc.status === "down" ? withAlpha(down, 0.22) : withAlpha(degraded, 0.18);
        ctx.fillRect(x1, top, Math.max(1, x2 - x1), h);
      }
      if (s.banded) {
        const trace = (ys: number[], reverse = false) => {
          const idx = s.xs.map((_, i) => i);
          for (const i of reverse ? idx.reverse() : idx) {
            const x = u.valToPos(s.xs[i], "x", true), y = u.valToPos(ys[i], "y", true);
            if (i === 0 && !reverse) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
          }
        };
        ctx.beginPath();
        trace(s.high);
        trace(s.low, true);
        ctx.closePath();
        ctx.fillStyle = withAlpha(ok, 0.13);
        ctx.fill();
        // A faint stroke along the peak edge so the envelope reads clearly.
        ctx.beginPath();
        trace(s.high);
        ctx.lineWidth = dprOf(u);
        ctx.strokeStyle = withAlpha(ok, 0.5);
        ctx.stroke();
      }
      ctx.restore();
    };

    // Accent dot + soft label at the peak. Hooks draw in device pixels.
    const drawPeak = (u: uPlot) => {
      const s = current;
      if (!s) return;
      const dpr = dprOf(u), ctx = u.ctx;
      const cx = u.valToPos(s.xs[s.peakIdx], "x", true);
      const cy = u.valToPos(s.high[s.peakIdx], "y", true);
      ctx.save();
      ctx.beginPath();
      ctx.arc(cx, cy, 4 * dpr, 0, Math.PI * 2);
      ctx.fillStyle = accent;
      ctx.fill();
      ctx.lineWidth = 1.5 * dpr;
      ctx.strokeStyle = surface;
      ctx.stroke();
      const right = cx > u.bbox.left + u.bbox.width * 0.6;
      ctx.fillStyle = axis;
      ctx.font = `${11 * dpr}px ui-monospace, "DejaVu Sans Mono", monospace`;
      ctx.textAlign = right ? "right" : "left";
      ctx.textBaseline = "bottom";
      ctx.fillText(t("online_peak", s.high[s.peakIdx].toLocaleString()), cx + (right ? -6 : 6) * dpr, Math.max(u.bbox.top + 12 * dpr, cy - 5 * dpr));
      ctx.restore();
    };

    const cursorTip = (u: uPlot) => {
      const s = current;
      const i = u.cursor.idx;
      if (!s || i == null) return hideTip();
      const d = new Date(s.xs[i] * 1000);
      const when = s.range === "24h"
        ? `${pad2(d.getHours())}:${pad2(d.getMinutes())}`
        : `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}` + (s.range === "7d" ? ` ${pad2(d.getHours())}:00` : "");
      const value = s.banded
        ? `${t("online_avg", s.avg[i].toLocaleString())} · ${t("online_peak", s.high[i].toLocaleString())}`
        : s.avg[i].toLocaleString();
      const rect = u.over.getBoundingClientRect();
      showTipAt({ text: `${when} · ${value}` }, rect.left + (u.cursor.left ?? 0), rect.top + u.valToPos(s.high[i], "y") - 2);
    };

    const u = new uPlot(
      {
        width: Math.max(200, el.clientWidth),
        height: el.clientHeight || 140,
        padding: [12, 10, 2, 6],
        legend: { show: false },
        cursor: { y: false, points: { show: false }, drag: { x: false, y: false } },
        scales: {
          x: { time: false },
          // Cover the band when bucketed; otherwise pad around the line.
          y: {
            range: (_u, dmin, dmax) => {
              const s = current;
              const lo = s?.banded ? Math.min(...s.low) : dmin;
              const hi = s?.banded ? Math.max(...s.high) : dmax;
              const pad = Math.max(1, (hi - lo) * 0.15);
              return [Math.max(0, lo - pad), hi + pad];
            },
          },
        },
        axes: [
          { stroke: axis, grid: { show: false }, ticks: { show: false }, gap: 4, size: 22, font: "11px ui-monospace, monospace",
            values: (_u, splits) => splits.map((ts) => xLabel(current?.range ?? "24h", ts)) },
          { stroke: axis, ticks: { show: false }, gap: 4, size: 34, font: "11px ui-monospace, monospace",
            grid: { stroke: withAlpha(grid, 0.7), width: 1, dash: [2, 3] },
            values: (_u, splits) => splits.map((v) => Math.round(v).toLocaleString()) },
        ],
        series: [
          {},
          {
            stroke: ok,
            width: 1.6,
            points: { show: false },
            // A held value until the next transition: draw it stepped.
            paths: uPlot.paths.stepped!({ align: 1 }),
            // Area fill on the raw 24h view; bucketed views use the band.
            fill: () => (current && !current.banded ? withAlpha(ok, 0.2) : "transparent"),
          },
        ],
        hooks: { drawClear: [drawBands], draw: [drawPeak], setCursor: [cursorTip] },
      },
      [m.xs, m.avg],
      el,
    );
    u.over.addEventListener("mouseleave", hideTip);
    return u;
  }

  $effect(() => {
    current = model;
    if (!host || !model) {
      chart?.destroy();
      chart = null;
      return;
    }
    if (chart) chart.setData([model.xs, model.avg]);
    else chart = create(host, model);
  });

  $effect(() => {
    if (!host) return;
    return autosize(host, (width, height) => chart?.setSize({ width, height: height || 140 }));
  });

  $effect(() => () => chart?.destroy());
</script>

<section class="activity" aria-label={t("section_online")}>
  <header class="section-head">
    <h2>{t("section_online")}</h2>
    <span class="section-head__hint">{summary}</span>
  </header>
  <div class="card">
    <div class="controls">
      <div class="seg" role="tablist">
        {#each [["bangumi", t("metric_bangumi")], ["traffic", t("metric_traffic")]] as [m, label] (m)}
          <button role="tab" aria-selected={metric === m} class:active={metric === m} onclick={() => (metric = m as Metric)}>{label}</button>
        {/each}
      </div>
      <div class="seg" role="tablist">
        {#each ["24h", "7d", "30d", "all"] as r (r)}
          <button role="tab" aria-selected={range === r} class:active={range === r} onclick={() => (range = r as Range)}>
            {r === "all" ? t("range_all") : r}
          </button>
        {/each}
      </div>
    </div>
    <div class="chart">
      <div class="host" bind:this={host}></div>
      {#if raw && !model}
        <div class="empty">{t("online_no_data")}</div>
      {/if}
    </div>
    <p class="note">{t(metric === "traffic" ? "traffic_note" : "online_note")}</p>
  </div>
</section>

<style>
  .activity { margin-bottom: 44px; }
  .card {
    overflow: hidden;
    padding: 0 0 14px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: var(--shadow-sm);
  }
  .controls {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px 16px;
    padding: 12px 14px;
    border-bottom: 1px solid var(--border);
  }
  .seg {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 2px;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    box-shadow: var(--shadow-sm);
  }
  .seg button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 26px;
    padding: 3px 11px;
    border: 0;
    border-radius: 999px;
    background: none;
    color: var(--text-faint);
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    cursor: pointer;
    transition: color 0.15s ease, background 0.15s ease;
  }
  .seg button:hover, .seg button.active { color: var(--text); background: var(--surface-2); }
  .chart { position: relative; width: 100%; height: 140px; margin-top: 14px; padding: 0 18px; }
  .host { width: 100%; height: 100%; }
  .host :global(.u-over) { cursor: crosshair; }
  .host :global(.u-cursor-x) { border-right: 1px dashed var(--text-dim); }
  .empty {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-faint);
    font-size: 12px;
    font-style: italic;
  }
  .note {
    margin: 8px 0 0;
    padding: 0 18px;
    color: var(--text-faint);
    font-family: var(--font-mono);
    font-size: 11px;
    letter-spacing: 0.01em;
  }
  @media (max-width: 560px) {
    .chart { height: 120px; }
  }
</style>
