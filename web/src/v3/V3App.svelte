<script lang="ts">
  import { journeys, kindLabel, type Coordinates, type L, type Lang, type Memento, type Theme } from '../data';
  import { cityDots, project } from './cityLookup';
  import TripMap from './TripMap.svelte';

  // v3 — the "techo" (手帳, paper notebook) front door. View 1 (landing) is the
  // journal index — a two-page spread with a paper sketch map. View 2 (detail)
  // is the journey on a REAL map (reused v1 MapLibre): mementos cluster by place
  // (map is the index — felicia:decision:map-first-landing), and opening a place
  // reveals its memories (a place holds several) as paper cards with essay +
  // gallery. Styled with Tailwind (felicia:decision:techo-paper-v3).
  let {
    lang = $bindable('ja'),
    theme = $bindable('dark'),
  }: { lang?: Lang; theme?: Theme } = $props();

  let selectedIndex = $state(0);
  let view = $state<'landing' | 'detail'>('landing');
  let selectedPlaceKey = $state<string | null>(null);

  const selectedJourney = $derived(journeys[selectedIndex]);

  function t(value: L): string {
    return value[lang];
  }

  // Group a journey's mementos by place — one map marker per place, and a place
  // can hold several memories.
  interface PlaceGroup {
    key: string;
    coords: Coordinates;
    seq: number;
    mementos: Memento[];
  }

  const placeGroups = $derived.by(() => {
    const groups = new Map<string, PlaceGroup>();
    selectedJourney.mementos.forEach(memento => {
      const key = memento.place.ja || memento.id;
      let group = groups.get(key);
      if (!group) {
        group = { key, coords: memento.coords, seq: groups.size + 1, mementos: [] };
        groups.set(key, group);
      }
      group.mementos.push(memento);
    });
    return Array.from(groups.values());
  });

  const mapPlaces = $derived(
    placeGroups.map(group => ({ key: group.key, coords: group.coords, seq: group.seq, count: group.mementos.length }))
  );

  const selectedPlace = $derived(placeGroups.find(group => group.key === selectedPlaceKey) ?? null);

  const transitPairs = $derived(
    selectedJourney.mementos
      .filter(memento => memento.transit)
      .map(memento => [memento.transit!.from.coords, memento.transit!.to.coords] as [Coordinates, Coordinates])
  );

  function selectJourney(index: number) {
    selectedIndex = index;
  }

  function openJourney() {
    selectedPlaceKey = placeGroups[0]?.key ?? null;
    view = 'detail';
  }

  function backToLanding() {
    view = 'landing';
  }

  function selectPlace(key: string) {
    selectedPlaceKey = key;
  }

  function onPhotoError(event: Event) {
    (event.currentTarget as HTMLImageElement).style.display = 'none';
  }

  // Washi tape texture: a small fixed palette cycled by card index.
  const washiColors = [
    'rgba(200, 120, 60, 0.35)',
    'rgba(120, 150, 108, 0.35)',
    'rgba(120, 138, 188, 0.32)',
    'rgba(196, 150, 88, 0.35)'
  ];
  const washiRotations = [-4, 3, -2, 5];

  const years = $derived.by(() => {
    const set = new Set<string>();
    for (const journey of journeys) {
      const match = journey.dates.ja.match(/(\d{4})年/);
      set.add(match ? match[1] : '2026');
    }
    return Array.from(set).sort((a, b) => Number(b) - Number(a));
  });

  const activeYear = $derived.by(() => {
    const match = selectedJourney.dates.ja.match(/(\d{4})年/);
    return match ? match[1] : years[0];
  });

  const journeyCountLabel = $derived.by(() => {
    const n = journeys.length;
    if (lang === 'en') return `${n} journeys`;
    if (lang === 'zh') return `${n}次旅程`;
    return `${n}つの旅`;
  });

  function mementoCountLabel(n: number): string {
    if (lang === 'en') return `${n} stop${n === 1 ? '' : 's'}`;
    if (lang === 'zh') return `${n}件`;
    return `${n}件`;
  }

  function placeMemoriesLabel(n: number): string {
    if (lang === 'en') return `${n} memor${n === 1 ? 'y' : 'ies'} here`;
    if (lang === 'zh') return `此地 ${n} 件记忆`;
    return `この場所の記憶 ${n}件`;
  }

  // Landing sketch map: dim every journey's city dots, brighten the selected.
  const journeyCityDots = $derived(cityDots.filter(dot => dot.journeyId === selectedJourney.id));
  const routePoints = $derived(journeyCityDots.map(dot => project(dot.coords)));
  const routePath = $derived(routePoints.map(p => `${p.x},${p.y}`).join(' '));

  const mapCaption = { ja: '地図の印をえらぶと、その旅がひらきます', en: 'Choose a mark on the map to open that journey', zh: '在地图上选择标记，即可打开这段旅程' } satisfies L;
  const seasonCaption = { ja: '冬－春の記録', en: 'Winter–spring notes', zh: '冬–春记录' } satisfies L;
  const brandTagline = { ja: 'Travel journal', en: 'Travel journal', zh: 'Travel journal' } satisfies L;
  const selectedBadge = { ja: '選択中', en: 'Selected', zh: '已选' } satisfies L;
  const openCta = { ja: 'この旅をひらく →', en: 'Open this journey →', zh: '打开这段旅程 →' } satisfies L;
  const backLabel = { ja: '手帳に戻る', en: 'Back to journal', zh: '返回手帳' } satisfies L;
