<script lang="ts">
  import { parseRoute, listHash, siteHash } from "./router"
  import JourneyList from "./views/JourneyList.svelte"
  import JourneyDetail from "./views/JourneyDetail.svelte"
  import MementoEditor from "./views/MementoEditor.svelte"
  import SiteDeploy from "./views/SiteDeploy.svelte"
  import { loadLocale, message, saveLocale, type Locale } from "./i18n"

  let locale = $state(loadLocale())
  let hash = $state(location.hash)

  function changeLocale(event: Event) {
    locale = (event.currentTarget as HTMLSelectElement).value as Locale
    saveLocale(locale)
  }
  const route = $derived(parseRoute(hash))

  $effect(() => {
    document.documentElement.lang = locale
  })
</script>

<svelte:window on:hashchange={() => (hash = location.hash)} />

<svelte:head>
  <title>{message(locale, "admin.shell.page_title")}</title>
</svelte:head>

<div class="admin-shell">
  <aside class="sidebar">
    <div class="brand">
      <span class="brand-mark">F</span>
      <span>felicia</span>
    </div>
    <p class="eyebrow">{message(locale, "admin.shell.workspace_label")}</p>
    <nav aria-label={message(locale, "admin.shell.navigation_label")}>
      <a class:active={route.name !== "site"} aria-current={route.name !== "site" ? "page" : undefined} href={listHash}>{message(locale, "admin.journeys.title")}</a>
      <a class:active={route.name === "site"} aria-current={route.name === "site" ? "page" : undefined} href={siteHash}>{message(locale, "admin.site.navigation")}</a>
    </nav>
    <div class="sidebar-footer">
      <span class="status-dot"></span>
      {message(locale, "admin.shell.local_workspace")}
    </div>
  </aside>

  <main class="content">
    <header class="topbar">
      <div>
        <p class="eyebrow">Felicia</p>
      </div>
      <label class="locale-control">
        <span class="sr-only">{message(locale, "admin.shell.language_label")}</span>
        <select aria-label={message(locale, "admin.shell.language_label")} value={locale} onchange={changeLocale}>
          <option value="ja">日本語</option>
          <option value="en">English</option>
          <option value="zh">中文</option>
        </select>
      </label>
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
        <JourneyDetail id={route.id} {locale} />
      {/key}
    {:else if route.name === "memento"}
      {#key `${route.journeyId}/${route.id}`}
        <MementoEditor journeyId={route.journeyId} id={route.id} {locale} />
      {/key}
    {:else if route.name === "site"}
      <SiteDeploy {locale} />
    {:else}
      <JourneyList {locale} />
    {/if}
  </main>
</div>
