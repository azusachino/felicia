<script lang="ts">
  import { onMount, setContext, type Snippet } from "svelte"
  import { page } from "$app/state"
  import { resolve } from "$app/paths"
  import { STUDIO_CONTEXT, type StudioState } from "./studio"
  import SampleButton from "$lib/components/SampleButton.svelte"
  import { Button } from "$lib/components/ui/button"
  import SelectField from "$lib/components/SelectField.svelte"
  import DiscardChangesDialog from "$lib/components/DiscardChangesDialog.svelte"
  import CompiledPreview from "$lib/components/CompiledPreview.svelte"
  import { getDesktopWorkspace, closeDesktopSample } from "../api"
  import { prepareWorkspaceSwitch } from "$lib/workspace"
  import { loadLocale, message, saveLocale, type Locale } from "../i18n"
  import * as Popover from "$lib/components/ui/popover"
  import { setMode, userPrefersMode } from "mode-watcher"
  import { Tooltip } from "bits-ui"
  import { Settings2, Languages, Monitor, Sun, Moon } from "@lucide/svelte"

  let { children }: { children: Snippet } = $props()

  let locale = $state(loadLocale())
  let settingsTrigger = $state<HTMLButtonElement | null>(null)
  let settingsOpen = $state(false)
  let settingsHintOpen = $state(false)
  const themeOptions = [
    { value: "system", Icon: Monitor, key: "admin.common.theme_system" },
    { value: "light", Icon: Sun, key: "admin.common.theme_light" },
    { value: "dark", Icon: Moon, key: "admin.common.theme_dark" },
  ] as const
  const library = $derived(page.route.id === "/" || page.route.id === "/[...missing]")
  const site = $derived(page.route.id === "/site")
  // Wails beta.24 identifies its macOS webview with this user-agent suffix.
  const desktop = navigator.userAgent.includes("wails.io")
  const nativeMac = desktop && navigator.userAgent.includes("Macintosh")
  let sample = $state(false)
  let isolated = $state(false)
  let sampleReturnAvailable = $state(false)
  let workspaceReady = $state(!desktop)
  let previewOpen = $state(false)
  let discardPrompt = $state<string | null>(null)
  let discardResolver: ((discard: boolean) => void) | null = null
  function decideDiscard(discard: boolean) {
    const resolve = discardResolver
    discardResolver = null
    discardPrompt = null
    resolve?.(discard)
  }
  function confirmDiscard(prompt: string): Promise<boolean> {
    if (discardResolver) return Promise.resolve(false)
    discardPrompt = prompt
    return new Promise((resolve) => {
      discardResolver = resolve
    })
  }
  let workspaceError = $state("")
  let closingSample = $state(false)
  onMount(() => {
    if (desktop)
      getDesktopWorkspace()
        .then((workspace) => {
          sample = workspace.sample
          isolated = workspace.isolated
          sampleReturnAvailable = workspace.return_available
          workspaceReady = true
        })
        .catch((cause) => {
          workspaceError = String(cause)
        })
  })
  async function closeSample() {
    closingSample = true
    try {
      if (!(await prepareWorkspaceSwitch())) {
        closingSample = false
        return
      }
      await closeDesktopSample()
      window.location.reload()
    } catch (cause) {
      workspaceError = String(cause)
      closingSample = false
    }
  }
  setContext<StudioState>(STUDIO_CONTEXT, {
    get locale() {
      return locale
    },
    desktop,
    get sample() {
      return sample
    },
    get isolated() {
      return isolated
    },
    get workspaceReady() {
      return workspaceReady
    },
    confirmDiscard,
    openPreview() {
      previewOpen = true
    },
  })

  function changeLocale(next: Locale) {
    locale = next
    saveLocale(locale)
  }
  $effect(() => {
    document.documentElement.lang = locale
  })
</script>

