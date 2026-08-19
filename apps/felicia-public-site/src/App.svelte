<script lang="ts">
  import { designs, resolveLocale, type ApiSiteSettings, type Lang, type Theme } from "@felicia/shared"
  import { loadJourneys, loadSiteSettings } from "./api/source"

  // The public reader is locked to a single design, chosen by the author from
  // the admin GUI (FELICIA-ADMIN-02 M2) and served as part of `/api/v1/site`.
  // The registry in designs.ts remains the source of truth for what designs
  // exist; this shell just resolves the configured one and renders it.
  let settings = $state<ApiSiteSettings | null>(null)

  $effect(() => {
    loadSiteSettings()
      .then((s) => (settings = s))
      .catch(() => {
        // Absent/unreachable settings = current demo behavior: fall back to
        // the default design (v1) and the existing lang/theme defaults below.
      })
  })

  const active = $derived(designs.find((d) => d.id === settings?.design) ?? designs[0])
  const Active = $derived(active.component)

  // lang/theme are shared across the mounted design so switching keeps your
  // reading state. lang keeps its existing localStorage override precedence
  // (captured before the persist effect below can write a fallback value
  // into storage); theme has no persistence, so it simply switches its
  // default source from a literal to the resolved site settings once they
  // load.
  const storedLocale = localStorage.getItem("felicia.locale")
  let lang: Lang = $state(resolveLocale(storedLocale ?? navigator.language))
  let theme: Theme = $state("dark")

  $effect(() => localStorage.setItem("felicia.locale", lang))

  $effect(() => {
    if (!settings) return
    if (!storedLocale) {
      lang = settings.default_language
    }
    theme = settings.default_theme

    if (/^#[0-9a-fA-F]{6}$/.test(settings.accent)) {
      document.documentElement.style.setProperty("--accent", settings.accent)
    }
  })
</script>

{#key active.id}
  <Active bind:lang bind:theme {loadJourneys} />
{/key}
