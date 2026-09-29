<script lang="ts">
  import StatusIcon from "./StatusIcon.svelte";
  import { t } from "../lib/i18n.svelte";
  import { fmtDurationLong, fmtUptime } from "../lib/format";
  import type { Overall, Status } from "../lib/api-types";

  let { overall, error }: { overall: Overall | null; error: string } = $props();

  // Guest/public endpoints never escalate the banner; they surface as a badge
  // on their group instead.
  const nonGuest = $derived((overall?.components ?? []).filter((c) => c.kind !== "guest"));
  const status = $derived(
    nonGuest.reduce<Status>((w, c) => (rank(c.status) > rank(w) ? c.status : w), "ok"),
  );
  const affected = $derived(nonGuest.filter((c) => c.status !== "ok").length);
  // How long the current outage has lasted: the earliest live-derived `since`
  // among affected components, the same stamp the backend reports.
  const since = $derived(
    nonGuest.filter((c) => c.status !== "ok" && c.since).reduce<number | null>((m, c) => (m == null || c.since! < m ? c.since! : m), null),
  );
  const avgUptime = $derived(nonGuest.reduce((a, c) => a + (c.uptime || 0), 0) / (nonGuest.length || 1));

  function rank(s: Status) {
    return s === "down" ? 2 : s === "degraded" ? 1 : 0;
  }

  const state = $derived(error ? "down" : overall ? status : "loading");
</script>

<section class="banner banner--{state}" aria-live="polite">
  <div class="left">
    <div class="icon"><StatusIcon status={state} /></div>
    <div>
      {#if error}
        <h1>{t("error_load")}</h1>
        <p>{error}</p>
      {:else if overall}
        <h1>{t(`banner_${status}`)}</h1>
        <p>{affected ? t("banner_sub_affected", affected, nonGuest.length) : t("banner_sub_all", nonGuest.length)}</p>
      {:else}
        <h1>{t("banner_loading")}</h1>
        <p>{t("banner_loading_sub")}</p>
      {/if}
    </div>
  </div>
  {#if overall}
    <dl>
      <!-- During an outage only its duration matters; otherwise the 30-day uptime. -->
      {#if status !== "ok"}
        {#if since != null}
          <div><dt>{t("held_label")}</dt><dd class="num">{fmtDurationLong(overall.updated_at - since)}</dd></div>
        {/if}
      {:else}
        <div><dt>{t("uptime_30d")}</dt><dd class="num">{fmtUptime(avgUptime)}</dd></div>
      {/if}
    </dl>
  {/if}
</section>

<style>
  .banner {
    --tone: var(--border-strong);
    position: relative;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 32px;
    margin-bottom: 30px;
    padding: 22px 26px;
    border: 1px solid var(--line, var(--border-strong));
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: var(--shadow-md);
    transition: border-color 0.2s ease;
  }
  .banner::before {
    content: "";
    position: absolute;
    inset: 0 auto 0 0;
    width: 4px;
    background: var(--tone);
    transition: background 0.2s ease;
  }
  .banner--ok { --tone: var(--ok); --line: var(--ok-line); }
  .banner--degraded { --tone: var(--degraded); --line: var(--degraded-line); }
  .banner--down { --tone: var(--down); --line: var(--down-line); }

  .left { display: flex; align-items: center; gap: 18px; min-width: 0; }
  .icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: var(--tone);
    box-shadow: inset 0 -2px 0 rgba(0,0,0,0.06);
    color: #fff;
    transition: background 220ms var(--ease-out);
  }
  .icon :global(svg) { display: block; width: 18px; height: 18px; }
  .banner--loading .icon { background: var(--surface-2); color: var(--text-faint); }
  .banner--loading .icon :global(svg) { animation: pulse 1.4s ease-in-out infinite; }
  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }

  h1 { font-size: 20px; letter-spacing: -0.02em; line-height: 1.2; }
  p { margin: 4px 0 0; font-size: 13.5px; color: var(--text-dim); }

  dl { display: flex; gap: 32px; margin: 0; text-align: right; }
  dl > div { display: flex; flex-direction: column; gap: 4px; }
  dt { font-size: 11px; font-weight: 500; letter-spacing: 0.09em; text-transform: uppercase; color: var(--text-faint); }
  dd { margin: 0; font-size: 20px; font-weight: 500; letter-spacing: -0.01em; }

  @media (max-width: 640px) {
    .banner { gap: 12px; padding: 14px 16px; }
    .left { gap: 12px; }
    .icon { width: 28px; height: 28px; }
    .icon :global(svg) { width: 14px; height: 14px; }
    h1 { font-size: 16px; }
    p { font-size: 12px; }
    dl { gap: 16px; flex-shrink: 0; }
    dt { font-size: 10px; }
    dd { font-size: 16px; }
  }
</style>
