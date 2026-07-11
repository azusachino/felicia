<script lang="ts">
  import { loadJourneys } from '../api/source'
  import type { Coordinates, Journey, Lang, Theme } from '../data'

  let { lang = $bindable('ja'), theme = $bindable('dark') }: { lang?: Lang; theme?: Theme } =
    $props()
  let journeys = $state<Journey[]>([])
  let selectedIndex = $state(0)
  let isLoading = $state(true)
  let error = $state<string | null>(null)
  let selectedMementoIndex = $state(0)
  const selectedJourney = $derived(journeys[selectedIndex] ?? null)
  const selectedMemento = $derived(selectedJourney?.mementos[selectedMementoIndex] ?? null)

  const ui = {
    ja: {
      title: '旅の地図帳',
      subtitle: '世界中の記憶を、場所からひらく',
      journeys: '旅',
      memories: '記憶',
      back: '一覧に戻る',
      previous: '前',
      next: '次',
      loading: '読み込み中…',
      retry: '再試行',
    },
    en: {
      title: 'The Atlas',
      subtitle: 'Open the archive from the places you remember',
      journeys: 'journeys',
      memories: 'memories',
      back: 'Back to atlas',
      previous: 'Previous',
      next: 'Next',
      loading: 'Loading…',
      retry: 'Retry',
    },
    zh: {
      title: '旅行图册',
      subtitle: '从记得的地方打开这份档案',
      journeys: '次旅程',
      memories: '段记忆',
      back: '返回图册',
      previous: '上一段',
      next: '下一段',
      loading: '加载中…',
      retry: '重试',
    },
  } as const
  const t = (value: { ja: string; en: string; zh: string }) => value[lang]
  const label = $derived(ui[lang])

  function world([lon, lat]: Coordinates) {
    return { x: ((lon + 180) / 360) * 100, y: (1 - (lat + 60) / 210) * 100 }
  }

  function routePoints(journey: Journey) {
    return journey.route
      .map((coord) => {
        const point = world(coord)
        return `${point.x},${point.y}`
      })
      .join(' ')
  }

  function loadData() {
    isLoading = true
    error = null
    loadJourneys()
      .then((data) => {
        journeys = data
        isLoading = false
      })
      .catch((reason) => {
        error = reason instanceof Error ? reason.message : String(reason)
        isLoading = false
      })
  }

  function selectJourney(index: number) {
    selectedIndex = index
    selectedMementoIndex = 0
  }

  function previousMemory() {
    if (selectedMementoIndex > 0) selectedMementoIndex -= 1
  }

  function nextMemory() {
    if (selectedJourney && selectedMementoIndex < selectedJourney.mementos.length - 1)
      selectedMementoIndex += 1
  }

  function handleKeydown(event: KeyboardEvent) {
    if (!selectedMemento) return
    if (event.key === 'ArrowLeft') {
      event.preventDefault()
      previousMemory()
    } else if (event.key === 'ArrowRight') {
      event.preventDefault()
      nextMemory()
    }
  }

  $effect(() => loadData())
</script>

<svelte:window on:keydown={handleKeydown} />

