<script lang="ts">
  import { fade, fly } from "svelte/transition"
  import { onMount } from "svelte"
  import { kindLabel, uiText, type Journey, type MementoCard, type L, type Lang, type Memento, type Theme } from "../../data"
  import { message, type MessageKey } from "../../i18n/catalog"
  import PhotoLightbox from "../PhotoLightbox.svelte"
  import TicketStub from "../TicketStub.svelte"

  // Cabinet — memento-first front door. The detailed memento "page" is the
  // centre; a preview carousel is the index. Cartography is reached as the
  // "more" view.
  export let lang: Lang = "ja"
  export let theme: Theme = "dark"
  export let loadJourneys: () => Promise<Journey[]>
  export let toMap: (() => void) | undefined = undefined

  const title = { ja: "旅の残り香", en: "What Lingers", zh: "旅途余香" }
  const label = {
    map: { ja: "← 地図", en: "← Map", zh: "← 地图" },
    onMap: { ja: "地図で見る →", en: "See on the map →", zh: "在地图上查看 →" },
    memories: { ja: "記憶", en: "Memories", zh: "回忆" },
  }

  let allMementos: MementoCard[] = []
  let shelf: MementoCard[] = []
  let selected: MementoCard | undefined
  let isLoading = true
  let error: string | null = null

  $: t = (value: L | MessageKey) => (typeof value === "string" ? message(lang, value) : value[lang])
  $: memento = selected?.memento as Memento | undefined

  onMount(() => {
    loadJourneys()
      .then((data) => {
        allMementos = data.flatMap((journey) => journey.mementos.map((item) => ({ memento: item, journey })))
        shelf = [...allMementos, ...allMementos]
        selected = allMementos[0]
      })
      .catch((reason) => {
        error = reason instanceof Error ? reason.message : String(reason)
      })
      .finally(() => {
        isLoading = false
      })
  })

  function select(card: MementoCard) {
    selected = card
  }

  function toggleTheme() {
    theme = theme === "dark" ? "light" : "dark"
  }
</script>

