<script lang="ts">
  import { onMount } from "svelte"
  import { compileSite, getSiteInfo, type AdminSiteInfo, type CompileReport } from "../api"

  let info = $state<AdminSiteInfo | null>(null)
  let loading = $state(true)
  let error = $state("")

  type BuildStatus = "idle" | "pending" | "success" | "error"
  interface BuildState {
    status: BuildStatus
    message: string
    report?: CompileReport
  }
  let build = $state<BuildState>({ status: "idle", message: "" })

  async function load() {
    loading = true
    error = ""
    try {
      info = await getSiteInfo()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "Unable to load site info"
    } finally {
      loading = false
    }
  }

  async function triggerBuild() {
    build = { status: "pending", message: "Building…" }
    try {
      const report = await compileSite()
      build = { status: "success", message: "Build complete.", report }
      // Refresh site info so the preview link appears/updates once the
      // artifact is ready (artifact_ready flips from false to true on the
      // very first build).
      await load()
    } catch (cause) {
      build = { status: "error", message: cause instanceof Error ? cause.message : "Build failed" }
    }
  }

  function previewUrl(port: string): string {
    return `http://${location.hostname}:${port}/`
  }

  onMount(load)
</script>

<section class="site">
  <header class="site-header">
    <p class="eyebrow">Felicia / Site &amp; Deploy</p>
    <h1>Site &amp; Deploy</h1>
  </header>

  <p class="hint">
    This is the offline, local deployment target (epic FELICIA-ADMIN-02 milestone M0): building compiles the published content into a static artifact directory on this machine — no admin data, drafts,
    or credentials leave it. That artifact directory can be hosted anywhere a static file server can reach (a CDN, a bucket, GitHub Pages, or just this machine) — this page's preview link is the
    built-in local server for verifying the result before you ship it anywhere.
  </p>

  {#if loading}
    <p class="hint">Loading site info…</p>
  {:else if error}
    <p class="api-error" role="alert">{error}. Start the local API to load site info.</p>
  {:else if info}
    <section class="site-info" aria-label="Build output">
      <div class="info-row">
        <span class="info-label">Output directory</span>
        <code class="info-value">{info.out_dir}</code>
      </div>

      {#if info.artifact_ready}
        <div class="info-row">
          <span class="info-label">Preview</span>
          <a class="preview-link" href={previewUrl(info.preview_port)} target="_blank" rel="noreferrer">{previewUrl(info.preview_port)}</a>
        </div>
      {/if}

      {#if !info.spa_ready}
        <p class="hint">The preview will serve the compiled JSON only until the public SPA is built (run <code>make web-build</code>) — the built-in server overlays the artifact on that build.</p>
      {/if}
    </section>

    <section class="build" aria-label="Build site">
      <div class="build-head">
        <h2>Build</h2>
        <button type="button" onclick={triggerBuild} disabled={build.status === "pending"}>{build.status === "pending" ? "Building…" : "Build site"}</button>
      </div>
      <p class="trigger-note">Compiles all published journeys and mementos into the output directory above.</p>

      {#if build.status === "success"}
        <p class="trigger-status trigger-status--success" role="status">{build.message}</p>
        {#if build.report}
          <dl class="report-grid">
            <div class="report-cell">
              <dt>Journeys</dt>
              <dd>{build.report.Journeys}</dd>
            </div>
            <div class="report-cell">
              <dt>Mementos</dt>
              <dd>{build.report.Mementos}</dd>
            </div>
            <div class="report-cell">
              <dt>Media</dt>
              <dd>{build.report.Media}</dd>
            </div>
            <div class="report-cell">
              <dt>Removed</dt>
              <dd>{build.report.Removed}</dd>
            </div>
          </dl>
        {/if}
      {:else if build.status === "error"}
        <p class="trigger-status trigger-status--error" role="alert">{build.message}</p>
      {/if}
    </section>
  {/if}
</section>

<style>
  .site-header h1 {
    margin-top: 4px;
  }
  .hint {
    margin-top: 16px;
    color: #766956;
  }
  .api-error {
    margin-top: 16px;
  }
  .site-info {
    margin-top: 32px;
    padding: 20px 24px;
    border: 1px solid #dfd4c1;
    border-radius: 12px;
    background: rgb(255 250 242 / 55%);
  }
  .info-row {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .info-row + .info-row {
    margin-top: 12px;
  }
  .info-label {
    min-width: 120px;
    color: #766956;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .info-value {
    padding: 4px 8px;
    border-radius: 6px;
    background: rgb(255 255 255 / 60%);
    font-size: 13px;
  }
  .preview-link {
    color: #9f522d;
    font-weight: 500;
  }
  .build {
    margin-top: 40px;
  }
  .build-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .build-head h2 {
    margin: 0;
    font-family: Georgia, serif;
    font-size: 22px;
    font-weight: 500;
  }
  .build-head button {
    border: 0;
    border-radius: 7px;
    padding: 8px 12px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .build-head button:disabled {
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
  .report-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
    gap: 12px;
    margin: 16px 0 0;
  }
  .report-cell {
    padding: 14px 16px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
    text-align: center;
  }
  .report-cell dt {
    color: #766956;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .report-cell dd {
    margin: 4px 0 0;
    font-family: Georgia, serif;
    font-size: 22px;
    font-weight: 500;
  }
</style>
