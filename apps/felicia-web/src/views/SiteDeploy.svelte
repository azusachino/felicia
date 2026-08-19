<script lang="ts">
  import { onMount } from "svelte"
  import {
    browseDirectories,
    compileSite,
    getSiteInfo,
    getSiteSettings,
    updateSiteOutDir,
    updateSiteSettings,
    type AdminBrowseResult,
    type AdminSiteInfo,
    type AdminSiteSettings,
    type CompileReport,
  } from "../api"

  let info = $state<AdminSiteInfo | null>(null)
  let loading = $state(true)
  let error = $state("")

  function actionErrorMessage(cause: unknown): string {
    return cause instanceof Error ? cause.message : "Request failed"
  }

  type BuildStatus = "idle" | "pending" | "success" | "error"
  interface BuildState {
    status: BuildStatus
    message: string
    report?: CompileReport
  }
  let build = $state<BuildState>({ status: "idle", message: "" })

  async function loadInfo() {
    info = await getSiteInfo()
  }

  // --- Site identity (ADMIN-02 M2 02.2c) -------------------------------------
  // `settings` is the last-saved-from-server record; `draft` is the editable
  // copy the form binds to. Kept separate so a page reload (or a re-fetch
  // after Build) can't silently clobber in-progress edits — only saveSettings
  // re-derives draft from a fresh server response.
  const designChoices: { id: AdminSiteSettings["design"]; label: string }[] = [
    { id: "v1", label: "Map" },
    { id: "v2", label: "Collection" },
    { id: "v3", label: "Journal" },
    { id: "v4", label: "Atlas" },
  ]

  type SiteIdentityDraft = Omit<AdminSiteSettings, "accent"> & { accent: string }

  // <input type="color"> requires a valid #rrggbb value at all times, but the
  // server allows an unset ("") accent. Falling back to a neutral default
  // here (and always sending whatever the swatch currently shows on save) is
  // simpler than tracking "did the user touch this field" — the tradeoff is
  // that saving once an accent has never been set will persist this default
  // rather than leaving accent "".
  const FALLBACK_ACCENT = "#ea580c"

  function draftFromSettings(source: AdminSiteSettings): SiteIdentityDraft {
    return { ...source, accent: source.accent || FALLBACK_ACCENT }
  }

  let settings = $state<AdminSiteSettings | null>(null)
  let draft = $state<SiteIdentityDraft | null>(null)

  type SaveStatus = "idle" | "pending" | "success" | "error"
  interface SaveState {
    status: SaveStatus
    message: string
  }
  let save = $state<SaveState>({ status: "idle", message: "" })

  async function loadSettings() {
    settings = await getSiteSettings()
    draft = draftFromSettings(settings)
  }

  async function saveSettings() {
    if (!draft) return
    save = { status: "pending", message: "" }
    try {
      const updated = await updateSiteSettings(draft)
      settings = updated
      draft = draftFromSettings(updated)
      save = { status: "success", message: "Saved." }
    } catch (cause) {
      save = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function load() {
    loading = true
    error = ""
    try {
      await Promise.all([loadInfo(), loadSettings()])
    } catch (cause) {
      error = actionErrorMessage(cause)
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
      // very first build). Deliberately not the full load() — that would
      // re-fetch settings and discard any unsaved site-identity edits.
      await loadInfo()
    } catch (cause) {
      build = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  function previewUrl(port: string): string {
    return `http://${location.hostname}:${port}/`
  }

  // --- Output-location picker (ADMIN-02 staged-rebuild GUI) -----------------
  // A normal-flow panel (not position:fixed) that opens inline below the
  // output-directory row: browseDirectories() lists the current directory's
  // subfolders, "Up" walks to `parent` (hidden once parent === "", i.e. the
  // server's configured root), and "Select this folder" repoints out_dir via
  // updateSiteOutDir, then closes and refreshes site info.
  type PickerStatus = "idle" | "loading" | "error"
  interface PickerState {
    open: boolean
    status: PickerStatus
    message: string
    browse: AdminBrowseResult | null
  }
  let picker = $state<PickerState>({ open: false, status: "idle", message: "", browse: null })

  async function openPicker() {
    picker = { open: true, status: "loading", message: "", browse: null }
    try {
      const result = await browseDirectories(info?.out_dir)
      picker = { open: true, status: "idle", message: "", browse: result }
    } catch {
      // The current out_dir may sit outside the browse root (or not exist
      // yet) — fall back to the root listing rather than leaving the picker
      // stuck on an error the moment it opens.
      try {
        const result = await browseDirectories()
        picker = { open: true, status: "idle", message: "", browse: result }
      } catch (rootCause) {
        picker = { open: true, status: "error", message: actionErrorMessage(rootCause), browse: null }
      }
    }
  }

  async function navigateTo(path: string) {
    picker = { ...picker, status: "loading", message: "" }
    try {
      const result = await browseDirectories(path)
      picker = { ...picker, status: "idle", message: "", browse: result }
    } catch (cause) {
      picker = { ...picker, status: "error", message: actionErrorMessage(cause) }
    }
  }

  function closePicker() {
    picker = { open: false, status: "idle", message: "", browse: null }
  }

  async function selectCurrentFolder() {
    if (!picker.browse) return
    const path = picker.browse.path
    picker = { ...picker, status: "loading", message: "" }
    try {
      await updateSiteOutDir(path)
      closePicker()
      await load()
    } catch (cause) {
      picker = { ...picker, status: "error", message: actionErrorMessage(cause) }
    }
  }

  function pickerKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") closePicker()
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
        <button type="button" class="secondary" onclick={openPicker}>Change location…</button>
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

      {#if picker.open}
        <!--
          A normal-flow panel, not position:fixed — it lives right in this
          section's document flow and pushes surrounding content down,
          avoiding the classic position:fixed pitfalls (stacking-context
          surprises, iOS viewport-resize jumps, needing a separate scroll
          lock). role="dialog" + aria-modal + the Escape handler keep it
          keyboard-accessible without those tradeoffs.
        -->
        <div class="picker-panel" role="dialog" aria-modal="true" aria-label="Choose output location" onkeydown={pickerKeydown} tabindex="-1">
          <div class="picker-head">
            <h3>Choose output location</h3>
            <button type="button" class="secondary" onclick={closePicker} aria-label="Close">Close</button>
          </div>

          {#if picker.status === "loading"}
            <p class="hint">Loading…</p>
          {:else if picker.status === "error"}
            <p class="trigger-status trigger-status--error" role="alert">{picker.message}</p>
          {:else if picker.browse}
            {@const browse = picker.browse}
            <p class="picker-path">
              <code>{browse.path || browse.root}</code>
            </p>
            <ul class="picker-dirs">
              {#if browse.parent !== ""}
                <li>
                  <button type="button" class="picker-entry" onclick={() => navigateTo(browse.parent)}>.. (up)</button>
                </li>
              {/if}
              {#each browse.dirs as dir (dir.path)}
                <li>
                  <button type="button" class="picker-entry" onclick={() => navigateTo(dir.path)}>{dir.name}</button>
                </li>
              {/each}
              {#if browse.dirs.length === 0}
                <li class="hint">No subfolders here.</li>
              {/if}
            </ul>
            <div class="picker-actions">
              <button type="button" onclick={selectCurrentFolder}>Select this folder</button>
              <button type="button" class="secondary" onclick={closePicker}>Cancel</button>
            </div>
          {/if}
        </div>
      {/if}
    </section>

    {#if settings && draft}
      {@const d = draft}
      <section class="site-identity" aria-label="Site identity">
        <h2>Site identity</h2>
        <p class="trigger-note">Design, title, and style projected to the public site's <code>site.json</code>.</p>

        <div class="design-cards">
          {#each designChoices as choice (choice.id)}
            <button type="button" class="design-card" class:selected={d.design === choice.id} onclick={() => (d.design = choice.id)}>
              <span class="design-card-id">{choice.id}</span>
              <span class="design-card-label">{choice.label}</span>
            </button>
          {/each}
        </div>

        <div class="identity-fields">
          <label class="field field-wide">
            <span class="field-label">Title</span>
            <input type="text" bind:value={d.title} placeholder="Site title" />
          </label>

          <label class="field field-wide">
            <span class="field-label">Description</span>
            <textarea bind:value={d.description} placeholder="Site description" rows="3"></textarea>
          </label>

          <label class="field">
            <span class="field-label">Default language</span>
            <select bind:value={d.default_language}>
              <option value="ja">Japanese</option>
              <option value="en">English</option>
              <option value="zh">Chinese</option>
            </select>
          </label>

          <label class="field">
            <span class="field-label">Default theme</span>
            <select bind:value={d.default_theme}>
              <option value="dark">Dark</option>
              <option value="light">Light</option>
            </select>
          </label>

          <label class="field field-accent">
            <span class="field-label">Accent color</span>
            <input type="color" bind:value={d.accent} />
          </label>
        </div>

        <div class="save-row">
          <button type="button" onclick={saveSettings} disabled={save.status === "pending"}>{save.status === "pending" ? "Saving…" : "Save site settings"}</button>
          {#if save.status === "success"}
            <span class="trigger-status trigger-status--success" role="status">{save.message}</span>
          {:else if save.status === "error"}
            <span class="trigger-status trigger-status--error" role="alert">{save.message}</span>
          {/if}
        </div>
      </section>
    {/if}

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
  .info-row .secondary {
    border: 1px solid #d8cdbb;
    border-radius: 7px;
    padding: 7px 12px;
    color: #6b5137;
    background: #fffaf2;
    font-size: 13px;
    white-space: nowrap;
  }
  .picker-panel {
    margin-top: 18px;
    padding: 16px 18px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 255 255 / 55%);
  }
  .picker-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .picker-head h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
  }
  .picker-head button {
    border: 1px solid #d8cdbb;
    border-radius: 7px;
    padding: 6px 10px;
    color: #6b5137;
    background: #fffaf2;
    font-size: 12px;
  }
  .picker-path {
    margin: 12px 0 0;
    font-size: 12px;
    color: #766956;
  }
  .picker-path code {
    padding: 3px 7px;
    border-radius: 6px;
    background: rgb(255 255 255 / 60%);
  }
  .picker-dirs {
    display: grid;
    gap: 4px;
    margin: 10px 0 0;
    padding: 0;
    list-style: none;
    max-height: 220px;
    overflow-y: auto;
  }
  .picker-entry {
    display: block;
    width: 100%;
    text-align: left;
    padding: 8px 10px;
    border: 1px solid transparent;
    border-radius: 7px;
    color: #342a1e;
    background: transparent;
    font-size: 13px;
  }
  .picker-entry:hover,
  .picker-entry:focus-visible {
    border-color: #d8cdbb;
    background: rgb(255 255 255 / 70%);
  }
  .picker-actions {
    display: flex;
    gap: 10px;
    margin-top: 14px;
  }
  .picker-actions button {
    border: 0;
    border-radius: 7px;
    padding: 8px 14px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
  }
  .picker-actions button.secondary {
    color: #6b5137;
    background: transparent;
    border: 1px solid #d8cdbb;
  }
  .site-identity {
    margin-top: 32px;
    padding: 20px 24px;
    border: 1px solid #dfd4c1;
    border-radius: 12px;
    background: rgb(255 250 242 / 55%);
  }
  .site-identity h2 {
    margin: 0;
    font-family: Georgia, serif;
    font-size: 22px;
    font-weight: 500;
  }
  .design-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
    gap: 10px;
    margin-top: 16px;
  }
  .design-card {
    display: flex;
    flex-direction: column;
    gap: 4px;
    align-items: flex-start;
    padding: 12px 14px;
    border: 1px solid #d8cdbb;
    border-radius: 9px;
    background: #fffaf2;
    text-align: left;
  }
  .design-card:hover,
  .design-card:focus-visible {
    border-color: #b98a5c;
  }
  .design-card.selected {
    border-color: #9f522d;
    background: rgb(159 82 45 / 8%);
    box-shadow: inset 0 0 0 1px #9f522d;
  }
  .design-card-id {
    font-family: Georgia, serif;
    font-size: 15px;
    font-weight: 600;
    text-transform: uppercase;
    color: #342a1e;
  }
  .design-card-label {
    font-size: 12px;
    color: #766956;
  }
  .identity-fields {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 16px;
    margin-top: 20px;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .field-wide {
    grid-column: 1 / -1;
  }
  .field-label {
    color: #766956;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .field input[type="text"],
  .field textarea,
  .field select {
    padding: 8px 10px;
    border: 1px solid #d8cdbb;
    border-radius: 7px;
    background: #fffaf2;
    color: #342a1e;
    font-size: 13px;
    font-family: inherit;
  }
  .field textarea {
    resize: vertical;
  }
  .field-accent input[type="color"] {
    width: 56px;
    height: 36px;
    padding: 2px;
    border: 1px solid #d8cdbb;
    border-radius: 7px;
    background: #fffaf2;
  }
  .save-row {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-top: 22px;
  }
  .save-row button {
    border: 0;
    border-radius: 7px;
    padding: 8px 14px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .save-row button:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .save-row .trigger-status {
    margin-top: 0;
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
