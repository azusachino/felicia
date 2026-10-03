<script lang="ts">
  import type { WrapOptions } from "svelte-spa-router/wrap"
  import type { StudioState } from "../routes"

  let {
    component: Page,
    studio,
    params = null,
  }: {
    component: NonNullable<WrapOptions["component"]>
    studio: StudioState
    params?: Record<string, string> | null
  } = $props()
</script>

<!-- Remount on parameter changes, as the former keyed page branches did.
     A language change updates props without discarding the current draft. -->
{#key JSON.stringify(params)}
  <Page {...params} locale={studio.locale} desktop={studio.desktop} />
{/key}
