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
      <a class:active={route.name !== "site"} aria-current={route.name !== "site" ? "page" : undefined} href={listHash}>Journeys</a>
      <a class:active={route.name === "site"} aria-current={route.name === "site" ? "page" : undefined} href={siteHash}>Site &amp; Deploy</a>
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
      <!--
        Not a button: felicia is single-user and local-only (no accounts, no
        auth — see docs/roadmap/admin-gui-v2-epic.md's "no credentials inside
        felicia" constraint), so there is no profile menu for this to open.
        It previously rendered as an interactive-looking `aria-label="Open
        profile menu"` button with no click handler — a control that promised
        a menu and did nothing, including for screen-reader users who heard
        the label. This is the honest version: a plain identity mark.
      -->
      <span class="profile" aria-hidden="true">YP</span>
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
