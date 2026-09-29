<script lang="ts">
  import IncidentDay, { groupByDay, type DayEntry } from "../components/IncidentDay.svelte";
  import { t } from "../lib/i18n.svelte";
  import { fmtDayLabel, fmtDurationLong, fmtMonthLabel, isoDayLocal, ymKey } from "../lib/format";
  import type { HistoryIncident, IncidentHistory } from "../lib/api-types";

  // The archive is paged by a fixed window of whole local months. Paging
  // replaces the window, so the range label is always the whole story, and the
  // request names exact unix bounds so a month header never describes a range
  // the server only partially returned.
  const PAGE_MONTHS = 4;

  function monthWindow(endYM: string, count: number) {
    const [y, m] = endYM.split("-").map(Number);
    const months = Array.from({ length: count }, (_, i) => ymKey(new Date(y, m - 1 - i, 1)));
    return {
      from: new Date(y, m - count, 1).getTime() / 1000,
      to: new Date(y, m, 1).getTime() / 1000,
      months,
    };
  }

  let endYM = $state("");
  let months = $state<string[]>([]);
  let incidents = $state<HistoryIncident[]>([]);
  let earliest = $state(0);
  let loading = $state(false);
  let failed = $state(false);

  async function load(end = ymKey(new Date())) {
    if (loading) return;
    loading = true;
    const win = monthWindow(end, PAGE_MONTHS);
    try {
      const res = await fetch(`/api/incidents?from=${win.from}&to=${win.to}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: IncidentHistory = await res.json();
      incidents = data.incidents ?? [];
      earliest = data.earliest_ts ?? 0;
      endYM = end;
      months = win.months;
      failed = false;
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }
  load();

  // The oldest archived incident is the floor, the current month the ceiling.
  const atOldest = $derived(!months.length || !earliest || earliest >= monthWindow(months[0], months.length).from);
  const atNewest = $derived(!endYM || endYM >= ymKey(new Date()));

  function shift(pages: number) {
    if (loading || !endYM || (pages < 0 ? atOldest : atNewest)) return;
    const [y, m] = endYM.split("-").map(Number);
    const next = ymKey(new Date(y, m - 1 + pages * PAGE_MONTHS, 1));
    const capped = next > ymKey(new Date()) ? ymKey(new Date()) : next;
    if (capped !== endYM) load(capped);
  }

  const view = $derived(
    months.map((ym) => {
      const entries: DayEntry[] = incidents.filter((i) => isoDayLocal(i.start_ts).startsWith(ym));
      const days = groupByDay(entries);
      // Guest incidents are folded away by default, so the summary counts only
      // what the reader can see.
      const visible = entries.filter((e) => e.kind !== "guest");
      const downtime = visible.reduce((s, e) => s + Math.max(60, e.end_ts - e.start_ts), 0);
      return { ym, days, count: visible.length, downtime };
    }),
  );
</script>

<section aria-label={t("history_title")}>
  <header class="hero">
    <div>
      <div class="titlerow">
        <h1>{t("history_title")}</h1>
        <nav class="pager">
          <button type="button" aria-label={t("history_prev")} disabled={loading || atOldest} onclick={() => shift(-1)}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15.75 19.5 8.25 12l7.5-7.5" /></svg>
          </button>
          <span class="range num" aria-live="polite">
            {#if months.length}
              {months.length === 1 ? fmtMonthLabel(months[0]) : `${fmtMonthLabel(months.at(-1)!)} – ${fmtMonthLabel(months[0])}`}
            {:else}—{/if}
          </span>
          <button type="button" aria-label={t("history_next")} disabled={loading || atNewest} onclick={() => shift(1)}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8.25 4.5 15.75 12l-7.5 7.5" /></svg>
          </button>
        </nav>
      </div>
      <p class="hero__intro">{t("history_intro")}</p>
    </div>
    <dl class="hero__meta">
      <div>
        <dt>{t("history_earliest_label")}</dt>
        <dd class="num">{earliest ? fmtDayLabel(isoDayLocal(earliest)) : "—"}</dd>
      </div>
    </dl>
  </header>

  {#if failed}
    <div class="notice">{t("error_load")}</div>
  {:else if months.length && !earliest}
    <div class="notice">{t("history_empty")}</div>
  {/if}

  <div class="months">
    {#each view as m (m.ym)}
      <section>
        <header class="month-head">
          <h2>{fmtMonthLabel(m.ym)}</h2>
          <span class="summary num" class:clear={!m.count}>
            {m.count ? t("history_month_summary", m.count, fmtDurationLong(m.downtime)) : t("history_month_clear")}
          </span>
        </header>
        {#if m.days.length}
          <div class="days">
            {#each m.days as d (d.iso)}
              <IncidentDay iso={d.iso} entries={d.entries} />
            {/each}
          </div>
        {:else}
          <p class="none">{t("history_month_none")}</p>
        {/if}
      </section>
    {/each}
  </div>
</section>

<style>
  .titlerow { display: flex; align-items: center; flex-wrap: wrap; gap: 16px; }
  .pager { display: flex; align-items: center; gap: 2px; }
  .pager button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    padding: 0;
    border: 0;
    border-radius: var(--radius-sm);
    background: none;
    color: var(--text-faint);
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }
  .pager svg { width: 16px; height: 16px; }
  .pager button:hover:not(:disabled) { color: var(--text); background: var(--surface-2); }
  .pager button:disabled { color: var(--border-strong); cursor: not-allowed; }
  .range { min-width: 148px; color: var(--text-dim); font-size: 12px; letter-spacing: 0.01em; text-align: center; white-space: nowrap; }

  .months { display: flex; flex-direction: column; gap: 40px; }
  .month-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 12px;
    margin-bottom: 18px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--border-strong);
  }
  h2 { font-size: 17px; letter-spacing: -0.01em; }
  .summary { color: var(--text-dim); font-size: 11.5px; letter-spacing: 0.01em; }
  .summary.clear { color: var(--text-faint); }
  .days { display: flex; flex-direction: column; gap: 22px; }
  .none { margin: 0; padding: 2px 0; color: var(--text-faint); font-size: 13px; font-style: italic; }
</style>
