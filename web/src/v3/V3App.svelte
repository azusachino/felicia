<script lang="ts">
  import { kindLabel, type Coordinates, type L, type Lang, type Memento, type Theme } from '../data'
  import { project } from './cityLookup'
  import TripMap from './TripMap.svelte'
  import { loadJourneys } from '../api/source'
  import type { Journey } from '../data'

  // v3 — the "techo" (手帳, paper notebook) front door. View 1 (landing) is the
  // journal index — a two-page spread with a paper sketch map. View 2 (detail)
  // is the journey on a REAL map (reused v1 MapLibre): mementos cluster by place
  // (map is the index — felicia:decision:map-first-landing), and opening a place
  // reveals its memories (a place holds several) as paper cards with essay +
  // gallery. Styled with Tailwind (felicia:decision:techo-paper-v3).
  let { lang = $bindable('ja'), theme = $bindable('dark') }: { lang?: Lang; theme?: Theme } =
    $props()

  let journeys = $state<Journey[]>([])
  let isLoading = $state(true)
  let error = $state<string | null>(null)
  let selectedIndex = $state(0)
  let view = $state<'landing' | 'detail'>('landing')
  let selectedPlaceKey = $state<string | null>(null)
  let selectedMementoIndex = $state(0)

  function handleKeydown(event: KeyboardEvent) {
    if (view !== 'detail' || !selectedMemento) return
    if (event.key === 'ArrowLeft') {
      event.preventDefault()
      goToPrevMemento()
    } else if (event.key === 'ArrowRight') {
      event.preventDefault()
      goToNextMemento()
    }
  }

  const selectedJourney = $derived(journeys[selectedIndex] ?? null)

  interface CityDot {
    id: string
    journeyId: string
    label: string
    coords: Coordinates
  }

  const cityDots = $derived.by<CityDot[]>(() => {
    const list: CityDot[] = []
    for (const journey of journeys) {
      if (journey.representativeDots) {
        journey.representativeDots.forEach((dot, index) => {
          list.push({
            id: `${journey.id}:dot:${index}`,
            journeyId: journey.id,
            label: dot.label.toUpperCase(),
            coords: dot.coord,
          })
        })
      }
    }
    return list
  })

  function loadData() {
    isLoading = true
    error = null
    loadJourneys()
      .then((data) => {
        journeys = data
        isLoading = false
      })
      .catch((err) => {
        error = err instanceof Error ? err.message : String(err)
        isLoading = false
      })
  }

  $effect(() => {
    loadData()
  })

  function t(value: L): string {
    return value[lang]
  }

  // Places ARE the journey's derived visits (felicia:decision:place-as-derived-visit);
  // mementos anchor to them by visitId. One map marker per visit; a visit can hold
  // several memories. No string-grouping — the visit is the stable place key.
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
    })),
  )

  const selectedPlace = $derived(
    placeGroups.find((group) => group.key === selectedPlaceKey) ?? null,
  )

  const selectedMemento = $derived(selectedJourney?.mementos[selectedMementoIndex] ?? null)

  const transitPairs = $derived.by(() => {
    if (!selectedJourney) return []
    return selectedJourney.mementos
      .filter((memento) => memento.transit)
      .map(
        (memento) =>
          [memento.transit!.from.coords, memento.transit!.to.coords] as [Coordinates, Coordinates],
      )
  })

  function selectJourneyById(id: string) {
    const idx = journeys.findIndex((j) => j.id === id)
    if (idx !== -1) {
      selectedIndex = idx
      selectedMementoIndex = 0
      selectedPlaceKey = null
    }
  }

  function selectYear(year: string) {
    const idx = journeys.findIndex((journey) => {
      const match = journey.dates.ja.match(/(\d{4})年/)
      const y = match ? match[1] : '2026'
      return y === year
    })
    if (idx !== -1) {
      selectedIndex = idx
      selectedMementoIndex = 0
      selectedPlaceKey = null
    }
  }

  function openJourney(index = selectedIndex) {
    selectedIndex = index
    selectedMementoIndex = 0
    selectedPlaceKey = selectedJourney?.mementos[0]?.visitId ?? null
    view = 'detail'
  }

  function backToLanding() {
    view = 'landing'
  }

  function selectPlace(key: string) {
    if (!key || selectedPlaceKey === key) {
      selectedPlaceKey = null
    } else {
      selectedPlaceKey = key
      const first = selectedJourney?.mementos.findIndex((memento) => memento.visitId === key) ?? -1
      if (first >= 0) selectedMementoIndex = first
    }
  }

  function goToPrevMemento() {
    if (!selectedJourney || selectedMementoIndex === 0) return
    selectedMementoIndex -= 1
    selectedPlaceKey = selectedJourney.mementos[selectedMementoIndex]?.visitId ?? null
  }

  function goToNextMemento() {
    if (!selectedJourney || selectedMementoIndex >= selectedJourney.mementos.length - 1) return
    selectedMementoIndex += 1
    selectedPlaceKey = selectedJourney.mementos[selectedMementoIndex]?.visitId ?? null
  }

  function onPhotoError(event: Event) {
    ;(event.currentTarget as HTMLImageElement).style.display = 'none'
  }

  // Washi tape texture: a small fixed palette cycled by card index.
  const washiColors = [
    'rgba(200, 120, 60, 0.35)',
    'rgba(120, 150, 108, 0.35)',
    'rgba(120, 138, 188, 0.32)',
    'rgba(196, 150, 88, 0.35)',
  ]
  const washiRotations = [-4, 3, -2, 5]

  const years = $derived.by(() => {
    // eslint-disable-next-line svelte/prefer-svelte-reactivity -- transient local dedupe set inside a pure $derived
    const set = new Set<string>()
    for (const journey of journeys) {
      const match = journey.dates.ja.match(/(\d{4})年/)
      set.add(match ? match[1] : '2026')
    }
    return Array.from(set).sort((a, b) => Number(b) - Number(a))
  })

  const activeYear = $derived.by(() => {
    if (!selectedJourney) return years[0] ?? '2026'
    const match = selectedJourney.dates.ja.match(/(\d{4})年/)
    return match ? match[1] : years[0]
  })

  const journeyCountLabel = $derived.by(() => {
    const n = journeys.length
    if (lang === 'en') return `${n} journeys`
    if (lang === 'zh') return `${n}次旅程`
    return `${n}つの旅`
  })

  function mementoCountLabel(n: number): string {
    if (lang === 'en') return `${n} stop${n === 1 ? '' : 's'}`
    if (lang === 'zh') return `${n}件`
    return `${n}件`
  }

  function placeMemoriesLabel(n: number): string {
    if (lang === 'en') return `${n} memor${n === 1 ? 'y' : 'ies'} here`
    if (lang === 'zh') return `此地 ${n} 件记忆`
    return `この場所の記憶 ${n}件`
  }

  // Landing sketch map: dim every journey's city dots, brighten the selected.
  const routePoints = $derived(selectedJourney?.route.map((coord) => project(coord)) ?? [])
  const routePath = $derived(routePoints.map((p) => `${p.x},${p.y}`).join(' '))

  const mapCaption = {
    ja: '地図の印をえらぶと、その旅がひらきます',
    en: 'Choose a mark on the map to open that journey',
    zh: '在地图上选择标记，即可打开这段旅程',
  } satisfies L
  const seasonCaption = {
    ja: '冬－春の記録',
    en: 'Winter–spring notes',
    zh: '冬–春记录',
  } satisfies L
  const brandTagline = {
    ja: 'Travel journal',
    en: 'Travel journal',
    zh: 'Travel journal',
  } satisfies L
  const selectedBadge = { ja: '選択中', en: 'Selected', zh: '已选' } satisfies L
  const openCta = {
    ja: 'この旅をひらく →',
    en: 'Open this journey →',
    zh: '打开这段旅程 →',
  } satisfies L
  const backLabel = { ja: '手帳に戻る', en: 'Back to journal', zh: '返回手帳' } satisfies L
  const prevMemoryLabel = { ja: '前の記憶', en: 'Previous memory', zh: '上一段记忆' } satisfies L
  const nextMemoryLabel = { ja: '次の記憶', en: 'Next memory', zh: '下一段记忆' } satisfies L
