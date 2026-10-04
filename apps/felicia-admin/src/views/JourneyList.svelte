<script lang="ts">
  import { onMount } from "svelte"
  import { Button } from "$lib/components/ui/button"
  import IconButton from "$lib/components/IconButton.svelte"
  import { FolderOpen, FolderSearch, Plus, Hammer, Download, X } from "@lucide/svelte"
  import { goto } from "$app/navigation"
  import { resolve } from "$app/paths"
  import { message, statusMessage, type Locale } from "../i18n"

  let { locale, desktop = false }: { locale: Locale; desktop?: boolean } = $props()
  import {
    compileSite,
    createJourney,
    pickDesktopFolder,
    describeLoadFailure,
    formatJourneyDate,
    getBuildStatus,
    importLocalJourney,
    loadJourneySummaries,
    scanLocalJourney,
    type AdminJourneySummary,
    type CompileReport,
    type LocalJourneyPlan,
    type MementoState,
  } from "../api"
  import { journeyDetailPath } from "../router"

  const stateOrder: MementoState[] = ["candidate", "draft", "authored", "published", "archived"]

  let summaries = $state<AdminJourneySummary[]>([])
  let loading = $state(true)
  let error = $state("")
  let showNewJourney = $state(false)
  let creationMode = $state<"create" | "scan">("create")

  // Create form state
  let createTitle = $state("")
  let createPlace = $state("")
  let createSlug = $state("")
  let createDateStart = $state("")
  let createDateEnd = $state("")
  let createCountry = $state("")
  let createRegion = $state("")
  let createState = $state<"idle" | "pending" | "error">("idle")
  let createError = $state("")
  let slugManuallyEdited = $state(false)
  const fallbackSlug = `journey-${crypto.getRandomValues(new Uint32Array(1))[0].toString(16)}`
  let creationDialog = $state<HTMLDialogElement>()
  let creationOpener: HTMLButtonElement | null = null

  function openCreation(mode: "create" | "scan", event: MouseEvent) {
    creationMode = mode
    creationOpener = event.currentTarget as HTMLButtonElement
    showNewJourney = true
  }

  function creationClosed() {
    showNewJourney = false
    creationOpener?.focus()
  }

  $effect(() => {
    if (!creationDialog) return
    if (showNewJourney && !creationDialog.open) {
      creationDialog.showModal()
      creationDialog.querySelector("input")?.focus()
    } else if (!showNewJourney && creationDialog.open) {
      creationDialog.close()
    }
  })

  function cancelCreation(event?: Event) {
    if (createState === "pending" || scanState === "importing") {
      event?.preventDefault()
      return
    }
    showNewJourney = false
  }

  function onTitleInput() {
    if (!slugManuallyEdited) {
      createSlug =
        createTitle
          .toLowerCase()
          .trim()
          .replace(/[^\w\s-]/g, "")
          .replace(/[\s_-]+/g, "-")
          .replace(/^-+|-+$/g, "") || fallbackSlug
    }
  }

  async function submitCreateJourney() {
    createState = "pending"
    createError = ""
    try {
      const res = await createJourney({
        title: createTitle.trim(),
        place: createPlace.trim(),
        slug: createSlug.trim(),
        date_start: createDateStart,
        date_end: createDateEnd,
        country: createCountry.trim() || undefined,
        region: createRegion.trim() || undefined,
      })
      showNewJourney = false
      await goto(resolve(journeyDetailPath(res.id)))
    } catch (cause) {
      createError = actionErrorMessage(cause)
      createState = "error"
    }
  }

  // Scan form state
  let workspace = $state("")
  let browseState = $state<"idle" | "pending">("idle")
  let slug = $state("")
  let title = $state("")
  let place = $state("")
  let scanState = $state<"idle" | "scanning" | "ready" | "importing" | "error">("idle")
  let scanError = $state("")
  let scanned = $state<LocalJourneyPlan | null>(null)

  // Desktop only: ask the native shell for a folder and fill the input.
  // The web admin has no /api/desktop bridge; the typed-path input stays.
  async function browseWorkspace() {
    browseState = "pending"
    try {
      const res = await pickDesktopFolder(message(locale, "admin.connectors.folder_path"))
      if (res.selected && res.path) workspace = res.path
    } catch (cause) {
      scanError = actionErrorMessage(cause)
    } finally {
      browseState = "idle"
    }
  }

  async function load() {
    loading = true
    error = ""
    try {
      summaries = await loadJourneySummaries()
    } catch (cause) {
      error = describeLoadFailure(cause, "journeys")
    } finally {
      loading = false
    }
  }

  // Pending-build tracking (memento-lifecycle staged rebuild — ADMIN-02
  // §6): per-journey pending counts drive the card highlight and the
  // bottom Build & preview action's count. Best-effort, same as the
  // journey-detail page — a failure here just leaves nothing highlighted.
  let pendingByJourney = $state<Record<string, number>>({})

  async function loadBuildStatus() {
    try {
      const status = await getBuildStatus()
      pendingByJourney = status.pending_by_journey
    } catch {
      pendingByJourney = {}
    }
  }

  function pendingJourneyCount(): number {
    return Object.keys(pendingByJourney).length
  }

  type BuildStatus = "idle" | "pending" | "success" | "error"
  interface BuildState {
    status: BuildStatus
    message: string
    report?: CompileReport
  }
  let buildState = $state<BuildState>({ status: "idle", message: "" })

  function actionErrorMessage(cause: unknown): string {
    return cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
  }

  async function triggerBuild() {
    buildState = { status: "pending", message: message(locale, "admin.build.building") }
    try {
      const report = await compileSite()
      buildState = { status: "success", message: message(locale, "admin.build.complete"), report }
      // One compile builds every journey, so this clears every pending
      // count/highlight, not just the ones visible on this page.
      await loadBuildStatus()
    } catch (cause) {
      buildState = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  function buildButtonLabel(pending: number, status: BuildStatus): string {
    if (status === "pending") return message(locale, "admin.build.building")
    return pending > 0 ? message(locale, "admin.build.label_pending", { count: pending }) : message(locale, "admin.build.label")
  }

  async function scanWorkspace() {
    scanState = "scanning"
    scanError = ""
    scanned = null
    try {
      scanned = await scanLocalJourney({ workspace })
      scanState = "ready"
    } catch (cause) {
      scanError = actionErrorMessage(cause)
      scanState = "error"
    }
  }

  async function importWorkspace() {
    if (!scanned) return
    scanState = "importing"
    scanError = ""
    try {
      await importLocalJourney({ workspace, journey_id: scanned.journey_id, slug, title, place })
      showNewJourney = false
      scanned = null
      scanState = "idle"
      await load()
      await loadBuildStatus()
    } catch (cause) {
      scanError = actionErrorMessage(cause)
      scanState = "error"
    }
  }

  onMount(() => {
    load()
    loadBuildStatus()
  })
</script>

<section class="journeys">
  <header class="journeys-header">
    <h1>{message(locale, "admin.journeys.title")}</h1>
    <div class="header-actions">
      <Button variant="outline" size="sm" type="button" onclick={(event) => openCreation("scan", event)} aria-label={message(locale, "admin.connectors.scan_trip_folder")}
        ><FolderSearch size={16} aria-hidden="true" />{message(locale, "admin.common.scan")}</Button
      >
      <Button size="sm" type="button" onclick={(event) => openCreation("create", event)}><Plus size={16} aria-hidden="true" />{message(locale, "admin.journeys.new_action")}</Button>
    </div>
  </header>

  <dialog class="studio-dialog new-journey" bind:this={creationDialog} aria-labelledby="new-journey-title" oncancel={cancelCreation} onclose={creationClosed}>
    <header class="sheet-header">
      <h2 id="new-journey-title">{message(locale, creationMode === "create" ? "admin.journeys.create_heading" : "admin.connectors.scan_heading")}</h2>
      <IconButton variant="ghost" label={message(locale, "admin.common.close")} onclick={() => cancelCreation()} disabled={createState === "pending" || scanState === "importing"}
        ><X size={16} aria-hidden="true" /></IconButton
      >
    </header>
    <div class="sheet-body">
      {#if creationMode === "create"}
        <p class="hint">{message(locale, "admin.journeys.create_note")}</p>
        <form
          id="journey-create-form"
          class="new-journey-form"
          onsubmit={(event) => {
            event.preventDefault()
            submitCreateJourney()
          }}
        >
          <label>{message(locale, "admin.common.title")}<input required bind:value={createTitle} oninput={onTitleInput} /></label>
          <label>{message(locale, "admin.common.place")}<input required bind:value={createPlace} /></label>
          <label>{message(locale, "admin.journeys.start_date")}<input required type="date" bind:value={createDateStart} /></label>
          <label>{message(locale, "admin.journeys.end_date")}<input required type="date" min={createDateStart} bind:value={createDateEnd} /></label>
          <details class="more-options">
            <summary>{message(locale, "admin.journeys.more_options")}</summary>
            <div class="optional-fields">
              <label>{message(locale, "admin.journeys.slug")}<input required bind:value={createSlug} oninput={() => (slugManuallyEdited = true)} /></label>
              <label>{message(locale, "admin.journeys.country_optional")}<input bind:value={createCountry} /></label>
              <label>{message(locale, "admin.journeys.region_optional")}<input bind:value={createRegion} /></label>
            </div>
          </details>
        </form>
        {#if createError}<p class="api-error" role="alert">{createError}</p>{/if}
      {:else}
        <p class="hint">
          {message(locale, "admin.connectors.scan_note")}
        </p>
        <div class="new-journey-form">
          <div class="source-folder">
            <label for="source-folder-path">{message(locale, "admin.connectors.folder_path")}</label>
            <div class="path-control">
              <input id="source-folder-path" bind:value={workspace} placeholder="/Users/you/trips/izu-trip-2026-08-01" />
              {#if desktop}<IconButton label={message(locale, "admin.connectors.browse")} onclick={browseWorkspace} disabled={browseState === "pending"}
                  ><FolderOpen size={16} aria-hidden="true" /></IconButton
                >{/if}
            </div>
          </div>
          <label>{message(locale, "admin.journeys.slug")}<input bind:value={slug} placeholder="izu-trip-2026-08-01" /></label>
          <label>{message(locale, "admin.common.title")}<input bind:value={title} placeholder="Izu · 2026-08-01 – 2026-08-02" /></label>
          <label>{message(locale, "admin.common.place")}<input bind:value={place} placeholder="Izu" /></label>
        </div>
        {#if scanError}<p class="api-error" role="alert">{scanError}</p>{/if}
        {#if scanned}
          <div class="scan-result">
            <strong>{message(locale, "admin.connectors.dry_run_result")}</strong>
            <span>{scanned.plan.date_start ? formatJourneyDate(scanned.plan.date_start) : "?"} – {scanned.plan.date_end ? formatJourneyDate(scanned.plan.date_end) : "?"}</span>
            <span>{message(locale, "admin.connectors.scan_counts", { routes: scanned.plan.routes.length, stops: scanned.plan.stops.length, mementos: scanned.plan.mementos.length })}</span>
            {#if scanned.plan.issues.length > 0}<span class="scan-warning">{message(locale, "admin.connectors.review_notes", { count: scanned.plan.issues.length })}</span>{/if}
          </div>
        {/if}
      {/if}
    </div>
    <footer class="sheet-actions">
      <button class="secondary" type="button" onclick={() => cancelCreation()} disabled={createState === "pending" || scanState === "importing"}>{message(locale, "admin.common.cancel")}</button>
      {#if creationMode === "create"}
        <Button
          type="submit"
          form="journey-create-form"
          aria-label={createState === "pending" ? message(locale, "admin.journeys.creating") : message(locale, "admin.journeys.create_action")}
          disabled={!createTitle.trim() || !createPlace.trim() || !createSlug.trim() || !createDateStart || !createDateEnd || createState === "pending"}
        >
          <Plus size={16} aria-hidden="true" />
          {createState === "pending" ? message(locale, "admin.journeys.creating") : message(locale, "admin.common.create")}
        </Button>
      {:else}
        <Button
          variant="outline"
          type="button"
          onclick={scanWorkspace}
          disabled={!workspace || scanState === "scanning"}
          aria-label={scanState === "scanning" ? message(locale, "admin.connectors.scanning") : message(locale, "admin.connectors.scan_preview")}
          ><FolderSearch size={16} aria-hidden="true" />{scanState === "scanning" ? message(locale, "admin.connectors.scanning") : message(locale, "admin.common.scan")}</Button
        >
        {#if scanned}<Button
            type="button"
            onclick={importWorkspace}
            disabled={!slug || !title || scanState === "importing"}
            aria-label={scanState === "importing" ? message(locale, "admin.connectors.importing") : message(locale, "admin.connectors.confirm_import")}
          >
            <Download size={16} aria-hidden="true" />{scanState === "importing" ? message(locale, "admin.connectors.importing") : message(locale, "admin.common.import")}</Button
          >{/if}
      {/if}
    </footer>
  </dialog>

  {#if loading}
    <p class="hint">{message(locale, "admin.journeys.loading")}</p>
  {:else if error}
    <p class="api-error" role="alert">{error}</p>
  {:else if summaries.length === 0}
    <p class="hint">{message(locale, "admin.journeys.empty")}</p>
  {:else}
    <ul class="journey-cards">
      {#each summaries as summary (summary.journey.id)}
        <li>
          <a class="journey-card" class:journey-card--pending={pendingByJourney[summary.journey.id] > 0} href={resolve(journeyDetailPath(summary.journey.id))}>
            <div class="journey-card-main">
              <p class="eyebrow">{summary.journey.slug}</p>
              <h2>{summary.journey.title}</h2>
              <p class="journey-card-dates">{formatJourneyDate(summary.journey.date_start)} – {formatJourneyDate(summary.journey.date_end)}</p>
              {#if pendingByJourney[summary.journey.id] > 0}
                <span class="pending-dot">{pendingByJourney[summary.journey.id]} {message(locale, "admin.journeys.pending_build")}</span>
              {/if}
            </div>
            <div class="journey-card-meta">
              <span class="stat">
                <strong>{summary.mementoCount}</strong>
                <span class="card-note">{message(locale, "admin.stats.mementos")}</span>
              </span>
              <span class="stat">
                <strong>{summary.stopCandidateCount ?? "—"}</strong>
                <span class="card-note">{message(locale, "admin.stats.stop_candidates")}</span>
              </span>
              <div class="badge-row">
                {#each stateOrder as state (state)}
                  {#if summary.stateCounts[state]}
                    <span class={`badge badge--${state}`}>{statusMessage(locale, state)} · {summary.stateCounts[state]}</span>
                  {/if}
                {/each}
              </div>
            </div>
          </a>
        </li>
      {/each}
    </ul>

    <section class="build-shortcut" aria-label={message(locale, "admin.build.preview_label")}>
      <div class="build-row">
        <span class="build-label">{message(locale, "admin.build.label")}</span>
        <Button type="button" onclick={triggerBuild} disabled={buildState.status === "pending"} aria-label={buildButtonLabel(pendingJourneyCount(), buildState.status)}
          ><Hammer size={16} aria-hidden="true" />{buildState.status === "pending" ? message(locale, "admin.build.building") : message(locale, "admin.common.build")}{#if pendingJourneyCount() > 0}
            ({pendingJourneyCount()}){/if}</Button
        >
        {#if buildState.status === "success"}
          {#if buildState.report}
            <span class="trigger-status trigger-status--success build-report">
              {message(locale, "admin.build.report_summary", {
                journeys: buildState.report.Journeys,
                mementos: buildState.report.Mementos,
                media: buildState.report.Media,
                removed: buildState.report.Removed,
              })}
            </span>
          {/if}
        {:else if buildState.status === "error"}
          <span class="trigger-status trigger-status--error">{buildState.message}</span>
        {/if}
      </div>
      <p class="trigger-note">{message(locale, "admin.build.list_note")}</p>
    </section>
  {/if}
</section>

<style>
  .source-folder {
    display: grid;
    align-content: start;
    gap: 6px;
    min-width: 0;
  }
  .path-control {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 8px;
  }
  .path-control input {
    min-width: 0;
    width: 100%;
  }
  .journeys {
    padding: 24px;
  }
  .journeys-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 16px;
    position: sticky;
    top: 0;
    z-index: 1;
    min-height: 52px;
    margin: -24px -24px 24px;
    padding: 6px 24px;
    border-bottom: 1px solid var(--line);
    background: var(--surface);
    --wails-draggable: drag;
  }
  .journeys-header h1 {
    font-size: 16px;
  }
  .header-actions {
    --wails-draggable: no-drag;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .new-journey-form {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    margin: 18px 0;
  }
  .new-journey-form label {
    display: grid;
    gap: 5px;
    color: var(--muted);
    font-size: 13px;
    font-weight: 500;
  }
  .new-journey-form input {
    min-width: 0;
    padding: 9px 10px;
    border: 1px solid var(--line);
    border-radius: 8px;
    color: var(--text);
    background: var(--surface);
    font: inherit;
  }
  .more-options {
    grid-column: 1 / -1;
    color: var(--muted);
  }
  .more-options summary {
    cursor: pointer;
    font-size: 13px;
    padding: 4px 0;
  }
  .optional-fields {
    display: grid;
    gap: 12px;
    margin-top: 12px;
  }
  .scan-result {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin-top: 16px;
    padding: 12px;
    color: var(--text);
    background: var(--surface-muted);
    border-radius: 8px;
    font-size: 13px;
  }
  .scan-warning {
    color: var(--accent);
  }
  .hint {
    margin: 16px 0;
    color: var(--muted);
    line-height: 1.5;
  }
  .sheet-body .hint {
    margin-top: 0;
  }
  .journey-cards {
    display: grid;
    gap: 12px;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .journey-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
    padding: 16px;
    border: 1px solid var(--line);
    border-radius: 10px;
    color: inherit;
    text-decoration: none;
    background: var(--surface-raised);
    transition: border-color 0.15s ease;
  }
  .journey-card:hover {
    border-color: var(--accent);
  }
  /* Pending-build highlight (memento-lifecycle staged rebuild, ADMIN-02
     §6) — the same visual language as the journey-detail memento row: a
     left border + subtle background, plus an inline "pending build"
     label rather than color alone. */
  .journey-card--pending {
    border-inline-start: 3px solid var(--accent);
    background: var(--accent-soft);
  }
  .pending-dot {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    margin-top: 8px;
    color: var(--accent);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    white-space: nowrap;
  }
  .pending-dot::before {
    content: "";
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--accent);
  }
  .journey-card-main h2 {
    margin: 2px 0 0;
    font-size: 16px;
  }
  .journey-card-dates {
    margin: 6px 0 0;
    color: var(--muted);
    font-size: 13px;
  }
  .journey-card-meta {
    display: flex;
    align-items: center;
    gap: 24px;
    flex-shrink: 0;
  }
  .stat {
    display: grid;
    gap: 2px;
    text-align: right;
  }
  .stat strong {
    font-size: 16px;
    font-weight: 600;
  }
  .badge-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    max-width: 220px;
  }
  @media (max-width: 820px) {
    .journeys {
      padding: 20px 16px;
    }
    .journeys-header {
      margin: -20px -16px 20px;
      padding-inline: 16px;
    }
  }
  @media (max-width: 720px) {
    .journey-card {
      flex-direction: column;
      align-items: flex-start;
    }
    .journey-card-meta {
      width: 100%;
      justify-content: space-between;
    }
  }
  /* Bottom Build & preview action (ADMIN-02 §6) — same shortcut styling
     as the journey-detail page's build-shortcut section. */
  .build-shortcut {
    margin-top: 24px;
    padding: 14px 18px;
    border: 1px solid var(--line);
    border-radius: 10px;
    background: var(--surface-raised);
  }
  .build-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 14px;
  }
  .build-label {
    color: var(--text);
    font-weight: 600;
    font-size: 14px;
  }
  .build-row .trigger-status {
    margin: 0;
  }
  .trigger-note {
    margin: 8px 0 0;
    color: var(--muted);
    font-size: 12px;
  }
  .trigger-status {
    margin: 10px 0 0;
    font-size: 13px;
  }
  /* #3f7a52 measured 4.45:1 on this background — just under the 4.5:1 AA
     floor (axe color-contrast, serious). */
  .trigger-status--success {
    color: var(--text);
  }
  .trigger-status--error {
    color: var(--danger);
  }
</style>
