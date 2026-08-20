<script lang="ts">
  import { kindLabel, type Coordinates, type Journey, type L, type Lang, type Memento, type MementoKind, type Theme } from "@felicia/model"
  import TabiMap from "./TabiMap.svelte"
  import { message, type MessageKey } from "@felicia/model"

  // Tabi (旅, journey) — the cartoon reader: a cat runs the route from stop to
  // stop, cuddles each one (a pattern keyed to that stop's own memento kind),
  // then runs the return leg home. This shell owns the journey picker, the
  // "Play" trigger, and the place-info card; TabiMap.svelte owns the actual
  // map, the D3 overlay, and the run/cuddle sequencer -- see
  // docs/tutorials/d3/index.md in harus-workstation for the D3 lessons behind it.
  let { lang = $bindable("ja"), theme = $bindable("dark"), loadJourneys }: { lang?: Lang; theme?: Theme; loadJourneys: () => Promise<Journey[]> } = $props()

  let journeys = $state<Journey[]>([])
  let isLoading = $state(true)
  let error = $state<string | null>(null)
  let selectedIndex = $state(0)
  let selectedPlaceKey = $state<string | null>(null)
  let runToken = $state(0)
  let cartoonFilter = $state(true)

  function t(value: L | MessageKey): string {
    return typeof value === "string" ? message(lang, value) : value[lang]
  }

  const journeyRunLabel = { ja: "旅をはじめる", en: "Run the trip", zh: "开始旅程" } satisfies L
  const cartoonOnLabel = { ja: "🎨 カートゥーン", en: "🎨 Cartoon", zh: "🎨 卡通" } satisfies L
  const cartoonOffLabel = { ja: "🗺️ 写実", en: "🗺️ Realistic", zh: "🗺️ 写实" } satisfies L

  function loadData() {
    isLoading = true
    error = null
    loadJourneys()
      .then((data) => {
        journeys = data
        selectedIndex = 0
        selectedPlaceKey = null
      })
      .catch((reason) => {
        error = reason instanceof Error ? reason.message : String(reason)
      })
      .finally(() => {
        isLoading = false
      })
  }

  $effect(() => loadData())

  const selectedJourney = $derived(journeys[selectedIndex] ?? null)

  // Same place-as-derived-visit shape as techo (felicia:decision:place-as-derived-visit).
  interface PlaceGroup {
    key: string
    label: L
    coords: Coordinates
    seq: number
    mementos: Memento[]
  }

  const placeGroups = $derived.by<PlaceGroup[]>(() => {
    if (!selectedJourney) return []
    return selectedJourney.visits.map((visit, index) => ({
      key: visit.id,
      label: visit.label,
      coords: visit.coords,
      seq: index + 1,
      mementos: selectedJourney.mementos.filter((memento) => memento.visitId === visit.id),
    }))
  })

  const mapPlaces = $derived(
    placeGroups.map((group) => ({
      key: group.key,
      coords: group.coords,
      seq: group.seq,
      count: group.mementos.length,
      // First memento's kind decides that stop's cuddle pattern -- a visit
      // is always derived from at least one memento (felicia:decision:
      // place-as-derived-visit), so this fallback is a defensive default,
      // not an expected path.
      kind: (group.mementos[0]?.kind ?? "souvenir") satisfies MementoKind,
    })),
  )

  const selectedPlace = $derived(placeGroups.find((group) => group.key === selectedPlaceKey) ?? null)

  function selectJourney(index: number) {
    selectedIndex = index
    selectedPlaceKey = null
  }

  function selectPlace(key: string) {
    selectedPlaceKey = key === selectedPlaceKey ? null : key
  }
</script>

