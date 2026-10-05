<script lang="ts">
  import { Compass } from "@lucide/svelte"
  import { Button } from "$lib/components/ui/button"
  import { getStudio } from "$lib/studio"
  import { openDesktopSample } from "../../api"
  import { prepareWorkspaceSwitch } from "$lib/workspace"
  import { message } from "../../i18n"
  const studio = getStudio()
  let pending = $state(false)
  let error = $state("")
  async function open() {
    pending = true
    error = ""
    try {
      if (!(await prepareWorkspaceSwitch())) return
      await openDesktopSample()
      window.location.reload()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : message(studio.locale, "admin.common.request_failed")
    } finally {
      pending = false
    }
  }
</script>

{#if studio.desktop && studio.workspaceReady && !studio.sample}
  <div class="sample-action">
    <Button size="sm" variant="outline" onclick={open} disabled={pending}><Compass size={16} aria-hidden="true" />{message(studio.locale, "admin.sample.open")}</Button>
    {#if error}<p class="api-error" role="alert">{error}</p>{/if}
  </div>
{/if}
