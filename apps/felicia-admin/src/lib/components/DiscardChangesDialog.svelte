<script lang="ts">
  import { AlertDialog } from "bits-ui"
  import { Button } from "$lib/components/ui/button"
  import { message, type Locale } from "../../i18n"
  let { prompt, locale, decide }: { prompt: string | null; locale: Locale; decide: (discard: boolean) => void } = $props()
</script>

<AlertDialog.Root
  open={prompt !== null}
  onOpenChange={(open) => {
    if (!open) decide(false)
  }}
>
  <AlertDialog.Portal>
    <AlertDialog.Overlay class="studio-dialog-overlay" />
    <AlertDialog.Content class="studio-dialog">
      <header class="sheet-header"><AlertDialog.Title>{message(locale, "admin.common.discard_title")}</AlertDialog.Title></header>
      <div class="sheet-body"><AlertDialog.Description>{prompt}</AlertDialog.Description></div>
      <footer class="sheet-actions">
        <AlertDialog.Cancel>
          {#snippet child({ props })}<Button {...props} variant="outline" onclick={() => decide(false)}>{message(locale, "admin.common.keep_editing")}</Button>{/snippet}
        </AlertDialog.Cancel>
        <AlertDialog.Action>
          {#snippet child({ props })}<Button {...props} onclick={() => decide(true)}>{message(locale, "admin.common.discard_changes")}</Button>{/snippet}
        </AlertDialog.Action>
      </footer>
    </AlertDialog.Content>
  </AlertDialog.Portal>
</AlertDialog.Root>
