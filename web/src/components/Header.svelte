<script lang="ts">
  import { t } from "../lib/i18n.svelte";

  let { route, onsubscribe }: { route: string; onsubscribe: () => void } = $props();
</script>

<div class="page-top">
  <a class="brand" href="/">
    <span class="badge" aria-hidden="true">
      <svg viewBox="0 0 28 18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1,14 6,9 10,11 15,4 20,7 27,3" /></svg>
    </span>
    <span class="name"><span class="italic">Bangumi</span>&nbsp;Status</span>
  </a>
  <div class="actions">
    <nav class="pills" aria-label={t("nav_label")}>
      <a href="/" aria-current={route === "status" ? "page" : undefined}>{t("nav_status")}</a>
      <a href="/stats" aria-current={route === "stats" ? "page" : undefined}>{t("nav_wiki_stats")}</a>
    </nav>
    <button class="subscribe" type="button" aria-haspopup="dialog" onclick={onsubscribe}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 11a9 9 0 0 1 9 9" /><path d="M4 4a16 16 0 0 1 16 16" /><circle cx="5" cy="19" r="1.4" fill="currentColor" /></svg>
      <span>{t("subscribe_btn")}</span>
    </button>
  </div>
</div>

<style>
  .page-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 16px;
    margin-bottom: 28px;
  }
  .actions { display: inline-flex; align-items: center; gap: 12px; min-width: 0; }

  .brand {
    display: inline-flex;
    align-items: baseline;
    font-size: 23px;
    color: var(--text);
    /* The wordmark keeps its own face. */
    font-family: "Instrument Sans", var(--font-sans);
  }
  .brand:hover { text-decoration: none; }
  .badge {
    display: inline-flex;
    align-items: center;
    width: 37px;
    height: 25px;
    color: var(--brand-ink);
    transform: translateY(4px);
  }
  .badge svg { width: 100%; height: 100%; }
  .name { font-weight: 600; letter-spacing: -0.01em; }
  .italic { font-style: italic; font-size: 26px; color: var(--brand-ink); margin-right: -2px; }

  .pills {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 2px;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    box-shadow: var(--shadow-sm);
  }
  .pills a {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 26px;
    padding: 3px 10px;
    border-radius: 999px;
    color: var(--text-faint);
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    transition: color 0.15s ease, background 0.15s ease;
  }
  .pills a:hover, .pills a[aria-current="page"] {
    color: var(--text);
    background: var(--surface-2);
    text-decoration: none;
  }

  .subscribe {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    min-height: 32px;
    padding: 2px 12px;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    box-shadow: var(--shadow-sm);
    color: var(--text);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: border-color 0.15s ease, background 0.15s ease;
  }
  .subscribe svg { width: 13px; height: 13px; color: var(--brand-ink); }
  .subscribe:hover { background: var(--surface-2); border-color: var(--border-strong); }

  @media (max-width: 640px) {
    .page-top { align-items: flex-start; gap: 12px; }
    .actions { gap: 8px; }
    .pills a { padding: 3px 8px; font-size: 11.5px; }
    .subscribe { width: 32px; padding: 2px; }
    .subscribe span { display: none; }
    .subscribe svg { width: 15px; height: 15px; }
  }
</style>
