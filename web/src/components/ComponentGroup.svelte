<script lang="ts">
  import { slide } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
  import ComponentRow from "./ComponentRow.svelte";
  import { t } from "../lib/i18n.svelte";
  import type { ComponentStatus, Status } from "../lib/api-types";

  let { domain, comps }: { domain: string; comps: ComponentStatus[] } = $props();

  const hasAuth = $derived(comps.some((c) => c.kind === "auth"));
  const hasGuest = $derived(comps.some((c) => c.kind === "guest"));
  // Guest rows fold away when the group also has an authenticated row.
  let showGuest = $state(false);
  const worst = $derived(comps.reduce<Status>((w, c) => (rank(c.status) > rank(w) ? c.status : w), "ok"));
  const guestBad = $derived(comps.filter((c) => c.kind === "guest" && c.status !== "ok"));

  function rank(s: Status) {
    return s === "down" ? 2 : s === "degraded" ? 1 : 0;
  }
</script>

<div class="group">
  <div class="hdr">
    <span class="domain"><span class="status-dot status-{worst}"></span>{domain}</span>
    <div class="hdr-right">
      {#if guestBad.length}
        <span class="badge" class:degraded={!guestBad.some((c) => c.status === "down")} title={t("collapsed_guest_hint", guestBad.length)}>
          {guestBad.length}
        </span>
      {/if}
      {#if hasGuest && hasAuth}
        <button class="guest-toggle" type="button" aria-pressed={showGuest} onclick={() => (showGuest = !showGuest)}>
          <span class="chevron" aria-hidden="true"></span>
          <span>{showGuest ? t("guest_hide") : t("guest_show")}</span>
        </button>
      {/if}
    </div>
  </div>
  {#each comps as c (c.kind)}
    {#if c.kind !== "guest" || showGuest || !hasAuth}
      <div class="row" transition:slide={{ duration: 240, easing: cubicOut }}>
        <ComponentRow {c} />
      </div>
    {/if}
  {/each}
</div>

<style>
  .group {
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: var(--shadow-sm);
  }
  .hdr {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 18px;
    background: var(--surface-sunk);
    border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
  }
  .domain {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-family: var(--font-mono);
    font-size: 12.5px;
    font-weight: 600;
    letter-spacing: 0.01em;
    color: var(--text-dim);
  }
  .hdr-right { display: inline-flex; align-items: center; gap: 10px; }
  .row + .row { border-top: 1px solid var(--border); }

  .badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    border-radius: 9px;
    background: var(--down);
    color: #fff;
    font-size: 11px;
    font-weight: 600;
    line-height: 1;
    transition: background 190ms ease;
  }
  .badge.degraded { background: var(--degraded); }

  .guest-toggle {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px 2px 5px;
    border: 1px solid transparent;
    border-radius: 999px;
    background: none;
    color: var(--text-faint);
    font-size: 11px;
    font-weight: 500;
    letter-spacing: 0.02em;
    cursor: pointer;
    transition: color 0.15s ease, background 0.15s ease, border-color 0.15s ease;
  }
  .guest-toggle:hover { color: var(--text); background: var(--surface); border-color: var(--border); }
  .guest-toggle[aria-pressed="true"] { color: var(--text); }
  .chevron {
    width: 0;
    height: 0;
    border-top: 4px solid transparent;
    border-bottom: 4px solid transparent;
    border-left: 5px solid currentColor;
    transition: transform 0.18s ease;
  }
  [aria-pressed="true"] .chevron { transform: rotate(90deg); }
</style>
