<script lang="ts">
  let { status }: { status: string } = $props();
</script>

<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  {#if status === "ok"}
    <polyline points="5 12.5 10 17.5 19 7.5" />
  {:else if status === "down"}
    <line x1="6.5" y1="6.5" x2="17.5" y2="17.5" /><line x1="17.5" y1="6.5" x2="6.5" y2="17.5" />
  {:else if status === "degraded"}
    <line x1="12" y1="6" x2="12" y2="13" /><circle cx="12" cy="17.5" r="0.2" stroke-width="3.2" />
  {:else}
    <circle cx="6" cy="12" r="1.2" /><circle cx="12" cy="12" r="1.2" /><circle cx="18" cy="12" r="1.2" />
  {/if}
</svg>
