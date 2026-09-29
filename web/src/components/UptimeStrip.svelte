<script lang="ts">
  import { tooltip } from "../lib/tooltip.svelte";
  import { isoDayLocal, nowSec } from "../lib/format";
  import type { DayBucket } from "../lib/api-types";

  let { days }: { days: DayBucket[] } = $props();

  // Pad with empty cells up to the viewer's today when the server's (CST)
  // calendar is behind the local one.
  const cells = $derived.by(() => {
    const out = days.slice();
    const today = isoDayLocal(nowSec());
    let last = out.at(-1)?.day;
    while (last && last < today) {
      const [y, m, d] = last.split("-").map(Number);
      last = isoDayLocal(new Date(y, m - 1, d + 1).getTime() / 1000);
      out.push({ day: last, total: 0, uptime: 0, down: 0, degrade: 0, status: "ok" });
    }
    return out;
  });
</script>

<div class="strip">
  {#each cells as d (d.day)}
    <div class="cell cell-{d.total ? d.status || 'none' : 'none'}" use:tooltip={() => ({ day: d })}></div>
  {/each}
</div>

<style>
  .strip { display: flex; flex: 1; align-items: stretch; gap: 1.5px; min-width: 0; height: 20px; }
  .cell {
    flex: 1 1 0;
    min-width: 0;
    border-radius: 2px;
    background: var(--none);
    cursor: help;
    transition: background 200ms var(--ease-out), filter 0.08s ease;
  }
  .cell:hover { filter: brightness(1.12); }
  .cell-ok { background: var(--ok); }
  .cell-degraded { background: var(--degraded); }
  .cell-down { background: var(--down); }
  .cell-none { background: var(--none-soft); border: 1px solid var(--border); }
  @media (max-width: 720px) {
    .strip { flex: 1 1 100%; }
  }
</style>
