<script lang="ts">
  import Header from "./components/Header.svelte";
  import Footer from "./components/Footer.svelte";
  import SubscribeModal from "./components/SubscribeModal.svelte";
  import Tooltip from "./components/Tooltip.svelte";
  import Status from "./pages/Status.svelte";
  import History from "./pages/History.svelte";

  // Three paths share this app; the server answers every unknown path with it.
  // There is no client-side navigation, so the route is read once.
  type Route = "status" | "stats" | "history";
  const route: Route = (
    { "/stats": "stats", "/history": "history" } as Record<string, Route>
  )[location.pathname.replace(/\/+$/, "")] ?? "status";

  let subscribing = $state(false);
</script>

<main class="container" class:wide={route === "stats"}>
  <Header {route} onsubscribe={() => (subscribing = true)} />
  {#if route === "stats"}
    {#await import("./pages/Stats.svelte") then { default: Stats }}
      <Stats />
    {/await}
  {:else if route === "history"}
    <History />
  {:else}
    <Status />
  {/if}
  <Footer />
</main>

<Tooltip />
{#if subscribing}
  <SubscribeModal onclose={() => (subscribing = false)} />
{/if}

<style>
  .container {
    max-width: 800px;
    margin: 0 auto;
    padding: 40px 24px 80px;
  }
  .wide { max-width: 1180px; }
  @media (max-width: 960px) {
    .wide { max-width: 800px; }
  }
  @media (max-width: 640px) {
    .container { padding-left: 18px; padding-right: 18px; }
  }
</style>
