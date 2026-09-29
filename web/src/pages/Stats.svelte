<script lang="ts">
  import WikiChart from "../components/WikiChart.svelte";
  import { t } from "../lib/i18n.svelte";
  import { fmtDayLabel, fmtRelative } from "../lib/format";
  import type { WikiStatsPoint } from "../lib/api-types";

  type Payload = { scraped_at: number; data: WikiStatsPoint[] };
  let payload = $state<Payload | null>(null);
  let failed = $state("");

  async function load() {
    try {
      const res = await fetch("/api/wiki-stats", { cache: "no-store" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      payload = await res.json();
      failed = "";
    } catch (err) {
      failed = err instanceof Error ? err.message : String(err);
    }
  }
  load();

  const points = $derived((payload?.data ?? []).toSorted((a, b) => a.date.localeCompare(b.date)));
  const latest = $derived(points.at(-1));

  const charts: { title: "wiki_metric_register" | "wiki_metric_collection" | "wiki_metric_topics" | "wiki_metric_replies";
    hint: "wiki_hint_daily" | "wiki_hint_by_type" | "wiki_hint_by_source"; series: [string, keyof WikiStatsPoint][] }[] = [
    { title: "wiki_metric_register", hint: "wiki_hint_daily", series: [["注册用户", "register_total"]] },
    { title: "wiki_metric_collection", hint: "wiki_hint_by_type", series: [
      ["想看/读/听/玩", "collection_1"], ["看/读/听/玩过", "collection_2"], ["在看/读/听/玩", "collection_3"],
      ["搁置", "collection_4"], ["抛弃", "collection_5"]] },
    { title: "wiki_metric_topics", hint: "wiki_hint_by_source", series: [
      ["小组主题", "topic_1"], ["条目主题", "topic_2"], ["日志发布", "topic_7"]] },
    { title: "wiki_metric_replies", hint: "wiki_hint_by_source", series: [
      ["小组回复", "reply_1"], ["条目回复", "reply_2"], ["角色吐槽", "reply_3"], ["人物吐槽", "reply_4"],
      ["目录留言", "reply_5"], ["时间线回复", "reply_6"], ["日志回复", "reply_7"], ["章节讨论", "reply_8"]] },
  ];
  const totals = $derived([
    [latest?.register_total, "wiki_metric_register"],
    [latest?.collection_total, "wiki_metric_collection"],
    [latest?.topic_total, "wiki_metric_topics"],
    [latest?.reply_total, "wiki_metric_replies"],
  ] as const);
</script>

<svelte:document onvisibilitychange={() => !document.hidden && load()} />

<section aria-label={t("wiki_title")}>
  <header class="hero">
    <div>
      <h1>{t("wiki_title")}</h1>
      <p class="hero__intro">{t("wiki_intro")}</p>
    </div>
    <dl class="hero__meta">
      <div><dt>{t("wiki_recent_scrape")}</dt><dd class="num">{payload?.scraped_at ? fmtRelative(payload.scraped_at) : "—"}</dd></div>
      <div><dt>{t("wiki_data_day")}</dt><dd class="num">{latest ? fmtDayLabel(latest.date) : "—"}</dd></div>
    </dl>
  </header>

  {#if latest}
    <div class="summary">
      {#each totals as [value, label] (label)}
        <div class="item"><span class="num">{Number(value || 0).toLocaleString()}</span><span class="desc">{t(label)}</span></div>
      {/each}
    </div>
  {/if}
  {#if failed || (payload && !points.length)}
    <div class="notice">{failed || t("wiki_empty")}</div>
  {/if}

  <div class="charts">
    {#each charts as c (c.title)}
      <section>
        <header class="section-head">
          <h2>{t(c.title)}</h2>
          <span class="section-head__hint">{t(c.hint)}</span>
        </header>
        {#if points.length}
          <WikiChart {points} series={c.series} />
        {:else}
          <div class="empty">{payload || failed ? t("wiki_empty") : ""}</div>
        {/if}
      </section>
    {/each}
  </div>
</section>

<style>
  .summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 32px; }
  .item {
    min-width: 0;
    padding: 13px 14px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--surface);
    box-shadow: var(--shadow-sm);
  }
  .item .num { display: block; font-size: 21px; font-weight: 600; line-height: 1.1; }
  .item:nth-child(2) .num { color: #459f74; }
  .item:nth-child(3) .num { color: #4f84bf; }
  .item:nth-child(4) .num { color: #b77a3b; }
  .desc { display: block; margin-top: 5px; color: var(--text-faint); font-size: 12px; }
  .charts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 36px 24px; }
  .charts > section { min-width: 0; }
  .charts .section-head { margin-bottom: 10px; }
  .empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 560px;
    color: var(--text-faint);
    font-size: 12px;
    font-style: italic;
  }
  @media (max-width: 960px) {
    .charts { grid-template-columns: 1fr; }
  }
  @media (max-width: 720px) {
    .summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .empty { height: 520px; }
  }
  @media (max-width: 420px) {
    .summary { grid-template-columns: 1fr; }
  }
</style>
