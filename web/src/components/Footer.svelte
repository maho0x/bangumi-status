<script lang="ts">
  import { locale, t, toggleLang } from "../lib/i18n.svelte";
</script>

<footer>
  <p class="desc">{t("footer_desc")}</p>
  <div class="right">
    <p class="meta">
      <span>{t("auto_refresh")}</span>
      <span aria-hidden="true">·</span>
      <a href="/api/feed.atom" class="link">{t("atom_feed")}</a>
      <span aria-hidden="true">·</span>
      <a href="https://github.com/maho0x/bangumi-status" class="link" target="_blank" rel="noopener">GitHub</a>
    </p>
    <button class="lang" type="button" aria-label="Switch language" onclick={toggleLang}>
      {locale.lang === "zh" ? "EN" : "中文"}
    </button>
  </div>
</footer>

<style>
  footer {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 18px;
    margin-top: 56px;
    padding-top: 24px;
    border-top: 1px solid var(--border);
  }
  .right { display: flex; align-items: center; flex-wrap: wrap; gap: 14px; }
  .desc { margin: 0; max-width: 540px; font-size: 12.5px; color: var(--text-dim); }
  .meta {
    display: inline-flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 10px;
    margin: 0;
    color: var(--text-faint);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .link { color: var(--text-dim); border-bottom: 1px dotted color-mix(in srgb, var(--text-dim) 45%, transparent); }
  .link:hover { color: var(--brand-ink); border-bottom-color: var(--brand); text-decoration: none; }
  .lang {
    padding: 5px 11px;
    border: 1px solid var(--border-strong);
    border-radius: 999px;
    background: transparent;
    color: var(--text-dim);
    font-size: 12px;
    font-weight: 500;
    letter-spacing: 0.03em;
    cursor: pointer;
    transition: border-color 0.15s ease, color 0.15s ease;
  }
  .lang:hover { border-color: var(--brand); color: var(--brand-ink); }
</style>
