<script lang="ts">
  import type { Snippet } from "svelte"
  import "../app.css"
  import StudioShell from "$lib/StudioShell.svelte"
  let { children }: { children: Snippet } = $props()
</script>

<StudioShell {children} />
