<script lang="ts">
  import { onMount } from "svelte"
  import { Button } from "$lib/components/ui/button"
  import IconButton from "$lib/components/IconButton.svelte"
  import { FolderOpen, Folder, ArrowUp, X, Save, Hammer, Check } from "@lucide/svelte"
  import { message, type Locale } from "../i18n"

  let { locale }: { locale: Locale } = $props()
  import {
    browseDirectories,
    compileSite,
    describeLoadFailure,
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
    return cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
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
      save = { status: "success", message: message(locale, "admin.common.saved") }
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
      error = describeLoadFailure(cause, "site info")
    } finally {
      loading = false
    }
  }

  async function triggerBuild() {
    build = { status: "pending", message: message(locale, "admin.build.building") }
    try {
      const report = await compileSite()
      build = { status: "success", message: message(locale, "admin.build.complete"), report }
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
    // Otherwise closing (Escape, Close, or a successful selection) drops
    // keyboard focus back to the document body, forcing the author to tab
    // in from the top of the page again.
    pickerTrigger?.focus()
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

  // role="dialog"/aria-modal implies both of these, and neither happened
  // without it: keyboard focus stayed on the "Change location…" trigger
  // when the panel opened, so Escape (bound to the panel's own keydown) did
  // nothing until the author had already tabbed or clicked into the panel —
  // and a screen reader user got no cue the dialog existed at all.
  let pickerPanel = $state<HTMLDivElement | undefined>(undefined)
  let pickerTrigger = $state<HTMLButtonElement | null>(null)
  $effect(() => {
    if (picker.open) pickerPanel?.focus()
  })

  onMount(load)
</script>

<section class="site">
  <header class="site-header">
    <p class="eyebrow">{message(locale, "admin.site.breadcrumb")}</p>
    <h1>{message(locale, "admin.site.navigation")}</h1>
  </header>

  <p class="hint">{message(locale, "admin.site.deploy_note")}</p>

  {#if loading}
    <p class="hint">{message(locale, "admin.site.loading")}</p>
  {:else if error}
    <p class="api-error" role="alert">{error}</p>
  {:else if info}
    <section class="site-info" aria-label={message(locale, "admin.site.output_directory")}>
      <div class="info-row">
        <span class="info-label">{message(locale, "admin.site.output_directory")}</span>
        <code class="info-value">{info.out_dir}</code>
        <IconButton bind:ref={pickerTrigger} label={message(locale, "admin.site.change_location")} onclick={openPicker}><FolderOpen size={16} aria-hidden="true" /></IconButton>
      </div>

      {#if info.artifact_ready}
        <div class="info-row">
          <span class="info-label">{message(locale, "admin.site.preview")}</span>
          <a class="preview-link" href={previewUrl(info.preview_port)} target="_blank" rel="external noreferrer">{previewUrl(info.preview_port)}</a>
        </div>
      {/if}

      {#if !info.spa_ready}
        <p class="hint">{message(locale, "admin.site.preview_unavailable_note")}</p>
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
        <div bind:this={pickerPanel} class="picker-panel" role="dialog" aria-modal="true" aria-label={message(locale, "admin.site.output_picker.title")} onkeydown={pickerKeydown} tabindex="-1">
          <div class="picker-head">
            <h3>{message(locale, "admin.site.output_picker.title")}</h3>
            <IconButton variant="ghost" onclick={closePicker} label={message(locale, "admin.site.output_picker.close")}><X size={16} aria-hidden="true" /></IconButton>
          </div>

          {#if picker.status === "loading"}
            <p class="hint">{message(locale, "admin.common.loading")}</p>
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
                  <Button type="button" variant="ghost" class="w-full justify-start" onclick={() => navigateTo(browse.parent)}
                    ><ArrowUp size={16} aria-hidden="true" />{message(locale, "admin.site.output_picker.up")}</Button
                  >
                </li>
              {/if}
              {#each browse.dirs as dir (dir.path)}
                <li>
                  <Button type="button" variant="ghost" class="w-full justify-start" onclick={() => navigateTo(dir.path)}><Folder size={16} aria-hidden="true" />{dir.name}</Button>
                </li>
              {/each}
              {#if browse.dirs.length === 0}
                <li class="hint">{message(locale, "admin.site.output_picker.empty")}</li>
              {/if}
            </ul>
            <div class="picker-actions">
              <Button type="button" onclick={selectCurrentFolder}><Check size={16} aria-hidden="true" />{message(locale, "admin.site.output_picker.select")}</Button>
              <Button type="button" variant="outline" onclick={closePicker}>{message(locale, "admin.common.cancel")}</Button>
            </div>
          {/if}
        </div>
      {/if}
    </section>

    {#if settings && draft}
      {@const d = draft}
      <section class="site-identity" aria-label={message(locale, "admin.site.identity")}>
        <h2>{message(locale, "admin.site.identity")}</h2>
        <p class="trigger-note">{message(locale, "admin.site.identity_note")}</p>

        <div class="design-cards">
          <button type="button" class="design-card" class:selected={d.design === "atlas"} onclick={() => (d.design = "atlas")}>
            <span class="design-card-id">atlas</span>
            <span class="design-card-label">{message(locale, "admin.site.design_atlas")}</span>
          </button>
        </div>

        <div class="identity-fields">
          <label class="field field-wide">
            <span class="field-label">{message(locale, "admin.site.title")}</span>
            <input type="text" bind:value={d.title} placeholder={message(locale, "admin.site.title")} />
          </label>

          <label class="field field-wide">
            <span class="field-label">{message(locale, "admin.site.description")}</span>
            <textarea bind:value={d.description} placeholder={message(locale, "admin.site.description")} rows="3"></textarea>
          </label>

          <label class="field">
            <span class="field-label">{message(locale, "admin.site.default_language")}</span>
            <select bind:value={d.default_language}>
              <option value="ja">{message(locale, "admin.common.language_japanese")}</option>
              <option value="en">{message(locale, "admin.common.language_english")}</option>
              <option value="zh">{message(locale, "admin.common.language_chinese")}</option>
            </select>
          </label>

          <label class="field">
            <span class="field-label">{message(locale, "admin.site.default_theme")}</span>
            <select bind:value={d.default_theme}>
              <option value="dark">{message(locale, "admin.common.theme_dark")}</option>
              <option value="light">{message(locale, "admin.common.theme_light")}</option>
            </select>
          </label>

          <label class="field field-accent">
            <span class="field-label">{message(locale, "admin.site.accent_color")}</span>
            <input type="color" bind:value={d.accent} />
          </label>
        </div>

        <div class="save-row">
          <Button
            type="button"
            onclick={saveSettings}
            disabled={save.status === "pending"}
            aria-label={save.status === "pending" ? message(locale, "admin.common.saving") : message(locale, "admin.site.save_settings")}
          >
            <Save size={16} aria-hidden="true" />{save.status === "pending" ? message(locale, "admin.common.saving") : message(locale, "admin.common.save")}</Button
          >
          {#if save.status === "success"}
            <span class="trigger-status trigger-status--success" role="status">{save.message}</span>
          {:else if save.status === "error"}
            <span class="trigger-status trigger-status--error" role="alert">{save.message}</span>
          {/if}
        </div>
      </section>
    {/if}

    <section class="build" aria-label={message(locale, "admin.site.build_action")}>
      <div class="build-head">
        <h2>{message(locale, "admin.site.build_heading")}</h2>
        <Button
          type="button"
          onclick={triggerBuild}
          disabled={build.status === "pending"}
          aria-label={build.status === "pending" ? message(locale, "admin.build.building") : message(locale, "admin.site.build_action")}
        >
          <Hammer size={16} aria-hidden="true" />{build.status === "pending" ? message(locale, "admin.build.building") : message(locale, "admin.common.build")}</Button
        >
      </div>
      <p class="trigger-note">{message(locale, "admin.site.build_note")}</p>

      {#if build.status === "success"}
        <p class="trigger-status trigger-status--success" role="status">{build.message}</p>
        {#if build.report}
          <dl class="report-grid">
            <div class="report-cell">
              <dt>{message(locale, "admin.stats.journeys")}</dt>
              <dd>{build.report.Journeys}</dd>
            </div>
            <div class="report-cell">
              <dt>{message(locale, "admin.stats.mementos")}</dt>
              <dd>{build.report.Mementos}</dd>
            </div>
            <div class="report-cell">
              <dt>{message(locale, "admin.stats.media")}</dt>
              <dd>{build.report.Media}</dd>
            </div>
            <div class="report-cell">
              <dt>{message(locale, "admin.stats.removed")}</dt>
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
  .site {
    max-width: 1040px;
    margin-inline: auto;
  }
  .site-header h1 {
    margin-top: 4px;
  }
  .hint,
  .trigger-note {
    color: var(--muted);
    font-size: 13px;
    line-height: 1.5;
  }
  .hint,
  .api-error {
    margin-top: 16px;
  }
  .site-info,
  .site-identity,
  .build {
    margin-top: 24px;
    padding: 20px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--surface-raised);
  }
  .info-row {
    display: grid;
    grid-template-columns: minmax(112px, auto) minmax(0, 1fr) auto;
    align-items: center;
    gap: 12px;
  }
  .info-row + .info-row {
    margin-top: 12px;
  }
  .info-label,
  .field-label {
    color: var(--muted);
    font-size: 13px;
  }
  .info-value,
  .picker-path code {
    min-width: 0;
    padding: 6px 8px;
    border-radius: 4px;
    background: var(--surface-muted);
    overflow-wrap: anywhere;
    font-size: 12px;
  }
  .preview-link {
    color: var(--accent);
    overflow-wrap: anywhere;
  }
  .picker-panel {
    margin-top: 16px;
    padding: 16px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--surface);
  }
  .picker-head,
  .build-head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .picker-head h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
  }
  .picker-path {
    margin: 12px 0 0;
    color: var(--muted);
    overflow-wrap: anywhere;
  }
  .picker-dirs {
    display: grid;
    gap: 4px;
    margin: 12px 0 0;
    padding: 0;
    list-style: none;
    max-height: 220px;
    overflow: auto;
  }
  .picker-actions,
  .save-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
    margin-top: 16px;
  }
  .design-cards {
    display: grid;
    gap: 8px;
    margin-top: 16px;
  }
  .design-card {
    display: flex;
    flex-direction: column;
    gap: 4px;
    align-items: flex-start;
    padding: 12px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--surface);
    text-align: left;
  }
  .design-card.selected {
    border-color: var(--accent);
  }
  .design-card-id {
    color: var(--text);
    font-size: 14px;
    font-weight: 600;
  }
  .design-card-label {
    color: var(--muted);
    font-size: 12px;
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
  .field input[type="text"],
  .field textarea,
  .field select {
    padding: 8px 10px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }
  .field textarea {
    resize: vertical;
  }
  .field-accent input[type="color"] {
    width: 56px;
    height: 32px;
    padding: 2px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--surface);
  }
  .trigger-note {
    margin: 8px 0 0;
  }
  .trigger-status {
    margin: 10px 0 0;
    font-size: 13px;
  }
  .trigger-status--success {
    color: var(--text);
  }
  .trigger-status--error {
    color: var(--danger);
  }
  .save-row .trigger-status {
    margin-top: 0;
  }
  .report-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
    gap: 12px;
    margin: 16px 0 0;
  }
  .report-cell {
    padding: 12px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--surface);
    text-align: center;
  }
  .report-cell dt {
    color: var(--muted);
    font-size: 12px;
  }
  .report-cell dd {
    margin: 4px 0 0;
    font-size: 20px;
    font-weight: 600;
  }
  @media (max-width: 900px) {
    .info-row {
      grid-template-columns: minmax(0, 1fr) auto;
    }
    .info-label {
      grid-column: 1 / -1;
    }
  }
</style>
