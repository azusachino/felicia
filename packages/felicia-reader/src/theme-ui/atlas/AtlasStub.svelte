<script lang="ts">
  import type { Lang, Memento } from "@felicia/model"
  import { templateFor } from "./stubs"
  import TicketStub from "@felicia/components/TicketStub.svelte"

  let {
    memento,
    lang,
    photoLabel,
    selected = false,
    onSelect,
  }: {
    memento: Memento
    lang: Lang
    photoLabel: string
    selected?: boolean
    onSelect: () => void
  } = $props()

  const t = (value: { ja: string; en: string; zh: string }) => value[lang]
  const template = $derived(templateFor(memento.kind))
</script>

<button
  class="stub"
  class:selected
  class:photo-fallback={!template}
  class:stub-transit={memento.kind === "transit"}
  class:stub-stamp={memento.kind === "stamp"}
  class:stub-goods={memento.kind === "goods"}
  class:stub-receipt={memento.kind === "receipt"}
  class:stub-souvenir={memento.kind === "souvenir"}
  class:stub-ticket={memento.kind === "ticket"}
  aria-label={t(memento.title)}
  onclick={onSelect}
>
  {#if !template && memento.photos[0]}
    <img class="fallback-image" src={memento.photos[0].src} alt={t(memento.title)} />
    <div class="fallback-copy">
      <span>{t(memento.date)}</span>
      <strong>{t(memento.title)}</strong>
      <small>{t(memento.place)}</small>
    </div>
  {:else if memento.kind === "transit" || memento.kind === "ticket"}
    <TicketStub {memento} {lang} />
  {:else if memento.kind === "stamp"}
    <div class="stamp-mark">印</div>
    <div class="stamp-copy">
      <span>{t(memento.place)}</span>
      <strong>{t(memento.title)}</strong>
      <small>{t(memento.date)}</small>
    </div>
  {:else if memento.kind === "goods"}
    <div class="tag-hole"></div>
    <span class="tag-kind">{template?.label}</span>
    <strong>{t(memento.title)}</strong>
    {#if t(memento.vendor) || memento.price}
      <small>{[t(memento.vendor), memento.price].filter(Boolean).join(" · ")}</small>
    {/if}
  {:else if memento.kind === "receipt"}
    <span class="receipt-vendor">{t(memento.vendor)}</span>
    <strong>{t(memento.title)}</strong>
    <div class="receipt-rule"></div>
    <small>{t(memento.date)} · {memento.price}</small>
  {:else}
    <span class="card-kind">{template?.label ?? "memento"}</span>
    <strong>{t(memento.title)}</strong>
    <small>{t(memento.place)} · {t(memento.date)}</small>
  {/if}

  {#if memento.photos.length}
    <span class="photo-count">{memento.photos.length} {photoLabel}</span>
  {/if}
</button>

<style>
  .stub {
    display: flex;
    position: relative;
    flex: 0 1 21rem;
    min-height: 10rem;
    flex-direction: column;
    align-items: flex-start;
    justify-content: space-between;
    overflow: hidden;
    padding: 1.25rem;
    border: 0;
    border-radius: 0.2rem;
    box-shadow: 0 1rem 2.5rem #0008;
    color: #191817;
    cursor: pointer;
    font: inherit;
    text-align: left;
    transform: rotate(-1deg);
    transition:
      transform 180ms ease,
      box-shadow 180ms ease;
  }

  .stub:nth-child(even) {
    transform: rotate(1deg);
  }

  .stub:hover,
  .stub:focus-visible,
  .stub.selected {
    z-index: 2;
    outline: 2px solid #ff9b72;
    outline-offset: 3px;
    box-shadow: 0 1rem 3rem #000b;
    transform: translateY(-0.35rem) rotate(0);
  }

  .stub strong {
    margin: 0.75rem 0;
    font-size: 1.15rem;
  }

  .stub small,
  .stub span {
    font-size: 0.7rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .stub-transit,
  .stub-ticket {
    display: block;
    padding: 0;
    background: transparent;
  }

  .tag-kind,
  .card-kind {
    color: #71695e;
    font-size: 0.65rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }

  .stub-stamp {
    background: #f3ece3;
    border: 0.5rem solid #efe5d9;
    outline: 1px solid #c84436;
    outline-offset: -0.8rem;
  }

  .stamp-mark {
    align-self: center;
    padding: 0.5rem;
    border: 2px solid #c84436;
    border-radius: 50%;
    color: #c84436;
    font-size: 1.8rem;
  }

  .stamp-copy {
    display: flex;
    width: 100%;
    flex-direction: column;
    align-items: center;
    gap: 0.25rem;
    text-align: center;
  }

  .stub-goods {
    background: #c99b62;
    background-image: radial-gradient(#9e7042 0.5px, transparent 0.5px);
    background-size: 5px 5px;
  }

  .tag-hole {
    align-self: center;
    width: 0.8rem;
    height: 0.8rem;
    border: 2px solid #644426;
    border-radius: 50%;
  }

  .stub-receipt {
    background: #fffdf5;
    background-image: linear-gradient(#e8e2d5 1px, transparent 1px);
    background-size: 100% 1.45rem;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  }

  .receipt-vendor {
    font-weight: 700;
  }

  .receipt-rule {
    width: 100%;
    border-top: 1px dashed #837a6e;
  }

  .stub-souvenir {
    background: #e6d2a8;
  }

  .photo-fallback {
    padding: 0;
    background: #e8e0d2;
  }

  .fallback-image {
    width: 100%;
    height: 7rem;
    object-fit: cover;
  }

  .fallback-copy {
    display: flex;
    width: 100%;
    flex-direction: column;
    gap: 0.3rem;
    padding: 0.8rem 1rem 1rem;
  }

  .photo-count {
    position: absolute;
    top: 0.8rem;
    right: 0.8rem;
    padding: 0.25rem 0.45rem;
    border-radius: 999px;
    background: #191817;
    color: #efe9d9;
    font-size: 0.65rem;
  }
</style>
