<script lang="ts">
  import AtlasDetail from "./AtlasDetail.svelte"
  import AtlasMap from "./AtlasMap.svelte"
  import FloatingToolbar from "../../components/FloatingToolbar.svelte"
  import JournalStream from "../../components/JournalStream.svelte"
  import PhotoLightbox from "../../components/PhotoLightbox.svelte"
  import TripPlaybackCard from "../../playback/TripPlaybackCard.svelte"
  import TripStage from "../../playback/TripStage.svelte"
  import ExportDialog from "../../playback/ExportDialog.svelte"
  import {
    generatePlaybackSteps,
    PlaybackController,
    type PlaybackControllerState,
    type PlaybackStep,
  } from "../../playback/engine"
  import type { Journey, Lang, Memento, Theme } from "@felicia/model"

  let {
    lang = $bindable("ja"),
    theme = $bindable("dark"),
    loadJourneys,
  }: {
    lang?: Lang
    theme?: Theme
    loadJourneys: () => Promise<Journey[]>
  } = $props()

  let journeys = $state<Journey[]>([])
  let isLoading = $state(true)
  let error = $state<string | null>(null)
  let mode = $state<"map" | "journal">("map")
  let activeJourneyId = $state<string | null>(null)
  let activeMementoId = $state<string | null>(null)
  let selectedMementoId = $state<string | null>(null)
  let showStage = $state(false)
  let showExport = $state(false)
  let lightboxPhoto = $state<{ src: string; video?: string; caption: string } | null>(null)

  const activeJourney = $derived(
    journeys.find((journey) => journey.id === activeJourneyId) ?? journeys[0] ?? null
  )
  const selectedMemento = $derived(
    activeJourney?.mementos.find((memento) => memento.id === selectedMementoId) ?? null
  )

  // Playback engine state
  let playbackController = $state<PlaybackController | null>(null)
  let currentStep = $state<PlaybackStep | null>(null)
  let lastJourneyId = $state<string | null>(null)

  function initPlayback(journey: Journey | null) {
    playbackController?.destroy()
    if (!journey) {
      playbackController = null
      currentStep = null
      lastJourneyId = null
      return
    }
    const steps = generatePlaybackSteps(journey, lang)
    const ctrl = new PlaybackController(steps)
    ctrl.subscribe((state: PlaybackControllerState, step: PlaybackStep) => {
      currentStep = step
      if (step.type === "stop") {
        activeMementoId = step.memento.id
      } else {
        activeMementoId = null
      }
    })
    playbackController = ctrl
    lastJourneyId = journey.id
  }

  $effect(() => {
    if (activeJourney && activeJourney.id !== lastJourneyId) {
      initPlayback(activeJourney)
    }
  })

  function handleSelectJourney(id: string | null) {
    activeJourneyId = id
    selectedMementoId = null
    activeMementoId = null
  }

  function handleSelectPhoto(memento: Memento, photoIndex: number) {
    const photo = memento.photos[photoIndex]
    if (photo) {
      lightboxPhoto = {
        src: photo.src,
        video: photo.video,
        caption: photo.caption[lang] || photo.caption.zh || photo.caption.en || "",
      }
    }
  }

  function closeMemento() {
    selectedMementoId = null
  }

  function loadData() {
    isLoading = true
    error = null
    loadJourneys()
      .then((data) => {
        journeys = data
        activeJourneyId = data[0]?.id ?? null
        activeMementoId = null
      })
      .catch((reason) => {
        error = reason instanceof Error ? reason.message : String(reason)
      })
      .finally(() => {
        isLoading = false
      })
  }

  $effect(() => loadData())
</script>

<svelte:window onkeydown={(event) => event.key === "Escape" && (closeMemento(), (showStage = false), (showExport = false), (lightboxPhoto = null))} />

<main class="felicia-canvas relative z-20 isolate h-screen w-full overflow-hidden bg-[#07131f] text-stone-100" class:light={theme === "light"}>
  {#if isLoading}
    <div class="flex h-screen w-full items-center justify-center text-sm text-stone-400">
      Loading journal...
    </div>
  {:else if error}
    <div class="flex h-screen w-full flex-col items-center justify-center gap-3 text-sm text-stone-400">
      <p class="text-rose-400">{error}</p>
      <button
        type="button"
        class="rounded-lg border border-white/20 px-3 py-1.5 text-xs text-white hover:bg-white/10"
        onclick={loadData}
      >
        Retry
      </button>
    </div>
  {:else if journeys.length}
    <!-- Shared Top Floating Toolbar HUD -->
    <FloatingToolbar
      {journeys}
      {activeJourneyId}
      {mode}
      {lang}
      onSelectJourney={handleSelectJourney}
      onModeChange={(newMode) => (mode = newMode)}
    />

    {#if mode === "map"}
      <!-- Fullscreen Map Surface -->
      <div class="absolute inset-0">
        <AtlasMap
          {journeys}
          {activeJourneyId}
          {activeMementoId}
          {currentStep}
          {lang}
          {theme}
          onSelect={(id) => {
            activeMementoId = id
            selectedMementoId = id
          }}
        />
      </div>

      <!-- Bottom-Left Playback HUD Card -->
      {#if playbackController && currentStep}
        <TripPlaybackCard
          controller={playbackController}
          {currentStep}
          {lang}
          onOpenStage={() => (showStage = true)}
          onOpenExport={() => (showExport = true)}
          onOpenDetail={(mementoId) => (selectedMementoId = mementoId)}
          onClose={() => {
            playbackController?.pause()
            activeMementoId = null
          }}
        />
      {/if}

      <!-- Stage Mode (Fullscreen Presenter View) -->
      {#if showStage && playbackController && currentStep}
        <TripStage
          controller={playbackController}
          {currentStep}
          {lang}
          onClose={() => (showStage = false)}
        />
      {/if}

      <!-- Export Video Dialog -->
      {#if showExport && playbackController && activeJourney}
        <ExportDialog
          controller={playbackController}
          journeyTitle={activeJourney.title[lang] || activeJourney.title.zh || activeJourney.title.ja}
          onClose={() => (showExport = false)}
        />
      {/if}

      <!-- Memento Detail Drawer -->
      {#if selectedMemento}
        <AtlasDetail
          memento={selectedMemento}
          {lang}
          photoLabel="Photos"
          photosHeading="Photos"
          closeLabel="Close"
          storyLabel="Story"
          onClose={closeMemento}
        />
      {/if}

    {:else if mode === "journal" && activeJourney}
      <!-- Vertical Reading Stream (手记 / 日历) -->
      <div class="relative z-10 h-screen w-full overflow-y-auto overscroll-contain">
        <JournalStream
          journey={activeJourney}
          {lang}
          onSelectPhoto={handleSelectPhoto}
        />
      </div>

      <!-- Lightbox for Journal Stream Photos -->
      {#if lightboxPhoto}
        <PhotoLightbox
          src={lightboxPhoto.src}
          video={lightboxPhoto.video}
          alt={lightboxPhoto.caption}
          caption={lightboxPhoto.caption}
          openLabel="Open"
          closeLabel="Close"
        />
      {/if}
    {/if}
  {:else}
    <div class="flex h-screen w-full items-center justify-center text-sm text-stone-400">
      No journeys found
    </div>
  {/if}
</main>
