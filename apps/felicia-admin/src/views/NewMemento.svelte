<script lang="ts">
  import { beforeNavigate, goto } from "$app/navigation"
  import { resolve } from "$app/paths"
  import { Button } from "$lib/components/ui/button"
  import { Input } from "$lib/components/ui/input"
  import { Label } from "$lib/components/ui/label"
  import SelectField from "$lib/components/SelectField.svelte"
  import { ArrowLeft, Plus } from "@lucide/svelte"
  import { getJourney, getTemplates, listMementos, upsertMemento, type AdminJourney, type AdminTemplateRegistry } from "../api"
  import { message, type Locale } from "../i18n"
  import { journeyDetailPath, mementoEditPath } from "../router"

  let { journeyId, locale }: { journeyId: string; locale: Locale } = $props()
  let journey = $state<AdminJourney | null>(null)
  let templates = $state<AdminTemplateRegistry | null>(null)
  let title = $state("")
  let kind = $state("")
  let initialKind = $state("")
  let seq = $state(0)
  let pending = $state(false)
  let saved = $state(false)
  let error = $state("")
  const id = crypto.randomUUID()
  const items = $derived(Object.keys(templates ?? {}).map((value) => ({ value, label: value })))
  const dirty = $derived(Boolean(title || kind !== initialKind))
  $effect(() => {
    const currentId = journeyId
    let active = true
    Promise.all([getJourney(currentId), getTemplates(), listMementos(currentId)])
      .then(([value, registry, mementos]) => {
        if (!active) return
        journey = value
        templates = registry
        kind = registry.goods ? "goods" : (Object.keys(registry)[0] ?? "")
        initialKind = kind
        seq = mementos.reduce((max, item) => Math.max(max, item.seq + 1), 0)
      })
      .catch((cause) => {
        if (active) error = cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
      })
    return () => {
      active = false
    }
  })
  beforeNavigate((navigation) => {
    if (saved || (!dirty && !pending)) return
    if (navigation.willUnload || pending || !confirm(message(locale, "admin.journeys.unsaved_leave"))) navigation.cancel()
  })
  async function create() {
    if (pending || saved || !journey || !kind || !title.trim()) return
    pending = true
    error = ""
    try {
      await upsertMemento({
        id,
        journey_id: journeyId,
        kind,
        seq,
        title: title.trim(),
        place: journey.place,
        occurred_at: `${journey.date_start.slice(0, 10)}T00:00:00Z`,
        occurred_tz: "UTC",
        kind_data: {},
        state: "draft",
      })
      saved = true
      await goto(resolve(mementoEditPath(journeyId, id)))
    } catch (cause) {
      error = cause instanceof Error ? cause.message : message(locale, "admin.common.request_failed")
    } finally {
      pending = false
    }
  }
</script>

<svelte:document
  onclickcapture={(event) => {
    if (pending && event.target instanceof Element && event.target.closest("a[href]")) {
      event.preventDefault()
      event.stopPropagation()
    }
  }}
/>
<section class="memento-creation" aria-labelledby="new-memento-title">
  <header>
    <Button variant="outline" disabled={pending} onclick={() => goto(resolve(journeyDetailPath(journeyId)))}><ArrowLeft size={16} aria-hidden="true" />{message(locale, "admin.common.return")}</Button>
    <h1 id="new-memento-title">{message(locale, "admin.mementos.add")}</h1>
  </header>
  {#if error}<p role="alert">{error}</p>{/if}
  {#if journey && templates}
    <form
      onsubmit={(event) => {
        event.preventDefault()
        create()
      }}
    >
      <div>
        <Label for="memento-kind">{message(locale, "admin.connectors.kind_label")}</Label><SelectField
          id="memento-kind"
          label={message(locale, "admin.connectors.kind_label")}
          {items}
          bind:value={kind}
          disabled={pending}
        />
      </div>
      <div><Label for="memento-title">{message(locale, "admin.common.title")}</Label><Input id="memento-title" required bind:value={title} disabled={pending} /></div>
      <footer>
        <Button type="submit" disabled={pending || saved || !kind}><Plus size={16} aria-hidden="true" />{message(locale, pending ? "admin.journeys.creating" : "admin.common.create")}</Button>
      </footer>
    </form>
  {:else if !error}<p role="status">{message(locale, "admin.common.loading")}</p>{/if}
</section>

<style>
  .memento-creation {
    max-width: 960px;
    margin: 0 auto;
  }
  header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;
  }
  h1 {
    font-size: 20px;
  }
  form {
    display: grid;
    gap: 20px;
    padding: 24px;
    border: 1px solid var(--line);
    border-radius: 12px;
    background: var(--surface);
  }
  form > div {
    display: grid;
    gap: 8px;
  }
  footer {
    display: flex;
    justify-content: flex-end;
  }
</style>
