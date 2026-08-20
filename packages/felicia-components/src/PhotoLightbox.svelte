<script lang="ts">
  import { tick } from "svelte"

  export let src: string
  export let alt: string
  export let caption = ""
  export let openLabel = "Open photo"
  export let closeLabel = "Close"
  export let imageClass = ""

  let open = false
  let closeButton: HTMLButtonElement

  async function show() {
    open = true
    await tick()
    closeButton?.focus()
  }

  function close() {
    open = false
  }

  function handleKeydown(event: KeyboardEvent) {
    if (open && event.key === "Escape") close()
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<button type="button" class="photo-trigger" aria-label={openLabel} on:click={show}>
  <img {src} {alt} class={imageClass} />
</button>

{#if open}
  <div class="photo-lightbox" role="dialog" aria-modal="true" aria-label={alt}>
    <div class="lightbox-frame">
      <button bind:this={closeButton} type="button" class="lightbox-close" aria-label={closeLabel} on:click={close}>×</button>
      <img class="lightbox-image" {src} {alt} />
      {#if caption}
        <p>{caption}</p>
      {/if}
    </div>
  </div>
{/if}

<style>
  .photo-trigger {
    display: block;
    width: 100%;
    overflow: hidden;
    padding: 0;
    border: 0;
    background: transparent;
    text-align: left;
  }

  .photo-trigger img {
    display: block;
    width: 100%;
    cursor: zoom-in;
  }

  .photo-lightbox {
    position: fixed;
    z-index: 100;
    inset: 0;
    display: grid;
    place-items: center;
    padding: 1.5rem;
    background: rgb(0 0 0 / 82%);
  }

  .lightbox-frame {
    position: relative;
    max-width: min(96vw, 96rem);
    max-height: 94vh;
    padding: 0.65rem;
    background: #111;
    box-shadow: 0 1.5rem 5rem rgb(0 0 0 / 45%);
  }

  .lightbox-image {
    display: block;
    max-width: 92vw;
    max-height: 84vh;
    width: auto;
    height: auto;
    object-fit: contain;
  }

  .lightbox-close {
    position: absolute;
    z-index: 1;
    top: 0.4rem;
    right: 0.4rem;
    width: 2rem;
    height: 2rem;
    border: 1px solid rgb(255 255 255 / 35%);
    border-radius: 999px;
    color: #fff;
    background: rgb(0 0 0 / 65%);
    font-size: 1.25rem;
    line-height: 1;
  }

  .lightbox-frame p {
    margin: 0.55rem 0 0.1rem;
    color: #ddd;
    font-size: 0.78rem;
  }
</style>
