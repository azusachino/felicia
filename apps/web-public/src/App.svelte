<script lang="ts">
  import { designs, designFromHash } from "./designs"
  import type { Lang, Theme } from "./data"
  import { message, resolveLocale } from "./i18n/catalog"

  // The demo is a deployable, immutable-data showcase that can switch between
  // multiple front-of-house DESIGNS (the PM may supply several). Every design
  // is a pure presentation over the same fixtures + {journey, memento} contract
  // (felicia:decision:presentation-agnostic-contract); the registry in
  // designs.ts is the single source of truth and this shell just resolves the
  // active one from the URL hash and renders it, with a persistent switcher so
  // any design is one click (and deep-linkable) away.
  let hash = $state(location.hash)
  const active = $derived(designFromHash(hash))
  const Active = $derived(active.component)

  // lang/theme are shared across designs so switching keeps your reading state.
  let lang: Lang = $state(
    resolveLocale(localStorage.getItem("felicia.locale") ?? navigator.language),
  )
  let theme: Theme = $state("dark")

  $effect(() => localStorage.setItem("felicia.locale", lang))

  function select(target: (typeof designs)[number]) {
    location.hash = target.hash
  }
</script>

<svelte:window on:hashchange={() => (hash = location.hash)} />

<nav class="design-switcher" aria-label={message(lang, "system.design")}>
  {#each designs as design (design.id)}
    <button
      type="button"
      class="design-switcher__btn"
      class:active={design.id === active.id}
      aria-pressed={design.id === active.id}
      onclick={() => select(design)}
    >
      {message(lang, design.labelKey)}
    </button>
  {/each}
</nav>

{#key active.id}
  <Active bind:lang bind:theme />
{/key}

<style>
  .design-switcher {
    position: fixed;
    z-index: 1000;
    bottom: 1rem;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    gap: 0.25rem;
    padding: 0.25rem;
    border-radius: 999px;
    background: rgba(20, 18, 15, 0.72);
    backdrop-filter: blur(8px);
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.35);
  }

  .design-switcher__btn {
    font: inherit;
    font-size: 0.8rem;
    line-height: 1;
    padding: 0.5rem 0.9rem;
    border: none;
    border-radius: 999px;
    background: transparent;
    color: rgba(255, 255, 255, 0.72);
    cursor: pointer;
    transition:
      background 0.15s ease,
      color 0.15s ease;
  }

  .design-switcher__btn:hover {
    color: #fff;
  }

  .design-switcher__btn.active {
    background: #b45f26;
    color: #fff;
  }
</style>
