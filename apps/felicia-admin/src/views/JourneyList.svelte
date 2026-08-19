<script lang="ts">
  import { onMount } from "svelte"
  import { compileSite, getBuildStatus, loadJourneySummaries, type AdminJourneySummary, type CompileReport, type MementoState } from "../api"
  import { journeyDetailHash } from "../router"

  const stateOrder: MementoState[] = ["candidate", "draft", "authored", "published", "archived"]

  let summaries = $state<AdminJourneySummary[]>([])
  let loading = $state(true)
  let error = $state("")

  async function load() {
    loading = true
    error = ""
    try {
      summaries = await loadJourneySummaries()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "Unable to load journeys"
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
    return cause instanceof Error ? cause.message : "Request failed"
  }

  async function triggerBuild() {
    buildState = { status: "pending", message: "Building…" }
    try {
      const report = await compileSite()
      buildState = { status: "success", message: "Build complete.", report }
      // One compile builds every journey, so this clears every pending
      // count/highlight, not just the ones visible on this page.
      await loadBuildStatus()
    } catch (cause) {
      buildState = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  function buildButtonLabel(pending: number, status: BuildStatus): string {
    if (status === "pending") return "Building…"
    return pending > 0 ? `Build & preview (${pending})` : "Build & preview"
  }

  onMount(() => {
    load()
    loadBuildStatus()
  })
</script>

<section class="journeys">
  <header class="journeys-header">
    <div>
      <p class="eyebrow">Felicia / Journeys</p>
      <h1>Journeys</h1>
    </div>
    <button class="secondary" type="button" onclick={load} disabled={loading}>{loading ? "Loading…" : "Refresh"}</button>
  </header>

  {#if loading}
    <p class="hint">Loading journeys…</p>
  {:else if error}
    <p class="api-error" role="alert">{error}. Start the local API to load authoring data.</p>
  {:else if summaries.length === 0}
    <p class="hint">No journeys yet.</p>
  {:else}
    <ul class="journey-cards">
      {#each summaries as summary (summary.journey.id)}
        <li>
          <a class="journey-card" class:journey-card--pending={pendingByJourney[summary.journey.id] > 0} href={journeyDetailHash(summary.journey.id)}>
            <div class="journey-card-main">
              <p class="eyebrow">{summary.journey.slug}</p>
              <h2>{summary.journey.title}</h2>
              <p class="journey-card-dates">{summary.journey.date_start} – {summary.journey.date_end}</p>
              {#if pendingByJourney[summary.journey.id] > 0}
                <span class="pending-dot">{pendingByJourney[summary.journey.id]} pending build</span>
              {/if}
            </div>
            <div class="journey-card-meta">
              <span class="stat">
                <strong>{summary.mementoCount}</strong>
                <span class="card-note">mementos</span>
              </span>
              <span class="stat">
                <strong>{summary.stopCandidateCount ?? "—"}</strong>
                <span class="card-note">stop candidates</span>
              </span>
              <div class="badge-row">
                {#each stateOrder as state (state)}
                  {#if summary.stateCounts[state]}
                    <span class={`badge badge--${state}`}>{state} · {summary.stateCounts[state]}</span>
                  {/if}
                {/each}
              </div>
            </div>
          </a>
        </li>
      {/each}
    </ul>

    <section class="build-shortcut" aria-label="Build and preview">
      <div class="build-row">
        <span class="build-label">Build &amp; preview</span>
        <button type="button" onclick={triggerBuild} disabled={buildState.status === "pending"}>{buildButtonLabel(pendingJourneyCount(), buildState.status)}</button>
        {#if buildState.status === "success"}
          {#if buildState.report}
            <span class="trigger-status trigger-status--success build-report">
              {buildState.report.Journeys} journeys · {buildState.report.Mementos} mementos · {buildState.report.Media} media · {buildState.report.Removed} removed
            </span>
          {/if}
        {:else if buildState.status === "error"}
          <span class="trigger-status trigger-status--error">{buildState.message}</span>
        {/if}
      </div>
      <p class="trigger-note">Compiles all published journeys and mementos in one pass — resolves every pending journey above.</p>
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
  .trigger-status--success {
    color: #3f7a52;
  }
  .trigger-status--error {
    color: #a84a34;
  }
</style>
