<script lang="ts">
  import {
    deleteMemento,
    getMemento,
    getTemplates,
    isConflict,
    listMementoPhotos,
    snapToRoute,
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
    unpublishActionLabel,
    FORM_LEVEL_ISSUE_KEY,
    type CommonFormFields,
    type KindFormState,
    type LatLngInput,
    type PhotoFormFields,
  } from "../mementoForm"
  import { journeyDetailHash } from "../router"

  let { journeyId, id }: { journeyId: string; id: string } = $props()

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
    return cause instanceof Error ? cause.message : "Request failed"
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
  // memento. Used both on initial load and after a conflict reload (which
  // must discard whatever the user had typed, per ADMIN-01.5 — no merge UI).
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
      loadError = actionErrorMessage(cause)
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

    saveState = { status: "pending", message: "Saving…", fieldErrors: {} }
    try {
      await upsertMemento(payload)
      // Re-fetch so the next save carries the fresh revision (ADMIN-01.5).
      const refreshed = await getMemento(memento.id)
      memento = refreshed
      hydrateForm(refreshed, templates)
      saveState = { status: "success", message: "Saved.", fieldErrors: {} }
    } catch (cause) {
      if (isConflict(cause)) {
        saveState = {
          status: "conflict",
          message: "Someone else changed this memento since it was loaded.",
          fieldErrors: {},
        }
        return
      }
      if (cause instanceof ApiError && cause.issues && cause.issues.length > 0) {
        saveState = { status: "error", message: cause.message, fieldErrors: groupIssuesByField(cause.issues) }
        return
      }
      saveState = { status: "error", message: actionErrorMessage(cause), fieldErrors: {} }
    }
  }

  async function reloadAfterConflict() {
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
    deleteState = { status: "pending", message: "Deleting…" }
    try {
      await deleteMemento(memento.id)
      location.hash = journeyDetailHash(journeyId)
    } catch (cause) {
      // A 422 (delete_requires_unpublish, or in principle invalid_transition)
      // carries a structured issue — surface its friendly message rather
      // than the raw error string.
      if (cause instanceof ApiError && cause.issues && cause.issues.length > 0) {
        deleteState = { status: "error", message: cause.issues.map(issueMessage).join(" ") }
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

  // --- Photos (metadata-only; no byte upload in this epic) -----------------
  // Existing photos load from GET /api/admin/mementos/{id}/photos; each row
  // saves through the POST /api/admin/photos upsert keyed by its own id.

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

  function addPhotoRow() {
    photoRows = [
      ...photoRows,
      {
        id: crypto.randomUUID(),
        fields: photoFormFieldsFromRequest({ seq: String(photoRows.length) }),
        status: "idle",
        message: "",
      },
    ]
  }

  async function savePhotoRow(row: PhotoRow) {
    if (!memento) return
    row.status = "pending"
    row.message = "Saving…"
    photoRows = [...photoRows]
    try {
      await upsertPhoto(buildPhotoPayload(row.id, memento.id, row.fields))
      row.status = "success"
      row.message = "Saved."
    } catch (cause) {
      row.status = "error"
      row.message = actionErrorMessage(cause)
    }
    photoRows = [...photoRows]
  }
</script>

<section class="editor">
  <a class="back-link" href={journeyDetailHash(journeyId)}>&larr; Back to journey</a>

  {#if loading}
    <p class="hint">Loading memento…</p>
  {:else if loadError}
    <p class="api-error" role="alert">{loadError}. Start the local API to load authoring data.</p>
  {:else if memento}
    <header class="editor-header">
      <p class="eyebrow">{memento.kind}</p>
      <h1>{memento.title || memento.place || "Untitled memento"}</h1>
      <span class={`badge badge--${memento.state}`}>{memento.state}</span>
    </header>

    {#if saveState.status === "conflict"}
      <div class="conflict-banner" role="alert">
        <p>{saveState.message}</p>
        <button type="button" onclick={reloadAfterConflict}>Reload and reapply</button>
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

    <section class="fields" aria-label="Common fields">
      <h2>Details</h2>
      <div class="field-grid">
        <label class="field">
          Title
          <input type="text" bind:value={common.title} />
        </label>
        <label class="field">
          Place
          <input type="text" bind:value={common.place} />
        </label>
        <label class="field">
          Occurred at
          <input type="datetime-local" bind:value={common.occurredAtLocal} />
        </label>
        <label class="field">
          Timezone
          <input type="text" placeholder="Asia/Tokyo" bind:value={common.occurredTz} />
          {#if issuesFor("occurred_tz").length > 0}
            <span class="field-error">{issuesFor("occurred_tz").join(" ")}</span>
          {/if}
        </label>
        <label class="field">
          Vendor
          <input type="text" bind:value={common.vendor} />
        </label>
        <label class="field">
          Price amount
          <input type="text" inputmode="decimal" bind:value={common.price.amount} />
        </label>
        <label class="field">
          Price currency
          <input type="text" placeholder="JPY" maxlength="3" bind:value={common.price.currency} />
        </label>
      </div>
      <label class="field field--wide">
        Essay
        <textarea rows="6" bind:value={common.essay}></textarea>
      </label>
    </section>

    <section class="fields" aria-label="Location">
      <h2>Location</h2>
      <p class="trigger-note">
        {anchor() === "edge" ? "This kind spans a from → to route — enter both endpoints." : "This kind sits at a single point."}
        Non-draft saves must resolve to a valid location.
      </p>
      {#if issuesFor("geom").length > 0}
        <p class="field-error">{issuesFor("geom").join(" ")}</p>
      {/if}
      <div class="point-grid">
        {#each points as point, index (index)}
          <div class="point-row">
            <span class="point-label">{anchor() === "edge" ? (index === 0 ? "From" : "To") : "Point"}</span>
            <label class="field">
              Lat
              <input type="text" bind:value={point.lat} />
            </label>
            <label class="field">
              Lng
              <input type="text" bind:value={point.lng} />
            </label>
            <button type="button" class="secondary" onclick={() => snapPoint(index)} disabled={snapStatus[index] === "pending"}>
              {snapStatus[index] === "pending" ? "Snapping…" : "Snap to route"}
            </button>
            {#if snapStatus[index] === "error"}
              <span class="field-error">Couldn't snap this point — enter a valid lat/lng first, or check the journey has a route.</span>
            {/if}
          </div>
        {/each}
      </div>
    </section>

    <section class="fields" aria-label="Kind data">
      <h2>{memento.kind} details</h2>
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
                    placeholder="amount"
                    value={moneyField(tplField.Name).amount}
                    oninput={(e) => setMoneyField(tplField.Name, "amount", (e.currentTarget as HTMLInputElement).value)}
                  />
                  <input
                    type="text"
                    placeholder="currency"
                    maxlength="3"
                    value={moneyField(tplField.Name).currency}
                    oninput={(e) => setMoneyField(tplField.Name, "currency", (e.currentTarget as HTMLInputElement).value)}
                  />
                </span>
              {:else if tplField.Type === "station" || tplField.Type === "venue"}
                <span class="place-inputs">
                  <input type="text" placeholder="name" value={placeField(tplField.Name).name} oninput={(e) => setPlaceField(tplField.Name, "name", (e.currentTarget as HTMLInputElement).value)} />
                  <input type="text" placeholder="lat" value={placeField(tplField.Name).lat} oninput={(e) => setPlaceField(tplField.Name, "lat", (e.currentTarget as HTMLInputElement).value)} />
                  <input type="text" placeholder="lng" value={placeField(tplField.Name).lng} oninput={(e) => setPlaceField(tplField.Name, "lng", (e.currentTarget as HTMLInputElement).value)} />
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
        <p class="hint">Kind registry has no template for "{memento.kind}" — kind_data can't be validated here.</p>
      {:else}
        <p class="trigger-note">Read-only — this kind doesn't have a dedicated form yet.</p>
        <pre class="kind-data-json">{otherKindDataText}</pre>
      {/if}
    </section>

    <section class="fields" aria-label="Photos">
      <div class="inbox-head">
        <h2>Photos</h2>
        <button type="button" onclick={addPhotoRow}>Add photo</button>
      </div>
      <p class="trigger-note">Metadata only — object key and content hash must reference bytes already in the media store.</p>
      {#if photoRows.length === 0}
        <p class="hint">No photos yet.</p>
      {:else}
        <ul class="photo-list">
          {#each photoRows as row (row.id)}
            <li class="photo-row">
              <div class="field-grid">
                <label class="field">
                  Object key
                  <input type="text" bind:value={row.fields.objectKey} />
                </label>
                <label class="field">
                  Content hash
                  <input type="text" bind:value={row.fields.contentHash} />
                </label>
                <label class="field">
                  Caption
                  <input type="text" bind:value={row.fields.caption} />
                </label>
                <label class="field">
                  Seq
                  <input type="text" inputmode="numeric" bind:value={row.fields.seq} />
                </label>
              </div>
              <button type="button" onclick={() => savePhotoRow(row)} disabled={row.status === "pending"}>{row.status === "pending" ? "Saving…" : "Save photo"}</button>
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

    <section class="actions" aria-label="Save and publish actions">
      <button type="button" onclick={() => save()} disabled={saveState.status === "pending"}>{saveState.status === "pending" ? "Saving…" : "Save"}</button>
      <button type="button" class="secondary" onclick={saveAndBack} disabled={saveState.status === "pending"}>
        {saveState.status === "pending" ? "Saving…" : "Save & back to journey"}
      </button>
      {#if unpublishActionLabel(memento.state) && previousLifecycleState(memento.state)}
        <button type="button" class="secondary" onclick={retreatLifecycle} disabled={saveState.status === "pending"}>
          {unpublishActionLabel(memento.state)}
        </button>
      {/if}
      {#if lifecycleActionLabel(memento.state) && nextLifecycleState(memento.state)}
        <button type="button" class="primary" onclick={advanceLifecycle} disabled={saveState.status === "pending"}>
          {lifecycleActionLabel(memento.state)}
        </button>
      {/if}
    </section>

    <section class="danger-zone" aria-label="Delete memento">
      <h2>Delete this memento</h2>
      <p class="trigger-note">Permanent — this cannot be undone, and its photos are removed with it. If this memento was derived from an import source, a future import may re-create it.</p>
      {#if deleteBlockedByPublishedState()}
        <p class="hint">Unpublish this memento before deleting it — a published memento can't be deleted directly.</p>
        <button type="button" class="danger" disabled title="Unpublish first">Delete</button>
      {:else if deleteState.status === "confirming" || deleteState.status === "pending"}
        <div class="confirm-strip" role="alert">
          <p>Delete this memento permanently? Photos are removed too. A future import may re-create a source-derived memento like this one.</p>
          <div class="confirm-actions">
            <button type="button" class="danger" onclick={confirmDelete} disabled={deleteState.status === "pending"}>
              {deleteState.status === "pending" ? "Deleting…" : "Confirm delete"}
            </button>
            <button type="button" class="secondary" onclick={cancelDelete} disabled={deleteState.status === "pending"}>Cancel</button>
          </div>
        </div>
      {:else}
        <button type="button" class="danger" onclick={requestDelete}>Delete</button>
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
  .money-inputs,
  .place-inputs {
    display: flex;
    gap: 8px;
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
  .inbox-head button,
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
  .photo-row {
    display: grid;
    gap: 10px;
    padding: 14px 16px;
    border: 1px solid #dfd4c1;
    border-radius: 10px;
    background: rgb(255 250 242 / 55%);
  }
  .trigger-status {
    font-size: 13px;
  }
  .trigger-status--success {
    color: #3f7a52;
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
  .badge--draft {
    color: #9f522d;
    background: rgb(231 162 96 / 24%);
  }
  .badge--authored {
    color: #7a6a1f;
    background: rgb(214 188 84 / 24%);
  }
  .badge--published {
    color: #3f7a52;
    background: rgb(120 184 135 / 24%);
  }
  .badge--candidate,
  .badge--archived {
    color: #766956;
    background: rgb(166 154 137 / 20%);
  }
</style>