</script>

<svelte:window on:keydown={handleKeydown} />

<main class="techo-shell" class:theme-light={theme === 'light'} class:is-detail={view === 'detail'}>
  {#if view === 'landing'}
    <!-- View 1: the journal index — sketch map on the left, journey cards on the right. -->
    <div class="techo-frame">
      <div class="techo-spread">
        {#if isLoading}
          <!-- Map page skeleton -->
          <section class="techo-page techo-page--map" aria-label="Journey map loading">
            <div class="map-grid skeleton-pulse"></div>
            <p
              class="map-caption skeleton-pulse"
              style="width: 70%; height: 0.8rem; border-radius: 0.2rem; background: var(--paper-3);"
            ></p>
          </section>

          <!-- Index page skeleton -->
          <section class="techo-page techo-page--index" aria-label="Journal index loading">
            <header class="index-header">
              <div>
                <p
                  class="brand-mark skeleton-pulse"
                  style="width: 6rem; height: 1.1rem; border-radius: 0.2rem; background: var(--paper-2);"
                ></p>
                <p
                  class="brand-tagline skeleton-pulse"
                  style="width: 8rem; height: 0.9rem; border-radius: 0.2rem; background: var(--paper-2); margin-top: 0.5rem;"
                ></p>
              </div>
              <p
                class="season-caption skeleton-pulse"
                style="width: 5rem; height: 0.8rem; border-radius: 0.2rem; background: var(--paper-2);"
              ></p>
            </header>

            <div class="year-row" style="margin-top: 2rem;">
              <div
                class="skeleton-pulse"
                style="width: 5rem; height: 2.4rem; border-radius: 0.2rem; background: var(--paper-2);"
              ></div>
              <div
                class="skeleton-pulse"
                style="width: 4rem; height: 0.8rem; border-radius: 0.2rem; background: var(--paper-2);"
              ></div>
            </div>

            <div
              class="journey-cards"
              style="margin-top: 1.5rem; display: flex; flex-direction: column; gap: 0.9rem;"
            >
              {#each [0, 1] as i (i)}
                <div class="skeleton-card">
                  <div
                    class="skeleton-line skeleton-line--title skeleton-pulse"
                    style="margin-top: 0.5rem;"
                  ></div>
                  <div class="skeleton-line skeleton-line--date skeleton-pulse"></div>
                  <div class="card-divider"></div>
                  <div class="skeleton-line skeleton-line--place skeleton-pulse"></div>
                </div>
              {/each}
            </div>
          </section>
        {:else if error}
          <section class="techo-page techo-page--map" aria-label="Journey map error">
            <div class="map-grid" style="opacity: 0.4;"></div>
          </section>
          <section class="techo-page techo-page--index" aria-label="Journal index error">
            <div class="error-container">
              <h2 class="error-title">読み込みエラー / Loading Error</h2>
              <p class="error-text">{error}</p>
              <button type="button" class="retry-button" onclick={loadData}>再試行 / Retry</button>
            </div>
          </section>
        {:else if journeys.length === 0}
          <section class="techo-page techo-page--map" aria-label="Journey map empty">
            <div class="map-grid" style="opacity: 0.4;"></div>
          </section>
          <section class="techo-page techo-page--index" aria-label="Journal index empty">
            <div class="empty-container">
              <h2 class="error-title">旅の記録がありません</h2>
              <p class="empty-text">
                現在、この手帳には記録された旅がありません。<br />There are no journeys recorded in
                this journal.
              </p>
            </div>
          </section>
        {:else}
          <section class="techo-page techo-page--map" aria-label="Journey map">
            <div class="map-grid">
              <svg
                class="map-svg"
                viewBox="0 0 100 100"
                preserveAspectRatio="none"
                aria-hidden="true"
              >
                {#if routePoints.length > 1}
                  <polyline class="map-route" points={routePath} />
                {/if}
              </svg>
              {#each cityDots as dot (dot.id)}
                {@const p = project(dot.coords)}
                {@const active = selectedJourney && dot.journeyId === selectedJourney.id}
                <button
                  type="button"
                  class="map-dot border-none bg-transparent p-0 cursor-pointer focus:outline-none animate-none"
                  class:active
                  style="left:{p.x}%; top:{p.y}%"
                  onclick={() => selectJourneyById(dot.journeyId)}
                >
                  <span class="map-dot-mark"></span>
                  <span class="map-dot-label">{dot.label}</span>
                </button>
              {/each}
            </div>
            <p class="map-caption">{t(mapCaption)}</p>
          </section>

          <section class="techo-page techo-page--index" aria-label="Journal index">
            <header class="index-header">
              <div>
                <p class="brand-mark">F E L I C I A</p>
                <p class="brand-tagline">{t(brandTagline)}</p>
              </div>
              <p class="season-caption">{t(seasonCaption)}</p>
            </header>

            <div class="year-row">
              <h1 class="year-heading">{activeYear}</h1>
              <p class="year-count">{journeyCountLabel}</p>
            </div>

            <button type="button" class="open-cta open-cta--top" onclick={() => openJourney()}>
              {t(openCta)}
              <span class="open-cta-subtitle"
                >{t(selectedJourney?.title ?? { ja: '', en: '', zh: '' })}</span
              >
            </button>

            <ol class="journey-cards">
              {#each journeys as journey, index (journey.id)}
                <li>
                  <button
                    type="button"
                    class="journey-card"
                    class:selected={index === selectedIndex}
                    onclick={() => openJourney(index)}
                    aria-label={`${t(journey.title)} — ${t(openCta)}`}
                  >
                    <span
                      class="washi-tape"
                      style="background:{washiColors[
                        index % washiColors.length
                      ]}; transform: translate(-50%, -55%) rotate({washiRotations[
                        index % washiRotations.length
                      ]}deg)"
                    ></span>
                    {#if index === selectedIndex}
                      <span class="selected-badge">{t(selectedBadge)}</span>
                    {/if}
                    <h2 class="card-title">{t(journey.title)}</h2>
                    <p class="card-dates">{t(journey.dates)}</p>
                    <div class="card-divider"></div>
                    <div class="card-footer">
                      <span class="card-place">{t(journey.place)}</span>
                      <span class="card-count">{mementoCountLabel(journey.mementos.length)}</span>
                    </div>
                  </button>
                </li>
              {/each}
            </ol>
          </section>
        {/if}

        <nav class="year-tabs" aria-label="Years">
          {#each years as year (year)}
            <button
              type="button"
              class="year-tab cursor-pointer border border-solid focus:outline-none"
              class:active={year === activeYear}
              onclick={() => selectYear(year)}
            >
              {year}
            </button>
          {/each}
        </nav>
      </div>
    </div>
  {:else}
    <!-- View 2: the journey on a real map; mementos cluster by place, opening a
         place reveals its memories. -->
    <section
      class="relative h-full w-full overflow-hidden bg-paper-2"
      aria-label={t(selectedJourney.title)}
    >
      <TripMap
        places={mapPlaces}
        route={selectedJourney.route}
        transit={transitPairs}
        activeKey={selectedPlaceKey}
        {theme}
        onSelect={selectPlace}
      />

      <header class="pointer-events-none absolute left-6 top-6 z-10 flex items-start gap-3">
        <div class="pointer-events-auto rounded-lg bg-paper-1/95 px-4 py-3 shadow-lg backdrop-blur">
          <p class="m-0 font-mono text-[0.7rem] tracking-[0.3em] text-terracotta">F E L I C I A</p>
          <h1 class="m-0 mt-1 font-mincho text-2xl font-bold text-ink">
            {t(selectedJourney.title)}
          </h1>
          <p class="m-0 text-sm text-ink-soft">
            {t(selectedJourney.dates)} · {t(selectedJourney.place)}
          </p>
        </div>
        <button
          type="button"
          class="pointer-events-auto rounded-md border border-black/10 bg-paper-1/95 px-3 py-2 font-mono text-xs tracking-wide text-terracotta shadow"
          onclick={backToLanding}>{t(backLabel)} ×</button
        >
      </header>

      {#if selectedMemento}
        <aside
          class="absolute right-0 top-0 z-10 flex h-full w-[min(30rem,46vw)] flex-col gap-5 overflow-y-auto bg-paper-1/95 px-6 py-6 shadow-2xl backdrop-blur"
          aria-label="Memories at this place"
          aria-keyshortcuts="ArrowLeft ArrowRight"
        >
          <div class="flex items-start justify-between">
            <div>
              <p class="m-0 font-mono text-[0.7rem] uppercase tracking-[0.18em] text-ink-faint">
                {placeMemoriesLabel(1)}
              </p>
              <h2 class="m-0 mt-1 font-mincho text-xl font-bold text-ink">
                {t(selectedPlace?.label ?? selectedMemento.place)}
              </h2>
            </div>
            <button
              type="button"
              class="flex h-7 w-7 items-center justify-center rounded-full border border-black/10 bg-paper-2 hover:bg-paper-3 text-base text-ink-soft cursor-pointer transition focus:outline-none"
              onclick={() => selectPlace('')}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          <div
            class="flex items-center justify-between border-b border-dashed border-black/15 pb-3"
          >
            <button
              type="button"
              class="flex items-center gap-1 font-mono text-xs text-ink-soft disabled:opacity-30 disabled:cursor-not-allowed hover:text-terracotta transition cursor-pointer bg-transparent border-none p-0 focus:outline-none"
              disabled={selectedMementoIndex === 0}
              onclick={goToPrevMemento}
            >
              ← {t(prevMemoryLabel)}
            </button>
            <span class="font-mono text-xs text-ink-faint">
              {selectedMementoIndex + 1} / {selectedJourney.mementos.length}
            </span>
            <button
              type="button"
              class="flex items-center gap-1 font-mono text-xs text-ink-soft disabled:opacity-30 disabled:cursor-not-allowed hover:text-terracotta transition cursor-pointer bg-transparent border-none p-0 focus:outline-none"
              disabled={selectedMementoIndex === selectedJourney.mementos.length - 1}
              onclick={goToNextMemento}
            >
              {t(nextMemoryLabel)} →
            </button>
          </div>

          <article class="rounded-lg border border-black/5 bg-paper-0 p-5 shadow-sm">
            <p class="m-0 font-mono text-[0.68rem] uppercase tracking-[0.2em] text-terracotta">
              {t(kindLabel[selectedMemento.kind])}
            </p>
            <h3 class="m-0 mt-1 font-mincho text-lg font-bold text-ink">
              {t(selectedMemento.title)}
            </h3>
            <p class="m-0 mt-0.5 text-xs text-ink-faint">
              {t(selectedMemento.date)}{selectedMemento.price ? ` · ${selectedMemento.price}` : ''}
            </p>
            <p class="m-0 mt-3 text-[0.9rem] leading-relaxed text-ink-soft">
              {t(selectedMemento.essay)}
            </p>
            {#if selectedMemento.photos.length}
              <div class="mt-4 flex flex-col gap-3">
                {#each selectedMemento.photos as photo (photo.src)}
                  <figure class="m-0 overflow-hidden rounded-md border border-black/5">
                    <img
                      src={photo.src}
                      alt={t(selectedMemento.title)}
                      class="block aspect-[4/3] w-full object-cover"
                      onerror={onPhotoError}
                    />
                    <figcaption class="px-3 py-2 text-xs text-ink-soft">
                      {t(photo.caption)}
                    </figcaption>
                  </figure>
                {/each}
              </div>
            {/if}
          </article>
        </aside>
      {/if}
    </section>
  {/if}
</main>

<style>
  .techo-shell {
    --ink: #3a2f1c;
    --ink-soft: #5a4622;
    --ink-faint: #7a5c30;
    --paper-0: #fbf7ee;
    --paper-1: #fdf9f0;
    --paper-2: #f3ecdb;
    --paper-3: #efe7d5;
    --terracotta: #b45f26;
    --hairline: rgba(90, 66, 30, 0.3);
    --hairline-strong: rgba(90, 66, 30, 0.4);

    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    padding: 2.5rem;
    box-sizing: border-box;
    background: radial-gradient(circle at 50% 20%, #d8cbb2, #b9a988 85%);
    font-family: 'Zen Old Mincho', 'Spectral', serif;
    color: var(--ink);
  }

  .techo-shell.theme-light {
    background: radial-gradient(circle at 50% 20%, #e7dcc4, #c9bb99 85%);
  }

  /* Detail goes full-bleed: the real map is the whole surface. Drop the flex
     centering so the map section fills 100% × 100% instead of being centered. */
  .techo-shell.is-detail {
    padding: 0;
    display: block;
  }

  .techo-frame {
    position: relative;
    width: min(94vw, 76rem);
    max-height: 92vh;
    border-radius: 0.9rem;
    background: var(--paper-3);
    box-shadow:
      0 2rem 4rem rgba(58, 47, 28, 0.35),
      0 0.5rem 1rem rgba(58, 47, 28, 0.2);
    padding: 0.5rem;
  }

  .techo-spread {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
    height: 100%;
  }

  .techo-page {
    position: relative;
    border-radius: 0.6rem;
    padding: 1.75rem 2rem;
    min-height: 34rem;
    max-height: 78vh;
    overflow-y: auto;
    box-sizing: border-box;
  }

  .techo-page--map {
    background: var(--paper-2);
    border-top-left-radius: 0.9rem;
    border-bottom-left-radius: 0.9rem;
  }

  .techo-page--index {
    background: var(--paper-1);
    border-top-right-radius: 0.9rem;
    border-bottom-right-radius: 0.9rem;
  }

  /* --- View 1: sketch map --- */

  .map-grid {
    position: relative;
    height: calc(100% - 2rem);
    min-height: 26rem;
    border-radius: 0.4rem;
    background-image:
      repeating-linear-gradient(0deg, transparent, transparent 39px, var(--hairline) 40px),
      repeating-linear-gradient(90deg, transparent, transparent 39px, var(--hairline) 40px);
    background-color: rgba(251, 247, 238, 0.35);
    opacity: 0.92;
  }

  .map-svg {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
  }

  .map-route {
    fill: none;
    stroke: var(--terracotta);
    stroke-width: 0.4;
    stroke-dasharray: 1.4 1.2;
    opacity: 0.85;
  }

  .map-dot {
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.3rem;
    transform: translate(-50%, -50%);
  }

  .map-dot-mark {
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 999px;
    background: #8a8272;
    box-shadow: 0 0 0 4px rgba(138, 130, 114, 0.15);
  }

  .map-dot-label {
    font-family: ui-monospace, 'SFMono-Regular', monospace;
    font-size: 0.6rem;
    letter-spacing: 0.08em;
    color: var(--ink-faint);
    white-space: nowrap;
  }

  .map-dot.active .map-dot-mark {
    background: var(--terracotta);
    box-shadow: 0 0 0 5px rgba(180, 95, 38, 0.22);
  }

  .map-dot.active .map-dot-label {
    color: var(--terracotta);
    font-weight: 700;
  }

  .map-caption {
    margin: 0.9rem 0 0;
    font-size: 0.72rem;
    color: var(--ink-faint);
  }

  /* --- View 1: index page --- */

  .index-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }

  .brand-mark {
    margin: 0;
    font-family: ui-monospace, 'SFMono-Regular', monospace;
    font-size: 0.95rem;
    letter-spacing: 0.4em;
    color: var(--terracotta);
  }

  .brand-tagline {
    margin: 0.2rem 0 0;
    font-family: 'Spectral', serif;
    font-style: italic;
    font-size: 1rem;
    color: var(--ink-soft);
  }

  .season-caption {
    margin: 0.15rem 0 0;
    font-size: 0.72rem;
    color: var(--ink-faint);
  }

  .year-row {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    margin-top: 1.5rem;
  }

  .year-heading {
    margin: 0;
    font-family: 'Zen Old Mincho', serif;
    font-size: 2.4rem;
    font-weight: 700;
    color: var(--ink);
  }

  .year-count {
    margin: 0;
    font-family: ui-monospace, 'SFMono-Regular', monospace;
    font-size: 0.72rem;
    color: var(--ink-faint);
  }

  .journey-cards {
    list-style: none;
    margin: 1.25rem 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
  }

  .journey-card {
    position: relative;
    display: block;
    width: 100%;
    text-align: left;
    background: var(--paper-0);
    border: 1px solid var(--hairline);
    border-radius: 0.35rem;
    padding: 1rem 1.25rem 0.9rem;
    box-shadow: 0 0.3rem 0.6rem rgba(58, 47, 28, 0.08);
    cursor: pointer;
    font: inherit;
    color: inherit;
  }

  .journey-card.selected {
    border-color: var(--terracotta);
    border-width: 2px;
    padding: calc(1rem - 1px) calc(1.25rem - 1px) calc(0.9rem - 1px);
  }

  .washi-tape {
    position: absolute;
    top: 0;
    left: 1.6rem;
    width: 3.6rem;
    height: 1.1rem;
    border-radius: 1px;
  }

  .selected-badge {
    position: absolute;
    top: 0.9rem;
    right: 1.1rem;
    font-family: ui-monospace, 'SFMono-Regular', monospace;
    font-size: 0.62rem;
    letter-spacing: 0.06em;
    color: #fff;
    background: var(--terracotta);
    border-radius: 0.25rem;
    padding: 0.2rem 0.5rem;
  }

  .card-title {
    margin: 0.6rem 0 0.3rem;
    font-family: 'Zen Old Mincho', serif;
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--ink);
  }

  .card-dates {
    margin: 0;
    font-size: 0.82rem;
    color: var(--ink-soft);
  }

  .card-divider {
    margin: 0.7rem 0;
    border-top: 1px dashed var(--hairline-strong);
  }

  .card-footer {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    font-size: 0.82rem;
  }

  .card-place {
    color: var(--ink-soft);
  }

  .card-count {
    font-weight: 700;
    color: var(--terracotta);
  }

  .open-cta {
    display: block;
    width: 100%;
    margin-top: 1.4rem;
    padding: 0.85rem 1rem;
    border: none;
    border-radius: 0.4rem;
    background: var(--terracotta);
    color: #fdf6ec;
    font-family: 'Zen Old Mincho', serif;
    font-size: 1rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    cursor: pointer;
  }

  .open-cta:hover {
    background: #a05420;
  }

  .open-cta--top {
    margin-top: 1rem;
    margin-bottom: 0.25rem;
  }

  .open-cta-subtitle {
    display: block;
    margin-top: 0.2rem;
    font-size: 0.7rem;
    font-weight: 400;
    opacity: 0.85;
  }

  .year-tabs {
    position: absolute;
    top: 3rem;
    right: -1.9rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .year-tab {
    display: block;
    padding: 0.5rem 0.35rem;
    font-family: ui-monospace, 'SFMono-Regular', monospace;
    font-size: 0.68rem;
    letter-spacing: 0.06em;
    text-align: center;
    color: var(--ink-faint);
    background: var(--paper-2);
    border: 1px solid var(--hairline);
    border-radius: 0.3rem;
    writing-mode: vertical-rl;
  }

  .year-tab.active {
    color: #fdf6ec;
    background: var(--terracotta);
    border-color: var(--terracotta);
  }

  @media (max-width: 900px) {
    .techo-spread {
      grid-template-columns: 1fr;
    }

    .techo-page--index {
      order: 1;
    }

    .techo-page--map {
      order: 2;
    }

    .year-tabs {
      display: none;
    }
  }

  /* --- Skeletons, Errors, Empty States --- */

  .skeleton-pulse {
    animation: skeleton-loading 1.5s infinite ease-in-out;
  }

  @keyframes skeleton-loading {
    0% {
      background-color: var(--paper-2);
      opacity: 0.6;
    }
    50% {
      background-color: var(--paper-3);
      opacity: 0.95;
    }
    100% {
      background-color: var(--paper-2);
      opacity: 0.6;
    }
  }

  .skeleton-line {
    height: 1rem;
    border-radius: 0.2rem;
    background-color: var(--paper-2);
    margin-bottom: 0.5rem;
  }

  .skeleton-line--title {
    width: 60%;
    height: 1.4rem;
  }

  .skeleton-line--date {
    width: 40%;
  }

  .skeleton-line--place {
    width: 30%;
  }

  .skeleton-card {
    background: var(--paper-0);
    border: 1px solid var(--hairline);
    border-radius: 0.35rem;
    padding: 1.6rem 1.25rem 0.9rem;
    box-shadow: 0 0.3rem 0.6rem rgba(58, 47, 28, 0.04);
    pointer-events: none;
  }

  .error-container,
  .empty-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    height: 100%;
    min-height: 26rem;
    padding: 2rem;
  }

  .error-title {
    font-family: 'Zen Old Mincho', serif;
    font-size: 1.4rem;
    font-weight: 700;
    color: var(--terracotta);
    margin: 0 0 1rem;
  }

  .error-text,
  .empty-text {
    font-size: 0.9rem;
    color: var(--ink-soft);
    margin: 0 0 1.5rem;
    line-height: 1.6;
  }

  .retry-button {
    padding: 0.6rem 1.5rem;
    border: 1px solid var(--terracotta);
    border-radius: 0.25rem;
    background: transparent;
    color: var(--terracotta);
    font-family: inherit;
    font-size: 0.85rem;
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .retry-button:hover {
    background: var(--terracotta);
    color: #fdf6ec;
  }
</style>
