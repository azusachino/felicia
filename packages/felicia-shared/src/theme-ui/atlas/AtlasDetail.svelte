<script lang="ts">
  import type { Lang, Memento } from "../../data"
  import AtlasStub from "./AtlasStub.svelte"
  import PhotoLightbox from "../PhotoLightbox.svelte"
  import { message } from "../../i18n/catalog"

  let {
    memento,
    lang,
    photoLabel,
    photosHeading,
    closeLabel,
    storyLabel,
    onClose,
  }: {
    memento: Memento
    lang: Lang
    photoLabel: string
    photosHeading: string
    closeLabel: string
    storyLabel: string
    onClose: () => void
  } = $props()

  const t = (value: { ja: string; en: string; zh: string }) => value[lang]
</script>

<aside class="detail-panel" aria-label={t(memento.title)}>
  <button class="detail-close" type="button" aria-label={closeLabel} onclick={onClose}>×</button>
  <div class="detail-scroll">
    <AtlasStub {memento} {lang} {photoLabel} onSelect={() => undefined} />

    <header class="detail-heading">
      <p>{t(memento.date)} · {t(memento.place)}</p>
      <h2>{t(memento.title)}</h2>
    </header>

    <article class="essay">
      <span>{storyLabel}</span>
      <p>{t(memento.essay)}</p>
    </article>

    {#if memento.photos.length}
      <section class="gallery" aria-label={photoLabel}>
        <h3>{photosHeading}</h3>
        <div class="gallery-grid">
          {#each memento.photos as photo, index (`${photo.src}:${index}`)}
            <figure class:tilt-left={index % 2 === 0}>
              <PhotoLightbox src={photo.src} alt={t(photo.caption)} caption={t(photo.caption)} openLabel={message(lang, "ui.zoom")} {closeLabel} imageClass="detail-photo" />
              <figcaption>{t(photo.caption)}</figcaption>
            </figure>
          {/each}
        </div>
      </section>
    {/if}
  </div>
</aside>

<style>
  .detail-panel {
    display: flex;
    position: absolute;
    z-index: 4;
    top: 1rem;
    right: 1rem;
    bottom: 1rem;
    width: min(31rem, calc(100% - 2rem));
    flex-direction: column;
    overflow: hidden;
    border: 1px solid #cdbda5;
    border-radius: 0.2rem;
    background: #f3ecdf;
    box-shadow: 0 1.5rem 4rem #000b;
    color: #362d25;
    animation: open-panel 220ms ease-out both;
  }

  .detail-close {
    position: absolute;
    z-index: 1;
    top: 0.7rem;
    right: 0.7rem;
    width: 2rem;
    height: 2rem;
    border: 1px solid #a99a87;
    border-radius: 50%;
    color: #594b3d;
    background: #f3ecdfdd;
    font-size: 1.25rem;
    line-height: 1;
  }

  .detail-scroll {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    overflow-y: auto;
    padding: 2.5rem 1.5rem 2rem;
  }

  .detail-scroll :global(.stub) {
    align-self: center;
    width: min(100%, 25rem);
    cursor: default;
    transform: rotate(-1deg);
  }

  .detail-scroll :global(.stub:hover) {
    transform: rotate(-1deg);
  }

  .detail-heading {
    border-bottom: 1px solid #cdbda5;
    padding-bottom: 1rem;
  }

  .detail-heading p,
  .essay > span,
  .gallery h3 {
    margin: 0;
    color: #9a5b37;
    font-size: 0.68rem;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }

  .detail-heading h2 {
    margin: 0.55rem 0 0;
    font-family: "Zen Old Mincho", Georgia, serif;
    font-size: clamp(1.5rem, 4vw, 2.1rem);
    line-height: 1.15;
  }

  .essay p {
    margin: 0.65rem 0 0;
    font-family: "Zen Old Mincho", Georgia, serif;
    font-size: 1.05rem;
    font-style: italic;
    line-height: 1.8;
  }

  .gallery {
    border-top: 1px solid #cdbda5;
    padding-top: 1rem;
  }

  .gallery-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: 1rem;
    margin-top: 0.85rem;
  }

  figure {
    margin: 0;
    padding: 0.55rem 0.55rem 1.1rem;
    background: #fffdf8;
    box-shadow: 0 0.5rem 1rem #7f6c5926;
    transform: rotate(1deg);
  }

  figure.tilt-left {
    transform: rotate(-2deg);
  }

  :global(.detail-photo) {
    width: 100%;
    aspect-ratio: 1 / 0.82;
    object-fit: cover;
  }

  figcaption {
    margin-top: 0.5rem;
    color: #6f6256;
    font-family: "Zen Old Mincho", Georgia, serif;
    font-size: 0.78rem;
    line-height: 1.4;
  }

  @keyframes open-panel {
    from {
      opacity: 0;
      transform: translateX(1.5rem);
    }

    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  @media (max-width: 700px) {
    .detail-panel {
      top: auto;
      right: 0.75rem;
      bottom: 0.75rem;
      left: 0.75rem;
      width: auto;
      max-height: min(80vh, 42rem);
    }
  }
</style>
