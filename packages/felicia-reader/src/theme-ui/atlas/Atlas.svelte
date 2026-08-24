<script lang="ts">
  import AtlasDetail from "./AtlasDetail.svelte"
  import AtlasMap from "./AtlasMap.svelte"
  import AtlasStub from "./AtlasStub.svelte"
  import type { Journey, Lang, Memento, Theme } from "@felicia/model"

  let { lang = $bindable("ja"), theme = $bindable("dark"), loadJourneys }: { lang?: Lang; theme?: Theme; loadJourneys: () => Promise<Journey[]> } = $props()
  let journeys = $state<Journey[]>([])
  let newestFirst = $state(true)
  let isLoading = $state(true)
  let error = $state<string | null>(null)
  let activeJourneyId = $state<string | null>(null)
  let activeMementoId = $state<string | null>(null)
  let selectedMementoId = $state<string | null>(null)
  let indexOpen = $state(true)

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
  const activeJourney = $derived(journeys.find((journey) => journey.id === activeJourneyId) ?? journeys[0] ?? null)
  const selectedMemento = $derived(activeJourney?.mementos.find((memento) => memento.id === selectedMementoId) ?? null)

  const ui = {
    ja: {
      title: "旅の地図帳",
      subtitle: "旅の記憶を、チケットのかたちで",
      newest: "Newest",
      oldest: "Oldest",
      journeys: "旅",
      mementos: "件",
      photos: "photos",
      photosHeading: "写真",
      indexOpen: "目録",
      indexClose: "目録を閉じる",
      loading: "読み込み中…",
      retry: "再試行",
      close: "閉じる",
      story: "Story",
    },
    en: {
      title: "Felicia's Waypoints",
      subtitle: "A travel journal in memento stubs",
      newest: "Newest",
      oldest: "Oldest",
      journeys: "journeys",
      mementos: "mementos",
      photos: "photos",
      photosHeading: "Photos",
      indexOpen: "Index",
      indexClose: "Hide index",
      loading: "Loading…",
      retry: "Retry",
      close: "Close",
      story: "Story",
    },
    zh: {
      title: "旅行图册",
      subtitle: "以纪念物记录旅途",
      newest: "Newest",
      oldest: "Oldest",
      journeys: "次旅程",
      mementos: "件纪念物",
      photos: "photos",
      photosHeading: "照片",
      indexOpen: "目录",
      indexClose: "关闭目录",
      loading: "加载中…",
      retry: "重试",
      close: "关闭",
      story: "故事",
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
      { rootMargin: "-35% 0px -55% 0px", threshold: 0 },
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
      { rootMargin: "-42% 0px -42% 0px", threshold: 0 },
    )
    observer.observe(node)
    return { destroy: () => observer.disconnect() }
  }

  function selectMemento(memento: Memento, journeyId: string) {
    activeJourneyId = journeyId
    activeMementoId = memento.id
    selectedMementoId = memento.id
  }

  function selectMementoById(mementoId: string) {
    for (const journey of journeys) {
      const memento = journey.mementos.find((item) => item.id === mementoId)
      if (memento) {
        selectMemento(memento, journey.id)
        return
      }
    }
  }

  function jumpToJourney(journeyId: string) {
    activeJourneyId = journeyId
    const target = document.querySelector<HTMLElement>(`[data-journey-id="${journeyId}"]`)
    target?.scrollIntoView({ behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth", block: "start" })
  }

  function closeMemento() {
    selectedMementoId = null
  }

  $effect(() => loadData())
</script>

<main class="waypoints" class:light={theme === "light"}>
  {#if isLoading}
    <div class="status" role="status">{label.loading}</div>
  {:else if error}
    <div class="status status-error" role="alert">
      <p>{error}</p>
      <button type="button" onclick={loadData}>{label.retry}</button>
    </div>
  {:else if orderedJourneys.length}
    <div class="map-surface" aria-label="Journey map">
      <AtlasMap {journeys} {activeJourneyId} {activeMementoId} {lang} {theme} onSelect={selectMementoById} />
    </div>

    <button class="index-toggle" type="button" aria-expanded={indexOpen} onclick={() => (indexOpen = !indexOpen)}>
      <span aria-hidden="true">{indexOpen ? "−" : "+"}</span>
      {indexOpen ? label.indexClose : label.indexOpen}
    </button>

    {#if indexOpen}
      <aside class="atlas-index" aria-label={label.journeys}>
        <div class="atlas-index-head">
          <div>
            <p class="index-kicker">FELICIA / ATLAS</p>
            <h2>{label.journeys}</h2>
          </div>
          <span class="index-total">{orderedJourneys.length}</span>
        </div>
        <div class="atlas-index-list">
          {#each orderedJourneys as entry (entry.journey.id)}
            <button class:active={entry.journey.id === activeJourneyId} class="atlas-journey" type="button" onclick={() => jumpToJourney(entry.journey.id)}>
              <span class="atlas-journey-title">{t(entry.journey.title)}</span>
              <span class="atlas-journey-meta">{t(entry.journey.place)} · {t(entry.journey.dates)}</span>
              <span class="atlas-journey-count">{entry.mementos.length} {label.mementos}</span>
            </button>
          {/each}
        </div>
        {#if activeJourney}
          <div class="atlas-index-mementos">
            <p class="index-kicker">{t(activeJourney.title)}</p>
            {#each activeJourney.mementos.slice(0, 4) as memento, index (memento.id)}
              <button class:active={memento.id === activeMementoId} type="button" onclick={() => selectMemento(memento, activeJourney.id)}>
                <span>{String(index + 1).padStart(2, "0")}</span>
                <strong>{t(memento.title)}</strong>
              </button>
            {/each}
          </div>
        {/if}
      </aside>
    {/if}

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
        <button type="button" class:active={newestFirst} aria-pressed={newestFirst} onclick={() => (newestFirst = true)}>{label.newest}</button>
        <button type="button" class:active={!newestFirst} aria-pressed={!newestFirst} onclick={() => (newestFirst = false)}>{label.oldest}</button>
      </div>
    </header>

    <div class="journey-stream">
      {#each orderedJourneys as entry (entry.journey.id)}
        <section class:active={entry.journey.id === activeJourneyId} class="journey-section" use:observeJourney={entry.journey} data-journey-id={entry.journey.id}>
          <header class="journey-heading">
            <p class="journey-number">
              {String(orderedJourneys.indexOf(entry) + 1).padStart(2, "0")}
            </p>
            <h2>{t(entry.journey.title)}</h2>
            <p>{t(entry.journey.place)} · {t(entry.journey.dates)}</p>
          </header>

          <div class="stub-field" aria-label={t(entry.journey.title)}>
            {#each entry.mementos as memento (memento.id)}
              <div class:active={memento.id === activeMementoId} class="memento-entry" data-journey-id={entry.journey.id} use:observeMemento={memento}>
                <AtlasStub {memento} {lang} photoLabel={label.photos} selected={memento.id === selectedMementoId} onSelect={() => selectMemento(memento, entry.journey.id)} />
              </div>
            {/each}
          </div>
        </section>
      {/each}
    </div>

    {#if selectedMemento}
      <AtlasDetail memento={selectedMemento} {lang} photoLabel={label.photos} photosHeading={label.photosHeading} closeLabel={label.close} storyLabel={label.story} onClose={closeMemento} />
    {/if}
  {:else}
    <div class="status">{label.journeys}: 0</div>
  {/if}
</main>

<style>
  .waypoints {
    --ink: #f5f5f5;
    --muted: #a8a8a8;
    --orange: var(--accent, #d46728);
    --waypoints-bg: #121212;
    min-height: 100%;
    height: 100%;
    overflow-y: auto;
    overscroll-behavior-y: contain;
    background: #121212;
    color: var(--ink);
    font-family: Outfit, ui-sans-serif, system-ui, sans-serif;
  }

  .waypoints.light {
    --ink: #2d2925;
    --muted: #706a65;
    --orange: var(--accent, #b45f26);
    --waypoints-bg: #e7e0d5;
    background: #e7e0d5;
  }

  .map-surface {
    position: fixed;
    z-index: 0;
    inset: 0;
    pointer-events: auto;
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

  .index-toggle {
    position: fixed;
    z-index: 5;
    bottom: 1.25rem;
    left: 1.25rem;
    display: flex;
    min-height: 2.75rem;
    align-items: center;
    gap: 0.5rem;
    padding: 0.65rem 0.9rem;
    border: 1px solid color-mix(in srgb, var(--ink) 30%, transparent);
    border-radius: 999px;
    color: var(--ink);
    background: color-mix(in srgb, var(--waypoints-bg) 82%, transparent);
    box-shadow: 0 0.75rem 2rem #0005;
    backdrop-filter: blur(12px);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .index-toggle span {
    display: grid;
    width: 1.25rem;
    height: 1.25rem;
    place-items: center;
    border: 1px solid currentColor;
    border-radius: 50%;
    font-size: 1rem;
    line-height: 1;
  }

  .atlas-index {
    position: fixed;
    z-index: 4;
    top: 5.5rem;
    left: 1.25rem;
    display: flex;
    width: min(21rem, calc(100vw - 2.5rem));
    max-height: min(70vh, 36rem);
    flex-direction: column;
    gap: 1rem;
    overflow-y: auto;
    padding: 1.15rem;
    border: 1px solid color-mix(in srgb, var(--ink) 20%, transparent);
    border-radius: 1rem;
    color: var(--ink);
    background: color-mix(in srgb, var(--waypoints-bg) 88%, transparent);
    box-shadow: 0 1.5rem 4rem #0007;
    backdrop-filter: blur(18px) saturate(120%);
  }

  .atlas-index-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }

  .atlas-index h2 {
    margin: 0.15rem 0 0;
    font-size: 1.4rem;
    letter-spacing: -0.04em;
  }

  .index-kicker {
    margin: 0;
    color: var(--orange);
    font-size: 0.62rem;
    font-weight: 700;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }

  .index-total {
    display: grid;
    min-width: 2rem;
    height: 2rem;
    place-items: center;
    border: 1px solid color-mix(in srgb, var(--ink) 24%, transparent);
    border-radius: 50%;
    color: var(--orange);
    font:
      700 0.75rem/1 ui-monospace,
      monospace;
  }

  .atlas-index-list,
  .atlas-index-mementos {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .atlas-journey,
  .atlas-index-mementos button {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.2rem;
    width: 100%;
    padding: 0.7rem;
    border: 1px solid transparent;
    border-radius: 0.6rem;
    color: var(--ink);
    background: transparent;
    text-align: left;
  }

  .atlas-journey:hover,
  .atlas-journey.active,
  .atlas-index-mementos button:hover,
  .atlas-index-mementos button.active {
    border-color: color-mix(in srgb, var(--orange) 65%, transparent);
    background: color-mix(in srgb, var(--orange) 18%, transparent);
  }

  .atlas-journey-title,
  .atlas-index-mementos strong {
    font-size: 0.88rem;
    font-weight: 700;
  }

  .atlas-journey-meta,
  .atlas-journey-count {
    color: var(--muted);
    font-size: 0.7rem;
  }

  .atlas-index-mementos {
    border-top: 1px solid color-mix(in srgb, var(--ink) 16%, transparent);
    padding-top: 0.9rem;
  }

  .atlas-index-mementos button {
    display: grid;
    grid-template-columns: 2rem minmax(0, 1fr);
    align-items: center;
    gap: 0.5rem;
  }

  .atlas-index-mementos button > span {
    color: var(--orange);
    font:
      700 0.68rem/1 ui-monospace,
      monospace;
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
    .atlas-index {
      top: auto;
      right: 0.75rem;
      bottom: 4.75rem;
      left: 0.75rem;
      width: auto;
      max-height: min(52vh, 27rem);
    }

    .index-toggle {
      right: 1rem;
      bottom: 1rem;
      left: auto;
    }

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
