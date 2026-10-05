<script lang="ts">
  import { page } from "$app/state"
  import { getStudio } from "$lib/studio"
  import NewMemento from "../../../../../views/NewMemento.svelte"
  const studio = getStudio()
</script>

{#key page.params.journeyId}<NewMemento journeyId={page.params.journeyId!} locale={studio.locale} />{/key}