</script>

<main class="techo-shell" class:theme-light={theme === 'light'} class:is-detail={view === 'detail'}>
  {#if view === 'landing'}
    <!-- View 1: the journal index — sketch map on the left, journey cards on the right. -->
    <div class="techo-frame">
      <div class="techo-spread">
        <section class="techo-page techo-page--map" aria-label="Journey map">
          <div class="map-grid">
            <svg class="map-svg" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
              {#if routePoints.length > 1}
                <polyline class="map-route" points={routePath} />
              {/if}
            </svg>
            {#each cityDots as dot (dot.id)}
              {@const p = project(dot.coords)}
              {@const active = dot.journeyId === selectedJourney.id}
              <div class="map-dot" class:active style="left:{p.x}%; top:{p.y}%">
                <span class="map-dot-mark"></span>
                <span class="map-dot-label">{dot.label}</span>
              </div>
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

          <ol class="journey-cards">
            {#each journeys as journey, index (journey.id)}
              <li>
                <button
                  type="button"
                  class="journey-card"
                  class:selected={index === selectedIndex}
                  onclick={() => selectJourney(index)}
                >
                  <span
                    class="washi-tape"
                    style="background:{washiColors[index % washiColors.length]}; transform: translate(-50%, -55%) rotate({washiRotations[
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

          <button type="button" class="open-cta" onclick={openJourney}>{t(openCta)}</button>
        </section>

        <nav class="year-tabs" aria-label="Years">
          {#each years as year (year)}
            <span class="year-tab" class:active={year === activeYear}>{year}</span>
          {/each}
        </nav>
      </div>
    </div>
  {:else}
    <!-- View 2: the journey on a real map; mementos cluster by place, opening a
         place reveals its memories. -->
    <section class="relative h-full w-full overflow-hidden bg-paper-2" aria-label={t(selectedJourney.title)}>
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
          <h1 class="m-0 mt-1 font-mincho text-2xl font-bold text-ink">{t(selectedJourney.title)}</h1>
          <p class="m-0 text-sm text-ink-soft">{t(selectedJourney.dates)} · {t(selectedJourney.place)}</p>
        </div>
        <button
          type="button"
          class="pointer-events-auto rounded-md border border-black/10 bg-paper-1/95 px-3 py-2 font-mono text-xs tracking-wide text-terracotta shadow"
          onclick={backToLanding}
        >{t(backLabel)} ×</button>
      </header>

      {#if selectedPlace}
        <aside
          class="absolute right-0 top-0 z-10 flex h-full w-[min(30rem,46vw)] flex-col gap-5 overflow-y-auto bg-paper-1/95 px-6 py-6 shadow-2xl backdrop-blur"
          aria-label="Memories at this place"
        >
          <div>
            <p class="m-0 font-mono text-[0.7rem] uppercase tracking-[0.18em] text-ink-faint">
              {placeMemoriesLabel(selectedPlace.mementos.length)}
            </p>
            <h2 class="m-0 mt-1 font-mincho text-xl font-bold text-ink">{t(selectedPlace.mementos[0].place)}</h2>
          </div>

          {#each selectedPlace.mementos as memento (memento.id)}
            <article class="rounded-lg border border-black/5 bg-paper-0 p-5 shadow-sm">
              <p class="m-0 font-mono text-[0.68rem] uppercase tracking-[0.2em] text-terracotta">
                {t(kindLabel[memento.kind])}
              </p>
              <h3 class="m-0 mt-1 font-mincho text-lg font-bold text-ink">{t(memento.title)}</h3>
              <p class="m-0 mt-0.5 text-xs text-ink-faint">
                {t(memento.date)}{memento.price ? ` · ${memento.price}` : ''}
              </p>
              <p class="m-0 mt-3 text-[0.9rem] leading-relaxed text-ink-soft">{t(memento.essay)}</p>
              {#if memento.photos.length}
                <div class="mt-4 flex flex-col gap-3">
                  {#each memento.photos as photo (photo.src)}
                    <figure class="m-0 overflow-hidden rounded-md border border-black/5">
                      <img
                        src={photo.src}
                        alt={t(memento.title)}
                        class="block aspect-[4/3] w-full object-cover"
                        onerror={onPhotoError}
                      />
                      <figcaption class="px-3 py-2 text-xs text-ink-soft">{t(photo.caption)}</figcaption>
                    </figure>
                  {/each}
                </div>
              {/if}
            </article>
          {/each}
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
    box-shadow: 0 2rem 4rem rgba(58, 47, 28, 0.35), 0 0.5rem 1rem rgba(58, 47, 28, 0.2);
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

    .year-tabs {
      display: none;
    }
  }
</style>
