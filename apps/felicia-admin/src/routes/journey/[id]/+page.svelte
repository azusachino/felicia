<script lang="ts">
  import { page } from "$app/state"
  import { getStudio } from "$lib/studio"
  import JourneyDetail from "../../../views/JourneyDetail.svelte"
  const studio = getStudio()
</script>

{#key page.params.id}
  <JourneyDetail id={page.params.id!} locale={studio.locale} />
{/key}
