<script lang="ts">
  import StatusIcon from "./StatusIcon.svelte";
  import { t } from "../lib/i18n.svelte";
  import { fmtDurationLong, incidentLabel, nowSec } from "../lib/format";
  import type { Overall } from "../lib/api-types";

  let { overall }: { overall: Overall | null } = $props();

  const affected = $derived((overall?.components ?? []).filter((c) => c.status !== "ok" && c.kind !== "guest"));
  const now = $derived(overall?.updated_at || nowSec());
</script>

{#if affected.length}
  <section class="unresolved" aria-label={t("section_unresolved")}>
    <header class="section-head">
      <h2>{t("section_unresolved")}</h2>
      <span class="section-head__hint">{t("affected", affected.length)}</span>
    </header>
    <div class="list">
      {#each affected as c (c.domain + "|" + c.kind)}
        {@const views = c.probe_views ?? []}
        {@const bad = views.filter((v) => v.status !== "ok").length}
        <div class="card card--{c.status}">
          <div class="icon"><StatusIcon status={c.status} /></div>
          <div class="body">
            <!-- Guest endpoints are filtered out, so the domain alone names it. -->
            <div class="title"><span class="sev">{incidentLabel(c.status)}</span> — {c.domain}</div>
            <div class="meta">
              <!-- How long the outage has run: the same live `since` the banner counts from. -->
              <span class="mono">{t("held_label")} {fmtDurationLong(now - (c.since ?? now))}</span>
              {#if bad && views.length}
                <span class="sep">·</span>
                <span>{t("probes_failing", bad, views.length)}</span>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  </section>
{/if}

<style>
  .unresolved {
    margin-bottom: 30px;
    padding: 14px 18px 16px;
    border: 1px solid var(--down-line);
    border-radius: var(--radius);
    background: linear-gradient(180deg, color-mix(in srgb, var(--down-soft) 85%, transparent), var(--surface) 80%);
    box-shadow: var(--shadow-sm);
  }
  .section-head { margin-bottom: 10px; padding: 0; }
  .section-head h2, .section-head__hint { color: var(--down); }
  .list { display: flex; flex-direction: column; gap: 10px; }
  .card {
    --tone: var(--none);
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 12px 14px;
    border: 1px solid var(--border);
    border-left: 3px solid var(--tone);
    border-radius: var(--radius-sm);
    background: var(--surface);
    transition: border-left-color 190ms ease;
  }
  .card--down { --tone: var(--down); }
  .card--degraded { --tone: var(--degraded); }
  .icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 22px;
    height: 22px;
    margin-top: 1px;
    border-radius: 50%;
    background: var(--tone);
    color: #fff;
  }
  .icon :global(svg) { display: block; width: 12px; height: 12px; }
  .body { flex: 1; min-width: 0; }
  .title { font-weight: 500; font-size: 14px; letter-spacing: -0.005em; line-height: 1.35; }
  .sev { font-weight: 600; }
  .meta {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 4px 8px;
    margin-top: 4px;
    font-size: 12px;
    color: var(--text-dim);
  }
  .sep { color: var(--text-faint); }
  .mono { font-family: var(--font-mono); font-size: 11.5px; }
</style>
