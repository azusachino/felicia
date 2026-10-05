<script lang="ts">
  import type { Snippet } from "svelte"
  import type { HTMLButtonAttributes } from "svelte/elements"
  import { Tooltip } from "bits-ui"
  import { Button, type ButtonVariant } from "$lib/components/ui/button"

  let {
    label,
    children,
    href,
    onclick,
    disabled = false,
    variant = "outline",
    ref = $bindable(null),
  }: {
    label: string
    children: Snippet
    href?: string
    onclick?: HTMLButtonAttributes["onclick"]
    disabled?: boolean
    variant?: ButtonVariant
    ref?: HTMLElement | null
  } = $props()
  let open = $state(false)
</script>

<Tooltip.Root bind:open {disabled}>
  <Tooltip.Trigger onclick={onclick ?? undefined} {disabled}>
    {#snippet child({ props })}
      <Button {...props} bind:ref {href} {variant} size="icon" aria-label={label}>
        {@render children()}
      </Button>
    {/snippet}
  </Tooltip.Trigger>
  {#if open}
    <!-- Mount after the trigger ref exists so native dialog hints stay in its top layer. -->
    <Tooltip.Portal to={ref?.closest('dialog[open], [role="dialog"]') ?? undefined}>
      <Tooltip.Content role="tooltip" side="top" sideOffset={6} class="studio-tooltip">
        {label}
      </Tooltip.Content>
    </Tooltip.Portal>
  {/if}
</Tooltip.Root>
