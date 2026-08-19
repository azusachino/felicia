<script lang="ts">
  import { parseRoute, listHash, siteHash } from "./router"
  import JourneyList from "./views/JourneyList.svelte"
  import JourneyDetail from "./views/JourneyDetail.svelte"
  import MementoEditor from "./views/MementoEditor.svelte"
  import SiteDeploy from "./views/SiteDeploy.svelte"

  let hash = $state(location.hash)
  const route = $derived(parseRoute(hash))
</script>

<svelte:window on:hashchange={() => (hash = location.hash)} />

<svelte:head>
  <title>Felicia Admin · Journeys</title>
</svelte:head>

<div class="admin-shell">
  <aside class="sidebar">
    <div class="brand">
      <span class="brand-mark">F</span>
      <span>felicia</span>
    </div>
    <p class="eyebrow">Authoring workspace</p>
    <nav aria-label="Admin navigation">
      <a class:active={route.name !== "site"} href={listHash}>Journeys</a>
      <a class:active={route.name === "site"} href={siteHash}>Site &amp; Deploy</a>
    </nav>
    <div class="sidebar-footer">
      <span class="status-dot"></span>
      Local workspace
    </div>
  </aside>

  <main class="content">
    <header class="topbar">
      <div>
        <p class="eyebrow">Felicia</p>
      </div>
      <button class="profile" type="button" aria-label="Open profile menu">YP</button>
    </header>

    {#if route.name === "detail"}
      {#key route.id}
        <JourneyDetail id={route.id} />
      {/key}
    {:else if route.name === "memento"}
      {#key `${route.journeyId}/${route.id}`}
        <MementoEditor journeyId={route.journeyId} id={route.id} />
      {/key}
    {:else if route.name === "site"}
      <SiteDeploy />
    {:else}
      <JourneyList />
    {/if}
  </main>
</div>
