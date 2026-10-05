<script lang="ts">
  import { page } from "$app/state"
  import { getStudio } from "$lib/studio"
  import NewJourney from "../../../../views/NewJourney.svelte"
  import { getJourney, describeLoadFailure, type AdminJourney } from "../../../../api"
  import { message } from "../../../../i18n"
  const studio = getStudio()
  let journey = $state<AdminJourney | null>(null)
  let error = $state("")
  $effect(() => {
    const id = page.params.id!
    let active = true
    journey = null
    error = ""
    getJourney(id)
      .then((value) => {
        if (active) journey = value
      })
      .catch((cause) => {
        if (active) error = describeLoadFailure(cause, "this journey")
      })
    return () => {
      active = false
    }
  })
</script>

{#if error}<p role="alert">{error}</p>
{:else if journey}
  {#key journey.id}<NewJourney locale={studio.locale} {journey} />{/key}
{:else}<p role="status">{message(studio.locale, "admin.common.loading")}</p>{/if}
