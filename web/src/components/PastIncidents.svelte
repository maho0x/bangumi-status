<script lang="ts">
  import IncidentDay, { groupByDay, type DayEntry } from "./IncidentDay.svelte";
  import { t } from "../lib/i18n.svelte";
  import { componentLabel, nowSec } from "../lib/format";
  import type { ComponentStatus } from "../lib/api-types";

  let { components }: { components: ComponentStatus[] } = $props();

  // The last 10 days, every day listed even when quiet.
  const days = $derived.by(() => {
    if (!components.length) return [];
    const cutoff = nowSec() - 10 * 86400;
    const seed = (components[0].days ?? []).slice(-10).map((b) => b.day);
    const entries: DayEntry[] = components.flatMap((c) =>
      (c.incidents ?? []).filter((i) => i.start_ts >= cutoff).map((i) => ({ ...i, label: componentLabel(c), kind: c.kind })),
    );
    return groupByDay(entries, seed);
  });
</script>

<section class="past" aria-label={t("section_past")}>
  <header class="section-head">
    <h2>{t("section_past")}</h2>
    <div class="right">
      <span class="section-head__hint">{t("hint_past")}</span>
      <a href="/history" class="more">{t("past_incidents_more")}</a>
    </div>
  </header>
  <div class="days">
    {#each days as d (d.iso)}
      <IncidentDay iso={d.iso} entries={d.entries} />
    {:else}
      <div class="none">{t("no_data")}</div>
    {/each}
  </div>
</section>

<style>
  .past { margin-bottom: 44px; }
  .right { display: inline-flex; align-items: baseline; gap: 10px; }
  .more { color: var(--text-faint); font-family: var(--font-mono); font-size: 11.5px; letter-spacing: 0.01em; }
  .more:hover { color: var(--text); text-decoration: none; }
  .days { display: flex; flex-direction: column; gap: 22px; }
  .none { color: var(--text-faint); font-size: 13px; font-style: italic; }
</style>
