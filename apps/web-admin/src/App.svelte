<script lang="ts">
  import { onMount } from "svelte"
  import { loadAdminOverview, type AdminOverview } from "./api"

  const navItems = ["Overview", "Inbox", "Journeys", "Mementos", "Settings"]
  let active = $state("Overview")
  let overview = $state<AdminOverview>({ journeys: [], mementos: [] })
  let loading = $state(true)
  let error = $state("")

  onMount(async () => {
    try {
      overview = await loadAdminOverview()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "Unable to load the local workspace"
    } finally {
      loading = false
    }
  })
</script>

<svelte:head>
  <title>Felicia Admin · {active}</title>
</svelte:head>

<div class="admin-shell">
  <aside class="sidebar">
    <div class="brand">
      <span class="brand-mark">F</span>
      <span>felicia</span>
    </div>
    <p class="eyebrow">Authoring workspace</p>
    <nav aria-label="Admin navigation">
      {#each navItems as item (item)}
        <button class:active={active === item} type="button" onclick={() => (active = item)}>
          {item}
        </button>
      {/each}
    </nav>
    <div class="sidebar-footer">
      <span class="status-dot"></span>
      Local workspace
    </div>
  </aside>

  <main class="content">
    <header class="topbar">
      <div>
        <p class="eyebrow">Felicia / {active}</p>
        <h1>{active}</h1>
      </div>
      <button class="profile" type="button" aria-label="Open profile menu">YP</button>
    </header>

    {#if active === "Overview"}
      <section class="welcome">
        <div>
          <p class="eyebrow">Personal travel journal</p>
          <h2>Turn raw memories into a journey.</h2>
          <p class="lede">Import a package, review the detected places, and curate the story before publishing.</p>
        </div>
        <button class="primary" type="button" onclick={() => (active = "Inbox")}>Open inbox</button>
      </section>

      <section class="cards" aria-label="Workspace summary">
        <article><span class="card-label">Inbox</span><strong>0</strong><span class="card-note">packages to review</span></article>
        <article><span class="card-label">Journeys</span><strong>{loading ? "—" : overview.journeys.length}</strong><span class="card-note">from the admin API</span></article>
        <article>
          <span class="card-label">Draft mementos</span><strong>{loading ? "—" : overview.mementos.filter((memento) => ["candidate", "draft"].includes(memento.state)).length}</strong><span
            class="card-note">ready for curation</span
          >
        </article>
      </section>
      {#if error}
        <p class="api-error" role="alert">{error}. Start the local API to load authoring data.</p>
      {/if}
    {:else}
      <section class="empty-state">
        <span class="empty-icon">{active.slice(0, 1)}</span>
        <h2>{active} is ready for the next slice.</h2>
        <p>The admin shell is in place; its runtime data boundary will connect to the shared Felicia API.</p>
      </section>
    {/if}
  </main>
</div>
