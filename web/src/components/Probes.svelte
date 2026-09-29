<script lang="ts">
  import { slide } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
  import { t } from "../lib/i18n.svelte";
  import { fmtRelative, regionFlag, regionLabel } from "../lib/format";
  import type { Probe } from "../lib/api-types";

  let { probes }: { probes: Probe[] } = $props();
  let open = $state(false);
  const online = $derived(probes.filter((p) => p.online).length);
</script>

<section class="probes" aria-label={t("section_probes")}>
  <button class="section-head toggle" type="button" aria-expanded={open} onclick={() => (open = !open)}>
    <h2>{t("section_probes")}</h2>
    <span class="section-head__hint">{probes.length ? t("probe_online_summary", online, probes.length) : "—"}</span>
  </button>
  {#if open}
    <div class="list" transition:slide={{ duration: 240, easing: cubicOut }}>
      {#each probes as p (p.name)}
        <div class="card" class:offline={!p.online}>
          <div>
            <div class="name">{p.name}</div>
            <div class="region">{p.region ? regionFlag(p.region) + " " : ""}{regionLabel(p.region)} · {fmtRelative(p.last_seen)}</div>
          </div>
          <span class="pill">{p.online ? t("probe_pill_online") : t("probe_pill_offline")}</span>
        </div>
      {:else}
        <div class="empty">{t("no_probes")}</div>
      {/each}
    </div>
  {/if}
</section>

<style>
  .probes { margin-bottom: 48px; }
  .toggle {
    width: 100%;
    border: 0;
    background: none;
    color: inherit;
    text-align: left;
    cursor: pointer;
    user-select: none;
  }
  .toggle h2::after {
    content: "▸";
    display: inline-block;
    margin-left: 6px;
    color: var(--text-faint);
    font-size: 11px;
    transition: transform 0.15s ease;
  }
  .toggle[aria-expanded="true"] h2::after { transform: rotate(90deg); }
  .list { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
  .card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 12px 14px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface);
    box-shadow: var(--shadow-sm);
    transition: border-color 0.15s ease;
  }
  .card:hover { border-color: var(--border-strong); }
  .card.offline { opacity: 0.75; }
  .name { font-weight: 600; font-size: 13.5px; letter-spacing: -0.01em; }
  .region { margin-top: 2px; color: var(--text-faint); font-family: var(--font-mono); font-size: 11.5px; }
  .pill {
    padding: 3px 9px;
    border: 1px solid var(--ok-line);
    border-radius: 999px;
    background: var(--ok-soft);
    color: var(--ok);
    font-size: 10.5px;
    font-weight: 500;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    white-space: nowrap;
    transition: background 190ms ease, color 190ms ease, border-color 190ms ease;
  }
  .offline .pill { border-color: var(--down-line); background: var(--down-soft); color: var(--down); }
  .empty { padding: 12px 2px; color: var(--text-faint); font-size: 13px; }
</style>
