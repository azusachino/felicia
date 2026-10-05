<script lang="ts">
  import { Dialog } from "bits-ui"
  import { ArrowLeft } from "@lucide/svelte"
  import { Button } from "$lib/components/ui/button"
  import { getSiteInfo } from "../../api"
  import { message, type Locale } from "../../i18n"
  let { open = $bindable(false), locale, nativeMac = false }: { open?: boolean; locale: Locale; nativeMac?: boolean } = $props()
  let frame = $state<HTMLIFrameElement | null>(null)
  function readerEscape(event: MessageEvent) {
    if (open && url && event.source === frame?.contentWindow && event.origin === new URL(url).origin && event.data === "felicia:preview:escape") open = false
  }
  let url = $state("")
  let error = $state("")
  let loading = $state(false)
  async function load() {
    loading = true
    url = ""
    error = ""
    try {
      const info = await getSiteInfo()
      const port = Number(info.preview_port)
      if (!info.artifact_ready || !info.spa_ready || !Number.isInteger(port) || port < 1 || port > 65535) {
        error = message(locale, "admin.preview.unavailable")
      } else url = `http://127.0.0.1:${port}/`
    } catch (cause) {
      error = cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
    } finally {
      loading = false
    }
  }
  $effect(() => {
    if (open) void load()
  })
</script>

<svelte:window onmessage={readerEscape} />

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="compiled-preview-overlay" />
    <Dialog.Content class={nativeMac ? "compiled-preview native-mac" : "compiled-preview"} onInteractOutside={(event) => event.preventDefault()}>
      <header>
        <div>
          <Dialog.Title>{message(locale, "admin.preview.title")}</Dialog.Title>
          <Dialog.Description>{message(locale, "admin.preview.note")}</Dialog.Description>
        </div>
        <Button variant="outline" size="sm" onclick={() => (open = false)}><ArrowLeft size={16} aria-hidden="true" />{message(locale, "admin.preview.close")}</Button>
      </header>
      {#if loading}<p class="hint">{message(locale, "admin.preview.loading")}</p>
      {:else if error}<p class="api-error" role="alert">{error}</p>
      {:else if url}<iframe bind:this={frame} src={url} title={message(locale, "admin.preview.title")} sandbox="allow-scripts allow-same-origin allow-downloads"></iframe>{/if}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  :global(.compiled-preview) {
    position: fixed;
    inset: 16px;
    z-index: 51;
    display: flex;
    flex-direction: column;
    width: calc(100vw - 32px);
    max-width: none;
    height: calc(100vh - 32px);
    max-height: none;
    margin: auto;
    padding: 0;
    border: 1px solid var(--line);
    border-radius: 12px;
    color: var(--text);
    background: var(--surface);
  }
  :global(.compiled-preview.native-mac) {
    top: calc(var(--studio-titlebar-height) + 16px);
    height: calc(100vh - var(--studio-titlebar-height) - 32px);
  }
  :global(.compiled-preview-overlay) {
    position: fixed;
    inset: 0;
    z-index: 50;
    background: rgb(0 0 0 / 24%);
  }
  header {
    --wails-draggable: drag;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 12px 16px;
    border-bottom: 1px solid var(--line);
  }
  header :global(button) {
    min-height: 36px;
    font-size: 13px;
  }
  header :global([role="heading"]) {
    font-size: 16px;
    font-weight: 600;
  }
  header :global(p) {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--muted);
  }
  iframe {
    flex: 1;
    min-height: 0;
    width: 100%;
    border: 0;
    background: var(--surface);
  }
  .hint,
  .api-error {
    margin: 16px;
  }
</style>
