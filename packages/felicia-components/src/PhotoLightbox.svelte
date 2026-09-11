<script lang="ts">
  // Runes, not `export let`. Svelte 5 picks a component's mode from its own
  // syntax: one `export let` puts the whole file in legacy mode, where
  // `onclick={...}` and `onclose={...}` below are inert DOM attributes rather
  // than event handlers. This component had exactly that mix -- the only one
  // in the repo -- so nothing could open it and nothing could close it.
  let {
    src,
    alt,
    caption = "",
    openLabel = "Open photo",
    closeLabel = "Close",
    imageClass = "",
  }: {
    src: string
    alt: string
    caption?: string
    openLabel?: string
    closeLabel?: string
    imageClass?: string
  } = $props()

  let triggerButton: HTMLButtonElement | undefined
  let dialogEl: HTMLDialogElement | undefined

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

  /* Only the open state gets a display, because the browser hides a closed
     <dialog> with `display: none` and any unconditional `display` here
     overrides it -- which is what painted this lightbox over every theme
     before anyone clicked a photo. */
  .photo-lightbox[open] {
    display: grid;
  }

  .photo-lightbox {
    position: fixed;
    z-index: 100;
    inset: 0;
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
    /* The reader's global `.gallery img` rule (felicia-reader/src/public.css)
       crops every thumbnail to a 4:3 tile. This dialog is markup *inside* that
       gallery figure, so the rule reached the full-size photo too: the box was
       forced to 4:3 and `object-fit: contain` then letterboxed a 16:9 photo
       inside it, with dead bands above and below. The modal shows a photo
       whole and at its own shape, so it restates both properties rather than
       relying on every theme to scope its gallery selector. */
    aspect-ratio: auto;
    outline: none;
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
