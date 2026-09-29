<script lang="ts">
  import { tip } from "../lib/tooltip.svelte";
  import { t } from "../lib/i18n.svelte";
  import { fmtDayLabel, fmtDayRelative, fmtUptime, statusLabel, weekdayShort } from "../lib/format";

  let el = $state<HTMLDivElement>();
  let left = $state(0);
  let top = $state(0);

  // Centre above the anchor, clamped to the viewport; flip below when there
  // is no room above.
  $effect(() => {
    if (!el || !tip.content) return;
    void tip.x, tip.top, tip.bottom, tip.content;
    const w = el.offsetWidth;
    const h = el.offsetHeight;
    left = Math.max(6, Math.min(tip.x - w / 2, window.innerWidth - w - 6));
    const above = tip.top - h - 8;
    top = above < 6 ? tip.bottom + 6 : above;
  });
</script>

{#if tip.content}
  <div class="tooltip" role="tooltip" bind:this={el} style:left="{left}px" style:top="{top}px">
    {#if "text" in tip.content}
      {tip.content.text}
    {:else}
      {@const d = tip.content.day}
      {@const rel = fmtDayRelative(d.day) || weekdayShort(d.day)}
      <div class="head">
        <span class="date">{fmtDayLabel(d.day)}</span>
        {#if rel}<span class="rel">{rel}</span>{/if}
      </div>
      <div class="divider"></div>
      {#if !d.total}
        <div class="empty"><span class="dot"></span><span>{t("no_probe_data")}</span></div>
      {:else}
        {@const status = d.status || "ok"}
        <div class="meta">
          <span class="status"><span class="dot status-{status}"></span><span>{statusLabel(status)}</span></span>
          <span class="uptime">{fmtUptime(d.uptime)}</span>
        </div>
        <div class="counts">
          <span>{t("checks", d.total)}</span>
          {#if d.down}<span class="sep">·</span><span class="down">{t("failed_count", d.down)}</span>{/if}
          {#if d.degrade}<span class="sep">·</span><span class="degrade">{t("degraded_count", d.degrade)}</span>{/if}
        </div>
      {/if}
    {/if}
  </div>
{/if}

<style>
  .tooltip {
    position: fixed;
    z-index: 50;
    max-width: 280px;
    padding: 9px 12px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: color-mix(in srgb, var(--surface) 92%, transparent);
    backdrop-filter: saturate(1.4) blur(10px);
    -webkit-backdrop-filter: saturate(1.4) blur(10px);
    box-shadow: 0 1px 2px rgba(20, 18, 16, 0.04), 0 12px 28px -12px rgba(20, 18, 16, 0.18);
    color: var(--text);
    font-size: 12px;
    letter-spacing: 0.01em;
    pointer-events: none;
  }
  .head { display: flex; align-items: baseline; gap: 8px; white-space: nowrap; }
  .date { font-weight: 600; font-size: 12.5px; }
  .rel { font-size: 11px; color: var(--text-faint); font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
  .divider { height: 1px; margin: 7px -2px; background: var(--border); }
  .meta { display: flex; align-items: center; justify-content: space-between; gap: 14px; white-space: nowrap; }
  .status { display: inline-flex; align-items: center; gap: 6px; }
  .dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; background: var(--status, var(--none)); }
  .empty .dot { border: 1px solid var(--border-strong); }
  .uptime {
    font-family: var(--font-mono);
    font-size: 13.5px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.005em;
  }
  .counts {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 4px 8px;
    margin-top: 6px;
    color: var(--text-dim);
    font-family: var(--font-mono);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
  }
  .sep { color: var(--text-faint); }
  .down { color: var(--down); font-weight: 500; }
  .degrade { color: var(--degraded); font-weight: 500; }
  .empty { display: inline-flex; align-items: center; gap: 7px; color: var(--text-faint); font-style: italic; }
</style>
