<script lang="ts">
  import type { Lang, Memento, Theme } from "@felicia/model"
  import AtlasStub from "./AtlasStub.svelte"
  import PhotoLightbox from "../../components/PhotoLightbox.svelte"
  import { message } from "@felicia/model"

  let {
    memento,
    lang,
    theme = "dark",
    photoLabel,
    photosHeading,
    closeLabel,
    storyLabel,
    onClose,
  }: {
    memento: Memento
    lang: Lang
    theme?: Theme
    photoLabel: string
    photosHeading: string
    closeLabel: string
    storyLabel: string
    onClose: () => void
  } = $props()

  const t = (value: { ja: string; en: string; zh: string }) => value[lang]
</script>

<aside class="detail-panel" class:theme-light={theme === "light"} aria-label={t(memento.title)}>
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
    z-index: 30;
    top: 4.5rem;
    right: 1rem;
    bottom: 1rem;
    width: min(31rem, calc(100% - 2rem));
    flex-direction: column;
    overflow: hidden;
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 1rem;
    background: rgba(11, 28, 40, 0.94);
    backdrop-filter: blur(20px);
    box-shadow: 0 1.5rem 4rem rgba(0, 0, 0, 0.7);
    color: #e2e8f0;
    animation: open-panel 220ms ease-out both;
  }

  .detail-panel.theme-light {
    border: 1px solid rgba(0, 0, 0, 0.1);
    background: rgba(255, 255, 255, 0.94);
    box-shadow: 0 1.5rem 4rem rgba(0, 0, 0, 0.18);
    color: #1e293b;
  }

  .detail-close {
    display: grid;
    position: absolute;
    z-index: 10;
    top: 0.75rem;
    right: 0.75rem;
    width: 2.25rem;
    height: 2.25rem;
    place-items: center;
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 50%;
    color: #cbd5e1;
    background: rgba(0, 0, 0, 0.4);
    font-size: 1.25rem;
    line-height: 1;
    cursor: pointer;
    transition: all 0.2s;
  }

  .detail-panel.theme-light .detail-close {
    border: 1px solid rgba(0, 0, 0, 0.12);
    color: #475569;
    background: rgba(255, 255, 255, 0.8);
  }

  .detail-close:hover {
    color: #39C5BB;
    border-color: #39C5BB;
  }

  .detail-scroll {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    overflow-y: auto;
    padding: 2.25rem 1.5rem 2rem;
  }

  .detail-scroll :global(.stub) {
    align-self: center;
    flex: 0 0 auto;
    width: min(100%, 25rem);
    cursor: default;
    transform: rotate(-1deg);
  }

  .detail-scroll :global(.stub:hover) {
    transform: rotate(-1deg);
  }

  .detail-heading {
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    padding-bottom: 1rem;
  }

  .detail-panel.theme-light .detail-heading {
    border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  }

  .detail-heading p,
  .essay > span,
  .gallery h3 {
    margin: 0;
    color: #39C5BB;
    font-size: 0.7rem;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }

  .detail-heading h2 {
    margin: 0.55rem 0 0;
    font-family: "Zen Old Mincho", Georgia, serif;
    font-size: clamp(1.5rem, 4vw, 2.1rem);
    line-height: 1.15;
    color: #ffffff;
  }

  .detail-panel.theme-light .detail-heading h2 {
    color: #0f172a;
  }

  .essay p {
    margin: 0.65rem 0 0;
    font-family: "Zen Old Mincho", Georgia, serif;
    font-size: 1.05rem;
    font-style: italic;
    line-height: 1.8;
    color: #cbd5e1;
  }

  .detail-panel.theme-light .essay p {
    color: #334155;
  }

  .gallery {
    border-top: 1px solid rgba(255, 255, 255, 0.1);
    padding-top: 1rem;
  }

  .detail-panel.theme-light .gallery {
    border-top: 1px solid rgba(0, 0, 0, 0.08);
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
    background: #0f2434;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 0.5rem;
    box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, 0.5);
    transform: rotate(1deg);
  }

  .detail-panel.theme-light figure {
    background: #ffffff;
    border: 1px solid rgba(0, 0, 0, 0.08);
    box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, 0.08);
  }

  figure.tilt-left {
    transform: rotate(-2deg);
  }

  :global(.detail-photo) {
    width: 100%;
    aspect-ratio: 1 / 0.82;
    object-fit: cover;
    border-radius: 0.25rem;
  }

  figcaption {
    margin-top: 0.5rem;
    color: #94a3b8;
    font-family: "Zen Old Mincho", Georgia, serif;
    font-size: 0.78rem;
    line-height: 1.4;
  }

  .detail-panel.theme-light figcaption {
    color: #64748b;
  }

  @keyframes open-panel {
    from {
      opacity: 0;
      transform: translateY(10px);
    }

    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (max-width: 700px) {
    .detail-panel {
      top: auto;
      right: 0.75rem;
      bottom: 0.75rem;
      left: 0.75rem;
      width: auto;
      max-height: min(75vh, 38rem);
    }
  }
</style>
