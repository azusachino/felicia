<script lang="ts">
  import { afterNavigate, beforeNavigate, goto } from "$app/navigation"
  import { resolve } from "$app/paths"
  import { Plus, ArrowLeft } from "@lucide/svelte"
  import { Button } from "$lib/components/ui/button"
  import IconButton from "$lib/components/IconButton.svelte"
  import { Input } from "$lib/components/ui/input"
  import { Label } from "$lib/components/ui/label"
  import { createJourney } from "../api"
  import { message, type Locale } from "../i18n"
  import { journeyDetailPath } from "../router"

  let { locale }: { locale: Locale } = $props()
  let title = $state("")
  let place = $state("")
  let slug = $state("")
  let dateStart = $state("")
  let dateEnd = $state("")
  let country = $state("")
  let region = $state("")
  let slugEdited = $state(false)
  let pending = $state(false)
  let leaving = $state(false)
  let created = $state(false)
  let error = $state("")
  let titleInput = $state<HTMLElement | null>(null)
  const fallbackSlug = `journey-${crypto.getRandomValues(new Uint32Array(1))[0].toString(16)}`
  const dirty = $derived(Boolean(title || place || slug || dateStart || dateEnd || country || region))
  const invalidDates = $derived(Boolean(dateStart && dateEnd && dateEnd < dateStart))

  beforeNavigate((navigation) => {
    if (leaving || created || (!dirty && !pending)) return
    if (navigation.willUnload || pending || !confirm(message(locale, "admin.journeys.unsaved_leave"))) navigation.cancel()
  })

  function deriveSlug() {
    if (!slugEdited)
      slug =
        title
          .toLowerCase()
          .trim()
          .replace(/[^\w\s-]/g, "")
          .replace(/[\s_-]+/g, "-")
          .replace(/^-+|-+$/g, "") || fallbackSlug
  }

  async function cancel() {
    if (pending) return
    leaving = true
    await goto(resolve("/"))
  }

  async function submit() {
    if (pending || created || invalidDates) return
    pending = true
    error = ""
    try {
      const result = await createJourney({
        title: title.trim(),
        place: place.trim(),
        slug: slug.trim(),
        date_start: dateStart,
        date_end: dateEnd,
        country: country.trim() || undefined,
        region: region.trim() || undefined,
      })
      created = true
      await goto(resolve(journeyDetailPath(result.id)))
    } catch (cause) {
      error = cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
    } finally {
      pending = false
    }
  }

  afterNavigate(() => titleInput?.focus())
</script>

<section class="creation-page" aria-labelledby="new-journey-title">
  <header class="page-header">
    <IconButton variant="ghost" href={resolve("/")} label={message(locale, "admin.journeys.back")} disabled={pending}><ArrowLeft size={16} aria-hidden="true" /></IconButton>
    <h1 id="new-journey-title">{message(locale, "admin.journeys.create_heading")}</h1>
  </header>
  <div class="task-body">
    <p class="hint">{message(locale, "admin.journeys.create_note")}</p>
    <form
      id="journey-create-form"
      class="panel field-grid"
      onsubmit={(event) => {
        event.preventDefault()
        submit()
      }}
    >
      <div class="field">
        <Label for="new-title">{message(locale, "admin.common.title")}</Label><Input id="new-title" bind:ref={titleInput} required bind:value={title} oninput={deriveSlug} disabled={pending} />
      </div>
      <div class="field"><Label for="new-place">{message(locale, "admin.common.place")}</Label><Input id="new-place" required bind:value={place} disabled={pending} /></div>
      <div class="field"><Label for="new-start">{message(locale, "admin.journeys.start_date")}</Label><Input id="new-start" required type="date" bind:value={dateStart} disabled={pending} /></div>
      <div class="field">
        <Label for="new-end">{message(locale, "admin.journeys.end_date")}</Label><Input
          id="new-end"
          required
          type="date"
          min={dateStart}
          bind:value={dateEnd}
          disabled={pending}
          aria-invalid={invalidDates}
          aria-describedby={invalidDates ? "date-range-error" : undefined}
        />
      </div>
      <details class="more-options">
        <summary>{message(locale, "admin.journeys.more_options")}</summary>
        <div class="field-grid optional-fields">
          <div class="field">
            <Label for="new-slug">{message(locale, "admin.journeys.slug")}</Label><Input id="new-slug" required bind:value={slug} oninput={() => (slugEdited = true)} disabled={pending} />
          </div>
          <div class="field"><Label for="new-country">{message(locale, "admin.journeys.country_optional")}</Label><Input id="new-country" bind:value={country} disabled={pending} /></div>
          <div class="field"><Label for="new-region">{message(locale, "admin.journeys.region_optional")}</Label><Input id="new-region" bind:value={region} disabled={pending} /></div>
        </div>
      </details>
    </form>
    {#if invalidDates}<p id="date-range-error" class="api-error" role="alert">{message(locale, "admin.journeys.invalid_dates")}</p>{/if}
    {#if error}<p class="api-error" role="alert">{error}</p>{/if}
  </div>
  <footer class="task-footer">
    <Button variant="outline" disabled={pending} onclick={cancel}>{message(locale, "admin.common.cancel")}</Button>
    <Button
      type="submit"
      form="journey-create-form"
      aria-label={message(locale, pending ? "admin.journeys.creating" : "admin.journeys.create_action")}
      disabled={pending || created || !title.trim() || !place.trim() || !slug.trim() || !dateStart || !dateEnd}
    >
      <Plus size={16} aria-hidden="true" />{message(locale, pending ? "admin.journeys.creating" : "admin.common.create")}
    </Button>
  </footer>
</section>

<style>
  .creation-page {
    display: flex;
    flex-direction: column;
    min-height: 100%;
    max-width: 760px;
    margin-inline: auto;
    gap: 24px;
  }
  .page-header {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .page-header h1 {
    font-size: 20px;
  }
  .task-body {
    min-width: 0;
  }
  .hint {
    color: var(--muted);
    margin: 0 0 20px;
    line-height: 1.6;
  }
  .panel {
    padding: 20px;
    border: 1px solid var(--line);
    border-radius: 12px;
    background: var(--surface-raised);
  }
  .field-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }
  .field {
    display: grid;
    align-content: start;
    gap: 8px;
    min-width: 0;
  }
  .more-options {
    grid-column: 1 / -1;
    color: var(--muted);
  }
  .more-options summary {
    cursor: pointer;
    padding: 4px 0;
  }
  .optional-fields {
    margin-top: 16px;
  }
  .task-footer {
    position: sticky;
    bottom: 0;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: auto;
    padding: 16px 0;
    border-top: 1px solid var(--line);
    background: var(--surface);
  }
  @media (max-width: 640px) {
    .field-grid {
      grid-template-columns: minmax(0, 1fr);
    }
    .panel {
      padding: 16px;
    }
  }
</style>
