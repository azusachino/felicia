<script lang="ts">
  import {
    getJourney,
    listMementos,
    sortMementosBySeq,
    syncRoute,
    syncVisits,
    photoTray,
    routePointCount,
    type AdminJourney,
    type AdminMemento,
    type AdminVisitPreview,
    type AdminPhotoTrayItem,
  } from "../api"
  import { listHash } from "../router"

  let { id }: { id: string } = $props()

  let journey = $state<AdminJourney | null>(null)
  let mementos = $state<AdminMemento[]>([])
  let loading = $state(true)
  let error = $state("")

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

  // Re-runs whenever `id` changes, including a deep link straight from one
  // journey's hash to another (no full page reload).
  $effect(() => {
    loadDetail(id)
  })

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

    <section class="mementos" aria-label="Mementos">
      <h2>Mementos</h2>
      {#if mementos.length === 0}
        <p class="hint">No mementos yet.</p>
      {:else}
        <ul class="memento-list">
          {#each mementos as memento (memento.id)}
            <li class="memento-row">
              <span class="memento-seq">#{memento.seq}</span>
              <span class="memento-title">{memento.title || memento.place || memento.kind}</span>
              <span class="memento-kind">{memento.kind}</span>
              <span class={`badge badge--${memento.state}`}>{memento.state}</span>
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
  .mementos {
    margin-top: 40px;
  }
  .triggers h2,
  .mementos h2 {
    margin: 0 0 16px;
    font-family: Georgia, serif;
    font-size: 22px;
    font-weight: 500;
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
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 16px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
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
</style>
