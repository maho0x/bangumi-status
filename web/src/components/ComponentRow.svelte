<script lang="ts">
  import { slide } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
  import UptimeStrip from "./UptimeStrip.svelte";
  import { t } from "../lib/i18n.svelte";
  import { fmtUptime, kindLabel, regionFlag, statusLabel } from "../lib/format";
  import type { ComponentStatus } from "../lib/api-types";

  let { c }: { c: ComponentStatus } = $props();
  let open = $state(false);
  const views = $derived(c.probe_views ?? []);
</script>

<div class="component" class:open>
  <div class="label">
    <span class="status-dot status-{c.status}"></span>
    <span class="kind">{kindLabel(c)}</span>
    <span class="status">· {statusLabel(c.status)}</span>
  </div>
  <div class="right">
    <UptimeStrip days={c.days ?? []} />
    <span class="uptime">{fmtUptime(c.uptime)} {t("uptime_label")}</span>
    <button class="probe-toggle" type="button" aria-expanded={open} aria-label="toggle probe detail" onclick={() => (open = !open)}>
      <span class="count">{views.length}</span><span class="chevron" aria-hidden="true">{open ? "▾" : "▸"}</span>
    </button>
  </div>
  {#if open}
    <div class="detail" transition:slide={{ duration: 240, easing: cubicOut }}>
      {#each views as v (v.region + "|" + v.probe)}
        <div class="probe">
          <span class="name">{v.region ? regionFlag(v.region) + " " : ""}{v.probe}</span>
          <span class="err">{v.err || (v.http_code ? `HTTP ${v.http_code}` : "—")}</span>
          <span class="lat">{v.latency_ms} ms</span>
          <span class="pstatus"><span class="status-dot status-{v.status}"></span><span>{statusLabel(v.status)}</span></span>
        </div>
      {:else}
        <div class="probe empty">{t("no_probe_data_detail")}</div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .component {
    display: grid;
    grid-template-columns: minmax(160px, 220px) minmax(0, 1fr);
    column-gap: 20px;
    align-items: start;
    padding: 12px 18px 8px;
    transition: background 0.15s ease;
  }
  .component:hover { background: color-mix(in srgb, var(--surface-2) 40%, transparent); }

  .label { display: flex; align-items: center; gap: 8px; min-width: 0; height: 20px; font-size: 13px; }
  .kind { font-weight: 500; }
  .status { margin-left: 1px; font-size: 11.5px; color: var(--text-dim); }

  .right { display: flex; align-items: center; gap: 12px; min-width: 0; overflow: hidden; }
  .uptime {
    flex: 0 0 80px;
    color: var(--text-dim);
    font-family: var(--font-mono);
    font-size: 11px;
    font-weight: 500;
    font-variant-numeric: tabular-nums;
    text-align: right;
    white-space: nowrap;
  }

  .probe-toggle {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    gap: 4px;
    padding: 2px 7px;
    border: 1px solid var(--border);
    border-radius: 20px;
    background: none;
    color: var(--text-faint);
    font-family: var(--font-mono);
    font-size: 11px;
    white-space: nowrap;
    cursor: pointer;
    transition: color 0.15s, border-color 0.15s, background 0.15s;
  }
  .probe-toggle:hover { color: var(--text); border-color: var(--border-strong); background: var(--surface-2); }
  .open .probe-toggle { color: var(--text); border-color: var(--border-strong); }
  .count { font-variant-numeric: tabular-nums; }
  .chevron { font-size: 9px; }

  .detail {
    grid-column: 1 / -1;
    min-width: 0;
    margin-top: 14px;
    padding-top: 14px;
    border-top: 1px dashed var(--border-strong);
  }
  .probe {
    display: grid;
    grid-template-columns: minmax(80px, 180px) minmax(0, 1fr) 80px 90px;
    align-items: center;
    gap: 10px;
    min-width: 0;
    padding: 7px 0;
    color: var(--text-dim);
    font-size: 12.5px;
  }
  .probe + .probe { border-top: 1px solid color-mix(in srgb, var(--border) 60%, transparent); }
  .probe.empty { grid-template-columns: 1fr; color: var(--text-faint); }
  .name {
    overflow: hidden;
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 12px;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .err {
    overflow: hidden;
    color: var(--text-faint);
    font-family: var(--font-mono);
    font-size: 11.5px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .lat { text-align: right; color: var(--text); font-family: var(--font-mono); font-size: 12px; font-variant-numeric: tabular-nums; }
  .pstatus { display: inline-flex; align-items: center; justify-content: flex-end; gap: 6px; font-size: 12px; }

  @media (max-width: 720px) {
    .component { grid-template-columns: 1fr; row-gap: 8px; padding: 10px 14px; }
    .right { flex-wrap: wrap; gap: 4px 8px; }
    .uptime { flex: 1 0 auto; font-size: 10.5px; text-align: left; }
  }
  @media (max-width: 560px) {
    .probe { grid-template-columns: 1fr auto; gap: 2px 10px; }
    .name { grid-area: 1 / 1; }
    .pstatus { grid-area: 1 / 2; }
    .err { grid-area: 2 / 1; white-space: normal; word-break: break-all; }
    .lat { grid-area: 2 / 2; }
  }
</style>
