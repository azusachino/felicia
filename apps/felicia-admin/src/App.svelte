<script lang="ts">
  import Router, { replace } from "svelte-spa-router"
  import { listHash, siteHash } from "./router"
  import { createRoutes } from "./routes"
  import { loadLocale, message, saveLocale, type Locale } from "./i18n"

  let locale = $state(loadLocale())
  let library = $state(true)
  let site = $state(false)
  let settingsOpen = $state(false)
  let settingsMenu: HTMLDivElement
  let settingsTrigger: HTMLButtonElement
  // Wails beta.24 identifies its macOS webview with this user-agent suffix.
  const desktop = navigator.userAgent.includes("wails.io")
  const nativeMac = desktop && navigator.userAgent.includes("Macintosh")
  const routes = createRoutes({
    get locale() {
      return locale
    },
    desktop,
  })

  function changeLocale(event: Event) {
    locale = (event.currentTarget as HTMLSelectElement).value as Locale
    saveLocale(locale)
  }
  function dismissSettingsOnEscape(event: KeyboardEvent) {
    if (event.key !== "Escape" || !settingsOpen) return
    settingsOpen = false
    settingsTrigger?.focus()
  }
  function dismissSettingsOutside(event: PointerEvent) {
    if (settingsOpen && settingsMenu && !settingsMenu.contains(event.target as Node)) settingsOpen = false
  }
  $effect(() => {
    document.documentElement.lang = locale
  })
</script>

<svelte:window on:keydown={dismissSettingsOnEscape} on:pointerdown={dismissSettingsOutside} />
<svelte:head><title>{message(locale, "admin.shell.page_title")}</title></svelte:head>

<div class="admin-shell" class:native-mac={nativeMac}>
  <header class="window-toolbar">
    <div class="brand"><img src="/felicia-mark.svg" alt="Felicia" width="24" height="24" /><span>Felicia</span></div>
  </header>
  <aside class="sidebar">
    <nav aria-label={message(locale, "admin.shell.navigation_label")}>
      <a class:active={!site} aria-current={!site ? "page" : undefined} href={listHash}>
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m3 5 6-2 6 2 6-2v16l-6 2-6-2-6 2V5ZM9 3v16M15 5v16" /></svg>
        {message(locale, "admin.journeys.title")}
      </a>
      <a class:active={site} aria-current={site ? "page" : undefined} href={siteHash}>
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 8h16v12H4V8ZM12 3v11m-4-7 4-4 4 4" /></svg>
        {message(locale, "admin.site.navigation")}
      </a>
    </nav>
    <div class="sidebar-footer">
      <div class="settings-menu" bind:this={settingsMenu}>
        <button bind:this={settingsTrigger} class="settings-trigger" type="button" aria-expanded={settingsOpen} aria-controls="settings-popover" onclick={() => (settingsOpen = !settingsOpen)}>
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16M4 17h16M8 4v6M16 14v6" /></svg>
          {message(locale, "admin.shell.settings")}
        </button>
        <div id="settings-popover" class="settings-popover" role="group" aria-label={message(locale, "admin.shell.settings")} hidden={!settingsOpen}>
          <label class="locale-control"
            >{message(locale, "admin.shell.language_label")}
            <select aria-label={message(locale, "admin.shell.language_label")} value={locale} onchange={changeLocale}>
              <option value="ja">日本語</option><option value="en">English</option><option value="zh">中文</option>
            </select>
          </label>
        </div>
      </div>
      <span class="workspace-status"><span class="status-dot"></span>{message(locale, "admin.shell.local_workspace")}</span>
    </div>
  </aside>
  <main class="content" class:library>
    <Router
      {routes}
      onConditionsFailed={() => replace(listHash)}
      onRouteLoaded={({ route }) => {
        library = route === "/" || route === "*"
        site = route === "/site"
      }}
    />
  </main>
</div>