<main class="app-shell cabinet-shell" class:theme-light={theme === "light"}>
  {#if isLoading}
    <div class="cabinet-status">Loading…</div>
  {:else if error}
    <div class="cabinet-status">{error}</div>
  {:else if selected && memento}
    <header class="cabinet-top">
      <div class="cabinet-brand">
        <p class="eyebrow">felicia</p>
        <h1>{t(title)}</h1>
      </div>
      <div class="cabinet-controls">
        <div class="lang-switch" role="group" aria-label="Language">
          <button class:active={lang === "ja"} on:click={() => (lang = "ja")}>日本語</button>
          <button class:active={lang === "en"} on:click={() => (lang = "en")}>EN</button>
          <button class:active={lang === "zh"} on:click={() => (lang = "zh")}>中文</button>
        </div>
        <button class="theme-toggle" on:click={toggleTheme} aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}>
          {theme === "dark" ? "☀" : "☾"}
        </button>
        {#if toMap}
          <button class="all-btn" on:click={toMap}>{t(label.map)}</button>
        {/if}
      </div>
    </header>

    <!-- The memento detail "page": the centre of Cabinet. -->
    <section class="cabinet-stage" aria-label="Memento detail">
      {#key memento.id}
        <div class="cabinet-detail" in:fade={{ duration: 200 }}>
          <div class="cabinet-stub-col" in:fly={{ y: 14, duration: 320, delay: 40 }}>
            <div class="stub-card {memento.kind}">
              {#if memento.kind === "transit" || memento.kind === "ticket"}
                <TicketStub {memento} {lang} />
              {:else if memento.kind === "stamp"}
                <div class="stamp-face">
                  <span>御朱印</span>
                  <strong>{t(memento.place)}</strong>
                  <small>{t(memento.date)}</small>
                </div>
              {:else}
                <div class="goods-face">
                  <span>{t(kindLabel.goods)}</span>
                  <strong>{t(memento.title)}</strong>
                  {#if t(memento.vendor) || memento.price}
                    <small>{[t(memento.vendor), memento.price].filter(Boolean).join(" · ")}</small>
                  {/if}
                </div>
              {/if}
            </div>

            <dl class="cabinet-facts">
              <div>
                <dt>{lang === "en" ? "Journey" : lang === "zh" ? "旅程" : "旅"}</dt>
                <dd>{t(selected.journey.title)}</dd>
              </div>
              <div>
                <dt>{lang === "en" ? "Date" : lang === "zh" ? "日期" : "日付"}</dt>
                <dd>{t(memento.date)}</dd>
              </div>
              <div>
                <dt>{lang === "en" ? "Place" : lang === "zh" ? "地点" : "場所"}</dt>
                <dd>{t(memento.place)}</dd>
              </div>
            </dl>
          </div>

          <div class="cabinet-story-col">
            <div class="section-head">
              <p class="eyebrow">{t(kindLabel[memento.kind])}</p>
              <h2>{t(memento.title)}</h2>
            </div>

            <article class="essay">
              <span>{t(uiText.story)}</span>
              <p>{t(memento.essay)}</p>
            </article>

            {#if memento.photos.length}
              <div class="gallery">
                {#each memento.photos as photo, index (`${photo.src}:${index}`)}
                  <figure>
                    <PhotoLightbox src={photo.src} alt={t(memento.title)} caption={t(photo.caption)} openLabel={t(uiText.zoom)} closeLabel={t(uiText.close)} />
                    <figcaption>{t(photo.caption)}</figcaption>
                  </figure>
                {/each}
              </div>
            {/if}

            {#if toMap}
              <button class="cabinet-onmap" on:click={toMap}>{t(label.onMap)}</button>
            {/if}
          </div>
        </div>
      {/key}
    </section>

    <!-- The preview carousel: the index. Auto-scrolls; pauses on hover. -->
    <footer class="cabinet-carousel" aria-label="Memento shelf">
      <p class="eyebrow cabinet-carousel-head">{t(label.memories)}</p>
      <div class="cabinet-shelf">
        <div class="cabinet-track">
          {#each shelf as card, i (i)}
            <button
              class="cabinet-preview cabinet-preview--{card.memento.kind}"
              class:active={card.memento.id === memento.id}
              aria-hidden={i >= allMementos.length}
              tabindex={i >= allMementos.length ? -1 : 0}
              on:click={() => select(card)}
            >
              <span class="cabinet-preview-kind">{t(kindLabel[card.memento.kind])}</span>
              <strong>{t(card.memento.title)}</strong>
              <span class="cabinet-preview-meta">{t(card.memento.date)} · {t(card.journey.title)}</span>
              <span class="cabinet-preview-place">{t(card.memento.place)}</span>
            </button>
          {/each}
        </div>
      </div>
    </footer>
  {:else}
    <div class="cabinet-status">No mementos</div>
  {/if}
</main>

<style>
  /* Cabinet overrides the shared .app-shell grid with a stacked layout. Theme tokens
     and the paper-stub / essay / gallery classes are inherited from index.css. */
  .cabinet-shell {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }

  .cabinet-status {
    display: grid;
    min-height: 100%;
    place-items: center;
    color: var(--muted);
  }

  .cabinet-top {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    padding: 1.25rem 1.75rem;
    border-bottom: 1px solid var(--border);
  }

  .cabinet-brand h1 {
    margin: 0.2rem 0 0;
    font-size: 1.5rem;
    font-weight: 700;
    letter-spacing: 0.01em;
    color: var(--text);
  }

  .cabinet-controls {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .cabinet-stage {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 2rem 1.75rem;
  }

  .cabinet-detail {
    display: grid;
    grid-template-columns: minmax(0, 22rem) minmax(0, 1fr);
    gap: 2.5rem;
    max-width: 68rem;
    margin: 0 auto;
    align-items: start;
  }

  .cabinet-stub-col {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    position: sticky;
    top: 0;
  }

  .cabinet-facts {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin: 0;
  }

  .cabinet-facts div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--border);
  }

  .cabinet-facts dt {
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.16em;
    color: var(--faint);
  }

  .cabinet-facts dd {
    margin: 0;
    text-align: right;
    font-size: 0.9rem;
    color: var(--text-soft);
  }

  .cabinet-story-col {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .cabinet-onmap {
    align-self: flex-start;
    padding: 0.55rem 1rem;
    border-radius: 0.5rem;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--accent-ink);
    background: transparent;
    border: 1px solid var(--border);
  }

  .cabinet-onmap:hover {
    background: var(--hover);
  }

  /* Carousel — an auto-scrolling shelf of memories that pauses on hover. --- */
  .cabinet-carousel {
    border-top: 1px solid var(--border);
    padding: 1.5rem 1.75rem 1.75rem;
    background: var(--bg);
  }

  .cabinet-carousel-head {
    margin: 0 0 1rem;
  }

  .cabinet-shelf {
    overflow: hidden;
    -webkit-mask-image: linear-gradient(90deg, transparent, #000 4%, #000 96%, transparent);
    mask-image: linear-gradient(90deg, transparent, #000 4%, #000 96%, transparent);
  }

  .cabinet-track {
    display: flex;
    width: max-content;
    padding: 0.5rem 0;
    animation: cabinet-marquee 48s linear infinite;
  }

  .cabinet-shelf:hover .cabinet-track,
  .cabinet-shelf:focus-within .cabinet-track {
    animation-play-state: paused;
  }

  @keyframes cabinet-marquee {
    from {
      transform: translateX(0);
    }
    to {
      transform: translateX(-50%);
    }
  }

  .cabinet-preview {
    /* margin (not flex gap) keeps the doubled track seamless at -50%. */
    margin-right: 1.1rem;
    flex: 0 0 auto;
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
    width: 17.5rem;
    min-height: 8.5rem;
    padding: 1.1rem 1.25rem;
    text-align: left;
    border-radius: 0.7rem;
    border: 1px solid var(--border);
    border-left: 4px solid var(--chip-border);
    background: var(--card);
    transition:
      transform 0.15s ease,
      border-color 0.15s ease,
      background 0.15s ease;
  }

  .cabinet-preview:hover {
    transform: translateY(-3px);
    background: var(--hover);
  }

  .cabinet-preview.active {
    border-color: rgba(253, 186, 116, 0.6);
    border-left-color: var(--accent-ink);
    background: rgba(251, 146, 60, 0.1);
  }

  .cabinet-preview--transit {
    border-left-color: #fb923c;
  }

  .cabinet-preview--stamp {
    border-left-color: #ef4444;
  }

  .cabinet-preview--goods {
    border-left-color: #a855f7;
  }

  .cabinet-preview-kind {
    font-size: 0.66rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.18em;
    color: var(--accent-ink);
  }

  .cabinet-preview strong {
    font-size: 1.02rem;
    font-weight: 600;
    color: var(--text);
    line-height: 1.3;
  }

  .cabinet-preview-meta {
    font-size: 0.78rem;
    color: var(--muted);
  }

  .cabinet-preview-place {
    font-size: 0.74rem;
    color: var(--faint);
    margin-top: auto;
  }

  @media (prefers-reduced-motion: reduce) {
    .cabinet-track {
      animation: none;
    }

    .cabinet-shelf {
      overflow-x: auto;
      -webkit-mask-image: none;
      mask-image: none;
    }
  }

  @media (max-width: 900px) {
    .cabinet-detail {
      grid-template-columns: minmax(0, 1fr);
      gap: 1.75rem;
    }

    .cabinet-stub-col {
      position: static;
    }
  }
</style>
