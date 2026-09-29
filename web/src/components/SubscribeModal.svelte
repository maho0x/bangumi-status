<script lang="ts">
  import { fade } from "svelte/transition";
  import { t } from "../lib/i18n.svelte";

  let { onclose }: { onclose: () => void } = $props();

  const feedURL = location.origin + "/api/feed.atom";
  let copied = $state(false);
  let input = $state<HTMLInputElement>();
  let closeBtn = $state<HTMLButtonElement>();

  $effect(() => {
    closeBtn?.focus({ preventScroll: true });
    document.body.style.overflow = "hidden";
    return () => (document.body.style.overflow = "");
  });

  async function copy() {
    try {
      await navigator.clipboard.writeText(feedURL);
    } catch {
      input?.select();
    }
    copied = true;
    setTimeout(() => (copied = false), 1600);
  }
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onclose()} />

<div class="modal" role="dialog" aria-modal="true" aria-labelledby="subscribe-title">
  <div class="backdrop" onclick={onclose} aria-hidden="true" transition:fade={{ duration: 180 }}></div>
  <div class="panel" role="document">
    <header>
      <h2 id="subscribe-title">{t("modal_title")}</h2>
      <button class="close" type="button" aria-label={t("close")} onclick={onclose} bind:this={closeBtn}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="6" y1="6" x2="18" y2="18" /><line x1="18" y1="6" x2="6" y2="18" /></svg>
      </button>
    </header>
    <p class="intro">{t("modal_intro")}</p>

    <div class="option">
      <div class="row">
        <div class="icon" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 11a9 9 0 0 1 9 9" /><path d="M4 4a16 16 0 0 1 16 16" /><circle cx="5" cy="19" r="1.4" fill="currentColor" /></svg>
        </div>
        <div class="text">
          <div class="title">{t("sub_atom_title")}</div>
          <div class="desc">{t("sub_atom_desc")}</div>
        </div>
      </div>
      <div class="action">
        <input type="text" class="url" readonly value={feedURL} bind:this={input} />
        <button type="button" class="btn" class:copied onclick={copy}>{copied ? t("copied") : t("copy")}</button>
      </div>
    </div>

    <div class="option">
      <div class="row inline">
        <div class="icon" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 5L2 12.5l7 1M21 5l-2.5 15L9 13.5M21 5L9 13.5m0 0v5.5l3.5-3" /></svg>
        </div>
        <div class="text">
          <div class="title">{t("sub_tg_title")}</div>
          <div class="desc">{t("sub_tg_desc")}</div>
        </div>
        <a href="https://t.me/bgmmonitor" target="_blank" rel="noopener" class="btn">{t("sub_tg_go")}</a>
      </div>
    </div>

    <div class="option">
      <div class="row">
        <div class="icon" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></svg>
        </div>
        <div class="text">
          <div class="title">{t("sub_live_title")}</div>
          <div class="desc">{t("sub_live_desc")}</div>
        </div>
      </div>
    </div>

    <p class="foot">{t("modal_foot")}</p>
  </div>
</div>

<style>
  .modal {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
  }
  .backdrop {
    position: absolute;
    inset: 0;
    background: rgba(20, 19, 16, 0.45);
    backdrop-filter: blur(3px);
    -webkit-backdrop-filter: blur(3px);
  }
  @media (prefers-color-scheme: dark) {
    .backdrop { background: rgba(0, 0, 0, 0.6); }
  }
  .panel {
    position: relative;
    width: 100%;
    max-width: 480px;
    padding: 22px 24px 24px;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: 0 24px 80px -20px rgba(26,24,20,0.25), 0 4px 12px -4px rgba(26,24,20,0.1);
    animation: modal-in 0.18s var(--ease-out);
  }
  @keyframes modal-in {
    from { opacity: 0; transform: translateY(8px) scale(0.98); }
  }
  header { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-bottom: 8px; }
  h2 { font-size: 18px; letter-spacing: -0.015em; }
  .close {
    display: inline-flex;
    padding: 6px;
    border: none;
    border-radius: 6px;
    background: none;
    color: var(--text-faint);
    cursor: pointer;
    transition: color 0.15s ease, background 0.15s ease;
  }
  .close svg { width: 14px; height: 14px; }
  .close:hover { color: var(--text); background: var(--surface-2); }
  .intro { margin: 0 0 18px; color: var(--text-dim); font-size: 13.5px; }
  .foot {
    margin: 16px 0 0;
    padding-top: 14px;
    border-top: 1px solid var(--border);
    color: var(--text-faint);
    font-size: 11.5px;
  }

  .option {
    padding: 12px 14px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    transition: border-color 0.15s ease;
  }
  .option + .option { margin-top: 12px; }
  .option:hover { border-color: var(--border-strong); }
  .row { display: flex; align-items: flex-start; gap: 12px; }
  .row.inline { align-items: center; }
  .row.inline .btn { flex-shrink: 0; }
  .icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 30px;
    height: 30px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--surface);
    color: var(--brand-ink);
  }
  .icon svg { width: 15px; height: 15px; }
  .text { flex: 1; min-width: 0; }
  .title { font-weight: 500; font-size: 13.5px; }
  .desc { margin-top: 3px; font-size: 12px; color: var(--text-dim); }
  .action { display: flex; align-items: stretch; gap: 8px; margin-top: 10px; }
  .url {
    flex: 1;
    min-width: 0;
    padding: 7px 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 11.5px;
    letter-spacing: -0.01em;
  }
  .url:focus { outline: 2px solid color-mix(in srgb, var(--brand) 60%, transparent); outline-offset: 1px; border-color: var(--brand); }
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 32px;
    padding: 4px 16px;
    border: 1px solid var(--text);
    border-radius: 999px;
    background: var(--text);
    color: var(--surface);
    font-size: 12px;
    font-weight: 600;
    white-space: nowrap;
    cursor: pointer;
    transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;
  }
  .btn:hover { background: var(--brand-ink); border-color: var(--brand-ink); color: #fff; text-decoration: none; }
  .btn.copied { background: var(--ok); border-color: var(--ok); color: #fff; }
</style>
