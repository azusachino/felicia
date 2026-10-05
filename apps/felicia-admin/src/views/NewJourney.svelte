<script lang="ts">
  import { afterNavigate, beforeNavigate, goto } from "$app/navigation"
  import { resolve } from "$app/paths"
  import { untrack } from "svelte"
  import { Collapsible } from "bits-ui"
  import { Plus, ArrowLeft, Save } from "@lucide/svelte"
  import { Button } from "$lib/components/ui/button"
  import DateTimeInput from "$lib/components/DateTimeInput.svelte"
  import { Input } from "$lib/components/ui/input"
  import { Label } from "$lib/components/ui/label"
  import { createJourney, type AdminJourney } from "../api"
  import { message, type Locale } from "../i18n"
  import { journeyDetailPath } from "../router"

  let { locale, journey }: { locale: Locale; journey?: AdminJourney } = $props()
  const initial = untrack(() => journey)
  const returnPath = initial ? journeyDetailPath(initial.id) : "/"
  let title = $state(initial?.title ?? "")
  let place = $state(initial?.place ?? "")
  let slug = $state(initial?.slug ?? "")
  const now = new Date()
  const defaultDate = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`
  let dateStart = $state(initial?.date_start.slice(0, 10) ?? defaultDate)
  let dateEnd = $state(initial?.date_end.slice(0, 10) ?? defaultDate)
  let country = $state(initial?.country ?? "")
  let region = $state(initial?.region ?? "")
  let slugEdited = $state(Boolean(initial))
  let pending = $state(false)
  let leaving = $state(false)
  let created = $state(false)
  let error = $state("")
  let titleInput = $state<HTMLElement | null>(null)
  let moreOptions = $state(false)
  const fallbackSlug = `journey-${crypto.getRandomValues(new Uint32Array(1))[0].toString(16)}`
  const originalFields = JSON.stringify(untrack(() => [title, place, slug, dateStart, dateEnd, country, region]))
  const dirty = $derived(JSON.stringify([title, place, slug, dateStart, dateEnd, country, region]) !== originalFields)
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
    await goto(resolve(returnPath))
  }

  async function submit() {
    if (pending || created || invalidDates) return
    if (!title.trim() || !place.trim() || !slug.trim() || !dateStart || !dateEnd) {
      error = message(locale, "admin.journeys.required_fields")
      if (!slug.trim()) moreOptions = true
      return
    }
    pending = true
    error = ""
    try {
      const result = await createJourney({
        id: initial?.id,
        journal_id: initial?.journal_id,
        source_ref: initial?.source_ref,
        expected_revision: initial?.revision,
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

<svelte:document
  onclickcapture={(event) => {
    // Stop the anchor before WebKit queues its hash navigation. The route
    // guard remains responsible for history and programmatic navigation.
    if (pending && event.target instanceof Element && event.target.closest("a[href]")) {
      event.preventDefault()
      event.stopPropagation()
    }
  }}
/>

<section class="creation-page" aria-labelledby="new-journey-title">
  <header class="page-header">
    <Button
      type="button"
      variant="outline"
      class="min-h-[36px]"
      aria-label={message(locale, initial ? "admin.mementos.back_to_journey" : "admin.journeys.back")}
      disabled={pending}
      onclick={() => goto(resolve(returnPath))}><ArrowLeft size={16} aria-hidden="true" />{message(locale, "admin.common.return")}</Button
    >
    <h1 id="new-journey-title">{message(locale, initial ? "admin.journeys.edit" : "admin.journeys.create_heading")}</h1>
  </header>
  <div class="task-body">
    {#if !initial}<p class="hint">{message(locale, "admin.journeys.create_note")}</p>{/if}
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
      <DateTimeInput label={message(locale, "admin.journeys.start_date")} {locale} granularity="day" name="date_start" required bind:value={dateStart} disabled={pending} />
      <DateTimeInput
        label={message(locale, "admin.journeys.end_date")}
        {locale}
        granularity="day"
        name="date_end"
        required
        bind:value={dateEnd}
        disabled={pending}
        invalid={invalidDates}
        errorId={invalidDates ? "date-range-error" : undefined}
      />
      <Collapsible.Root class="more-options" bind:open={moreOptions}>
        <Collapsible.Trigger>{#snippet child({ props })}<Button {...props} variant="outline">{message(locale, "admin.journeys.more_options")}</Button>{/snippet}</Collapsible.Trigger>
        <Collapsible.Content>
          <div class="field-grid optional-fields">
            <div class="field">
              <Label for="new-slug">{message(locale, "admin.journeys.slug")}</Label><Input id="new-slug" bind:value={slug} oninput={() => (slugEdited = true)} disabled={pending} />
            </div>
            <div class="field"><Label for="new-country">{message(locale, "admin.journeys.country_optional")}</Label><Input id="new-country" bind:value={country} disabled={pending} /></div>
            <div class="field"><Label for="new-region">{message(locale, "admin.journeys.region_optional")}</Label><Input id="new-region" bind:value={region} disabled={pending} /></div>
          </div>
        </Collapsible.Content>
      </Collapsible.Root>
    </form>
    {#if invalidDates}<p id="date-range-error" class="api-error" role="alert">{message(locale, "admin.journeys.invalid_dates")}</p>{/if}
    {#if error}<p class="api-error" role="alert">{error}</p>{/if}
  </div>
  <footer class="task-footer">
    <Button variant="outline" disabled={pending} onclick={cancel}>{message(locale, "admin.common.cancel")}</Button>
    <Button
      type="submit"
      form="journey-create-form"
      aria-label={message(locale, pending ? (initial ? "admin.common.saving" : "admin.journeys.creating") : initial ? "admin.journeys.save" : "admin.journeys.create_action")}
      disabled={pending || created}
    >
      {#if initial}<Save size={16} aria-hidden="true" />{:else}<Plus size={16} aria-hidden="true" />{/if}{message(
        locale,
        pending ? (initial ? "admin.common.saving" : "admin.journeys.creating") : initial ? "admin.common.save" : "admin.common.create",
      )}
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
  :global(.more-options) {
    grid-column: 1 / -1;
    color: var(--muted);
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
