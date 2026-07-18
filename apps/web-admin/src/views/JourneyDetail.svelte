<script lang="ts">
  import {
    getJourney,
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
    type AdminStopCandidate,
    type AdminTemplateRegistry,
    type AdminVisitPreview,
    type AdminPhotoTrayItem,
    type PlanIntakeResult,
  } from "../api"
  import { listHash, mementoEditHash } from "../router"

  let { id }: { id: string } = $props()

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
    return cause instanceof Error ? cause.message : "Request failed"
  }

  async function triggerSyncRoute() {
    routeAction = { status: "pending", message: "Syncing route…" }
    try {
      const result = await syncRoute(id)
      const count = routePointCount(result)
      routeAction = { status: "success", message: `${count} route point${count === 1 ? "" : "s"} written`, data: count }
    } catch (cause) {
      routeAction = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function triggerSyncVisits() {
    visitsAction = { status: "pending", message: "Loading visits…" }
    try {
      const visits = await syncVisits(id)
      visitsAction = { status: "success", message: `${visits.length} visit${visits.length === 1 ? "" : "s"} found`, data: visits }
    } catch (cause) {
      visitsAction = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function triggerPhotoTray() {
    trayAction = { status: "pending", message: "Loading photo tray…" }
    try {
      const assets = await photoTray(id)
      trayAction = { status: "success", message: `${assets.length} photo${assets.length === 1 ? "" : "s"} found`, data: assets }
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
      return { status: "conflict", message: "Someone else changed this candidate — reload the journey and try again." }
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
    planAction = { status: "pending", message: "Planning intake…" }
    try {
      const result = await planIntake(id)
      const issueNote = result.issues.length > 0 ? `, ${result.issues.length} issue${result.issues.length === 1 ? "" : "s"}` : ""
      planAction = { status: "success", message: `${result.stops.length} stop${result.stops.length === 1 ? "" : "s"} proposed${issueNote}`, data: result }
      await loadStopCandidates(id)
    } catch (cause) {
      planAction = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function promoteCandidate(candidate: AdminStopCandidate) {
    const kind = kindFor(candidate.id)
    if (!kind) {
      candidateActions = { ...candidateActions, [candidate.id]: { status: "error", message: "No kind available to promote into — check the kind registry." } }
      return
    }
    candidateActions = { ...candidateActions, [candidate.id]: { status: "pending", message: "Promoting…" } }
    try {
      await promoteStopCandidate(candidate.id, kind, candidate.revision)
      candidateActions = { ...candidateActions, [candidate.id]: { status: "success", message: "Promoted to a draft memento." } }
      // Both the inbox (this candidate leaves the actionable list) and the
      // memento list (the new draft appears) refresh without a reload.
      await Promise.all([loadStopCandidates(id), refreshMementos(id)])
    } catch (cause) {
      candidateActions = { ...candidateActions, [candidate.id]: candidateFailure(cause) }
    }
  }

  async function reviewCandidate(candidate: AdminStopCandidate, state: "ignored" | "merged", mergedInto?: string) {
    candidateActions = { ...candidateActions, [candidate.id]: { status: "pending", message: state === "ignored" ? "Ignoring…" : "Merging…" } }
    try {
      const updated = await reviewStopCandidate(candidate.id, { state, expectedRevision: candidate.revision, mergedInto })
      stopCandidates = stopCandidates.map((existing) => (existing.id === updated.id ? updated : existing))
      candidateActions = { ...candidateActions, [candidate.id]: { status: "success", message: state === "ignored" ? "Ignored." : "Merged." } }
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
      error = cause instanceof Error ? cause.message : "Unable to load this journey"
    } finally {
      loading = false
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

  // Re-runs whenever `id` changes, including a deep link straight from one
  // journey's hash to another (no full page reload).
  $effect(() => {
    loadDetail(id)
    loadStopCandidates(id)
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
</script>

<section class="detail">
  <a class="back-link" href={listHash}>&larr; Journeys</a>

  {#if loading}
    <p class="hint">Loading journey…</p>
  {:else if error}
    <p class="api-error" role="alert">{error}. Start the local API to load authoring data.</p>
  {:else if journey}
    <header class="detail-header">
      <p class="eyebrow">{journey.slug}</p>
      <h1>{journey.title}</h1>
      <p class="detail-meta">{journey.place} · {journey.date_start} – {journey.date_end}</p>
    </header>

    <section class="triggers" aria-label="Import and preview triggers">
      <h2>Import &amp; preview</h2>
      <div class="trigger-grid">
        <article class="trigger">
          <div class="trigger-head">
            <h3>Sync route</h3>
            <button type="button" onclick={triggerSyncRoute} disabled={routeAction.status === "pending"}>{routeAction.status === "pending" ? "Syncing…" : "Sync route"}</button>
          </div>
          <p class="trigger-note">Pulls the GPS track and writes gps_route.</p>
          {#if routeAction.status === "success"}
            <p class="trigger-status trigger-status--success" role="status">{routeAction.message}</p>
          {:else if routeAction.status === "error"}
            <p class="trigger-status trigger-status--error" role="alert">{routeAction.message}</p>
          {/if}
        </article>

        <article class="trigger">
          <div class="trigger-head">
            <h3>Preview visits</h3>
            <button type="button" onclick={triggerSyncVisits} disabled={visitsAction.status === "pending"}>{visitsAction.status === "pending" ? "Loading…" : "Preview visits"}</button>
          </div>
          <p class="trigger-note">Read-only — derived places, nothing is saved.</p>
          {#if visitsAction.status === "success"}
            <p class="trigger-status trigger-status--success" role="status">{visitsAction.message}</p>
            {#if visitsAction.data && visitsAction.data.length > 0}
              <ul class="preview-list">
                {#each visitsAction.data as visit, index (index)}
                  <li>
                    <strong>{visit.label || "Unlabeled visit"}</strong>
                    <span class="preview-meta">{visit.arrive} → {visit.depart} · {Math.round(visit.confidence * 100)}%</span>
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
            <h3>Preview photo tray</h3>
            <button type="button" onclick={triggerPhotoTray} disabled={trayAction.status === "pending"}>{trayAction.status === "pending" ? "Loading…" : "Preview photo tray"}</button>
          </div>
          <p class="trigger-note">Read-only — Immich assets, nothing is saved.</p>
          {#if trayAction.status === "success"}
            <p class="trigger-status trigger-status--success" role="status">{trayAction.message}</p>
            {#if trayAction.data && trayAction.data.length > 0}
              <ul class="preview-list">
                {#each trayAction.data as asset (asset.id)}
                  <li>
                    <strong>{asset.at}</strong>
                    <span class="preview-meta">{asset.coord ? `${asset.coord[1].toFixed(4)}, ${asset.coord[0].toFixed(4)}` : "no GPS"} · {asset.checksum.slice(0, 10)}</span>
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

    <section class="inbox" aria-label="Intake inbox">
      <div class="inbox-head">
        <h2>Intake inbox</h2>
        <button type="button" onclick={triggerPlanIntake} disabled={planAction.status === "pending"}>{planAction.status === "pending" ? "Planning…" : "Plan intake"}</button>
      </div>
      <p class="trigger-note">Runs the intake planner over the journey's sources and proposes stop candidates for review.</p>
      {#if planAction.status === "success"}
        <p class="trigger-status trigger-status--success" role="status">{planAction.message}</p>
      {:else if planAction.status === "error"}
        <p class="trigger-status trigger-status--error" role="alert">{planAction.message}</p>
      {/if}

      {#if templatesError}
        <p class="trigger-status trigger-status--error" role="alert">Kind registry unavailable: {templatesError}. Promoting is disabled until this loads.</p>
      {/if}

      {#if stopCandidatesError}
        <p class="trigger-status trigger-status--error" role="alert">{stopCandidatesError}</p>
      {:else if stopCandidates.length === 0}
        <p class="hint">No stop candidates yet. Run "Plan intake" to generate some.</p>
      {:else}
        <ul class="candidate-list">
          {#each stopCandidates as candidate (candidate.id)}
            <li class="candidate-row">
              <div class="candidate-summary">
                <div class="candidate-main">
                  <strong>{candidate.label || "Unlabeled stop"}</strong>
                  <span class="candidate-meta">{candidate.arrive} → {candidate.depart} · {Math.round(candidate.confidence * 100)}% confidence</span>
                </div>
                <span class={`badge badge--${candidate.state}`}>{candidate.state}</span>
              </div>

              {#if candidate.state === "proposed"}
                <div class="candidate-actions">
                  <label class="candidate-field">
                    Kind
                    <select
                      aria-label={`Kind for ${candidate.label || "stop"}`}
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
                  <button type="button" onclick={() => promoteCandidate(candidate)} disabled={candidateAction(candidate.id).status === "pending" || !templates}>Promote</button>
                  <button type="button" class="secondary" onclick={() => reviewCandidate(candidate, "ignored")} disabled={candidateAction(candidate.id).status === "pending"}>Ignore</button>
                  <label class="candidate-field">
                    Merge into
                    <select aria-label={`Merge target for ${candidate.label || "stop"}`} bind:value={mergeTarget[candidate.id]}>
                      <option value="">Choose a candidate…</option>
                      {#each stopCandidates.filter((other) => other.id !== candidate.id) as other (other.id)}
                        <option value={other.id}>{other.label || "Unlabeled"} ({other.state})</option>
                      {/each}
                    </select>
                  </label>
                  <button
                    type="button"
                    class="secondary"
                    onclick={() => reviewCandidate(candidate, "merged", mergeTarget[candidate.id])}
                    disabled={candidateAction(candidate.id).status === "pending" || !mergeTarget[candidate.id]}>Merge</button
                  >
                </div>
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

    <section class="mementos" aria-label="Mementos">
      <h2>Mementos</h2>
      {#if mementos.length === 0}
        <p class="hint">No mementos yet.</p>
      {:else}
        <ul class="memento-list">
          {#each mementos as memento (memento.id)}
            <li class="memento-row">
              <a class="memento-link" href={mementoEditHash(id, memento.id)}>
                <span class="memento-seq">#{memento.seq}</span>
                <span class="memento-title">{memento.title || memento.place || memento.kind}</span>
                <span class="memento-kind">{memento.kind}</span>
                <span class={`badge badge--${memento.state}`}>{memento.state}</span>
              </a>
            </li>
          {/each}
        </ul>
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
  .trigger-status--success {
    color: #3f7a52;
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
  .memento-link {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 16px;
    color: inherit;
    text-decoration: none;
  }
  .memento-seq {
    color: #a69a89;
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
  .badge--proposed {
    color: #9f522d;
    background: rgb(231 162 96 / 24%);
  }
  .badge--kept {
    color: #3f7a52;
    background: rgb(120 184 135 / 24%);
  }
  .badge--ignored,
  .badge--merged {
    color: #766956;
    background: rgb(166 154 137 / 20%);
  }
</style>
