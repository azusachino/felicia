<script lang="ts">
  import { Reader, resolveLocale, themeFromId, type ApiSiteSettings, type Lang, type Theme } from "@felicia/reader"
  import { loadJourneys, loadSiteSettings } from "./api/source"

  let settings = $state<ApiSiteSettings | null>(null)
  const markUrl = `${import.meta.env.BASE_URL}felicia-mark.svg`

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
</script>

<div class="public-reader-shell" class:theme-light={theme === "light"}>
  <a class="public-brand" href="/" aria-label="Felicia home">
    <img src={markUrl} alt="" aria-hidden="true" />
    <span>felicia</span>
  </a>

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

  .public-brand {
    position: absolute;
    top: 1rem;
    left: 1rem;
    z-index: 10;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.375rem 0.75rem;
    border-radius: 9999px;
    background: rgba(9, 25, 37, 0.72);
    border: 1px solid rgba(184, 232, 221, 0.2);
    backdrop-filter: blur(8px);
    color: #c8d9dc;
    text-decoration: none;
    font-size: 0.8125rem;
    font-weight: 500;
    letter-spacing: 0.05em;
    transition:
      color 0.15s ease,
      border-color 0.15s ease;
  }

  .public-brand:hover {
    color: #fff;
    border-color: rgba(184, 232, 221, 0.4);
  }

  .public-brand img {
    width: 1rem;
    height: 1rem;
  }

  .public-reader-shell.theme-light .public-brand {
    background: rgba(241, 248, 246, 0.85);
    border-color: rgba(13, 41, 55, 0.16);
    color: #41606a;
  }

  .public-reader-shell.theme-light .public-brand:hover {
    color: #17202a;
    border-color: rgba(13, 41, 55, 0.32);
  }
</style>
