<script lang="ts">
  import { onMount } from "svelte"
  import { loadJourneySummaries, type AdminJourneySummary, type MementoState } from "../api"
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

  onMount(load)
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
          <a class="journey-card" href={journeyDetailHash(summary.journey.id)}>
            <div class="journey-card-main">
              <p class="eyebrow">{summary.journey.slug}</p>
              <h2>{summary.journey.title}</h2>
              <p class="journey-card-dates">{summary.journey.date_start} – {summary.journey.date_end}</p>
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
</style>
