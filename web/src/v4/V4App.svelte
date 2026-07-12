<script lang="ts">
  import V4Detail from './V4Detail.svelte'
  import V4Map from './V4Map.svelte'
  import V4Stub from './V4Stub.svelte'
  import { loadJourneys } from '../api/source'
  import type { Journey, Lang, Memento, Theme } from '../data'

  let { lang = $bindable('ja'), theme = $bindable('dark') }: { lang?: Lang; theme?: Theme } =
    $props()
  let journeys = $state<Journey[]>([])
  let newestFirst = $state(true)
  let isLoading = $state(true)
  let error = $state<string | null>(null)
  let activeJourneyId = $state<string | null>(null)
  let activeMementoId = $state<string | null>(null)
  let selectedMementoId = $state<string | null>(null)

  const orderedJourneys = $derived(
    [...journeys]
      .sort((left, right) => {
        const comparison = journeyStart(left).localeCompare(journeyStart(right))
        return newestFirst ? -comparison : comparison
      })
      .map((journey) => ({
        journey,
        mementos: [...journey.mementos].sort((left, right) => {
          const comparison = left.date.en.localeCompare(right.date.en)
          return newestFirst ? -comparison : comparison
        }),
      })),
  )
  const activeJourney = $derived(
    journeys.find((journey) => journey.id === activeJourneyId) ?? journeys[0] ?? null,
  )
  const selectedMemento = $derived(
    activeJourney?.mementos.find((memento) => memento.id === selectedMementoId) ?? null,
  )

  const ui = {
    ja: {
      title: '旅の地図帳',
      subtitle: '旅の記憶を、チケットのかたちで',
      newest: 'Newest',
      oldest: 'Oldest',
      journeys: '旅',
      photos: 'photos',
      loading: '読み込み中…',
      retry: '再試行',
      close: '閉じる',
      story: 'Story',
    },
    en: {
      title: "Felicia's Waypoints",
      subtitle: 'A travel journal in memento stubs',
      newest: 'Newest',
      oldest: 'Oldest',
      journeys: 'journeys',
      photos: 'photos',
      loading: 'Loading…',
      retry: 'Retry',
      close: 'Close',
      story: 'Story',
    },
    zh: {
      title: '旅行图册',
      subtitle: '以纪念物记录旅途',
      newest: 'Newest',
      oldest: 'Oldest',
      journeys: '次旅程',
      photos: 'photos',
      loading: '加载中…',
      retry: '重试',
      close: '关闭',
      story: '故事',
    },
  } as const

  const t = (value: { ja: string; en: string; zh: string }) => value[lang]
  const label = $derived(ui[lang])

  function journeyStart(journey: Journey) {
    return journey.dates.en.slice(0, 10)
  }

  function loadData() {
    isLoading = true
    error = null
    loadJourneys()
      .then((data) => {
        journeys = data
        activeJourneyId = data[0]?.id ?? null
        activeMementoId = data[0]?.mementos[0]?.id ?? null
      })
      .catch((reason) => {
        error = reason instanceof Error ? reason.message : String(reason)
      })
      .finally(() => {
        isLoading = false
      })
  }

  function observeJourney(node: HTMLElement, journey: Journey) {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) activeJourneyId = journey.id
      },
      { rootMargin: '-35% 0px -55% 0px', threshold: 0 },
    )
    observer.observe(node)
    return { destroy: () => observer.disconnect() }
  }

  function observeMemento(node: HTMLElement, memento: Memento) {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          activeMementoId = memento.id
          activeJourneyId = node.dataset.journeyId ?? activeJourneyId
        }
      },
      { rootMargin: '-42% 0px -42% 0px', threshold: 0 },
    )
    observer.observe(node)
    return { destroy: () => observer.disconnect() }
  }

  function selectMemento(memento: Memento, journeyId: string) {
    activeJourneyId = journeyId
    activeMementoId = memento.id
    selectedMementoId = memento.id
  }

  function closeMemento() {
    selectedMementoId = null
  }

  $effect(() => loadData())
</script>

