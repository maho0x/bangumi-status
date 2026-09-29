<script lang="ts">
  import { live, onReactionBurst, react, REACTION_IDS } from "../lib/live.svelte";
  import { t } from "../lib/i18n.svelte";

  let open = $state(false);
  let root = $state<HTMLDivElement>();
  let trigger = $state<HTMLAnchorElement>();
  const chips: Record<number, HTMLElement> = {};
  const lastClick = new Map<number, number>();

  // Floating emoji, launched from the reaction's chip (or the trigger).
  type Float = { key: number; id: number; x: number; y: number; rot: number };
  let floats = $state<Float[]>([]);
  let floatKey = 0;

  function float(id: number, anchor?: HTMLElement | null) {
    const el = anchor ?? chips[id] ?? trigger;
    const r = el?.getBoundingClientRect();
    if (!r || (!r.width && !r.height)) return;
    floats.push({
      key: ++floatKey,
      id,
      x: r.left + r.width / 2 - 15 + (Math.random() - 0.5) * 28,
      y: r.top - 20,
      rot: (Math.random() - 0.5) * 40,
    });
  }

  // Reactions from other visitors float up too (at most 4 per update).
  $effect(() =>
    onReactionBurst((id, n) => {
      for (let i = 0; i < Math.min(n, 4); i++) setTimeout(() => float(id), i * 80);
    }),
  );

  const byID = $derived(new Map(live.reactions.map((r) => [r.emoji_id, r])));
  const active = $derived(REACTION_IDS.filter((id) => (byID.get(id)?.count ?? 0) > 0));

  async function add(id: number, e: MouseEvent) {
    e.stopPropagation();
    const now = Date.now();
    if ((lastClick.get(id) ?? 0) + 100 > now) return;
    lastClick.set(id, now);
    const anchor = e.currentTarget as HTMLElement;
    float(id, anchor);
    if (!(await react(id))) {
      anchor.classList.add("shake");
      anchor.addEventListener("animationend", () => anchor.classList.remove("shake"), { once: true });
    }
  }
</script>

<svelte:document
  onclick={(e) => open && !root?.contains(e.target as Node) && (open = false)}
  onkeydown={(e) => e.key === "Escape" && (open = false)}
/>