<svelte:head>
  <title>{message(locale, "admin.shell.page_title")}</title>
  {#if desktop}<script type="module" src="/wails/runtime.js"></script>{/if}
</svelte:head>

<DiscardChangesDialog prompt={discardPrompt} {locale} decide={decideDiscard} />

<div class="admin-shell" class:native-mac={nativeMac}>
  <header class="window-toolbar">
    <div class="brand"><img src="/felicia-mark.svg" alt="Felicia" width="24" height="24" /><span>Felicia</span></div>
  </header>
  <aside class="sidebar">
    {#if sample}
      <section class="sample-banner" aria-label={message(locale, "admin.sample.title")}>
        <strong>{message(locale, "admin.sample.title")}</strong>
        <p>{message(locale, "admin.sample.note")}</p>
        <Button variant="outline" size="sm" class="sample-return" disabled={closingSample} onclick={closeSample}
          >{message(locale, sampleReturnAvailable ? "admin.sample.return" : "admin.sample.close")}</Button
        >
      </section>
    {/if}
    {#if isolated && !sample}<p class="hint" role="status">{message(locale, "admin.sample.empty_studio")}</p>{/if}
    {#if workspaceError}<p class="api-error" role="alert">{workspaceError}</p>{/if}
    <nav aria-label={message(locale, "admin.shell.navigation_label")}>
      <a class:active={!site} aria-current={!site ? "page" : undefined} href={resolve("/")}>
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m3 5 6-2 6 2 6-2v16l-6 2-6-2-6 2V5ZM9 3v16M15 5v16" /></svg>
        {message(locale, "admin.journeys.title")}
      </a>
      <a class:active={site} aria-current={site ? "page" : undefined} href={resolve("/site")}>
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 8h16v12H4V8ZM12 3v11m-4-7 4-4 4 4" /></svg>
        {message(locale, "admin.site.navigation")}
      </a>
    </nav>
    <div class="sidebar-footer">
      <Popover.Root
        bind:open={settingsOpen}
        onOpenChange={(open) => {
          if (open) settingsHintOpen = false
        }}
      >
        <Tooltip.Root bind:open={settingsHintOpen} disabled={settingsOpen}>
          <Tooltip.Trigger>
            {#snippet child({ props })}
              <Popover.Trigger {...props} bind:ref={settingsTrigger} class="settings-trigger" aria-label={message(locale, "admin.shell.settings")}>
                <Settings2 size={18} aria-hidden="true" />
              </Popover.Trigger>
            {/snippet}
          </Tooltip.Trigger>
          {#if settingsHintOpen && !settingsOpen}
            <Tooltip.Portal>
              <Tooltip.Content role="tooltip" side="top" sideOffset={6} class="studio-tooltip">{message(locale, "admin.shell.settings")}</Tooltip.Content>
            </Tooltip.Portal>
          {/if}
        </Tooltip.Root>
        <Popover.Content
          side="top"
          align="start"
          role="dialog"
          class="settings-panel w-72"
          aria-label={message(locale, "admin.shell.settings")}
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            settingsTrigger?.focus()
          }}
        >
          <header class="preferences-heading">
            <Settings2 size={18} aria-hidden="true" />
            <h2>{message(locale, "admin.shell.settings")}</h2>
          </header>
          <label class="locale-control">
            <span class="preference-label"><Languages size={16} aria-hidden="true" />{message(locale, "admin.shell.language_label")}</span>
            <SelectField
              label={message(locale, "admin.shell.language_label")}
              value={locale}
              onValueChange={changeLocale}
              items={[
                { value: "ja", label: "日本語" },
                { value: "en", label: "English" },
                { value: "zh", label: "中文" },
              ]}
            />
          </label>
          <fieldset class="appearance-control">
            <legend>{message(locale, "admin.shell.appearance")}</legend>
            <div class="theme-options">
              {#each themeOptions as { value, Icon, key } (value)}
                <Button
                  type="button"
                  variant="outline"
                  class={`theme-choice h-auto ${userPrefersMode.current === value ? "chosen" : ""}`}
                  aria-pressed={userPrefersMode.current === value}
                  onclick={() => setMode(value)}
                >
                  <Icon size={18} aria-hidden="true" />
                  <span>{message(locale, key)}</span>
                </Button>
              {/each}
            </div>
          </fieldset>
          {#if desktop && !sample}<div class="sample-setting"><SampleButton /></div>{/if}
        </Popover.Content>
      </Popover.Root>
    </div>
  </aside>
  <main class="content" class:library>
    {@render children()}
  </main>
</div>
<CompiledPreview bind:open={previewOpen} {locale} {nativeMac} />

<style>
  .sample-banner {
    margin-bottom: 16px;
    padding: 10px;
    border: 1px solid var(--line);
    border-radius: 10px;
    background: var(--surface-raised);
    font-size: 12px;
  }
  .sample-banner p {
    color: var(--muted);
    line-height: 1.5;
  }
  .sample-banner :global(.sample-return) {
    width: 100%;
    min-width: 0;
    height: auto;
    min-height: 32px;
    padding-block: 6px;
    white-space: normal;
    overflow-wrap: anywhere;
  }
  @media (max-width: 820px) {
    .sample-banner :global(.sample-return) {
      width: auto;
      max-width: 100%;
    }
    .sample-banner {
      display: flex;
      align-items: center;
      gap: 8px;
      margin: 0;
      padding: 6px 8px;
    }
    .sample-banner p {
      display: none;
    }
  }
  .sample-setting {
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--line);
  }
  .preferences-heading,
  .preference-label {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .preferences-heading {
    padding-bottom: 12px;
    margin-bottom: 12px;
    border-bottom: 1px solid var(--line);
  }
  .preferences-heading h2 {
    font-size: 14px;
  }
  .appearance-control {
    min-width: 0;
    margin: 16px 0 0;
    padding: 0;
    border: 0;
  }
  .appearance-control legend {
    margin-bottom: 8px;
    font-size: 13px;
    color: var(--muted);
  }
  .theme-options {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 6px;
  }
  :global(.theme-choice) {
    display: grid;
    justify-items: center;
    gap: 6px;
    padding: 10px 4px;
    border: 1px solid var(--line);
    border-radius: 8px;
    color: var(--muted);
    background: var(--surface);
    font-size: 12px;
  }
  :global(.theme-choice:hover) {
    background: var(--surface-muted);
  }
  :global(.theme-choice.chosen) {
    border-color: var(--accent);
    color: var(--accent);
    background: var(--accent-soft);
  }
</style>
