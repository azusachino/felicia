<script lang="ts">
  import { Reader, resolveLocale, themeFromId, type ApiSiteSettings, type Lang, type Theme } from "@felicia/reader"
  import { loadJourneys, loadSiteSettings } from "./api/source"

  let settings = $state<ApiSiteSettings | null>(null)

  $effect(() => {
    loadSiteSettings()
      .then((value) => (settings = value))
      .catch(() => {
        // The private reader keeps its default composition until the API is ready.
      })
  })

  const storedLocale = localStorage.getItem("felicia.locale")
  let lang: Lang = $state(resolveLocale(storedLocale ?? navigator.language))
  let theme: Theme = $state("dark")

  $effect(() => localStorage.setItem("felicia.locale", lang))

  $effect(() => {
    if (!settings) return
    if (!storedLocale) lang = settings.default_language
    theme = themeFromId(settings.default_theme).id
    if (/^#[0-9a-fA-F]{6}$/.test(settings.accent)) {
      document.documentElement.style.setProperty("--accent", settings.accent)
    }
  })
</script>

<Reader bind:lang bind:theme {loadJourneys} />