<div class="reactions">
  <div class="chips">
    {#each active as id (id)}
      {@const r = byID.get(id)!}
      <button type="button" class="chip" class:mine={r.mine} title={t("react_again")} bind:this={chips[id]} onclick={(e) => add(id, e)}>
        <span class="emoji" style:background-image="url('/smiles/{id}.gif')"></span>
        <span class="count">{r.count}</span>
      </button>
    {/each}
  </div>
  <div class="dropdown" class:open bind:this={root}>
    <a href="#react" class="trigger" role="button" aria-haspopup="true" aria-expanded={open} bind:this={trigger}
      onclick={(e) => { e.preventDefault(); e.stopPropagation(); open = !open; }}>
      <span class="heart" aria-hidden="true"></span>
      <span class="title">{t("react")}</span>
    </a>
    <ul class="grid" role="menu">
      {#each REACTION_IDS as id, i (id)}
        <li style:animation-delay="{0.02 + (i % 4) * 0.02 + Math.floor(i / 4) * 0.03}s">
          <button type="button" role="menuitem" class:mine={byID.get(id)?.mine} title={t("react")} onclick={(e) => add(id, e)}>
            <img src="/smiles/{id}.gif" alt="" loading="lazy" decoding="async" />
          </button>
        </li>
      {/each}
    </ul>
  </div>
</div>

{#each floats as f (f.key)}
  <img class="float" src="/smiles/{f.id}.gif" alt="" style:left="{f.x}px" style:top="{f.y}px" style:--rot="{f.rot}deg"
    onanimationend={() => (floats = floats.filter((x) => x.key !== f.key))} />
{/each}

<style>
  .reactions {
    position: relative;
    z-index: 10;
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 10px;
    margin: -20px 0 16px;
  }
  .chips { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; }
  .chip {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 3px 9px 3px 8px;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--text-dim);
    font-size: 11.5px;
    line-height: 1;
    cursor: pointer;
    transition: color 0.18s ease, background 0.18s ease, border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
  }
  .chip:hover {
    color: var(--text);
    border-color: var(--border-strong);
    background: var(--surface-2);
    transform: translateY(-1px);
    box-shadow: var(--shadow-sm);
  }
  .chip.mine {
    color: var(--brand-ink);
    border-color: color-mix(in srgb, var(--brand) 60%, var(--border));
    background: color-mix(in srgb, var(--brand) 10%, var(--surface));
  }
  .emoji {
    flex-shrink: 0;
    width: 16px;
    height: 16px;
    background: no-repeat 50% 50% / 100% auto;
    image-rendering: pixelated;
    transition: transform 0.2s var(--ease-out);
  }
  .chip:hover .emoji { transform: scale(1.12) rotate(-4deg); }
  .count {
    font-style: italic;
    font-size: 13px;
    line-height: 1;
    font-variant-numeric: tabular-nums;
    color: var(--text-faint);
    transition: color 0.18s ease;
  }
  .chip.mine .count { color: var(--brand-ink); }

  .dropdown { position: relative; }
  .trigger {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 3px 11px 3px 9px;
    border: 1px solid transparent;
    border-radius: 999px;
    color: var(--text-faint);
    font-size: 11px;
    font-weight: 500;
    letter-spacing: 0.04em;
    white-space: nowrap;
    user-select: none;
    cursor: pointer;
    transition: color 0.18s ease, background 0.18s ease, border-color 0.18s ease;
  }
  .trigger:hover, .open .trigger {
    color: var(--text);
    background: var(--surface);
    border-color: var(--border);
    text-decoration: none;
  }
  .title { font-style: italic; font-size: 13px; letter-spacing: 0; }
  .heart {
    flex-shrink: 0;
    width: 12px;
    height: 11px;
    background: no-repeat center / contain url("data:image/svg+xml;charset=UTF-8,%3Csvg width='14' height='12' viewBox='0 0 14 12' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M12.0307 1.69986C11.4024 1.07447 10.5705 0.732655 9.68264 0.732655C8.79479 0.732655 7.96036 1.077 7.332 1.70239L7.00382 2.02901L6.67056 1.69733C6.0422 1.07194 5.20523 0.72506 4.31738 0.72506C3.43207 0.72506 2.59764 1.0694 1.97182 1.69226C1.34346 2.31766 0.997478 3.14814 1.00002 4.03179C1.00002 4.91544 1.34855 5.74339 1.97691 6.36878L6.75451 11.1238C6.82066 11.1896 6.9097 11.2251 6.99619 11.2251C7.08269 11.2251 7.17173 11.1921 7.23787 11.1263L12.0256 6.37891C12.654 5.75351 13 4.92303 13 4.03938C13.0025 3.15573 12.6591 2.32525 12.0307 1.69986ZM11.5423 5.8953L6.99619 10.4022L2.46027 5.88771C1.96165 5.39145 1.6869 4.73314 1.6869 4.03179C1.6869 3.33044 1.9591 2.67213 2.45772 2.1784C2.9538 1.68467 3.61524 1.41122 4.31738 1.41122C5.02206 1.41122 5.68604 1.68467 6.18466 2.18093L6.7596 2.75315C6.89443 2.88735 7.11067 2.88735 7.2455 2.75315L7.81535 2.186C8.31397 1.68973 8.97795 1.41628 9.68009 1.41628C10.3822 1.41628 11.0437 1.68973 11.5423 2.18346C12.0409 2.67973 12.3131 3.33803 12.3131 4.03938C12.3157 4.74073 12.0409 5.39904 11.5423 5.8953Z' fill='currentColor'/%3E%3C/svg%3E");
    opacity: 0.7;
    transition: opacity 0.18s ease, transform 0.25s var(--ease-out);
  }
  .trigger:hover .heart, .open .heart { opacity: 1; transform: scale(1.08); }

  /* Picker: 4 columns, opens upward. */
  .grid {
    position: absolute;
    right: 0;
    bottom: calc(100% + 6px);
    z-index: 100;
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 2px;
    min-width: 168px;
    margin: 0;
    padding: 6px;
    list-style: none;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: var(--shadow-md);
    visibility: hidden;
    opacity: 0;
    transform: translateY(4px) scale(0.98);
    transform-origin: bottom right;
    pointer-events: none;
    transition: opacity 0.15s ease, transform 0.18s var(--ease-out), visibility 0s 0.18s;
  }
  .open .grid {
    visibility: visible;
    opacity: 1;
    transform: none;
    pointer-events: auto;
    transition: opacity 0.15s ease, transform 0.18s var(--ease-out), visibility 0s;
  }
  li { opacity: 0; transform: translateY(2px); }
  .open li { animation: item-in 0.2s var(--ease-out) forwards; }
  @keyframes item-in { to { opacity: 1; transform: none; } }
  .grid button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    padding: 8px;
    border: 0;
    border-radius: var(--radius-sm);
    background: none;
    cursor: pointer;
    transition: background 0.15s ease, transform 0.1s ease;
  }
  .grid button:hover { background: var(--surface-2); transform: scale(1.12); }
  .grid button:active { transform: scale(0.94); }
  .grid button.mine {
    background: color-mix(in srgb, var(--brand) 14%, var(--surface));
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--brand) 40%, transparent);
  }
  .grid img { display: block; width: 22px; height: 22px; image-rendering: pixelated; }

  :global(.shake) { animation: shake 0.3s ease; }
  @keyframes shake {
    25% { transform: translateX(-4px); }
    75% { transform: translateX(4px); }
  }

  .float {
    position: fixed;
    z-index: 9999;
    width: 30px;
    height: 30px;
    pointer-events: none;
    image-rendering: pixelated;
    animation: float-up 0.9s var(--ease-out) forwards;
  }
  @keyframes float-up {
    0% { opacity: 1; transform: translateY(0) scale(1.5); }
    30% { opacity: 1; transform: translateY(-24px) scale(1.2) rotate(calc(var(--rot) * 0.5)); }
    100% { opacity: 0; transform: translateY(-72px) scale(0.5) rotate(var(--rot)); }
  }
</style>
