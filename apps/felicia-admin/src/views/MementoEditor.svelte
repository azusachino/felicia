<script lang="ts">
  import { message, statusMessage, type Locale } from "../i18n"
  import {
    deleteMemento,
    describeLoadFailure,
    getMemento,
    getTemplates,
    isConflict,
    listMementoPhotos,
    snapToRoute,
    photoContentURL,
    uploadPhoto,
    upsertMemento,
    upsertPhoto,
    ApiError,
    type AdminMementoDetail,
    type AdminMementoPhoto,
    type AdminTemplate,
    type AdminTemplateField,
    type AdminTemplateRegistry,
  } from "../api"
  import {
    buildKindData,
    buildPhotoPayload,
    buildUpsertPayload,
    emptyKindFormState,
    fromRFC3339,
    geomToLatLngInputs,
    groupIssuesByField,
    issueMessage,
    lifecycleActionLabel,
    nextLifecycleState,
    parseKindData,
    parseLatLng,
    photoFormFieldsFromRequest,
    previousLifecycleState,
    priceFormFieldsFromMemento,
    serverChangedFields,
    unpublishActionLabel,
    FORM_LEVEL_ISSUE_KEY,
    type CommonFormFields,
    type KindFormState,
    type LatLngInput,
    type PhotoFormFields,
  } from "../mementoForm"
  import { journeyDetailHash } from "../router"

  let { journeyId, id, locale }: { journeyId: string; id: string; locale: Locale } = $props()

  // Only these two kinds get a hardcoded, registry-aligned form (ADMIN-01.4).
  // Every other kind falls back to a read-only pretty-printed kind_data view.
  const HARDCODED_KINDS = new Set(["transit", "goods"])

  let memento = $state<AdminMementoDetail | null>(null)
  let templates = $state<AdminTemplateRegistry | null>(null)
  let loading = $state(true)
  let loadError = $state("")

  let common = $state<CommonFormFields>({
    title: "",
    place: "",
    occurredAtLocal: "",
    occurredTz: "",
    essay: "",
    vendor: "",
    price: { amount: "", currency: "" },
  })
  let points = $state<LatLngInput[]>([{ lat: "", lng: "" }])
  let kindFormState = $state<KindFormState>({})
  let otherKindDataText = $state("")

  type SaveStatus = "idle" | "pending" | "success" | "error" | "conflict"
  interface SaveState {
    status: SaveStatus
    message: string
    fieldErrors: Record<string, string[]>
  }
  let saveState = $state<SaveState>({ status: "idle", message: "", fieldErrors: {} })

  type SnapStatus = "idle" | "pending" | "error"
  let snapStatus = $state<SnapStatus[]>([])

  function template(): AdminTemplate | null {
    if (!memento || !templates) return null
    return templates[memento.kind] ?? null
  }

  function anchor(): string {
    return template()?.Anchor ?? "point"
  }

  function templateFields(): AdminTemplateField[] {
    return template()?.Fields ?? []
  }

  function isHardcodedKind(): boolean {
    return memento !== null && HARDCODED_KINDS.has(memento.kind)
  }

  // Typed accessors into kindFormState for the template — kept here rather
  // than as inline casts in the markup so the .svelte template stays plain
  // property access, matching the rest of this app's views.
  function moneyField(name: string): { amount: string; currency: string } {
    return kindFormState[name] as { amount: string; currency: string }
  }
  function placeField(name: string): { name: string; lat: string; lng: string } {
    return kindFormState[name] as { name: string; lat: string; lng: string }
  }
  // Compound fields can't use bind:value (a function call isn't a bindable
  // expression), so their inputs write back through these setters instead.
  function setMoneyField(name: string, key: "amount" | "currency", value: string) {
    moneyField(name)[key] = value
  }
  function setPlaceField(name: string, key: "name" | "lat" | "lng", value: string) {
    placeField(name)[key] = value
  }
  // A plain text kind_data field's value is a bare string, unlike the
  // money/station shapes above — bound manually (rather than via bind:value)
  // since kindFormState's values are a union and this keeps that union out
  // of the template's type-checking entirely.
  function textFieldValue(name: string): string {
    return (kindFormState[name] as string) ?? ""
  }
  function setTextField(name: string, value: string) {
    kindFormState[name] = value
  }

  function actionErrorMessage(cause: unknown): string {
    return cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
  }

  // Wraps the lifecycle save so the template's onclick closure never touches
  // the nullable `memento` directly (TS can't narrow it inside a closure).
  function advanceLifecycle() {
    if (!memento) return
    const next = nextLifecycleState(memento.state)
    if (next) void save(next)
  }

  // The one backward step (ADMIN-02 M1 02.1a): published -> authored, via
  // the same save(targetState) path forward moves already use — it already
  // refetches and handles conflicts/validation, so unpublishing needs no new
  // machinery, just a different target state.
  function retreatLifecycle() {
    if (!memento) return
    const previous = previousLifecycleState(memento.state)
    if (previous) void save(previous)
  }

  // Rehydrates every piece of local form state from a freshly-fetched
  // memento. Used on initial load, after a successful save, and when the
  // author explicitly chooses to discard their draft. A conflict does NOT
  // call this: ADMIN-01.5 rules out a merge UI, which is a reason not to build
  // a diff editor, not a reason to delete the only copy of an essay.
  function hydrateForm(fetched: AdminMementoDetail, registry: AdminTemplateRegistry | null) {
    common = {
      title: fetched.title ?? "",
      place: fetched.place ?? "",
      occurredAtLocal: fromRFC3339(fetched.occurred_at),
      occurredTz: fetched.occurred_tz ?? "",
      essay: fetched.essay ?? "",
      vendor: fetched.vendor ?? "",
      price: priceFormFieldsFromMemento(fetched.price_amount, fetched.price_currency),
    }
    const tpl = registry?.[fetched.kind] ?? null
    const anchorValue = tpl?.Anchor ?? "point"
    points = geomToLatLngInputs(fetched.geom, anchorValue)
    snapStatus = points.map(() => "idle")
    if (tpl && HARDCODED_KINDS.has(fetched.kind)) {
      kindFormState = parseKindData(tpl.Fields, fetched.kind_data)
    } else {
      kindFormState = tpl ? emptyKindFormState(tpl.Fields) : {}
      otherKindDataText = JSON.stringify(fetched.kind_data ?? {}, null, 2)
    }
  }

  async function loadAll() {
    loading = true
    loadError = ""
    saveState = { status: "idle", message: "", fieldErrors: {} }
    try {
      const [fetchedMemento, registry, photos] = await Promise.all([getMemento(id), templates ?? getTemplates(), listMementoPhotos(id)])
      templates = registry
      memento = fetchedMemento
      hydrateForm(fetchedMemento, registry)
      photoRows = photos.map(photoRowFromExisting)
    } catch (cause) {
      loadError = describeLoadFailure(cause, "this memento")
    } finally {
      loading = false
    }
  }

  $effect(() => {
    loadAll()
  })

  function issuesFor(field: string): string[] {
    return saveState.fieldErrors[field] ?? []
  }

  function formLevelIssues(): string[] {
    return saveState.fieldErrors[FORM_LEVEL_ISSUE_KEY] ?? []
  }

  async function save(targetState?: string) {
    if (!memento) return
    const tpl = template()
    const kindData = isHardcodedKind() && tpl ? buildKindData(tpl.Fields, kindFormState) : (memento.kind_data ?? {})
    const payload = buildUpsertPayload({
      identity: {
        id: memento.id,
        journey_id: memento.journey_id,
        kind: memento.kind,
        seq: memento.seq,
        source_ref: memento.source_ref,
        authored_fields: memento.authored_fields ?? [],
        orphaned_at: memento.orphaned_at,
      },
      common,
      anchor: anchor(),
      points,
      kindData,
      state: targetState ?? memento.state,
      expectedRevision: memento.revision,
    })

    saveState = { status: "pending", message: message(locale, "admin.common.saving"), fieldErrors: {} }
    try {
      await upsertMemento(payload)
      // Re-fetch so the next save carries the fresh revision (ADMIN-01.5).
      const refreshed = await getMemento(memento.id)
      memento = refreshed
      hydrateForm(refreshed, templates)
      saveState = { status: "success", message: message(locale, "admin.common.saved"), fieldErrors: {} }
    } catch (cause) {
      if (isConflict(cause)) {
        // The draft stays exactly as typed. Fetch the server copy only to
        // report what moved, so the author can judge whether their text still
        // applies before retrying.
        let changed: string[]
        try {
          const current = await getMemento(memento.id)
          changed = serverChangedFields(memento, current)
        } catch {
          // Reporting what moved is a courtesy; failing to fetch it must not
          // turn a recoverable conflict into a lost draft.
          changed = []
        }
        saveState = {
          status: "conflict",
          message: changed.length > 0 ? message(locale, "admin.mementos.conflict.changed", { fields: changed.join(", ") }) : message(locale, "admin.mementos.conflict.unknown"),
          fieldErrors: {},
        }
        return
      }
      if (cause instanceof ApiError && cause.issues && cause.issues.length > 0) {
        saveState = { status: "error", message: cause.message, fieldErrors: groupIssuesByField(cause.issues, locale) }
        return
      }
      saveState = { status: "error", message: actionErrorMessage(cause), fieldErrors: {} }
    }
  }

  // Keeps the typed draft and refreshes only the revision the next save must
  // carry, so retrying submits the author's work against current state. This
  // is not a merge: the draft wins wholesale, which is the author's own most
  // recent intent.
  async function retryKeepingDraft() {
    if (!memento) return
    saveState = { status: "pending", message: message(locale, "admin.common.rechecking"), fieldErrors: {} }
    try {
      memento = await getMemento(memento.id)
      saveState = { status: "idle", message: "", fieldErrors: {} }
      await save()
    } catch (cause) {
      saveState = { status: "error", message: actionErrorMessage(cause), fieldErrors: {} }
    }
  }

  // The old behaviour, now only when asked for by name.
  async function discardDraftAndReload() {
    await loadAll()
  }

  // "Save & back to journey" (ADMIN-02 staged-rebuild GUI): the same save
  // path as the plain Save button, then — only on success, so a conflict or
  // validation error keeps the author on this page to fix it — navigate
  // back to the journey detail, where the pending-build highlight/count
  // now reflects any published<->authored toggle this save just made.
  async function saveAndBack() {
    await save()
    if (saveState.status === "success") {
      location.hash = journeyDetailHash(journeyId)
    }
  }

  // Delete (ADMIN-02 M1 02.1b): a permanent, hard delete with an inline
  // two-step confirm rather than a native confirm() dialog, so the copy can
  // spell out what's irreversible (photos cascade) and what isn't (a future
  // import may re-seed a source-derived memento with the same identity —
  // this is a plain delete, no tombstone).
  type DeleteStatus = "idle" | "confirming" | "pending" | "error"
  let deleteState = $state<{ status: DeleteStatus; message: string }>({ status: "idle", message: "" })

  // Delete is gated to candidate/draft/authored (contract §3) — a published
  // memento must be unpublished first. The GUI mirrors that guard rather
  // than only relying on the server's 422, so the "unpublish first" hint is
  // always visible instead of only appearing after a failed attempt.
  function deleteBlockedByPublishedState(): boolean {
    return memento?.state === "published"
  }

  function requestDelete() {
    deleteState = { status: "confirming", message: "" }
  }

  function cancelDelete() {
    deleteState = { status: "idle", message: "" }
  }

  async function confirmDelete() {
    if (!memento) return
    deleteState = { status: "pending", message: message(locale, "admin.common.deleting") }
    try {
      await deleteMemento(memento.id)
      location.hash = journeyDetailHash(journeyId)
    } catch (cause) {
      // A 422 (delete_requires_unpublish, or in principle invalid_transition)
      // carries a structured issue — surface its friendly message rather
      // than the raw error string.
      if (cause instanceof ApiError && cause.issues && cause.issues.length > 0) {
        deleteState = { status: "error", message: cause.issues.map((issue) => issueMessage(issue, locale)).join(" ") }
        return
      }
      deleteState = { status: "error", message: actionErrorMessage(cause) }
    }
  }

  async function snapPoint(index: number) {
    const point = points[index]
    const parsed = parseLatLng(point)
    if (!parsed) {
      snapStatus[index] = "error"
      return
    }
    snapStatus[index] = "pending"
    try {
      const result = await snapToRoute(journeyId, parsed)
      const [lng, lat] = result.point.coordinates
      points[index] = { lat: String(lat), lng: String(lng) }
      snapStatus[index] = "idle"
    } catch {
      snapStatus[index] = "error"
    }
  }

  // --- Original photo upload and curation -----------------
  // Originals stay private in the media store; publication emits sanitized copies.
  let photoUploadStatus = $state("")
  let photoUploadError = $state(false)

  interface PhotoRow {
    id: string
    fields: PhotoFormFields
    status: "idle" | "pending" | "success" | "error"
    message: string
  }
  let photoRows = $state<PhotoRow[]>([])

  function photoRowFromExisting(photo: AdminMementoPhoto): PhotoRow {
    return {
      id: photo.id,
      fields: photoFormFieldsFromRequest({
        objectKey: photo.object_key,
        contentHash: photo.content_hash,
        caption: photo.caption ?? "",
        seq: String(photo.seq),
        takenAt: photo.taken_at ? fromRFC3339(photo.taken_at) : "",
        sourceRef: photo.source_ref ?? "",
      }),
      status: "idle",
      message: "",
    }
  }

  async function handlePhotoUpload(event: Event) {
    if (!memento) return
    const input = event.currentTarget as HTMLInputElement
    const files = Array.from(input.files ?? [])
    if (files.length === 0) return
    photoUploadStatus = message(locale, "admin.mementos.photo_uploading", { count: files.length })
    photoUploadError = false
    const failures: string[] = []
    for (const file of files) {
      try {
        const uploaded = await uploadPhoto(memento.id, file)
        photoRows = [...photoRows, photoRowFromExisting(uploaded)]
      } catch (cause) {
        failures.push(`${file.name}: ${actionErrorMessage(cause)}`)
      }
    }
    photoUploadError = failures.length > 0
    photoUploadStatus = failures.length > 0 ? failures.join("; ") : message(locale, "admin.mementos.photo_upload_complete")
    input.value = ""
  }

  async function movePhoto(index: number, delta: number) {
    const target = index + delta
    if (!memento || target < 0 || target >= photoRows.length) return
    const reordered = [...photoRows]
    ;[reordered[index], reordered[target]] = [reordered[target], reordered[index]]
    photoRows = reordered
    for (const [seq, row] of reordered.entries()) {
      row.fields = photoFormFieldsFromRequest({ ...row.fields, seq: String(seq) })
      await savePhotoRow(row)
    }
  }

  async function savePhotoRow(row: PhotoRow) {
    if (!memento) return
    row.status = "pending"
    row.message = message(locale, "admin.common.saving")
    photoRows = [...photoRows]
    try {
      await upsertPhoto(buildPhotoPayload(row.id, memento.id, row.fields))
      row.status = "success"
      row.message = message(locale, "admin.common.saved")
    } catch (cause) {
      row.status = "error"
      row.message = actionErrorMessage(cause)
    }
    photoRows = [...photoRows]
  }
