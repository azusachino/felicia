<script lang="ts">
  import { Sun, Moon } from "@lucide/svelte"
  import { Reader, message, resolveLocale, themeFromId, type ApiSiteSettings, type Lang, type Theme } from "@felicia/reader"
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
  let appearanceEdited = $state(false)
  function chooseTheme(value: Theme) {
    appearanceEdited = true
    theme = value
  }

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
  <a class="public-brand" href="/" aria-label="Felicia home">
    <img src={markUrl} alt="" aria-hidden="true" />
    <span>felicia</span>
  </a>

  <div class="reader-appearance">
    <button type="button" aria-label={message(lang, "theme.light")} title={message(lang, "theme.light")} aria-pressed={theme === "light"} onclick={() => chooseTheme("light")}
      ><Sun size={18} aria-hidden="true" /></button
    >
    <button type="button" aria-label={message(lang, "theme.dark")} title={message(lang, "theme.dark")} aria-pressed={theme === "dark"} onclick={() => chooseTheme("dark")}
      ><Moon size={18} aria-hidden="true" /></button
    >
  </div>
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

  .reader-appearance {
    position: absolute;
    top: 1rem;
    right: 1rem;
    z-index: 10;
    display: flex;
    padding: 3px;
    border: 1px solid rgba(184, 232, 221, 0.2);
    border-radius: 8px;
    background: rgba(9, 25, 37, 0.9);
    color: #c8d9dc;
  }
  .reader-appearance button {
    display: grid;
    place-items: center;
    width: 44px;
    height: 44px;
    border: 0;
    border-radius: 5px;
    color: inherit;
    background: transparent;
    cursor: pointer;
  }
  .reader-appearance button[aria-pressed="true"] {
    color: #07131f;
    background: #ff9b72;
  }
  .reader-appearance button:focus-visible {
    outline: 2px solid currentColor;
    outline-offset: 2px;
  }
  .public-reader-shell.theme-light .reader-appearance {
    background: rgba(241, 248, 246, 0.95);
    border-color: rgba(13, 41, 55, 0.16);
    color: #23424b;
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
