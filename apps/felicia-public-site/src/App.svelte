<script lang="ts">
  import { onMount } from "svelte"
  import { designLanguageFromHash, designLanguageFromId, designLanguages, message, resolveLocale, themeFromId, type ApiSiteSettings, type Lang, type Theme } from "@felicia/reader"
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
  const markUrl = `${import.meta.env.BASE_URL}felicia-mark.svg`

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

<div class="public-reader-shell" class:theme-light={theme === "light"} class:design-cabinet={active.id === "cabinet"} class:design-cartography={active.id === "cartography"}>
  <a class="public-brand" href="/" aria-label="Felicia home">
    <img src={markUrl} alt="" aria-hidden="true" />
    <span>felicia</span>
  </a>

  <nav class="public-design-switcher" aria-label={message(lang, "system.design")}>
    {#each designLanguages as design (design.id)}
      <button
        type="button"
        class:active={active.id === design.id}
        aria-current={active.id === design.id ? "page" : undefined}
        aria-pressed={active.id === design.id}
        onclick={() => selectDesign(design.id)}
      >
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
    --switcher-bg: rgba(9, 25, 37, 0.72);
    --switcher-border: rgba(184, 232, 221, 0.2);
    --switcher-text: #c8d9dc;
    --switcher-active: #ff9b72;
    --switcher-active-text: #17202a;
    --glass-highlight: rgba(255, 255, 255, 0.2);
    --glass-shadow: rgba(1, 9, 17, 0.3);
    position: relative;
    width: 100%;
    height: 100%;
    color-scheme: dark;
  }

  .public-reader-shell.theme-light {
    --switcher-bg: rgba(241, 248, 246, 0.78);
    --switcher-border: rgba(13, 41, 55, 0.16);
    --switcher-text: #41606a;
    --switcher-active: #f08f69;
    --switcher-active-text: #17202a;
    --glass-highlight: rgba(255, 255, 255, 0.78);
    --glass-shadow: rgba(34, 73, 82, 0.16);
    color-scheme: light;
  }

  .public-brand {
    position: fixed;
    z-index: 60;
    top: 1rem;
    left: 1rem;
    display: inline-flex;
    min-height: 3.25rem;
    align-items: center;
    gap: 0.6rem;
    padding: 0.35rem 0.85rem 0.35rem 0.35rem;
    border: 1px solid var(--switcher-border);
    border-radius: 999px;
    color: var(--switcher-text);
    background: var(--switcher-bg);
    box-shadow:
      inset 0 1px 0 var(--glass-highlight),
      0 1rem 3rem var(--glass-shadow);
    backdrop-filter: blur(20px) saturate(140%);
    -webkit-backdrop-filter: blur(20px) saturate(140%);
    font-size: 0.8rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-decoration: none;
    text-transform: lowercase;
  }

  .public-brand img {
    width: 2.45rem;
    height: 2.45rem;
    border-radius: 0.72rem;
  }

  .public-brand:hover {
    border-color: color-mix(in srgb, var(--switcher-active) 58%, transparent);
  }

  .public-reader-shell.design-cabinet :global(.cabinet-top) {
    padding-left: 10rem;
  }

  .public-reader-shell.design-cartography :global(.index-rail) {
    padding-top: 6rem;
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
    min-height: 3.25rem;
    padding: 0.35rem;
    border: 1px solid var(--switcher-border);
    border-radius: 999px;
    background: var(--switcher-bg);
    box-shadow:
      inset 0 1px 0 var(--glass-highlight),
      0 1rem 3rem var(--glass-shadow);
    backdrop-filter: blur(20px) saturate(140%);
    -webkit-backdrop-filter: blur(20px) saturate(140%);
    transform: translateX(-50%);
  }

  .public-design-switcher button {
    border: 0;
    border-radius: 999px;
    padding: 0.38rem 0.65rem;
    color: var(--switcher-text);
    background: transparent;
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    white-space: nowrap;
    transition:
      color 180ms ease,
      background 180ms ease,
      transform 180ms ease;
  }

  .public-design-switcher button:hover,
  .public-design-switcher button.active {
    color: var(--switcher-active-text);
    background: var(--switcher-active);
  }

  .public-design-switcher button:active {
    transform: scale(0.97);
  }

  .public-design-switcher button:focus-visible {
    outline-color: var(--switcher-active);
    outline-offset: -2px;
  }

  @media (max-width: 700px) {
    .public-reader-shell.design-cabinet :global(.cabinet-top) {
      padding-left: 1.25rem;
    }

    .public-reader-shell.design-cartography :global(.index-rail) {
      padding-top: 7.5rem;
    }

    .public-brand {
      top: 0.65rem;
      left: 0.65rem;
      min-height: 2.75rem;
      padding-right: 0.65rem;
    }

    .public-brand img {
      width: 2rem;
      height: 2rem;
    }

    .public-design-switcher {
      top: 4rem;
      right: 0.65rem;
      left: 0.65rem;
      justify-content: center;
      transform: none;
    }
  }
</style>
