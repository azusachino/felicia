<script lang="ts">
  export let src: string
  export let alt: string
  export let caption = ""
  export let openLabel = "Open photo"
  export let closeLabel = "Close"
  export let imageClass = ""

  let triggerButton: HTMLButtonElement
  let dialogEl: HTMLDialogElement

  function show() {
    dialogEl?.showModal()
  }

  function close() {
    dialogEl?.close()
  }

  // Fires on Escape and on our own close() call alike -- native <dialog>
  // already supplies the focus trap and Escape handling; this only needs to
  // return focus to whatever opened it.
  function onDialogClose() {
    triggerButton?.focus()
  }
</script>

<button bind:this={triggerButton} type="button" class="photo-trigger" aria-label={openLabel} onclick={show}>
  <img {src} {alt} class={imageClass} />
</button>

<dialog bind:this={dialogEl} class="photo-lightbox" aria-label={alt} onclose={onDialogClose}>
  <div class="lightbox-frame">
    <button type="button" class="lightbox-close" aria-label={closeLabel} onclick={close}>×</button>
    <img class="lightbox-image" {src} {alt} />
    {#if caption}
      <p>{caption}</p>
    {/if}
  </div>
</dialog>

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
    overflow: auto;
    width: 100%;
    max-width: 100%;
    height: 100%;
    max-height: 100%;
    place-items: center;
    padding: 1.5rem;
    border: 0;
    background: transparent;
  }

  .photo-lightbox::backdrop {
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
    width: 2.75rem;
    height: 2.75rem;
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
