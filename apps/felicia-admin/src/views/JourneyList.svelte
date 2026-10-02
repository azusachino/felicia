<script lang="ts">
  import { onMount } from "svelte"
  import { message, statusMessage, type Locale } from "../i18n"

  let { locale }: { locale: Locale } = $props()
  import {
    compileSite,
    createJourney,
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
  import { journeyDetailHash } from "../router"

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

  function onTitleInput() {
    if (!slugManuallyEdited) {
      createSlug = createTitle
        .toLowerCase()
        .trim()
        .replace(/[^\w\s-]/g, "")
        .replace(/[\s_-]+/g, "-")
        .replace(/^-+|-+$/g, "")
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
      window.location.hash = journeyDetailHash(res.id)
    } catch (cause) {
      createError = actionErrorMessage(cause)
      createState = "error"
    }
  }

  // Scan form state
  let workspace = $state("")
  let slug = $state("")
  let title = $state("")
  let place = $state("")
  let scanState = $state<"idle" | "scanning" | "ready" | "importing" | "error">("idle")
  let scanError = $state("")
  let scanned = $state<LocalJourneyPlan | null>(null)

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
    <div>
      <p class="eyebrow">{message(locale, "admin.journeys.breadcrumb")}</p>
      <h1>{message(locale, "admin.journeys.title")}</h1>
    </div>
    <div class="header-actions">
      <button class="secondary" type="button" onclick={() => (showNewJourney = !showNewJourney)}
        >{showNewJourney ? message(locale, "admin.common.close") : message(locale, "admin.journeys.new_action")}</button
      >
      <button class="secondary" type="button" onclick={load} disabled={loading}>{loading ? message(locale, "admin.common.loading") : message(locale, "admin.common.refresh")}</button>
    </div>
  </header>

  {#if showNewJourney}
    <section class="new-journey" aria-labelledby="new-journey-title">
      <div class="new-journey-tabs">
        <button type="button" class:active={creationMode === "create"} onclick={() => (creationMode = "create")}>{message(locale, "admin.journeys.create_blank")}</button>
        <button type="button" class:active={creationMode === "scan"} onclick={() => (creationMode = "scan")}>{message(locale, "admin.connectors.scan_trip_folder")}</button>
      </div>

      {#if creationMode === "create"}
        <p class="eyebrow">{message(locale, "admin.journeys.direct_authoring")}</p>
        <h2 id="new-journey-title">{message(locale, "admin.journeys.create_heading")}</h2>
        <p class="hint">{message(locale, "admin.journeys.create_note")}</p>
        <div class="new-journey-form">
          <label>{message(locale, "admin.common.title")}<input bind:value={createTitle} oninput={onTitleInput} /></label>
          <label>{message(locale, "admin.common.place")}<input bind:value={createPlace} /></label>
          <label>{message(locale, "admin.journeys.slug")}<input bind:value={createSlug} oninput={() => (slugManuallyEdited = true)} placeholder="hakone-weekend-walk-2026" /></label>
          <label>{message(locale, "admin.journeys.start_date")}<input type="date" bind:value={createDateStart} /></label>
          <label>{message(locale, "admin.journeys.end_date")}<input type="date" bind:value={createDateEnd} /></label>
          <label>{message(locale, "admin.journeys.country_optional")}<input bind:value={createCountry} /></label>
          <label>{message(locale, "admin.journeys.region_optional")}<input bind:value={createRegion} /></label>
        </div>
        <div class="new-journey-actions">
          <button class="primary" type="button" onclick={submitCreateJourney} disabled={!createTitle || !createPlace || !createSlug || !createDateStart || !createDateEnd || createState === "pending"}>
            {createState === "pending" ? message(locale, "admin.journeys.creating") : message(locale, "admin.journeys.create_action")}
          </button>
          <button class="secondary" type="button" onclick={() => (showNewJourney = false)}>{message(locale, "admin.common.cancel")}</button>
        </div>
        {#if createError}<p class="api-error" role="alert">{createError}</p>{/if}
      {:else}
        <p class="eyebrow">{message(locale, "admin.connectors.local_source_intake")}</p>
        <h2 id="new-journey-title">{message(locale, "admin.connectors.scan_heading")}</h2>
        <p class="hint">
          {message(locale, "admin.connectors.scan_note")}
        </p>
        <div class="new-journey-form">
          <label>{message(locale, "admin.connectors.folder_path")}<input bind:value={workspace} placeholder="/Users/you/trips/izu-trip-2026-08-01" /></label>
          <label>{message(locale, "admin.journeys.slug")}<input bind:value={slug} placeholder="izu-trip-2026-08-01" /></label>
          <label>{message(locale, "admin.common.title")}<input bind:value={title} placeholder="Izu · 2026-08-01 – 2026-08-02" /></label>
          <label>{message(locale, "admin.common.place")}<input bind:value={place} placeholder="Izu" /></label>
        </div>
        <div class="new-journey-actions">
          <button class="secondary" type="button" onclick={scanWorkspace} disabled={!workspace || scanState === "scanning"}
            >{scanState === "scanning" ? message(locale, "admin.connectors.scanning") : message(locale, "admin.connectors.scan_preview")}</button
          >
          {#if scanned}
            <button class="primary" type="button" onclick={importWorkspace} disabled={!slug || !title || scanState === "importing"}
              >{scanState === "importing" ? message(locale, "admin.connectors.importing") : message(locale, "admin.connectors.confirm_import")}</button
            >
          {/if}
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
    </section>
  {/if}

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
          <a class="journey-card" class:journey-card--pending={pendingByJourney[summary.journey.id] > 0} href={journeyDetailHash(summary.journey.id)}>
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
        <button type="button" onclick={triggerBuild} disabled={buildState.status === "pending"}>{buildButtonLabel(pendingJourneyCount(), buildState.status)}</button>
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
  .journeys-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }
  .header-actions,
  .new-journey-actions {
    display: flex;
    gap: 8px;
  }
  .new-journey {
    margin-top: 24px;
    padding: 20px 24px;
    border: 1px solid #dfd4c1;
    border-radius: 12px;
    background: rgb(255 250 242 / 70%);
  }
  .new-journey-tabs {
    display: inline-flex;
    gap: 4px;
    padding: 3px;
    background: #ede6d8;
    border-radius: 8px;
    margin-bottom: 16px;
  }
  .new-journey-tabs button {
    border: 0;
    border-radius: 6px;
    padding: 6px 14px;
    font-size: 13px;
    color: #6b5137;
    background: transparent;
    cursor: pointer;
  }
  .new-journey-tabs button.active {
    background: #fffaf2;
    color: #342a1e;
    font-weight: 600;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  }
  .new-journey h2 {
    margin: 2px 0 0;
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
    color: #6b5137;
    font-size: 12px;
    font-weight: 600;
  }
  .new-journey-form input {
    min-width: 0;
    padding: 9px 10px;
    border: 1px solid #d8cdbb;
    border-radius: 6px;
    font: inherit;
  }
  .scan-result {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin-top: 16px;
    padding: 12px;
    color: #6b5137;
    background: #f7eddd;
    border-radius: 6px;
    font-size: 13px;
  }
  .scan-warning {
    color: #9f522d;
  }
  .secondary {
    border: 1px solid #d8cdbb;
    border-radius: 7px;
    padding: 9px 14px;
    color: #6b5137;
    background: #fffaf2;
  }
  .secondary:disabled {
    opacity: 0.6;
    cursor: default;
  }
  /* "Confirm import" is this form's primary action and previously had no
     class at all — an unstyled <button> next to a bordered .secondary one,
     so it rendered as plain text with no button affordance. Matches the
     app's other primary-action buttons (e.g. .build-row button). */
  .primary {
    border: 0;
    border-radius: 7px;
    padding: 9px 14px;
    color: #fffaf2;
    background: #9f522d;
  }
  .primary:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .hint {
    margin-top: 24px;
    color: #766956;
  }
  .journey-cards {
    display: grid;
    gap: 12px;
    margin: 24px 0 0;
    padding: 0;
    list-style: none;
  }
  .journey-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
    padding: 20px 24px;
    border: 1px solid #dfd4c1;
    border-radius: 12px;
    color: inherit;
    text-decoration: none;
    background: rgb(255 250 242 / 55%);
    transition: border-color 0.15s ease;
  }
  .journey-card:hover {
    border-color: #b3673a;
  }
  /* Pending-build highlight (memento-lifecycle staged rebuild, ADMIN-02
     §6) — the same visual language as the journey-detail memento row: a
     left border + subtle background, plus an inline "pending build"
     label rather than color alone. */
  .journey-card--pending {
    border-left: 3px solid #b3673a;
    background: rgb(231 162 96 / 14%);
  }
  .pending-dot {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    margin-top: 8px;
    color: #9f522d;
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
    background: #b3673a;
  }
  .journey-card-main h2 {
    margin: 2px 0 0;
    font-size: 22px;
  }
  .journey-card-dates {
    margin: 6px 0 0;
    color: #766956;
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
    font-family: Georgia, serif;
    font-size: 24px;
    font-weight: 500;
  }
  .badge-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    max-width: 220px;
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
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
  }
  .build-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 14px;
  }
  .build-label {
    color: #6b5137;
    font-weight: 600;
    font-size: 14px;
  }
  .build-row button {
    border: 0;
    border-radius: 7px;
    padding: 8px 12px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .build-row button:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .build-row .trigger-status {
    margin: 0;
  }
  .trigger-note {
    margin: 8px 0 0;
    color: #766956;
    font-size: 12px;
  }
  .trigger-status {
    margin: 10px 0 0;
    font-size: 13px;
  }
  /* #3f7a52 measured 4.45:1 on this background — just under the 4.5:1 AA
     floor (axe color-contrast, serious). */
  .trigger-status--success {
    color: #2f5e40;
  }
  .trigger-status--error {
    color: #a84a34;
  }
</style>
