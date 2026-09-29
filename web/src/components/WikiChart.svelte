<script lang="ts">
  import uPlot from "uplot";
  import "uplot/dist/uPlot.min.css";
  import { autosize, cssVar, withAlpha } from "../lib/chart";
  import type { WikiStatsPoint } from "../lib/api-types";

  type Field = keyof WikiStatsPoint;
  let { points, series }: { points: WikiStatsPoint[]; series: [string, Field][] } = $props();

  // Distinct line colours; the longest set is the 8-series replies breakdown.
  const PALETTE = ["#f09199", "#85c99b", "#6CA0DC", "#D9C775", "#C58FD9", "#E0936B", "#5FC9C2", "#D97A92", "#9DC468", "#B08CC4"];
  const HEIGHT = 460;

  let host = $state<HTMLDivElement>();

  // Multi-series line chart over a category axis (one x per day). The legend
  // toggles series and shows hovered values; drag zooms x, double-click resets.
  $effect(() => {
    if (!host) return;
    const labels = points.map((p) => p.title || p.date.slice(5));
    const data: uPlot.AlignedData = [points.map((_, i) => i), ...series.map(([, f]) => points.map((p) => Number(p[f]) || 0))];
    const axis = (extra: Partial<uPlot.Axis>): uPlot.Axis => ({
      stroke: cssVar("--text-faint", "#9a9c94"),
      ticks: { show: false },
      gap: 6,
      font: "11px ui-monospace, monospace",
      grid: { stroke: withAlpha(cssVar("--border", "#272a26"), 0.7), width: 1, dash: [2, 3] },
      ...extra,
    });
    const fill = cssVar("--surface", "#ffffff");
    const u = new uPlot(
      {
        width: Math.max(200, host.clientWidth),
        height: HEIGHT,
        padding: [14, 16, 0, 6],
        scales: { x: { time: false }, y: { range: (_u, _min, max) => [0, (max || 1) * 1.03] } },
        legend: { show: true, live: true },
        cursor: { drag: { x: true, y: false }, focus: { prox: 30 }, points: { size: 7 } },
        axes: [
          axis({ space: 64, values: (_u, splits) => splits.map((i) => labels[Math.round(i)] ?? "") }),
          axis({ size: 54, values: (_u, splits) => splits.map((v) => Math.round(v).toLocaleString()) }),
        ],
        series: [
          {},
          ...series.map(([label], i): uPlot.Series => {
            const color = PALETTE[i % PALETTE.length];
            return {
              label,
              stroke: color,
              width: 2,
              points: { show: true, size: 5, stroke: color, fill, width: 1.5 },
              value: (_u, v) => (v == null ? "—" : v.toLocaleString()),
            };
          }),
        ],
      },
      data,
      host,
    );
    const stop = autosize(host, (width) => u.setSize({ width, height: HEIGHT }));
    return () => {
      stop();
      u.destroy();
    };
  });
</script>

<div class="chart" bind:this={host}></div>

<style>
  .chart { width: 100%; min-height: 560px; }
  @media (max-width: 720px) {
    .chart { min-height: 520px; }
  }
</style>
