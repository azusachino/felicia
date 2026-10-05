<script lang="ts">
  import { DateField } from "bits-ui"
  import { CalendarDateTime, parseDate, parseDateTime, type DateValue } from "@internationalized/date"
  import type { Locale } from "../../i18n"

  let {
    value = $bindable(""),
    label,
    locale,
    disabled = false,
    granularity = "second",
    required = false,
    name,
    invalid = false,
    errorId,
  }: { value?: string; label: string; locale: Locale; disabled?: boolean; granularity?: "day" | "second"; required?: boolean; name?: string; invalid?: boolean; errorId?: string } = $props()
  const datetime = $derived(value ? (granularity === "day" ? parseDate(value) : parseDateTime(value.replace(" ", "T"))) : undefined)
  const allParts = ["year", "month", "day", "hour", "minute", "second"] as const
  const parts = $derived(allParts.slice(0, granularity === "day" ? 3 : 6))
  function setValue(next: DateValue | undefined) {
    value = next ? (granularity === "day" ? next.toString().slice(0, 10) : next.toString().replace("T", " ")) : ""
  }
  function displaySegment(part: string, text: string) {
    return /^\d+$/.test(text) ? text.padStart(part === "year" ? 4 : 2, "0") : text
  }
</script>

<div class="datetime-control">
  <DateField.Root bind:value={() => datetime, setValue} {locale} {disabled} {granularity} {required} hourCycle={24} placeholder={new CalendarDateTime(2000, 1, 1)}>
    <DateField.Label class="datetime-label">{label}</DateField.Label>
    <DateField.Input class="datetime-input" {name} aria-invalid={invalid || undefined} aria-describedby={errorId}>
      {#snippet children({ segments })}
        {#each parts as part, index (part)}
          {@const segment = segments.find((item) => item.part === part)}
          {#if segment}
            <DateField.Segment {part} class="datetime-segment">{displaySegment(part, segment.value)}</DateField.Segment>
            {#if index < parts.length - 1}<span aria-hidden="true">{index < 2 ? "-" : index === 2 ? "\u00a0" : ":"}</span>{/if}
          {/if}
        {/each}
      {/snippet}
    </DateField.Input>
  </DateField.Root>
</div>

<style>
  .datetime-control {
    display: grid;
    gap: 8px;
    min-width: 0;
  }
  .datetime-control :global(.datetime-label) {
    color: var(--muted);
    font-size: 13px;
  }
  .datetime-control :global(.datetime-input) {
    display: flex;
    align-items: center;
    min-height: 40px;
    padding: 8px 12px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--surface);
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  .datetime-control :global(.datetime-segment) {
    border-radius: 2px;
    outline: none;
  }
  .datetime-control :global(.datetime-segment:focus-visible) {
    background: var(--accent-soft);
    color: var(--accent);
  }
  .datetime-control :global(.datetime-input:focus-within) {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
  .datetime-control :global([data-disabled]) {
    opacity: 0.5;
  }
</style>
