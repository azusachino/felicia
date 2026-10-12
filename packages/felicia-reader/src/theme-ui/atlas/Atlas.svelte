<script lang="ts">
  import AtlasDetail from "./AtlasDetail.svelte"
  import AtlasMap from "./AtlasMap.svelte"
  import FloatingToolbar from "../../components/FloatingToolbar.svelte"
  import JournalStream from "../../components/JournalStream.svelte"
  import PhotoLightbox from "../../components/PhotoLightbox.svelte"
  import TripPlaybackCard from "../../playback/TripPlaybackCard.svelte"
  import TripStage from "../../playback/TripStage.svelte"
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
  let lightboxPhoto = $state<{ src: string; video?: string; caption: string } | null>(null)

  const activeJourney = $derived(
    activeJourneyId ? (journeys.find((journey) => journey.id === activeJourneyId) ?? null) : null
  )
  const selectedMemento = $derived(
    activeJourney
      ? (activeJourney.mementos.find((memento) => memento.id === selectedMementoId) ?? null)
      : (journeys.flatMap((j) => j.mementos).find((m) => m.id === selectedMementoId) ?? null)
  )

  // Playback engine state
  let playbackController = $state<PlaybackController | null>(null)
  let currentStep = $state<PlaybackStep | null>(null)
  let isPlaying = $state(false)
  let lastJourneyId = $state<string | null>(null)
  let legPhase = $state<"transit" | "arrived">("arrived")
  let legDistanceKm = $state<number>(0)
  let isPlaybackDismissed = $state(false)

  function initPlayback(journey: Journey | null) {
    playbackController?.destroy()
    isPlaybackDismissed = false
    if (!journey) {
      playbackController = null
      currentStep = null
      isPlaying = false
      lastJourneyId = null
      legPhase = "arrived"
      legDistanceKm = 0
      return
    }
    const steps = generatePlaybackSteps(journey, lang)
    const ctrl = new PlaybackController(steps)
    ctrl.subscribe((state: PlaybackControllerState, step: PlaybackStep) => {
      currentStep = step
      isPlaying = state.isPlaying
      if (step.type === "stop") {
        activeMementoId = step.memento.id
        legDistanceKm = step.travelDistanceKm ?? 0
        legPhase = state.isPlaying && (step.travelDistanceKm ?? 0) > 0.05 ? "transit" : "arrived"
      } else {
        activeMementoId = null
        legPhase = "arrived"
        legDistanceKm = 0
      }
    })
    playbackController = ctrl
    lastJourneyId = journey.id
  }

  $effect(() => {
    if (activeJourney && activeJourney.id !== lastJourneyId) {
      initPlayback(activeJourney)
    } else if (!activeJourney && playbackController) {
      initPlayback(null)
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

<svelte:window onkeydown={(event) => event.key === "Escape" && (closeMemento(), (showStage = false), (lightboxPhoto = null))} />

<main
  class="felicia-canvas waypoints relative z-20 isolate h-screen w-full overflow-hidden transition-colors duration-300 {theme === 'light' ? 'bg-[#f8fafc] text-stone-900' : 'bg-[#07131f] text-stone-100'}"
  class:light={theme === "light"}
>
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
      {theme}
      onSelectJourney={handleSelectJourney}
      onModeChange={(newMode) => (mode = newMode)}
      onToggleTheme={() => (theme = theme === "light" ? "dark" : "light")}
    />

    {#if mode === "map"}
      <!-- Fullscreen Map Surface -->
      <div class="absolute inset-0">
        <AtlasMap
          {journeys}
          {activeJourneyId}
          {activeMementoId}
          {currentStep}
          {isPlaying}
          {lang}
          {theme}
          onSelect={(id) => {
            activeMementoId = id
            selectedMementoId = id
          }}
          onLegStart={() => {
            if (isPlaying) legPhase = "transit"
          }}
          onLegArrive={() => {
            legPhase = "arrived"
          }}
        />
      </div>

      <!-- Bottom-Left Playback HUD Card -->
      {#if playbackController && currentStep && !isPlaybackDismissed}
        <TripPlaybackCard
          controller={playbackController}
          {currentStep}
          {lang}
          {theme}
          {legPhase}
          {legDistanceKm}
          onOpenStage={() => (showStage = true)}
          onOpenDetail={(mementoId) => (selectedMementoId = mementoId)}
          onClose={() => {
            playbackController?.pause()
            isPlaybackDismissed = true
          }}
        />
      {:else if playbackController && isPlaybackDismissed}
        <!-- Re-open Playback Pill -->
        <div class="pointer-events-auto absolute bottom-4 left-4 z-20">
          <button
            type="button"
            class="flex items-center gap-2 rounded-full border px-3.5 py-2 text-xs font-semibold shadow-xl backdrop-blur-xl transition {theme === 'light' ? 'border-black/10 bg-white/90 text-stone-800 hover:bg-white' : 'border-white/20 bg-black/75 text-white hover:bg-black/90'}"
            onclick={() => (isPlaybackDismissed = false)}
          >
            <span class="text-orange-400">▶</span>
            <span>{lang === 'ja' ? '旅の再生を開く' : lang === 'zh' ? '恢复行程回放' : 'Open Trip Playback'}</span>
          </button>
        </div>
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

    {:else if mode === "journal"}
      <!-- Vertical Reading Stream (手记 / 日历) -->
      <div class="relative z-10 h-screen w-full overflow-y-auto overscroll-contain">
        {#if activeJourney}
          <JournalStream
            journey={activeJourney}
            {lang}
            {theme}
            onSelectPhoto={handleSelectPhoto}
          />
        {:else}
          {#each journeys as j}
            <div class="border-b last:border-b-0 {theme === 'light' ? 'border-black/10' : 'border-white/10'}">
              <JournalStream
                journey={j}
                {lang}
                {theme}
                onSelectPhoto={handleSelectPhoto}
              />
            </div>
          {/each}
        {/if}
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