<main class="tabi-shell" class:theme-light={theme === "light"}>
  {#if isLoading}
    <div class="tabi-status" role="status">…</div>
  {:else if error}
    <div class="tabi-status tabi-status-error" role="alert">
      <p>{error}</p>
      <button type="button" onclick={loadData}>{t("ui.all")}</button>
    </div>
  {:else if !selectedJourney}
    <div class="tabi-status">{t("ui.journeys")}: 0</div>
  {:else}
    <nav class="tabi-journeys" aria-label={t("ui.journeys")}>
      {#each journeys as journey, index (journey.id)}
        <button type="button" class:active={index === selectedIndex} onclick={() => selectJourney(index)}>
          {t(journey.title)}
        </button>
      {/each}
    </nav>

    <section class="tabi-map-surface" aria-label={t(selectedJourney.title)}>
      <TabiMap places={mapPlaces} route={selectedJourney.route} activeKey={selectedPlaceKey} {runToken} {cartoonFilter} onSelect={selectPlace} />
    </section>

    <button type="button" class="tabi-filter-toggle" onclick={() => (cartoonFilter = !cartoonFilter)} aria-pressed={cartoonFilter}>
      {t(cartoonFilter ? cartoonOffLabel : cartoonOnLabel)}
    </button>

    <button type="button" class="tabi-play" onclick={() => (runToken += 1)}>▶ {t(journeyRunLabel)}</button>

    {#if selectedPlace}
      <aside class="tabi-place-card">
        <p class="tabi-place-label">{t(selectedPlace.label)}</p>
        {#each selectedPlace.mementos as memento (memento.id)}
          <p class="tabi-memento">
            <span class="tabi-memento-kind">{t(kindLabel[memento.kind])}</span>
            {t(memento.title)}
          </p>
        {/each}
      </aside>
    {/if}
  {/if}
</main>

<style>
  .tabi-shell {
    position: relative;
    width: 100%;
    height: 100%;
    background: #fff1f4;
    font-family: ui-rounded, "Nunito", system-ui, sans-serif;
    color: #4c1d3a;
  }

  .tabi-shell.theme-light {
    background: #fff8f0;
  }

  .tabi-status {
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    color: #be185d;
  }

  .tabi-status-error {
    gap: 1rem;
  }

  /* Every chrome surface below shares one treatment now: solid fill (no
     translucency, no backdrop-filter blur), a thick dark ink border matching
     the cat's own outline color, and a hard offset shadow with no blur --
     the same language "SUPER NYANCO RUN"'s own UI uses (a real pixel-game
     reference, not an invented approximation of one). The soft
     rgba(255,255,255,0.85)-plus-blur "glass pill" treatment this replaced
     was left over from the flat-vector version of this theme and never
     updated when the map and cat moved to the bolder pixel-sprite look --
     exactly the kind of per-component drift that's easy to miss without
     auditing every surface against the current design language, not just
     the ones that changed most recently. */
  .tabi-status button {
    padding: 0.5rem 1rem;
    border: 2.5px solid #3d2510;
    border-radius: 0.4rem;
    background: #fff;
    color: #be185d;
    font-weight: 700;
    box-shadow: 0.15rem 0.15rem 0 #3d2510;
  }

  .tabi-journeys {
    position: absolute;
    z-index: 10;
    top: 1rem;
    left: 1rem;
    display: flex;
    gap: 0.4rem;
    max-width: calc(100% - 2rem);
    overflow-x: auto;
    padding: 0.35rem;
    border: 2.5px solid #3d2510;
    border-radius: 0.5rem;
    background: #fff;
    box-shadow: 0.2rem 0.2rem 0 #3d2510;
  }

  .tabi-journeys button {
    padding: 0.4rem 0.9rem;
    border: 0;
    border-radius: 0.3rem;
    color: #be185d;
    background: transparent;
    font-weight: 700;
    font-size: 0.8rem;
    white-space: nowrap;
  }

  .tabi-journeys button.active {
    color: #fff;
    background: #fb7185;
  }

  .tabi-map-surface {
    position: absolute;
    inset: 0;
  }

  .tabi-filter-toggle {
    position: absolute;
    z-index: 10;
    top: 1rem;
    right: 1rem;
    padding: 0.4rem 0.8rem;
    border: 2.5px solid #3d2510;
    border-radius: 0.4rem;
    background: #fff;
    color: #be185d;
    font-family: inherit;
    font-weight: 700;
    font-size: 0.75rem;
    box-shadow: 0.15rem 0.15rem 0 #3d2510;
    cursor: pointer;
  }

  .tabi-filter-toggle:hover,
  .tabi-filter-toggle:focus-visible {
    background: #fce7f3;
    outline: none;
  }

  .tabi-play {
    position: absolute;
    z-index: 10;
    bottom: 1rem;
    left: 50%;
    transform: translateX(-50%);
    padding: 0.55rem 1.2rem;
    border: 2.5px solid #3d2510;
    border-radius: 0.5rem;
    background: #fb7185;
    color: #fff;
    font-family: inherit;
    font-weight: 800;
    font-size: 0.85rem;
    box-shadow: 0.25rem 0.25rem 0 #3d2510;
    cursor: pointer;
  }

  .tabi-play:hover,
  .tabi-play:focus-visible {
    background: #f472b6;
    outline: none;
  }

  .tabi-place-card {
    position: absolute;
    z-index: 10;
    right: 1rem;
    bottom: 1rem;
    max-width: min(22rem, calc(100% - 2rem));
    padding: 0.9rem 1.1rem;
    border: 2.5px solid #3d2510;
    border-radius: 0.6rem;
    background: #fff;
    box-shadow: 0.25rem 0.25rem 0 #3d2510;
  }

  .tabi-place-label {
    margin: 0 0 0.35rem;
    font-weight: 800;
    color: #be185d;
  }

  .tabi-memento {
    margin: 0.15rem 0;
    font-size: 0.85rem;
  }

  .tabi-memento-kind {
    margin-right: 0.4rem;
    padding: 0.1rem 0.45rem;
    border: 1.5px solid #3d2510;
    border-radius: 0.3rem;
    background: #fce7f3;
    color: #be185d;
    font-size: 0.65rem;
    font-weight: 800;
    text-transform: uppercase;
  }
</style>
