<script lang="ts" module>
  import type { Incident, Kind } from "../lib/api-types";
  import {
    fmtDayLabel, fmtDayRelative, fmtDuration, fmtRelative, fmtTimeLocal, incidentLabel, isoDayLocal, nowSec, severityRank,
  } from "../lib/format";

  export type DayEntry = Incident & { label: string; kind: Kind };

  /** Groups incidents by local start day, newest day and newest incident first. */
  export function groupByDay(entries: DayEntry[], seedDays: string[] = []): { iso: string; entries: DayEntry[] }[] {
    const byDay = new Map<string, DayEntry[]>(seedDays.map((d) => [d, []]));
    for (const e of entries) {
      const iso = isoDayLocal(e.start_ts);
      byDay.set(iso, [...(byDay.get(iso) ?? []), e]);
    }
    return [...byDay.keys()]
      .sort()
      .reverse()
      .map((iso) => ({
        iso,
        entries: byDay.get(iso)!.sort((a, b) => b.start_ts - a.start_ts || severityRank(a.status) - severityRank(b.status)),
      }));
  }
</script>

<script lang="ts">
  import { slide } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
  import { t } from "../lib/i18n.svelte";

  let { iso, entries }: { iso: string; entries: DayEntry[] } = $props();

  // Public-endpoint incidents fold away by default.
  let showGuest = $state(false);
  const guestCount = $derived(entries.filter((e) => e.kind === "guest").length);
  const rel = $derived(fmtDayRelative(iso));

  function describe(e: DayEntry) {
    const ongoing = nowSec() - e.end_ts < 240;
    const crossDay = isoDayLocal(e.end_ts) !== isoDayLocal(e.start_ts);
    const end = ongoing ? t("ongoing") : fmtTimeLocal(e.end_ts) + (crossDay ? "⁺¹" : "");
    const bits = [];
    if (e.peak_down && e.peak_total) bits.push(t("peak_failing", e.peak_down, e.peak_total));
    bits.push(fmtDuration(e.end_ts - e.start_ts));
    return `${fmtTimeLocal(e.start_ts)} – ${end} · ${bits.join(" · ")}`;
  }
</script>

<div class="day">
  <div class="date">{fmtDayLabel(iso)}{#if rel}<span class="rel">{rel}</span>{/if}</div>
  {#if entries.length === 0}
    <div class="none">{t("no_incidents")}</div>
  {:else}
    <div class="list">
      {#each entries as e (`${e.kind}|${e.start_ts}|${e.label}`)}
        {#if e.kind !== "guest" || showGuest}
          <div class="entry" transition:slide={{ duration: 240, easing: cubicOut }}>
            <span class="dot status-{e.status}"></span>
            <div class="label">
              <span class="sev sev--{e.status}">{incidentLabel(e.status)}</span> — <span>{e.label}</span>
              <span class="desc">{describe(e)}</span>
            </div>
            <span class="metric">{fmtRelative(e.end_ts)}</span>
          </div>
        {/if}
      {/each}
    </div>
    {#if guestCount}
      <button class="fold" type="button" onclick={() => (showGuest = !showGuest)}>
        {showGuest ? t("expanded_guest_hint", guestCount) : t("collapsed_guest_hint", guestCount)}
      </button>
    {/if}
  {/if}
</div>

<style>
  .day { padding-bottom: 18px; border-bottom: 1px solid var(--border); }
  .day:last-child { padding-bottom: 0; border-bottom: none; }
  .date {
    display: flex;
    align-items: baseline;
    gap: 10px;
    margin-bottom: 10px;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: -0.005em;
  }
  .rel { color: var(--text-faint); font-family: var(--font-mono); font-size: 11px; font-weight: 400; letter-spacing: 0.02em; }
  .none, .fold { padding: 2px 0; color: var(--text-faint); font-size: 13px; font-style: italic; }
  .fold {
    display: block;
    padding: 4px 0;
    border: 0;
    background: none;
    cursor: pointer;
    user-select: none;
  }
  .fold:hover { color: var(--text); }
  .fold:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: 3px; }
  .list { display: flex; flex-direction: column; }
  .entry {
    display: grid;
    grid-template-columns: 18px 1fr auto;
    align-items: baseline;
    gap: 10px;
    padding: 7px 0;
    font-size: 13px;
  }
  .entry + .entry { margin-top: 8px; border-top: 1px dashed color-mix(in srgb, var(--border) 70%, transparent); }
  .dot {
    justify-self: center;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--status);
    transform: translateY(1px);
    transition: background 190ms ease;
  }
  .label { min-width: 0; }
  .sev { font-weight: 600; }
  .sev--down { color: var(--down); }
  .sev--degraded { color: var(--degraded); }
  .desc { display: block; margin-top: 2px; color: var(--text-dim); font-size: 12px; }
  .metric {
    color: var(--text-faint);
    font-family: var(--font-mono);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  @media (max-width: 560px) {
    .entry { grid-template-columns: 18px 1fr; }
    .metric { grid-column: 2; color: var(--text-dim); }
  }
</style>