</script>

<section class="editor">
  <a class="back-link" href={journeyDetailHash(journeyId)}>&larr; {message(locale, "admin.mementos.back_to_journey")}</a>

  {#if loading}
    <p class="hint">{message(locale, "admin.mementos.loading")}</p>
  {:else if loadError}
    <p class="api-error" role="alert">{loadError}</p>
  {:else if memento}
    <header class="editor-header">
      <p class="eyebrow">{memento.kind}</p>
      <h1>{memento.title || memento.place || message(locale, "admin.mementos.untitled")}</h1>
      <span class={`badge badge--${memento.state}`}>{statusMessage(locale, memento.state)}</span>
    </header>

    {#if saveState.status === "conflict"}
      <div class="conflict-banner" role="alert">
        <p>{saveState.message}</p>
        <button type="button" onclick={retryKeepingDraft}>{message(locale, "admin.mementos.conflict.save_mine")}</button>
        <button type="button" class="secondary" onclick={discardDraftAndReload}>{message(locale, "admin.mementos.conflict.discard_mine")}</button>
      </div>
    {/if}

    {#if formLevelIssues().length > 0}
      <div class="form-errors" role="alert">
        {#each formLevelIssues() as issue (issue)}
          <p>{issue}</p>
        {/each}
      </div>
    {:else if saveState.status === "error"}
      <p class="trigger-status trigger-status--error" role="alert">{saveState.message}</p>
    {/if}

    {#if saveState.status === "success"}
      <p class="trigger-status trigger-status--success" role="status">{saveState.message}</p>
    {/if}

    <section class="fields" aria-label={message(locale, "admin.mementos.details_heading")}>
      <h2>{message(locale, "admin.mementos.details_heading")}</h2>
      <div class="field-grid">
        <label class="field">
          {message(locale, "admin.common.title")}
          <input type="text" bind:value={common.title} />
        </label>
        <label class="field">
          {message(locale, "admin.common.place")}
          <input type="text" bind:value={common.place} />
        </label>
        <label class="field">
          {message(locale, "admin.mementos.occurred_at")}
          <input type="datetime-local" bind:value={common.occurredAtLocal} />
        </label>
        <label class="field">
          {message(locale, "admin.mementos.timezone")}
          <input type="text" placeholder="Asia/Tokyo" bind:value={common.occurredTz} />
          {#if issuesFor("occurred_tz").length > 0}
            <span class="field-error">{issuesFor("occurred_tz").join(" ")}</span>
          {/if}
        </label>
        <label class="field">
          {message(locale, "admin.mementos.vendor")}
          <input type="text" bind:value={common.vendor} />
        </label>
        <label class="field">
          {message(locale, "admin.mementos.price_amount")}
          <input type="text" inputmode="decimal" bind:value={common.price.amount} />
        </label>
        <label class="field">
          {message(locale, "admin.mementos.price_currency")}
          <input type="text" placeholder="JPY" maxlength="3" bind:value={common.price.currency} />
        </label>
      </div>
      <label class="field field--wide">
        {message(locale, "admin.mementos.essay")}
        <textarea rows="6" bind:value={common.essay}></textarea>
      </label>
    </section>

    <section class="fields" aria-label={message(locale, "admin.mementos.location_heading")}>
      <h2>{message(locale, "admin.mementos.location_heading")}</h2>
      <p class="trigger-note">
        {anchor() === "edge" ? message(locale, "admin.mementos.location_edge_note") : message(locale, "admin.mementos.location_point_note")}
        {message(locale, "admin.mementos.location_validation_note")}
      </p>
      {#if issuesFor("geom").length > 0}
        <p class="field-error">{issuesFor("geom").join(" ")}</p>
      {/if}
      <div class="point-grid">
        {#each points as point, index (index)}
          <div class="point-row">
            <span class="point-label"
              >{anchor() === "edge" ? (index === 0 ? message(locale, "admin.mementos.from") : message(locale, "admin.mementos.to")) : message(locale, "admin.mementos.point")}</span
            >
            <label class="field">
              {message(locale, "admin.mementos.latitude")}
              <input type="text" bind:value={point.lat} />
            </label>
            <label class="field">
              {message(locale, "admin.mementos.longitude")}
              <input type="text" bind:value={point.lng} />
            </label>
            <button type="button" class="secondary" onclick={() => snapPoint(index)} disabled={snapStatus[index] === "pending"}>
              {snapStatus[index] === "pending" ? message(locale, "admin.mementos.snapping") : message(locale, "admin.mementos.snap_action")}
            </button>
            {#if snapStatus[index] === "error"}
              <span class="field-error">{message(locale, "admin.mementos.snap_error")}</span>
            {/if}
          </div>
        {/each}
      </div>
    </section>

    <section class="fields" aria-label={message(locale, "admin.mementos.kind_data_heading")}>
      <h2>{memento.kind} {message(locale, "admin.mementos.details_heading")}</h2>
      {#if isHardcodedKind() && templates}
        <div class="field-grid">
          {#each templateFields() as tplField (tplField.Name)}
            <label class="field">
              {tplField.Name}{tplField.Required ? " *" : ""}
              {#if tplField.Type === "money"}
                <span class="money-inputs">
                  <input
                    type="text"
                    inputmode="decimal"
                    placeholder={message(locale, "admin.common.amount")}
                    value={moneyField(tplField.Name).amount}
                    oninput={(e) => setMoneyField(tplField.Name, "amount", (e.currentTarget as HTMLInputElement).value)}
                  />
                  <input
                    type="text"
                    placeholder={message(locale, "admin.common.currency")}
                    maxlength="3"
                    value={moneyField(tplField.Name).currency}
                    oninput={(e) => setMoneyField(tplField.Name, "currency", (e.currentTarget as HTMLInputElement).value)}
                  />
                </span>
              {:else if tplField.Type === "station" || tplField.Type === "venue"}
                <span class="place-inputs">
                  <input
                    type="text"
                    placeholder={message(locale, "admin.common.name")}
                    value={placeField(tplField.Name).name}
                    oninput={(e) => setPlaceField(tplField.Name, "name", (e.currentTarget as HTMLInputElement).value)}
                  />
                  <input
                    type="text"
                    placeholder={message(locale, "admin.mementos.latitude")}
                    value={placeField(tplField.Name).lat}
                    oninput={(e) => setPlaceField(tplField.Name, "lat", (e.currentTarget as HTMLInputElement).value)}
                  />
                  <input
                    type="text"
                    placeholder={message(locale, "admin.mementos.longitude")}
                    value={placeField(tplField.Name).lng}
                    oninput={(e) => setPlaceField(tplField.Name, "lng", (e.currentTarget as HTMLInputElement).value)}
                  />
                </span>
              {:else}
                <input type="text" value={textFieldValue(tplField.Name)} oninput={(e) => setTextField(tplField.Name, (e.currentTarget as HTMLInputElement).value)} />
              {/if}
            </label>
            {#if issuesFor(tplField.Name).length > 0}
              <span class="field-error">{issuesFor(tplField.Name).join(" ")}</span>
            {/if}
          {/each}
        </div>
      {:else if templates && !template()}
        <p class="hint">{message(locale, "admin.mementos.kind_registry_missing", { kind: memento.kind })}</p>
      {:else}
        <p class="trigger-note">{message(locale, "admin.mementos.kind_read_only")}</p>
        <pre class="kind-data-json">{otherKindDataText}</pre>
      {/if}
    </section>

    <section class="fields" aria-label={message(locale, "admin.mementos.photos_heading")}>
      <div class="inbox-head">
        <h2>{message(locale, "admin.mementos.photos_heading")}</h2>
        <label class="photo-upload">
          {message(locale, "admin.mementos.add_photos")}
          <input type="file" accept="image/jpeg,image/png,image/webp" multiple onchange={handlePhotoUpload} />
        </label>
      </div>
      <p class="trigger-note">{message(locale, "admin.mementos.photo_upload_note")}</p>
      {#if photoUploadStatus}
        <p class={photoUploadError ? "trigger-status trigger-status--error" : "trigger-status"} role={photoUploadError ? "alert" : "status"}>{photoUploadStatus}</p>
      {/if}
      {#if photoRows.length === 0}
        <p class="hint">{message(locale, "admin.mementos.empty_photos")}</p>
      {:else}
        <ul class="photo-list">
          {#each photoRows as row (row.id)}
            <li class="photo-row">
              <img class="photo-preview" src={photoContentURL(row.id)} alt={row.fields.caption || message(locale, "admin.mementos.photo_alt", { number: Number(row.fields.seq) + 1 })} loading="lazy" />
              <div class="field-grid">
                <label class="field">
                  {message(locale, "admin.mementos.photo_caption")}
                  <input type="text" bind:value={row.fields.caption} />
                </label>
              </div>
              <div class="photo-actions">
                <button type="button" class="secondary" onclick={() => movePhoto(photoRows.indexOf(row), -1)} disabled={photoRows.indexOf(row) === 0 || row.status === "pending"}
                  >{message(locale, "admin.mementos.move_photo_up")}</button
                >
                <button type="button" class="secondary" onclick={() => movePhoto(photoRows.indexOf(row), 1)} disabled={photoRows.indexOf(row) === photoRows.length - 1 || row.status === "pending"}
                  >{message(locale, "admin.mementos.move_photo_down")}</button
                >
                <button type="button" onclick={() => savePhotoRow(row)} disabled={row.status === "pending"}
                  >{row.status === "pending" ? message(locale, "admin.common.saving") : message(locale, "admin.mementos.photo_save_caption")}</button
                >
              </div>
              {#if row.status === "success"}
                <span class="trigger-status trigger-status--success">{row.message}</span>
              {:else if row.status === "error"}
                <span class="trigger-status trigger-status--error">{row.message}</span>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="actions" aria-label={message(locale, "admin.mementos.actions_label")}>
      <button type="button" onclick={() => save()} disabled={saveState.status === "pending"}
        >{saveState.status === "pending" ? message(locale, "admin.common.saving") : message(locale, "admin.mementos.save")}</button
      >
      <button type="button" class="secondary" onclick={saveAndBack} disabled={saveState.status === "pending"}>
        {saveState.status === "pending" ? message(locale, "admin.common.saving") : message(locale, "admin.mementos.save_back")}
      </button>
      {#if unpublishActionLabel(memento.state) && previousLifecycleState(memento.state)}
        <button type="button" class="secondary" onclick={retreatLifecycle} disabled={saveState.status === "pending"}>
          {message(locale, "admin.mementos.unpublish")}
        </button>
      {/if}
      {#if lifecycleActionLabel(memento.state) && nextLifecycleState(memento.state)}
        <button type="button" class="primary" onclick={advanceLifecycle} disabled={saveState.status === "pending"}>
          {memento.state === "draft" ? message(locale, "admin.mementos.mark_authored") : message(locale, "admin.mementos.publish")}
        </button>
      {/if}
    </section>

    <section class="danger-zone" aria-label={message(locale, "admin.mementos.delete_heading")}>
      <h2>{message(locale, "admin.mementos.delete_heading")}</h2>
      <p class="trigger-note">{message(locale, "admin.mementos.delete_note")}</p>
      {#if deleteBlockedByPublishedState()}
        <p class="hint">{message(locale, "admin.mementos.unpublish_first")}</p>
        <button type="button" class="danger" disabled title={message(locale, "admin.mementos.unpublish_first")}>{message(locale, "admin.common.delete")}</button>
      {:else if deleteState.status === "confirming" || deleteState.status === "pending"}
        <div class="confirm-strip" role="alert">
          <p>{message(locale, "admin.mementos.delete_confirmation")}</p>
          <div class="confirm-actions">
            <button type="button" class="danger" onclick={confirmDelete} disabled={deleteState.status === "pending"}>
              {deleteState.status === "pending" ? message(locale, "admin.common.deleting") : message(locale, "admin.common.confirm_delete")}
            </button>
            <button type="button" class="secondary" onclick={cancelDelete} disabled={deleteState.status === "pending"}>{message(locale, "admin.common.cancel")}</button>
          </div>
        </div>
      {:else}
        <button type="button" class="danger" onclick={requestDelete}>{message(locale, "admin.common.delete")}</button>
      {/if}
      {#if deleteState.status === "error"}
        <p class="trigger-status trigger-status--error" role="alert">{deleteState.message}</p>
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
  .editor-header {
    display: flex;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;
  }
  .editor-header h1 {
    margin: 4px 0 0;
  }
  .conflict-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 20px;
    padding: 14px 16px;
    border: 1px solid #d8b28a;
    border-radius: 10px;
    background: rgb(231 162 96 / 16%);
    color: #7a3f1d;
  }
  .conflict-banner button {
    border: 0;
    border-radius: 7px;
    padding: 8px 12px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .form-errors {
    margin-top: 16px;
    padding: 12px 16px;
    border: 1px solid #e0a598;
    border-radius: 10px;
    background: rgb(168 74 52 / 10%);
    color: #a84a34;
    font-size: 13px;
  }
  .form-errors p {
    margin: 2px 0;
  }
  .fields {
    margin-top: 32px;
  }
  .fields h2 {
    margin: 0 0 14px;
    font-family: Georgia, serif;
    font-size: 20px;
    font-weight: 500;
  }
  .field-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 14px;
  }
  .field {
    display: grid;
    gap: 6px;
    color: #766956;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .field--wide {
    margin-top: 14px;
  }
  .field input,
  .field textarea {
    padding: 9px 11px;
    border: 1px solid #d8cdbb;
    border-radius: 7px;
    color: #342a1e;
    background: #fffaf2;
    font-size: 14px;
    text-transform: none;
    letter-spacing: normal;
    font-family: inherit;
  }
  .field-error {
    color: #a84a34;
    font-size: 12px;
    text-transform: none;
    letter-spacing: normal;
  }
  /* min-width: 0 on the flex container itself, not just its children, is
     required here: it also sits as a grid item of .field below, and a grid
     item's automatic minimum size is content-based unless overridden — so
     without this the row refuses to shrink below its children's natural
     width no matter what the children's own flex/min-width say. */
  .money-inputs,
  .place-inputs {
    display: flex;
    gap: 8px;
    min-width: 0;
  }
  /* Without this, each sub-input keeps its browser-default intrinsic width
     (~180px) and the row overflows the ~200px field-grid column — two or
     three of them spill visibly into the next field over. */
  .money-inputs input,
  .place-inputs input {
    flex: 1;
    min-width: 0;
  }
  .point-grid {
    display: grid;
    gap: 14px;
    margin-top: 12px;
  }
  .point-row {
    display: flex;
    align-items: flex-end;
    gap: 12px;
    flex-wrap: wrap;
    padding: 14px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
  }
  .point-label {
    color: #9f522d;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    min-width: 40px;
  }
  .point-row button,
  .actions button,
  .photo-row button,
  .danger-zone button {
    border: 0;
    border-radius: 7px;
    padding: 9px 14px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    white-space: nowrap;
  }
  .point-row button.secondary,
  .actions button.secondary,
  .danger-zone button.secondary {
    color: #6b5137;
    background: transparent;
    border: 1px solid #d8cdbb;
  }
  .danger-zone button.danger {
    color: #fffaf2;
    background: #a84a34;
  }
  .kind-data-json {
    margin: 0;
    padding: 14px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
    font-size: 12px;
    overflow-x: auto;
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
  .photo-list {
    display: grid;
    gap: 12px;
    margin: 14px 0 0;
    padding: 0;
    list-style: none;
  }
  .photo-upload {
    display: inline-flex;
    align-items: center;
    border: 0;
    border-radius: 7px;
    padding: 9px 14px;
    color: #fffaf2;
    background: #9f522d;
    font-size: 13px;
    cursor: pointer;
  }
  .photo-upload:focus-within {
    outline: 2px solid #6b5137;
    outline-offset: 2px;
  }
  .photo-upload input {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
  }
  .photo-row {
    display: grid;
    gap: 10px;
    padding: 14px 16px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
  }
  .photo-preview {
    display: block;
    width: min(100%, 480px);
    max-height: 320px;
    object-fit: contain;
    border-radius: 6px;
    background: #eee5d7;
  }
  .photo-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .photo-row button.secondary {
    color: #6b5137;
    background: transparent;
    border: 1px solid #d8cdbb;
  }
  .photo-row button:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
  .trigger-status {
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
  .trigger-note {
    margin: 0 0 8px;
    color: #766956;
    font-size: 12px;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin: 32px 0 8px;
  }
  .actions button.primary {
    background: #3f7a52;
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
  /* .trigger-note's shared #766956 measures 4.3:1 against this section's
     reddish tint — just under the 4.5:1 AA floor (axe color-contrast,
     serious). Plain cream backgrounds elsewhere are light enough that the
     shared color already passes there. */
  .danger-zone > .trigger-note {
    color: #5c4f3d;
  }
  .confirm-strip {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 12px;
    padding: 12px 14px;
    border: 1px solid rgb(168 74 52 / 45%);
    border-radius: 8px;
    background: rgb(168 74 52 / 10%);
  }
  .confirm-strip p {
    margin: 0;
    color: #7a3524;
    font-size: 13px;
  }
  .confirm-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }
  button:disabled {
    opacity: 0.6;
    cursor: default;
  }
  /* Darkened from the original tones (axe color-contrast, serious): those
     landed 4.15-4.27:1 against these translucent tinted backgrounds, under
     the 4.5:1 AA floor. */
  .badge--draft {
    color: #8a431f;
    background: rgb(231 162 96 / 24%);
  }
  .badge--authored {
    color: #5f5116;
    background: rgb(214 188 84 / 24%);
  }
  .badge--published {
    color: #2f5e40;
    background: rgb(120 184 135 / 24%);
  }
  .badge--candidate,
  .badge--archived {
    color: #5c5142;
    background: rgb(166 154 137 / 20%);
  }
</style>
