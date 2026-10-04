<script lang="ts">
  import type { Snippet } from "svelte"
  import "../app.css"
  import StudioShell from "$lib/StudioShell.svelte"
  import { ModeWatcher } from "mode-watcher"
  import { Tooltip } from "bits-ui"
  let { children }: { children: Snippet } = $props()
</script>

<ModeWatcher defaultMode="system" modeStorageKey="felicia.admin.appearance" />
<Tooltip.Provider delayDuration={350}>
  <StudioShell {children} />
</Tooltip.Provider>
