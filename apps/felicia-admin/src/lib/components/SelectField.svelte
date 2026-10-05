<script lang="ts" generics="T extends string">
  import { Select } from "bits-ui"
  import { ChevronDown, Check } from "@lucide/svelte"
  let {
    value = $bindable(),
    items,
    label,
    id,
    disabled = false,
    placeholder = "",
    onValueChange,
  }: { value?: T; items: { value: T; label: string }[]; label: string; id?: string; disabled?: boolean; placeholder?: string; onValueChange?: (value: T) => void } = $props()
  const descriptionId = $props.id()
  let trigger = $state<HTMLButtonElement | null>(null)
  function select(next: string) {
    const item = items.find((item) => item.value === next)
    if (item) {
      value = item.value
      onValueChange?.(item.value)
    }
  }
</script>

<span id={descriptionId} class="sr-only">{items.find((item) => item.value === value)?.label ?? placeholder}</span>
<Select.Root type="single" {items} value={value ?? ""} onValueChange={select} {disabled}>
  <Select.Trigger {id} bind:ref={trigger} aria-label={label} aria-describedby={descriptionId} class="select-trigger">
    <span>{items.find((item) => item.value === value)?.label ?? placeholder}</span><ChevronDown size={16} aria-hidden="true" />
  </Select.Trigger>
  <Select.Portal to={trigger?.closest('dialog, [role="dialog"]') ?? undefined}>
    <Select.Content class="select-content" sideOffset={4} collisionPadding={12}>
      <Select.Viewport>
        {#each items as item (item.value)}
          <Select.Item value={item.value} label={item.label} data-option-value={item.value} class="select-item">
            {#snippet children({ selected })}<span>{item.label}</span>{#if selected}<Check size={16} aria-hidden="true" />{/if}{/snippet}
          </Select.Item>
        {/each}
      </Select.Viewport>
    </Select.Content>
  </Select.Portal>
</Select.Root>

<style>
  :global(.select-trigger) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    width: 100%;
    min-width: 0;
    min-height: 40px;
    padding: 8px 12px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 14px;
    text-transform: none;
    letter-spacing: normal;
    --wails-draggable: no-drag;
  }
  :global(.select-trigger:focus-visible) {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
  :global(.select-content) {
    z-index: 100;
    min-width: var(--bits-select-anchor-width);
    max-height: min(300px, var(--bits-select-content-available-height));
    overflow: auto;
    padding: 4px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--surface);
    color: var(--text);
    box-shadow: 0 8px 24px #0003;
    --wails-draggable: no-drag;
  }
  :global(.select-item) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    min-height: 36px;
    padding: 8px 12px;
    border-radius: 4px;
    cursor: pointer;
  }
  :global(.select-item[data-highlighted]) {
    background: var(--accent-soft);
    color: var(--accent);
  }
</style>