<main class="waypoints" class:light={theme === 'light'}>
  {#if isLoading}
    <div class="status" role="status">{label.loading}</div>
  {:else if error}
    <div class="status status-error" role="alert">
      <p>{error}</p>
      <button type="button" onclick={loadData}>{label.retry}</button>
    </div>
  {:else if orderedJourneys.length}
    <div class="map-surface" aria-hidden="true">
      <V4Map
        {journeys}
        {activeJourneyId}
        {activeMementoId}
        {theme}
        onSelect={(id) => (selectedMementoId = id)}
      />
    </div>

    <header class="hero">
      <p class="brand">F E L I C I A / ATLAS</p>
      <h1>{label.title}</h1>
      <p>{label.subtitle}</p>
      <nav class="social" aria-label="Links">
        <a href="https://github.com" aria-label="GitHub">◉</a>
        <a href="https://x.com" aria-label="X">𝕏</a>
        <a href="https://telegram.org" aria-label="Telegram">➤</a>
        <a href="mailto:hello@example.com" aria-label="Email">✉</a>
      </nav>
      <div class="sort" role="group" aria-label="Sort journeys">
        <button
          type="button"
          class:active={newestFirst}
          aria-pressed={newestFirst}
          onclick={() => (newestFirst = true)}>{label.newest}</button
        >
        <button
          type="button"
          class:active={!newestFirst}
          aria-pressed={!newestFirst}
          onclick={() => (newestFirst = false)}>{label.oldest}</button
        >
      </div>
    </header>

    <div class="journey-stream">
      {#each orderedJourneys as entry (entry.journey.id)}
        <section
          class:active={entry.journey.id === activeJourneyId}
          class="journey-section"
          use:observeJourney={entry.journey}
          data-journey-id={entry.journey.id}
        >
          <header class="journey-heading">
            <p class="journey-number">
              {String(orderedJourneys.indexOf(entry) + 1).padStart(2, '0')}
            </p>
            <h2>{t(entry.journey.title)}</h2>
            <p>{t(entry.journey.place)} · {t(entry.journey.dates)}</p>
          </header>

          <div class="stub-field" aria-label={t(entry.journey.title)}>
            {#each entry.mementos as memento (memento.id)}
              <div
                class:active={memento.id === activeMementoId}
                class="memento-entry"
                data-journey-id={entry.journey.id}
                use:observeMemento={memento}
              >
                <V4Stub
                  {memento}
                  {lang}
                  photoLabel={label.photos}
                  selected={memento.id === selectedMementoId}
                  onSelect={() => selectMemento(memento, entry.journey.id)}
                />
              </div>
            {/each}
          </div>
        </section>
      {/each}
    </div>

    {#if selectedMemento}
      <V4Detail
        memento={selectedMemento}
        {lang}
        photoLabel={label.photos}
        closeLabel={label.close}
        storyLabel={label.story}
        onClose={closeMemento}
      />
    {/if}
  {:else}
    <div class="status">{label.journeys}: 0</div>
  {/if}
</main>

<style>
  .waypoints {
    --ink: #f5f5f5;
    --muted: #a8a8a8;
    --orange: #d46728;
    min-height: 100%;
    height: 100%;
    overflow-y: auto;
    overscroll-behavior-y: contain;
    background: #121212;
    color: var(--ink);
    font-family: Inter, ui-sans-serif, system-ui, sans-serif;
  }

  .waypoints.light {
    --ink: #2d2925;
    --muted: #706a65;
    --orange: #b45f26;
    background: #e7e0d5;
  }

  .map-surface {
    position: fixed;
    z-index: 0;
    inset: 0;
    pointer-events: none;
  }

  .map-surface :global(.maplibregl-map) {
    position: absolute;
    inset: 0;
  }

  .hero,
  .journey-stream {
    position: relative;
    z-index: 1;
  }

  .hero {
    display: flex;
    align-items: center;
    flex-direction: column;
    min-height: 100vh;
    padding: clamp(7rem, 18vh, 12rem) 1rem 5rem;
    text-align: center;
  }

  .brand,
  .journey-number {
    margin: 0 0 0.8rem;
    color: var(--orange);
    font-size: 0.68rem;
    letter-spacing: 0.18em;
  }

  .hero h1 {
    margin: 0;
    font-size: clamp(2.8rem, 5vw, 5rem);
    font-weight: 800;
    letter-spacing: -0.06em;
    text-shadow: 0 5px 18px #000;
  }

  .hero > p:not(.brand),
  .journey-heading > p:last-child {
    margin: 1rem 0 0;
    color: var(--muted);
    font-size: clamp(1rem, 1.7vw, 1.35rem);
  }

  .social {
    display: flex;
    gap: 1.35rem;
    margin-top: 1.25rem;
  }

  .social a {
    color: var(--muted);
    font-size: 1.35rem;
    text-decoration: none;
  }

  .sort {
    display: flex;
    margin-top: 1.5rem;
    padding: 0.2rem;
    border: 1px solid #444;
    border-radius: 0.7rem;
    background: #353535b8;
  }

  .sort button {
    padding: 0.55rem 1.15rem;
    border: 0;
    border-radius: 0.5rem;
    color: var(--muted);
    background: transparent;
  }

  .sort button.active {
    color: #fff;
    background: #686868;
  }

  .journey-section {
    max-width: 80rem;
    min-height: 100vh;
    margin: 0 auto;
    padding: 7rem 1.25rem 9rem;
    opacity: 0.62;
    transition: opacity 240ms ease;
  }

  .journey-section.active {
    opacity: 1;
  }

  .journey-heading {
    margin: 0 auto 3rem;
    text-align: center;
  }

  .journey-heading h2 {
    margin: 0;
    font-size: clamp(2.2rem, 4vw, 4rem);
    text-shadow: 0 4px 15px #000;
  }

  .journey-heading p:last-child {
    margin-bottom: 0;
  }

  .stub-field {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 2rem 1.5rem;
  }

  .memento-entry {
    scroll-margin-top: 38vh;
    opacity: 0.72;
    transition:
      opacity 180ms ease,
      transform 180ms ease;
  }

  .memento-entry.active {
    opacity: 1;
    transform: translateY(-0.25rem);
  }

  .status {
    display: grid;
    min-height: 100vh;
    place-items: center;
    color: var(--muted);
  }

  .status-error {
    gap: 1rem;
    align-content: center;
  }

  .status button {
    padding: 0.6rem 1rem;
    border: 1px solid #555;
    border-radius: 0.4rem;
    color: var(--ink);
    background: transparent;
  }

  @media (max-width: 640px) {
    .hero {
      min-height: 100svh;
      padding-top: 8rem;
    }

    .hero h1 {
      max-width: 18rem;
      line-height: 1.05;
    }

    .journey-section {
      padding-top: 5rem;
    }

    .stub-field {
      display: grid;
      grid-template-columns: 1fr;
    }

    .memento-entry :global(.stub) {
      width: min(100%, 25rem);
      margin-inline: auto;
    }
  }
</style>
