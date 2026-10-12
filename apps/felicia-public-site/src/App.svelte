<script lang="ts">
  import { Reader, resolveLocale, themeFromId, type ApiSiteSettings, type Lang, type Theme } from "@felicia/reader"
  import { loadJourneys, loadSiteSettings } from "./api/source"

  let settings = $state<ApiSiteSettings | null>(null)

  $effect(() => {
    loadSiteSettings()
      .then((s) => (settings = s))
      .catch(() => {
        // Fall back to defaults if settings unavailable
      })
  })

  const storedLocale = localStorage.getItem("felicia.locale")
  let lang: Lang = $state(resolveLocale(storedLocale ?? navigator.language))
  let theme: Theme = $state("dark")
  let appearanceEdited = $state(false)

  $effect(() => localStorage.setItem("felicia.locale", lang))

  $effect(() => {
    if (!settings) return
    if (!storedLocale) {
      lang = settings.default_language
    }
    if (!appearanceEdited) theme = themeFromId(settings.default_theme).id

    if (/^#[0-9a-fA-F]{6}$/.test(settings.accent)) {
      document.documentElement.style.setProperty("--accent", settings.accent)
    }
  })
</script>

<!-- Dialog key events do not cross the reader iframe's origin boundary. -->
<svelte:window
  onkeydown={(event) => {
    if (event.key === "Escape" && !event.defaultPrevented && window.parent !== window) window.parent.postMessage("felicia:preview:escape", "*")
  }}
/>

<div class="public-reader-shell" class:theme-light={theme === "light"}>
  <Reader bind:lang bind:theme {loadJourneys} />
</div>

<style>
  .public-reader-shell {
    position: relative;
    width: 100%;
    height: 100%;
    color-scheme: dark;
  }

  .public-reader-shell.theme-light {
    color-scheme: light;
  }
</style>
