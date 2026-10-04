<script lang="ts">
  import { setContext, type Snippet } from "svelte"
  import { page } from "$app/state"
  import { resolve } from "$app/paths"
  import { STUDIO_CONTEXT, type StudioState } from "./studio"
  import { loadLocale, message, saveLocale, type Locale } from "../i18n"
  import * as Popover from "$lib/components/ui/popover"
  import { setMode, userPrefersMode } from "mode-watcher"
  import { Tooltip } from "bits-ui"
  import { Settings2 } from "@lucide/svelte"

  let { children }: { children: Snippet } = $props()

  let locale = $state(loadLocale())
  let settingsTrigger = $state<HTMLButtonElement | null>(null)
  let settingsOpen = $state(false)
  let settingsHintOpen = $state(false)
  const library = $derived(page.route.id === "/" || page.route.id === "/[...missing]")
  const site = $derived(page.route.id === "/site")
  // Wails beta.24 identifies its macOS webview with this user-agent suffix.
  const desktop = navigator.userAgent.includes("wails.io")
  const nativeMac = desktop && navigator.userAgent.includes("Macintosh")
  setContext<StudioState>(STUDIO_CONTEXT, {
    get locale() {
      return locale
    },
    desktop,
  })

  function changeLocale(event: Event) {
    locale = (event.currentTarget as HTMLSelectElement).value as Locale
    saveLocale(locale)
  }
  $effect(() => {
    document.documentElement.lang = locale
  })
</script>

<svelte:head><title>{message(locale, "admin.shell.page_title")}</title></svelte:head>

<div class="admin-shell" class:native-mac={nativeMac}>
  <header class="window-toolbar">
    <div class="brand"><img src="/felicia-mark.svg" alt="Felicia" width="24" height="24" /><span>Felicia</span></div>
  </header>
  <aside class="sidebar">
    <nav aria-label={message(locale, "admin.shell.navigation_label")}>
      <a class:active={!site} aria-current={!site ? "page" : undefined} href={resolve("/")}>
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m3 5 6-2 6 2 6-2v16l-6 2-6-2-6 2V5ZM9 3v16M15 5v16" /></svg>
        {message(locale, "admin.journeys.title")}
      </a>
      <a class:active={site} aria-current={site ? "page" : undefined} href={resolve("/site")}>
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 8h16v12H4V8ZM12 3v11m-4-7 4-4 4 4" /></svg>
        {message(locale, "admin.site.navigation")}
      </a>
    </nav>
    <div class="sidebar-footer">
      <Popover.Root
        bind:open={settingsOpen}
        onOpenChange={(open) => {
          if (open) settingsHintOpen = false
        }}
      >
        <Tooltip.Root bind:open={settingsHintOpen} disabled={settingsOpen}>
          <Tooltip.Trigger>
            {#snippet child({ props })}
              <Popover.Trigger {...props} bind:ref={settingsTrigger} class="settings-trigger" aria-label={message(locale, "admin.shell.settings")}>
                <Settings2 size={18} aria-hidden="true" />
              </Popover.Trigger>
            {/snippet}
          </Tooltip.Trigger>
          {#if settingsHintOpen && !settingsOpen}
            <Tooltip.Portal>
              <Tooltip.Content role="tooltip" side="top" sideOffset={6} class="studio-tooltip">{message(locale, "admin.shell.settings")}</Tooltip.Content>
            </Tooltip.Portal>
          {/if}
        </Tooltip.Root>
        <Popover.Content
          side="top"
          align="start"
          role="dialog"
          class="settings-panel w-64"
          aria-label={message(locale, "admin.shell.settings")}
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            settingsTrigger?.focus()
          }}
        >
          <label class="locale-control">
            {message(locale, "admin.shell.language_label")}
            <select aria-label={message(locale, "admin.shell.language_label")} value={locale} onchange={changeLocale}>
              <option value="ja">日本語</option><option value="en">English</option><option value="zh">中文</option>
            </select>
          </label>
          <label class="locale-control">
            {message(locale, "admin.shell.appearance")}
            <select aria-label={message(locale, "admin.shell.appearance")} value={userPrefersMode.current} onchange={(event) => setMode(event.currentTarget.value as "system" | "light" | "dark")}>
              <option value="system">{message(locale, "admin.common.theme_system")}</option>
              <option value="light">{message(locale, "admin.common.theme_light")}</option>
              <option value="dark">{message(locale, "admin.common.theme_dark")}</option>
            </select>
          </label>
        </Popover.Content>
      </Popover.Root>
    </div>
  </aside>
  <main class="content" class:library>
    {@render children()}
  </main>
</div>
