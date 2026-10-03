<script lang="ts">
  import { push } from "svelte-spa-router"
  import { message, statusMessage, type Locale } from "../i18n"
  import {
    compileSite,
    deleteJourney,
    describeLoadFailure,
    formatJourneyDate,
    getJourney,
    getJourneyBuildStatus,
    getSiteInfo,
    getTemplates,
    isConflict,
    listMementos,
    listStopCandidates,
    planIntake,
    promoteStopCandidate,
    reviewStopCandidate,
    sortMementosBySeq,
    syncRoute,
    syncVisits,
    photoTray,
    routePointCount,
    type AdminJourney,
    type AdminMemento,
    type AdminSiteInfo,
    type AdminStopCandidate,
    type AdminTemplateRegistry,
    type AdminVisitPreview,
    type AdminPhotoTrayItem,
    type CompileReport,
    type PlanIntakeResult,
  } from "../api"
  import { listHash, mementoEditHash } from "../router"

  let { id, locale }: { id: string; locale: Locale } = $props()

  let journey = $state<AdminJourney | null>(null)
  let mementos = $state<AdminMemento[]>([])
  let loading = $state(true)
  let error = $state("")

  type ActionStatus = "idle" | "pending" | "success" | "error"

  interface ActionState<T> {
    status: ActionStatus
    message: string
    data?: T
  }

  let routeAction = $state<ActionState<number>>({ status: "idle", message: "" })
  let visitsAction = $state<ActionState<AdminVisitPreview[]>>({ status: "idle", message: "" })
  let trayAction = $state<ActionState<AdminPhotoTrayItem[]>>({ status: "idle", message: "" })

  function actionErrorMessage(cause: unknown): string {
    return cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
  }

  async function triggerSyncRoute() {
    routeAction = { status: "pending", message: message(locale, "admin.connectors.route.pending") }
    try {
      const result = await syncRoute(id)
      const count = routePointCount(result)
      routeAction = { status: "success", message: message(locale, "admin.connectors.route.success", { count }), data: count }
    } catch (cause) {
      routeAction = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function triggerSyncVisits() {
    visitsAction = { status: "pending", message: message(locale, "admin.connectors.visits.loading") }
    try {
      const visits = await syncVisits(id)
      visitsAction = { status: "success", message: message(locale, "admin.connectors.visits.success", { count: visits.length }), data: visits }
    } catch (cause) {
      visitsAction = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function triggerPhotoTray() {
    trayAction = { status: "pending", message: message(locale, "admin.connectors.photos.loading") }
    try {
      const assets = await photoTray(id)
      trayAction = { status: "success", message: message(locale, "admin.connectors.photos.success", { count: assets.length }), data: assets }
    } catch (cause) {
      trayAction = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  // Intake inbox (ADMIN-01.3b). "Plan intake" persists proposed stop
  // candidates; each proposed candidate can then be promoted (kind picker,
  // via the dedicated promote endpoint), ignored, or merged into another
  // candidate (both via the review endpoint).
  let stopCandidates = $state<AdminStopCandidate[]>([])
  let stopCandidatesError = $state("")
  let templates = $state<AdminTemplateRegistry | null>(null)
  let templatesError = $state("")
  let planAction = $state<ActionState<PlanIntakeResult>>({ status: "idle", message: "" })

  // Per-candidate action state (promote/ignore/merge), keyed by candidate
  // id. "conflict" is distinct from "error" so the inline message can use
  // the "someone else changed this" phrasing rather than a generic failure.
  type CandidateActionStatus = "idle" | "pending" | "success" | "error" | "conflict"
  interface CandidateActionState {
    status: CandidateActionStatus
    message: string
  }
  let candidateActions = $state<Record<string, CandidateActionState>>({})
  // The kind picker's current selection and the merge-target selection,
  // both keyed by candidate id.
  let selectedKind = $state<Record<string, string>>({})
  let mergeTarget = $state<Record<string, string>>({})

  function candidateAction(candidateId: string): CandidateActionState {
    return candidateActions[candidateId] ?? { status: "idle", message: "" }
  }

  function firstKind(): string {
    return templates ? (Object.keys(templates)[0] ?? "") : ""
  }

  function kindFor(candidateId: string): string {
    return selectedKind[candidateId] ?? firstKind()
  }

  // Maps a promote/review failure to the inline message shown next to the
  // candidate: a 409 gets the "someone else changed this" conflict
  // phrasing (no merge UI, per ADMIN-01.5/01.3b); anything else gets the
  // server's own error message.
  function candidateFailure(cause: unknown): CandidateActionState {
    if (isConflict(cause)) {
      return { status: "conflict", message: message(locale, "admin.connectors.candidate.conflict") }
    }
    return { status: "error", message: actionErrorMessage(cause) }
  }

  async function loadStopCandidates(journeyId: string) {
    stopCandidatesError = ""
    try {
      stopCandidates = await listStopCandidates(journeyId)
    } catch (cause) {
      stopCandidates = []
      stopCandidatesError = actionErrorMessage(cause)
    }
  }

  async function triggerPlanIntake() {
    planAction = { status: "pending", message: message(locale, "admin.connectors.inbox.pending") }
    try {
      const result = await planIntake(id)
      planAction = {
        status: "success",
        message: message(locale, "admin.connectors.inbox.proposed", { stops: result.stops.length, issues: result.issues.length }),
        data: result,
      }
      await loadStopCandidates(id)
    } catch (cause) {
      planAction = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function promoteCandidate(candidate: AdminStopCandidate) {
    const kind = kindFor(candidate.id)
    if (!kind) {
      candidateActions = { ...candidateActions, [candidate.id]: { status: "error", message: message(locale, "admin.connectors.candidate.no_kind") } }
      return
    }
    candidateActions = { ...candidateActions, [candidate.id]: { status: "pending", message: message(locale, "admin.connectors.candidate.promoting") } }
    try {
      await promoteStopCandidate(candidate.id, kind, candidate.revision)
      candidateActions = { ...candidateActions, [candidate.id]: { status: "success", message: message(locale, "admin.connectors.candidate.promoted") } }
      // Both the inbox (this candidate leaves the actionable list) and the
      // memento list (the new draft appears) refresh without a reload.
      await Promise.all([loadStopCandidates(id), refreshMementos(id)])
    } catch (cause) {
      candidateActions = { ...candidateActions, [candidate.id]: candidateFailure(cause) }
    }
  }

  async function reviewCandidate(candidate: AdminStopCandidate, state: "ignored" | "merged", mergedInto?: string) {
    candidateActions = {
      ...candidateActions,
      [candidate.id]: { status: "pending", message: state === "ignored" ? message(locale, "admin.connectors.candidate.discarding") : message(locale, "admin.connectors.candidate.merging") },
    }
    try {
      const updated = await reviewStopCandidate(candidate.id, { state, expectedRevision: candidate.revision, mergedInto })
      stopCandidates = stopCandidates.map((existing) => (existing.id === updated.id ? updated : existing))
      candidateActions = {
        ...candidateActions,
        [candidate.id]: { status: "success", message: state === "ignored" ? message(locale, "admin.connectors.candidate.discarded") : message(locale, "admin.connectors.candidate.merged") },
      }
    } catch (cause) {
      candidateActions = { ...candidateActions, [candidate.id]: candidateFailure(cause) }
    }
  }

  async function loadDetail(journeyId: string) {
    loading = true
    error = ""
    journey = null
    mementos = []
    try {
      const [journeyResult, mementoResult] = await Promise.all([getJourney(journeyId), listMementos(journeyId)])
      journey = journeyResult
      mementos = sortMementosBySeq(mementoResult)
    } catch (cause) {
      error = describeLoadFailure(cause, "this journey")
    } finally {
      loading = false
    }
  }

  let deleteState = $state<{ status: "idle" | "confirming" | "pending" | "error"; message?: string }>({ status: "idle" })

  function startDeleteJourney() {
    deleteState = { status: "confirming" }
  }

  function cancelDeleteJourney() {
    deleteState = { status: "idle" }
  }

  async function confirmDeleteJourney() {
    if (!journey) return
    deleteState = { status: "pending" }
    try {
      await deleteJourney(journey.id)
      await push(listHash)
    } catch (cause) {
      deleteState = { status: "error", message: cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed") }
    }
  }

  async function refreshMementos(journeyId: string) {
    try {
      mementos = sortMementosBySeq(await listMementos(journeyId))
    } catch {
      // Best-effort refresh after a promote — the memento list keeps its
      // last-known state if this particular re-fetch fails.
    }
  }

  // Pending-build tracking (memento-lifecycle staged rebuild — ADMIN-02
  // §6). Publish/unpublish no longer eagerly rebuild the artifact; instead
  // this drives the memento-row highlight and the Build button's count
  // suffix below. Best-effort: a failure here just leaves nothing
  // highlighted rather than failing the whole page.
  let pendingMementoIds = $state<Set<string>>(new Set())
  let pendingCount = $state(0)
  // Whether the artifact is current, stale, absent, or unreadable. Distinct
  // from buildState below, which tracks an in-flight build action.
  let artifactState = $state("")

  async function loadBuildStatus(journeyId: string) {
    try {
      const status = await getJourneyBuildStatus(journeyId)
      pendingMementoIds = new Set(status.pending_memento_ids)
      pendingCount = status.pending_count
      artifactState = status.build_state ?? ""
    } catch {
      // A status we could not read is unknown, not current. Reporting zero
      // pending here made a failed request indistinguishable from a site with
      // nothing to publish, which is the more reassuring of the two and the
      // wrong one.
      pendingMementoIds = new Set()
      pendingCount = 0
      artifactState = "unknown"
    }
  }

  // Re-runs whenever `id` changes, including a deep link straight from one
  // journey's hash to another (no full page reload).
  $effect(() => {
    loadDetail(id)
    loadStopCandidates(id)
    loadBuildStatus(id)
  })

  // The kind registry is journey-independent, so it loads once per
  // component instance rather than re-running on every `id` change.
  getTemplates()
    .then((result) => {
      templates = result
    })
    .catch((cause) => {
      templatesError = actionErrorMessage(cause)
    })

  // Build shortcut (ADMIN-02 M1 02.1d): the same compile endpoint the Site
  // page uses, surfaced here so publish -> build -> preview doesn't require
  // switching pages. Site info (for the preview link) loads once up front —
  // journey-independent, same as the kind registry above — so the link can
  // appear immediately if a previous build already made the artifact ready,
  // not only after this component triggers one itself.
  type BuildStatus = "idle" | "pending" | "success" | "error"
  interface BuildState {
    status: BuildStatus
    message: string
    report?: CompileReport
  }
  let buildState = $state<BuildState>({ status: "idle", message: "" })
  let siteInfo = $state<AdminSiteInfo | null>(null)

  getSiteInfo()
    .then((result) => {
      siteInfo = result
    })
    .catch(() => {
      // Best-effort: the preview link just stays hidden until a build here
      // succeeds and refreshes siteInfo itself.
    })

  async function triggerJourneyBuild() {
    buildState = { status: "pending", message: message(locale, "admin.build.building") }
    try {
      const report = await compileSite()
      buildState = { status: "success", message: message(locale, "admin.build.complete"), report }
      siteInfo = await getSiteInfo()
      // The build just resolved every published<->authored toggle since the
      // last one — refresh so the pending highlights/count clear.
      await loadBuildStatus(id)
    } catch (cause) {
      buildState = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  function buildButtonLabel(pending: number, status: BuildStatus): string {
    if (status === "pending") return message(locale, "admin.build.building")
    return pending > 0 ? message(locale, "admin.build.label_pending", { count: pending }) : message(locale, "admin.build.label")
  }

  // Says which of the four situations the author is in. "Built" is the only
  // one that means the artifact matches what a build would now produce, and
  // none of them claims anything about a remote deployment.
  function artifactStateLabel(state: string, pending: number): string {
    switch (state) {
      case "never_built":
        return pending > 0 ? message(locale, "admin.build.never_built_pending", { count: pending }) : message(locale, "admin.build.never_built")
      case "changed":
        return message(locale, "admin.build.changed_pending", { count: pending })
      case "built":
        return message(locale, "admin.build.built_current")
      case "unknown":
        return message(locale, "admin.build.unknown")
      default:
        return ""
    }
  }

  function previewUrl(port: string): string {
    return `http://${location.hostname}:${port}/`
  }
</script>

<section class="detail">
  <a class="back-link" href={listHash}>&larr; {message(locale, "admin.journeys.title")}</a>

  {#if loading}
    <p class="hint">{message(locale, "admin.journeys.loading")}</p>
  {:else if error}
    <p class="api-error" role="alert">{error}</p>
  {:else if journey}
    <header class="detail-header">
      <p class="eyebrow">{journey.slug}</p>
      <h1>{journey.title}</h1>
      <p class="detail-meta">{journey.place} · {formatJourneyDate(journey.date_start)} – {formatJourneyDate(journey.date_end)}</p>
    </header>

    <section class="triggers" aria-label={message(locale, "admin.connectors.import_preview")}>
      <h2>{message(locale, "admin.connectors.import_preview")}</h2>
      <div class="trigger-grid">
        <article class="trigger">
          <div class="trigger-head">
            <h3>{message(locale, "admin.connectors.route.title")}</h3>
            <button type="button" onclick={triggerSyncRoute} disabled={routeAction.status === "pending"}
              >{routeAction.status === "pending" ? message(locale, "admin.connectors.route.pending") : message(locale, "admin.connectors.route.title")}</button
            >
          </div>
          <p class="trigger-note">{message(locale, "admin.connectors.route.description")}</p>
          {#if routeAction.status === "success"}
            <p class="trigger-status trigger-status--success" role="status">{routeAction.message}</p>
          {:else if routeAction.status === "error"}
            <p class="trigger-status trigger-status--error" role="alert">{routeAction.message}</p>
          {/if}
        </article>

        <article class="trigger">
          <div class="trigger-head">
            <h3>{message(locale, "admin.connectors.visits.title")}</h3>
            <button type="button" onclick={triggerSyncVisits} disabled={visitsAction.status === "pending"}
              >{visitsAction.status === "pending" ? message(locale, "admin.common.loading") : message(locale, "admin.connectors.visits.title")}</button
            >
          </div>
          <p class="trigger-note">{message(locale, "admin.connectors.visits.description")}</p>
          {#if visitsAction.status === "success"}
            <p class="trigger-status trigger-status--success" role="status">{visitsAction.message}</p>
            {#if visitsAction.data && visitsAction.data.length > 0}
              <ul class="preview-list">
                {#each visitsAction.data as visit, index (index)}
                  <li>
                    <strong>{visit.label || message(locale, "admin.connectors.visits.unlabeled")}</strong>
                    <span class="preview-meta">{visit.arrive} → {visit.depart} · {message(locale, "admin.connectors.candidate.confidence", { percent: Math.round(visit.confidence * 100) })}</span>
                  </li>
                {/each}
              </ul>
            {/if}
          {:else if visitsAction.status === "error"}
            <p class="trigger-status trigger-status--error" role="alert">{visitsAction.message}</p>
          {/if}
        </article>

        <article class="trigger">
          <div class="trigger-head">
            <h3>{message(locale, "admin.connectors.photos.title")}</h3>
            <button type="button" onclick={triggerPhotoTray} disabled={trayAction.status === "pending"}
              >{trayAction.status === "pending" ? message(locale, "admin.common.loading") : message(locale, "admin.connectors.photos.title")}</button
            >
          </div>
          <p class="trigger-note">{message(locale, "admin.connectors.photos.description")}</p>
          {#if trayAction.status === "success"}
            <p class="trigger-status trigger-status--success" role="status">{trayAction.message}</p>
            {#if trayAction.data && trayAction.data.length > 0}
              <ul class="preview-list">
                {#each trayAction.data as asset (asset.id)}
                  <li>
                    <strong>{asset.at}</strong>
                    <span class="preview-meta"
                      >{asset.coord ? `${asset.coord[1].toFixed(4)}, ${asset.coord[0].toFixed(4)}` : message(locale, "admin.connectors.photos.no_gps")} · {asset.checksum.slice(0, 10)}</span
                    >
                  </li>
                {/each}
              </ul>
            {/if}
          {:else if trayAction.status === "error"}
            <p class="trigger-status trigger-status--error" role="alert">{trayAction.message}</p>
          {/if}
        </article>
      </div>
    </section>

    <section class="inbox" aria-label={message(locale, "admin.connectors.inbox.title")}>
      <div class="inbox-head">
        <h2>{message(locale, "admin.connectors.inbox.title")}</h2>
        <button type="button" onclick={triggerPlanIntake} disabled={planAction.status === "pending"}
          >{planAction.status === "pending" ? message(locale, "admin.connectors.inbox.pending") : message(locale, "admin.connectors.inbox.action")}</button
        >
      </div>
      <p class="trigger-note">{message(locale, "admin.connectors.inbox.description")}</p>
      {#if planAction.status === "success"}
        <p class="trigger-status trigger-status--success" role="status">{planAction.message}</p>
      {:else if planAction.status === "error"}
        <p class="trigger-status trigger-status--error" role="alert">{planAction.message}</p>
      {/if}

      {#if templatesError}
        <p class="trigger-status trigger-status--error" role="alert">{message(locale, "admin.connectors.registry_unavailable", { error: templatesError })}</p>
      {/if}

      {#if stopCandidatesError}
        <p class="trigger-status trigger-status--error" role="alert">{stopCandidatesError}</p>
      {:else if stopCandidates.length === 0}
        <p class="hint">{message(locale, "admin.connectors.inbox.empty")}</p>
      {:else}
        <ul class="candidate-list">
          {#each stopCandidates as candidate (candidate.id)}
            <li class="candidate-row">
              <div class="candidate-summary">
                <div class="candidate-main">
                  <strong>{candidate.label || message(locale, "admin.connectors.unlabeled_stop")}</strong>
                  <span class="candidate-meta"
                    >{candidate.arrive} → {candidate.depart} · {message(locale, "admin.connectors.candidate.confidence", { percent: Math.round(candidate.confidence * 100) })}</span
                  >
                </div>
                <span class={`badge badge--${candidate.state}`}>{statusMessage(locale, candidate.state)}</span>
              </div>

              {#if candidate.state === "proposed"}
                <div class="candidate-actions">
                  <label class="candidate-field">
                    {message(locale, "admin.connectors.kind_label")}
                    <select
                      aria-label={message(locale, "admin.connectors.candidate.kind_for", { label: candidate.label || message(locale, "admin.connectors.unlabeled_stop") })}
                      value={kindFor(candidate.id)}
                      onchange={(event) => (selectedKind[candidate.id] = (event.currentTarget as HTMLSelectElement).value)}
                      disabled={!templates}
                    >
                      {#if templates}
                        {#each Object.keys(templates) as kind (kind)}
                          <option value={kind}>{kind}</option>
                        {/each}
                      {/if}
                    </select>
                  </label>
                  <button type="button" onclick={() => promoteCandidate(candidate)} disabled={candidateAction(candidate.id).status === "pending" || !templates}
                    >{message(locale, "admin.connectors.promote")}</button
                  >
                  <button type="button" class="secondary" onclick={() => reviewCandidate(candidate, "ignored")} disabled={candidateAction(candidate.id).status === "pending"}
                    >{message(locale, "admin.connectors.discard")}</button
                  >
                  <label class="candidate-field">
                    {message(locale, "admin.connectors.merge_into")}
                    <select
                      aria-label={message(locale, "admin.connectors.candidate.merge_target_for", { label: candidate.label || message(locale, "admin.connectors.unlabeled_stop") })}
                      bind:value={mergeTarget[candidate.id]}
                    >
                      <option value="">{message(locale, "admin.connectors.merge_target_prompt")}</option>
                      {#each stopCandidates.filter((other) => other.id !== candidate.id) as other (other.id)}
                        <option value={other.id}>{other.label || message(locale, "admin.connectors.unlabeled_stop")} ({statusMessage(locale, other.state)})</option>
                      {/each}
                    </select>
                  </label>
                  <button
                    type="button"
                    class="secondary"
                    onclick={() => reviewCandidate(candidate, "merged", mergeTarget[candidate.id])}
                    disabled={candidateAction(candidate.id).status === "pending" || !mergeTarget[candidate.id]}>{message(locale, "admin.connectors.merge")}</button
                  >
                </div>
                <p class="trigger-note candidate-discard-hint">{message(locale, "admin.connectors.discard_note")}</p>
              {/if}

              {#if candidateAction(candidate.id).status === "error" || candidateAction(candidate.id).status === "conflict"}
                <p class="trigger-status trigger-status--error" role="alert">{candidateAction(candidate.id).message}</p>
              {:else if candidateAction(candidate.id).status === "success"}
                <p class="trigger-status trigger-status--success" role="status">{candidateAction(candidate.id).message}</p>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="mementos" aria-label={message(locale, "admin.mementos.title")}>
      <h2>{message(locale, "admin.mementos.title")}</h2>
      {#if mementos.length === 0}
        <p class="hint">{message(locale, "admin.mementos.empty")}</p>
      {:else}
        <ul class="memento-list">
          {#each mementos as memento (memento.id)}
            <li class="memento-row" class:memento-row--pending={pendingMementoIds.has(memento.id)}>
              <a class="memento-link" href={mementoEditHash(id, memento.id)}>
                <span class="memento-seq">#{memento.seq}</span>
                <span class="memento-title">{memento.title || memento.place || memento.kind}</span>
                <span class="memento-kind">{memento.kind}</span>
                {#if pendingMementoIds.has(memento.id)}
                  <span class="pending-dot" title={message(locale, "admin.connectors.pending_build")}>{message(locale, "admin.mementos.pending_build")}</span>
                {/if}
                <span class={`badge badge--${memento.state}`}>{statusMessage(locale, memento.state)}</span>
              </a>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="build-shortcut" aria-label={message(locale, "admin.build.preview_label")}>
      <div class="build-row">
        <span class="build-label">{message(locale, "admin.build.label")}</span>
        <button type="button" onclick={triggerJourneyBuild} disabled={buildState.status === "pending"}>{buildButtonLabel(pendingCount, buildState.status)}</button>
        {#if artifactStateLabel(artifactState, pendingCount)}
          <p class="hint" class:api-error={artifactState === "unknown"}>{artifactStateLabel(artifactState, pendingCount)}</p>
        {/if}
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
        {#if siteInfo?.artifact_ready}
          <a class="preview-link" href={previewUrl(siteInfo.preview_port)} target="_blank" rel="noreferrer">{message(locale, "admin.site.open_preview")}</a>
        {/if}
      </div>
      <p class="trigger-note">{message(locale, "admin.build.detail_note")}</p>
    </section>

    <section class="danger-zone" aria-label={message(locale, "admin.journeys.delete_heading")}>
      <h2>{message(locale, "admin.journeys.delete_heading")}</h2>
      <p class="trigger-note">{message(locale, "admin.journeys.delete_note")}</p>
      {#if deleteState.status === "confirming" || deleteState.status === "pending"}
        <div class="confirm-strip" role="alert">
          <p>{message(locale, "admin.journeys.delete_confirmation", { title: journey.title })}</p>
          <div class="confirm-actions">
            <button type="button" class="danger" onclick={confirmDeleteJourney} disabled={deleteState.status === "pending"}>
              {deleteState.status === "pending" ? message(locale, "admin.common.deleting") : message(locale, "admin.journeys.delete_confirm_action")}
            </button>
            <button type="button" class="secondary" onclick={cancelDeleteJourney} disabled={deleteState.status === "pending"}>{message(locale, "admin.common.cancel")}</button>
          </div>
        </div>
      {:else}
        <button type="button" class="danger" onclick={startDeleteJourney}>{message(locale, "admin.journeys.delete_action")}</button>
      {/if}
      {#if deleteState.status === "error"}
        <p class="api-error" role="alert">{deleteState.message}</p>
      {/if}
    </section>
  {/if}
</section>

<style>
  .back-link {
    display: inline-block;
    margin-bottom: 18px;
    color: #9f522d;
    font-size: 13px;
    text-decoration: none;
  }
  .back-link:hover {
    text-decoration: underline;
  }
  .hint {
    color: #766956;
  }
  .detail-header h1 {
    margin-top: 4px;
  }
  .detail-meta {
    margin: 8px 0 0;
    color: #766956;
  }
  .triggers,
  .inbox,
  .mementos {
    margin-top: 40px;
  }
  .triggers h2,
  .inbox h2,
  .mementos h2 {
    margin: 0 0 16px;
    font-family: Georgia, serif;
    font-size: 22px;
    font-weight: 500;
  }
  /* Deliberately lighter than the sections above (no Georgia display
     heading) — this is a shortcut to the Site page's build action, not a
     second home for it (ADMIN-02 M1 02.1d). */
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
  .preview-link {
    color: #9f522d;
    font-weight: 500;
    font-size: 13px;
    text-decoration: none;
  }
  .preview-link:hover {
    text-decoration: underline;
  }
  .candidate-discard-hint {
    margin-top: 8px;
  }
  .inbox-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .inbox-head h2 {
    margin: 0;
  }
  .inbox-head button {
    border: 0;
    border-radius: 7px;
    padding: 8px 12px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .inbox-head button:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .trigger-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 16px;
  }
  .trigger {
    padding: 18px;
    border: 1px solid #dfd4c1;
    border-radius: 12px;
    background: rgb(255 250 242 / 55%);
  }
  .trigger-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .trigger-head h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
  }
  .trigger-head button {
    border: 0;
    border-radius: 7px;
    padding: 8px 12px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .trigger-head button:disabled {
    opacity: 0.6;
    cursor: default;
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
  .preview-list {
    display: grid;
    gap: 6px;
    margin: 10px 0 0;
    padding: 0;
    list-style: none;
    max-height: 180px;
    overflow-y: auto;
  }
  .preview-list li {
    display: grid;
    gap: 2px;
    padding: 6px 8px;
    border-radius: 6px;
    background: rgb(255 255 255 / 50%);
    font-size: 12px;
  }
  .preview-meta {
    color: #766956;
  }
  .memento-list {
    display: grid;
    gap: 8px;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .memento-row {
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
    transition: border-color 0.15s ease;
  }
  .memento-row:hover {
    border-color: #b3673a;
  }
  /* Pending-build highlight (memento-lifecycle staged rebuild, ADMIN-02
     §6): a distinct left border + subtle background, plus the "pending
     build" label inline — not color alone, so it doesn't depend on the
     badge's hue to read. */
  .memento-row--pending {
    border-left: 3px solid #b3673a;
    background: rgb(231 162 96 / 14%);
  }
  .pending-dot {
    display: inline-flex;
    align-items: center;
    gap: 5px;
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
  .memento-link {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 16px;
    color: inherit;
    text-decoration: none;
  }
  .memento-seq {
    /* #a69a89 measured 2.54:1 on the memento row's card background — well
       under the 4.5:1 AA floor (axe color-contrast, serious). */
    color: #766956;
    font-size: 12px;
    min-width: 28px;
  }
  .memento-title {
    flex: 1;
    font-weight: 500;
  }
  .memento-kind {
    color: #766956;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .candidate-list {
    display: grid;
    gap: 10px;
    margin: 16px 0 0;
    padding: 0;
    list-style: none;
  }
  .candidate-row {
    padding: 14px 16px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
  }
  .candidate-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
  }
  .candidate-main {
    display: grid;
    gap: 2px;
  }
  .candidate-meta {
    color: #766956;
    font-size: 12px;
  }
  .candidate-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: 10px;
    margin-top: 12px;
  }
  .candidate-field {
    display: grid;
    gap: 4px;
    color: #766956;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .candidate-field select {
    padding: 7px 9px;
    border: 1px solid #d8cdbb;
    border-radius: 7px;
    color: #342a1e;
    background: #fffaf2;
    font-size: 13px;
    text-transform: none;
    letter-spacing: normal;
  }
  .candidate-actions button {
    border: 0;
    border-radius: 7px;
    padding: 8px 12px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .candidate-actions button.secondary {
    color: #6b5137;
    background: transparent;
    border: 1px solid #d8cdbb;
  }
  .candidate-actions button:disabled {
    opacity: 0.6;
    cursor: default;
  }
  /* Darkened from the original tones (axe color-contrast, serious): those
     landed 3.77-4.27:1 against these translucent tinted backgrounds, under
     the 4.5:1 AA floor — same fix as the shared badges in app.css. */
  .badge--proposed {
    color: #8a431f;
    background: rgb(231 162 96 / 24%);
  }
  .badge--kept {
    color: #2f5e40;
    background: rgb(120 184 135 / 24%);
  }
  .badge--ignored,
  .badge--merged {
    color: #5c5142;
    background: rgb(166 154 137 / 20%);
  }
  .danger-zone {
    margin: 36px 0 8px;
    padding: 18px 20px;
    border: 1px solid rgb(168 74 52 / 35%);
    border-radius: 10px;
    background: rgb(168 74 52 / 6%);
  }
  .danger-zone h2 {
    margin: 0 0 8px;
    color: #a84a34;
    font-family: Georgia, serif;
    font-size: 16px;
    font-weight: 600;
  }
  .danger-zone > .trigger-note {
    color: #5c4f3d;
  }
  .danger-zone button {
    border: 0;
    border-radius: 7px;
    padding: 9px 14px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .danger-zone button.secondary {
    color: #6b5137;
    background: transparent;
    border: 1px solid #d8cdbb;
  }
  .danger-zone button.danger {
    color: #fffaf2;
    background: #a84a34;
  }
  .confirm-strip {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 12px;
    padding: 12px 14px;
    border-radius: 8px;
    background: #fffaf2;
    border: 1px solid #dfd4c1;
  }
  .confirm-strip p {
    margin: 0;
    font-size: 13px;
    color: #6b5137;
  }
  .confirm-actions {
    display: flex;
    gap: 8px;
  }
</style>
