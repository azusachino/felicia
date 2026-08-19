<script lang="ts">
  import { onMount } from "svelte"
  import { designLanguageFromHash, designLanguageFromId, designLanguages, message, resolveLocale, themeFromId, type ApiSiteSettings, type Lang, type Theme } from "@felicia/shared"
  import { loadJourneys, loadSiteSettings } from "./api/source"

  let settings = $state<ApiSiteSettings | null>(null)
  let routeHash = $state(typeof window === "undefined" ? "" : window.location.hash)

  onMount(() => {
    const syncHash = () => (routeHash = window.location.hash)
    window.addEventListener("hashchange", syncHash)
    window.addEventListener("popstate", syncHash)
    return () => {
      window.removeEventListener("hashchange", syncHash)
      window.removeEventListener("popstate", syncHash)
    }
  })

  $effect(() => {
    loadSiteSettings()
      .then((s) => (settings = s))
      .catch(() => {
        // Absent/unreachable settings = current demo behavior: fall back to
        // the default design and the existing lang/theme defaults below.
      })
  })

  // The saved site design is the canonical default. A hash is an explicit,
  // shareable preview override, so authors can compare every registered design
  // without changing the published setting.
  const configured = $derived(designLanguageFromId(settings?.design))
  const active = $derived(routeHash ? designLanguageFromHash(routeHash) : configured)
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
    theme = themeFromId(settings.default_theme).id

    if (/^#[0-9a-fA-F]{6}$/.test(settings.accent)) {
      document.documentElement.style.setProperty("--accent", settings.accent)
    }
  })

  function selectDesign(id: string) {
    const design = designLanguageFromId(id)
    const url = design.hash ? design.hash : window.location.pathname + window.location.search
    window.history.pushState({}, "", url)
    routeHash = design.hash
  }
</script>

<div class="public-reader-shell">
  <nav class="public-design-switcher" aria-label={message(lang, "system.design")}>
    {#each designLanguages as design (design.id)}
      <button type="button" class:active={active.id === design.id} aria-pressed={active.id === design.id} onclick={() => selectDesign(design.id)}>
        {message(lang, design.labelKey)}
      </button>
    {/each}
  </nav>

  {#key active.id}
    <Active bind:lang bind:theme {loadJourneys} />
  {/key}
</div>

<style>
  .public-reader-shell {
    position: relative;
    width: 100%;
    height: 100%;
  }

  .public-design-switcher {
    position: fixed;
    z-index: 60;
    top: 1rem;
    left: 50%;
    display: flex;
    gap: 0.2rem;
    max-width: calc(100vw - 2rem);
    overflow-x: auto;
    padding: 0.25rem;
    border: 1px solid color-mix(in srgb, #fff 18%, transparent);
    border-radius: 999px;
    background: color-mix(in srgb, #09090b 78%, transparent);
    backdrop-filter: blur(12px);
    transform: translateX(-50%);
  }

  .public-design-switcher button {
    border: 0;
    border-radius: 999px;
    padding: 0.38rem 0.65rem;
    color: #a1a1aa;
    background: transparent;
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    white-space: nowrap;
  }

  .public-design-switcher button:hover,
  .public-design-switcher button.active {
    color: #18120d;
    background: #fdba74;
  }

  @media (max-width: 700px) {
    .public-design-switcher {
      top: 0.65rem;
      right: 0.65rem;
      left: 0.65rem;
      justify-content: center;
      transform: none;
    }
  }
</style>
