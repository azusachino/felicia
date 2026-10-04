<script lang="ts">
  import { getStudio } from "$lib/studio"
  import JourneyList from "../views/JourneyList.svelte"
  const studio = getStudio()
</script>

<JourneyList locale={studio.locale} desktop={studio.desktop} />
