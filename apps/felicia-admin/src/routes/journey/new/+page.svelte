<script lang="ts">
  import { getStudio } from "$lib/studio"
  import NewJourney from "../../../views/NewJourney.svelte"
  const studio = getStudio()
</script>

<NewJourney locale={studio.locale} />
