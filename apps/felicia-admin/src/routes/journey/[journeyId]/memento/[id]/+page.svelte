<script lang="ts">
  import { page } from "$app/state"
  import { getStudio } from "$lib/studio"
  import MementoEditor from "../../../../../views/MementoEditor.svelte"
  const studio = getStudio()
</script>

{#key JSON.stringify(page.params)}
  <MementoEditor journeyId={page.params.journeyId!} id={page.params.id!} locale={studio.locale} />
{/key}
