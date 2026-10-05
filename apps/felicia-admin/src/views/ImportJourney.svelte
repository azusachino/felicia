<script lang="ts">
  import { Button } from "$lib/components/ui/button"
  import { Input } from "$lib/components/ui/input"
  import { importLocalJourney, scanLocalJourney, pickDesktopFolder, formatJourneyDate, type LocalJourneyPlan } from "../api"
  import { message, type Locale } from "../i18n"
  import { listHash, journeyDetailHash } from "../router"

  let { locale, desktop }: { locale: Locale; desktop: boolean } = $props()
  let workspace = $state("")
  let slug = $state("")
  let title = $state("")
  let place = $state("")
  let pending = $state(false)
  let browsing = $state(false)
  let importing = $state(false)
  let error = $state("")
  let scanned = $state<LocalJourneyPlan | null>(null)
  let reviewedWorkspace = ""

  function failure(cause: unknown) {
    error = cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
  }
  async function browse() {
    browsing = true
    try {
      const result = await pickDesktopFolder(message(locale, "admin.connectors.folder_path"))
      if (result.selected && result.path) {
        workspace = result.path
        scanned = null
      }
    } catch (cause) {
      failure(cause)
    } finally {
      browsing = false
    }
  }
  async function scan() {
    pending = true
    scanned = null
    error = ""
    const input = workspace.trim()
    try {
      const result = await scanLocalJourney({ workspace: input })
      if (input === workspace.trim()) {
        scanned = result
        reviewedWorkspace = input
      }
    } catch (cause) {
      failure(cause)
    } finally {
      pending = false
    }
  }
  async function apply() {
    if (!scanned || reviewedWorkspace !== workspace.trim()) return
    importing = true
    error = ""
    try {
      await importLocalJourney({ workspace: reviewedWorkspace, journey_id: scanned.journey_id, slug: slug.trim(), title: title.trim(), place: place.trim() })
      location.hash = journeyDetailHash(scanned.journey_id)
    } catch (cause) {
      failure(cause)
    } finally {
      importing = false
    }
  }
</script>

<section class="studio-page task-page" aria-labelledby="import-title">
  <header class="page-header">
    <Button variant="ghost" disabled={importing} onclick={() => (location.hash = listHash)}>{message(locale, "admin.journeys.back")}</Button>
    <h1 id="import-title">{message(locale, "admin.connectors.import_journey")}</h1>
  </header>
  <div class="task-body">
    <p class="hint">{message(locale, "admin.connectors.import_intro")}</p>
    <p class="feedback">{message(locale, "admin.connectors.import_legacy_note")}</p>
    <section class="panel" aria-labelledby="trip-folder-title">
      <h2 id="trip-folder-title">{message(locale, "admin.connectors.trip_folder_advanced")}</h2>
      <p class="hint">{message(locale, "admin.connectors.scan_note")}</p>
      <div class="field-grid">
        <label class="field col-span-full"
          >{message(locale, "admin.connectors.folder_path")}<Input
            bind:value={workspace}
            oninput={() => (scanned = null)}
            disabled={pending || importing}
            placeholder="/path/to/synthetic-trip"
          /></label
        >
        {#if desktop}<Button variant="outline" onclick={browse} disabled={browsing || pending || importing}>{message(locale, "admin.connectors.browse")}</Button>{/if}
        <label class="field">{message(locale, "admin.journeys.slug")}<Input bind:value={slug} disabled={importing} /></label>
        <label class="field">{message(locale, "admin.common.title")}<Input bind:value={title} disabled={importing} /></label>
        <label class="field">{message(locale, "admin.common.place")}<Input bind:value={place} disabled={importing} /></label>
      </div>
    </section>
    {#if error}<p class="feedback feedback--error" role="alert">{error}</p>{/if}
    {#if scanned}
      <section class="panel" aria-label={message(locale, "admin.connectors.dry_run_result")}>
        <h2>{message(locale, "admin.connectors.dry_run_result")}</h2>
        <p>{scanned.plan.date_start ? formatJourneyDate(scanned.plan.date_start) : "?"} – {scanned.plan.date_end ? formatJourneyDate(scanned.plan.date_end) : "?"}</p>
        <p>{message(locale, "admin.connectors.scan_counts", { routes: scanned.plan.routes.length, stops: scanned.plan.stops.length, mementos: scanned.plan.mementos.length })}</p>
        {#if scanned.plan.issues.length}<p class="hint">{message(locale, "admin.connectors.review_notes", { count: scanned.plan.issues.length })}</p>{/if}
      </section>
    {/if}
  </div>
  <footer class="task-footer">
    <Button variant="outline" disabled={importing} onclick={() => (location.hash = listHash)}>{message(locale, "admin.common.cancel")}</Button>
    <Button variant="outline" disabled={!workspace.trim() || pending || browsing || importing} onclick={scan}
      >{message(locale, pending ? "admin.connectors.scanning" : "admin.connectors.scan_preview")}</Button
    >
    {#if scanned}<Button disabled={!slug.trim() || !title.trim() || importing} onclick={apply}>{message(locale, importing ? "admin.connectors.importing" : "admin.connectors.confirm_import")}</Button
      >{/if}
  </footer>
</section>