<main class="atlas" class:light={theme === 'light'}>
  {#if isLoading}
    <div class="atlas-loading">{label.loading}</div>
  {:else if error}
    <div class="atlas-error">
      <p>{error}</p>
      <button onclick={loadData}>{label.retry}</button>
    </div>
  {:else}
    <section class="atlas-map" aria-label={label.title}>
      <div class="atlas-heading">
        <p class="eyebrow">F E L I C I A / ATLAS</p>
        <h1>{label.title}</h1>
        <p>{label.subtitle}</p>
      </div>
      <svg class="world-map" viewBox="0 0 100 100" role="img" aria-label={label.title}>
        <defs>
          <pattern id="atlas-grid" width="5" height="5" patternUnits="userSpaceOnUse">
            <path
              d="M 5 0 L 0 0 0 5"
              fill="none"
              stroke="currentColor"
              stroke-opacity=".13"
              stroke-width=".12"
            />
          </pattern>
        </defs>
        <rect width="100" height="100" fill="url(#atlas-grid)" />
        {#each journeys as journey, index (journey.id)}
          <polyline
            class:active={index === selectedIndex}
            class="atlas-route"
            points={routePoints(journey)}
          />
          {@const first = journey.route[0]}
          {@const point = world(first)}
          <circle
            class:active={index === selectedIndex}
            class="atlas-dot"
            cx={point.x}
            cy={point.y}
            r={index === selectedIndex ? 1.1 : 0.65}
            role="button"
            tabindex="0"
            aria-label={t(journey.title)}
            onclick={() => selectJourney(index)}
            onkeydown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') selectJourney(index)
            }}
          />
        {/each}
      </svg>
      <div class="map-note">
        {journeys.length}
        {label.journeys} · {journeys.reduce((sum, journey) => sum + journey.mementos.length, 0)}
        {label.memories}
      </div>
    </section>

    <aside class="atlas-rail" aria-label={label.title}>
      <div class="rail-head">
        <span>{journeys.length} {label.journeys}</span>
        <span>← →</span>
      </div>
      <div class="journey-list">
        {#each journeys as journey, index (journey.id)}
          <button
            class:active={index === selectedIndex}
            class="journey-row"
            onclick={() => selectJourney(index)}
          >
            <span class="row-number">{String(index + 1).padStart(2, '0')}</span>
            <span class="row-copy"
              ><strong>{t(journey.title)}</strong><small
                >{t(journey.dates)} · {t(journey.place)}</small
              ></span
            >
            <span class="row-count">{journey.mementos.length}</span>
          </button>
        {/each}
      </div>

      {#if selectedJourney && selectedMemento}
        <section
          class="memory-card"
          aria-label={t(selectedMemento.title)}
          aria-keyshortcuts="ArrowLeft ArrowRight"
        >
          <div class="memory-toolbar">
            <span>{selectedMementoIndex + 1} / {selectedJourney.mementos.length}</span>
            <span>{t(selectedJourney.title)}</span>
          </div>
          <p class="eyebrow">{selectedMemento.kind}</p>
          <h2>{t(selectedMemento.title)}</h2>
          <p class="memory-meta">{t(selectedMemento.place)} · {t(selectedMemento.date)}</p>
          <p class="memory-essay">{t(selectedMemento.essay)}</p>
          <div class="memory-nav">
            <button disabled={selectedMementoIndex === 0} onclick={previousMemory}
              >← {label.previous}</button
            >
            <button
              disabled={selectedMementoIndex === selectedJourney.mementos.length - 1}
              onclick={nextMemory}>{label.next} →</button
            >
          </div>
        </section>
      {/if}
    </aside>
  {/if}
</main>

<style>
  .atlas {
    --ink: #f1eadc;
    --muted: #a69e90;
    --line: rgba(241, 234, 220, 0.18);
    --accent: #f28b35;
    display: grid;
    grid-template-columns: minmax(0, 1fr) 27rem;
    min-height: 100%;
    background: #171614;
    color: var(--ink);
    font-family: 'Zen Old Mincho', Georgia, serif;
  }
  .atlas.light {
    background: #eee5d5;
    color: #332a20;
    --muted: #766c5d;
    --line: rgba(51, 42, 32, 0.18);
  }
  .atlas-map {
    position: relative;
    min-height: 100vh;
    overflow: hidden;
    background: radial-gradient(circle at 48% 45%, #27231d, #11100f 75%);
  }
  .light .atlas-map {
    background: radial-gradient(circle at 48% 45%, #fff8ea, #ded0b9 75%);
  }
  .atlas-heading {
    position: absolute;
    z-index: 2;
    top: 2rem;
    left: 2.5rem;
    max-width: 24rem;
  }
  .eyebrow {
    margin: 0 0 0.55rem;
    color: var(--accent);
    font:
      0.68rem/1.2 ui-monospace,
      SFMono-Regular,
      monospace;
    letter-spacing: 0.22em;
    text-transform: uppercase;
  }
  h1,
  h2,
  p {
    margin-top: 0;
  }
  h1 {
    margin-bottom: 0.35rem;
    font-size: clamp(2.2rem, 4vw, 4.6rem);
    line-height: 0.98;
  }
  .atlas-heading p:last-child {
    color: var(--muted);
    font-size: 1rem;
  }
  .world-map {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    padding: 9rem 3rem 3rem;
    box-sizing: border-box;
    color: var(--line);
  }
  .atlas-route {
    fill: none;
    stroke: #d6b18c;
    stroke-width: 0.22;
    stroke-dasharray: 0.8 0.7;
    opacity: 0.28;
    cursor: pointer;
    transition:
      stroke 0.2s,
      opacity 0.2s,
      stroke-width 0.2s;
  }
  .atlas-route.active {
    stroke: var(--accent);
    stroke-width: 0.55;
    opacity: 1;
  }
  .atlas-dot {
    fill: #d6b18c;
    opacity: 0.65;
    cursor: pointer;
  }
  .atlas-dot.active {
    fill: var(--accent);
    opacity: 1;
  }
  .map-note {
    position: absolute;
    bottom: 1.75rem;
    left: 2.5rem;
    color: var(--muted);
    font:
      0.7rem ui-monospace,
      SFMono-Regular,
      monospace;
    letter-spacing: 0.08em;
  }
  .atlas-rail {
    display: flex;
    min-height: 100vh;
    flex-direction: column;
    border-left: 1px solid var(--line);
    background: rgba(22, 20, 17, 0.92);
  }
  .light .atlas-rail {
    background: rgba(255, 250, 240, 0.94);
  }
  .rail-head {
    display: flex;
    justify-content: space-between;
    padding: 1.6rem 1.4rem 1rem;
    color: var(--muted);
    font:
      0.68rem ui-monospace,
      SFMono-Regular,
      monospace;
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }
  .journey-list {
    flex: 1;
    overflow-y: auto;
    padding: 0 0.75rem 0.75rem;
  }
  .journey-row {
    display: grid;
    width: 100%;
    grid-template-columns: 2rem 1fr auto;
    gap: 0.75rem;
    align-items: center;
    padding: 1rem 0.7rem;
    border: 0;
    border-top: 1px solid var(--line);
    background: transparent;
    color: inherit;
    cursor: pointer;
    text-align: left;
    font: inherit;
  }
  .journey-row:hover,
  .journey-row.active {
    background: rgba(242, 139, 53, 0.1);
  }
  .row-number {
    color: var(--accent);
    font:
      0.7rem ui-monospace,
      SFMono-Regular,
      monospace;
  }
  .row-copy {
    display: grid;
    gap: 0.25rem;
    min-width: 0;
  }
  .row-copy strong {
    overflow: hidden;
    font-size: 1rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row-copy small {
    overflow: hidden;
    color: var(--muted);
    font:
      0.68rem ui-monospace,
      SFMono-Regular,
      monospace;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row-count {
    color: var(--muted);
    font:
      0.7rem ui-monospace,
      SFMono-Regular,
      monospace;
  }
  .memory-card {
    margin: 0 0.75rem 0.75rem;
    padding: 1.25rem;
    border: 1px solid var(--line);
    border-radius: 0.65rem;
    background: rgba(255, 250, 240, 0.07);
  }
  .light .memory-card {
    background: rgba(255, 255, 255, 0.55);
  }
  .memory-toolbar {
    display: flex;
    justify-content: space-between;
    gap: 0.75rem;
    margin-bottom: 1.2rem;
    color: var(--muted);
    font:
      0.65rem ui-monospace,
      SFMono-Regular,
      monospace;
  }
  .memory-card h2 {
    margin-bottom: 0.35rem;
    font-size: 1.5rem;
  }
  .memory-meta {
    color: var(--muted);
    font:
      0.7rem ui-monospace,
      SFMono-Regular,
      monospace;
  }
  .memory-essay {
    color: var(--muted);
    font-size: 0.9rem;
    line-height: 1.65;
  }
  .memory-nav {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    margin-top: 1.3rem;
  }
  .memory-nav button {
    border: 1px solid var(--line);
    border-radius: 0.3rem;
    padding: 0.55rem 0.65rem;
    background: transparent;
    color: inherit;
    cursor: pointer;
    font:
      0.7rem ui-monospace,
      SFMono-Regular,
      monospace;
  }
  .memory-nav button:disabled {
    cursor: not-allowed;
    opacity: 0.3;
  }
  .atlas-loading,
  .atlas-error {
    display: grid;
    min-height: 100%;
    place-items: center;
    background: #171614;
    color: var(--ink);
  }
  .atlas-error {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .atlas-error button {
    padding: 0.6rem 1rem;
  }
  @media (max-width: 900px) {
    .atlas {
      display: block;
    }
    .atlas-map {
      min-height: 52vh;
    }
    .atlas-rail {
      min-height: 48vh;
      border-top: 1px solid var(--line);
      border-left: 0;
    }
    .world-map {
      padding: 7rem 1rem 1rem;
    }
    .atlas-heading {
      top: 1.25rem;
      left: 1.25rem;
    }
    .map-note {
      bottom: 1rem;
      left: 1.25rem;
    }
  }
</style>
