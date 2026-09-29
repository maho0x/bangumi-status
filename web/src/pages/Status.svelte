<script lang="ts">
  import Banner from "../components/Banner.svelte";
  import Reactions from "../components/Reactions.svelte";
  import Unresolved from "../components/Unresolved.svelte";
  import ComponentGroup from "../components/ComponentGroup.svelte";
  import ActivityChart from "../components/ActivityChart.svelte";
  import PastIncidents from "../components/PastIncidents.svelte";
  import Probes from "../components/Probes.svelte";
  import { live, startLive } from "../lib/live.svelte";
  import { t } from "../lib/i18n.svelte";
  import { statusLabel } from "../lib/format";
  import type { ComponentStatus } from "../lib/api-types";

  $effect(() => startLive());

  const components = $derived(live.overall?.components ?? []);
  const groups = $derived.by(() => {
    const byDomain = new Map<string, ComponentStatus[]>();
    for (const c of components) byDomain.set(c.domain, [...(byDomain.get(c.domain) ?? []), c]);
    return [...byDomain].map(([domain, comps]) => ({ domain, comps }));
  });
</script>

<Banner overall={live.overall} error={live.error} />
<Reactions />
<Unresolved overall={live.overall} />

<section class="components" aria-label={t("section_current")}>
  <header class="section-head">
    <h2>{t("section_current")}</h2>
    <span class="section-head__hint">{t("hint_30d")}</span>
  </header>
  <div class="legend" aria-hidden="true">
    {#each ["ok", "degraded", "down", "none"] as s (s)}
      <span class="legend__item"><span class="status-dot status-{s}" class:none={s === "none"}></span>{statusLabel(s)}</span>
    {/each}
  </div>
  <div class="groups">
    {#each groups as g (g.domain)}
      <ComponentGroup domain={g.domain} comps={g.comps} />
    {/each}
  </div>
</section>

<ActivityChart overall={live.overall} />
<PastIncidents {components} />
<Probes probes={live.overall?.probes ?? []} />

<style>
  .components { margin-bottom: 44px; }
  .groups { display: flex; flex-direction: column; gap: 10px; }
  .legend {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 18px;
    margin-bottom: 14px;
    padding: 0 2px;
    color: var(--text-dim);
    font-size: 12px;
  }
  .legend__item { display: inline-flex; align-items: center; gap: 7px; }
  .none { border: 1px solid var(--border-strong); }
</style>
