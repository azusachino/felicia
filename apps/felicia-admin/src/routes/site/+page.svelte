<script lang="ts">
  import { getStudio } from "$lib/studio"
  import SiteDeploy from "../../views/SiteDeploy.svelte"
  const studio = getStudio()
</script>

<SiteDeploy locale={studio.locale} />
