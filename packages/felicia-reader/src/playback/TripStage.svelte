<script lang="ts">
  import { untrack } from "svelte"
  import type { Lang } from "@felicia/model"
  import type { PlaybackController, PlaybackControllerState, PlaybackStep } from "./engine"

  let {
    controller,
    currentStep,
    lang = "en",
    onClose,
  }: {
    controller: PlaybackController
    currentStep: PlaybackStep
    lang: Lang
    onClose: () => void
  } = $props()

  let state = $state<PlaybackControllerState>(untrack(() => controller.getState()))

  $effect(() => {
    return controller.subscribe((s) => {
      state = s
    })
  })

  const i18n = $derived(
    {
      zh: {
        stage: "旅程放映",
        summary: "旅程回顾",
        departure: "旅程出发",
        arrival: "归途到达",
        dayPrefix: "第",
        daySuffix: "天",
        days: "天",
        records: "条记录",
        photos: "张照片",
        close: "退出放映",
      },
      ja: {
        stage: "旅の上映",
        summary: "旅の振り返り",
        departure: "出発",
        arrival: "到着・帰路",
        dayPrefix: "第",
        daySuffix: "日",
        days: "日",
        records: "件の記録",
        photos: "枚の写真",
        close: "上映を終了",
      },
      en: {
        stage: "Trip Cinema",
        summary: "Trip Summary",
        departure: "Departure",
        arrival: "Arrival / Return",
        dayPrefix: "Day",
        daySuffix: "",
        days: "days",
        records: "records",
        photos: "photos",
        close: "Exit Stage",
      },
    }[lang]
  )
</script>

<div class="fixed inset-0 z-50 flex flex-col justify-between overflow-hidden bg-black/90 p-6 text-white backdrop-blur-2xl max-[48rem]:p-4">
  <!-- Top Bar -->
  <header class="flex items-center justify-between">
    <div>
      {#if currentStep.type === "intro"}
        <p class="text-xs tracking-wider text-[#39C5BB] uppercase">{i18n.stage}</p>
        <h1 class="text-xl font-bold">{currentStep.title}</h1>
      {:else if currentStep.type === "day"}
        <p class="text-xs tracking-wider text-[#39C5BB] uppercase">{i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}</p>
        <h1 class="text-xl font-bold">{currentStep.date}</h1>
      {:else if currentStep.type === "waypoint"}
        <p class="text-xs tracking-wider text-[#39C5BB] uppercase">
          {currentStep.role === "start" ? i18n.departure : i18n.arrival} · {i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}
        </p>
        <h1 class="text-xl font-bold">{currentStep.title}</h1>
      {:else if currentStep.type === "stop"}
        <p class="text-xs tracking-wider text-[#39C5BB] uppercase">
          {i18n.dayPrefix} {currentStep.day} {i18n.daySuffix} · {currentStep.memento.place[lang] || currentStep.memento.place.ja || currentStep.memento.place.en || currentStep.memento.place.zh}
        </p>
        <h1 class="text-xl font-bold">{currentStep.memento.title[lang] || currentStep.memento.title.ja || currentStep.memento.title.en || currentStep.memento.title.zh}</h1>
      {:else if currentStep.type === "summary"}
        <p class="text-xs tracking-wider text-[#39C5BB] uppercase">{i18n.summary}</p>
        <h1 class="text-xl font-bold">{currentStep.title}</h1>
      {/if}
    </div>

    <button
      type="button"
      class="flex size-10 cursor-pointer items-center justify-center rounded-full border border-white/20 bg-white/10 text-white transition hover:bg-white/25"
      onclick={onClose}
      aria-label={i18n.close}
    >
      ✕
    </button>
  </header>

  <!-- Center Stage Media -->
  <main class="relative flex flex-1 items-center justify-center py-4">
    {#if currentStep.type === "intro"}
      <!-- Cover Grid in Stage -->
      <div class="flex max-w-xl flex-wrap items-center justify-center gap-4">
        {#each currentStep.covers.slice(0, 3) as cover}
          <img src={cover.src} alt="" class="size-44 rounded-xl border border-white/20 object-cover shadow-2xl" />
        {/each}
      </div>

    {:else if currentStep.type === "day"}
      <div class="text-center">
        <div class="text-5xl">☁️</div>
        {#if currentStep.weather}
          <p class="mt-4 text-lg text-stone-300">{currentStep.weather.label}</p>
        {/if}
        {#if currentStep.places.length > 0}
          <p class="mt-2 text-sm text-stone-400">{currentStep.places.join(" → ")}</p>
        {/if}
      </div>

    {:else if currentStep.type === "stop"}
      {#if currentStep.memento.photos.length > 0}
        {@const photo = currentStep.memento.photos[0]}
        <div class="relative flex max-h-[60vh] max-w-[85vw] items-center justify-center overflow-hidden rounded-2xl border border-white/20 bg-stone-900 shadow-2xl">
          {#if photo.video}
            <video
              src={photo.video}
              poster={photo.src}
              autoplay
              loop
              muted
              playsinline
              class="max-h-[60vh] max-w-full object-contain"
            ></video>
            <span class="absolute top-3 left-3 flex items-center gap-1 rounded-full bg-black/60 px-2.5 py-1 text-xs font-medium text-white backdrop-blur">
              <span class="size-2 rounded-full bg-[#FF5E98]"></span> LIVE
            </span>
          {:else}
            <img src={photo.src} alt="" class="max-h-[60vh] max-w-full object-contain" />
          {/if}
        </div>
      {:else}
        <div class="max-w-lg text-center text-stone-400">
          <p class="text-lg">📍 {currentStep.memento.place[lang] || currentStep.memento.place.ja || currentStep.memento.place.en || currentStep.memento.place.zh}</p>
        </div>
      {/if}

    {:else if currentStep.type === "waypoint"}
      <div class="text-center">
        <div class="text-6xl">{currentStep.role === "start" ? "🚗" : "🏁"}</div>
        <h2 class="mt-4 text-3xl font-bold">{currentStep.place}</h2>
        <p class="mt-2 text-base text-stone-300">{currentStep.title}</p>
        {#if currentStep.time}
          <p class="mt-2 text-lg font-mono font-semibold text-[#39C5BB]">
            🕒 {currentStep.date ? `${currentStep.date} ` : ""}{currentStep.time}
          </p>
        {/if}
        <div class="mt-4 inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/10 px-4 py-1.5 text-xs font-mono text-stone-200">
          <span>{currentStep.coords[0].toFixed(4)}, {currentStep.coords[1].toFixed(4)}</span>
          {#if currentStep.distanceKm !== undefined}
            <span class="text-stone-400">·</span>
            <span class="text-teal-300">{currentStep.distanceKm.toLocaleString()} km</span>
          {/if}
        </div>
      </div>

    {:else if currentStep.type === "summary"}
      <div class="grid grid-cols-3 gap-4 text-center max-[48rem]:grid-cols-2">
        <div class="rounded-xl border border-white/10 bg-white/5 p-4">
          <div class="font-mono text-3xl font-bold text-[#39C5BB]">{currentStep.dayCount}</div>
          <div class="mt-1 text-xs text-stone-400">{i18n.days}</div>
        </div>
        <div class="rounded-xl border border-white/10 bg-white/5 p-4">
          <div class="font-mono text-3xl font-bold text-[#39C5BB]">{currentStep.stopCount}</div>
          <div class="mt-1 text-xs text-stone-400">{i18n.records}</div>
        </div>
        <div class="rounded-xl border border-white/10 bg-white/5 p-4">
          <div class="font-mono text-3xl font-bold text-[#39C5BB]">{currentStep.photoCount}</div>
          <div class="mt-1 text-xs text-stone-400">{i18n.photos}</div>
        </div>
      </div>
    {/if}
  </main>

  <!-- Bottom Stage HUD -->
  <footer class="flex flex-col items-center gap-3">
    {#if currentStep.type === "stop" && (currentStep.memento.essay[lang] || currentStep.memento.essay.ja || currentStep.memento.essay.en || currentStep.memento.essay.zh)}
      <p class="max-w-2xl text-center text-sm leading-relaxed text-stone-300">
        {currentStep.memento.essay[lang] || currentStep.memento.essay.ja || currentStep.memento.essay.en || currentStep.memento.essay.zh}
      </p>
    {/if}

    <!-- Segmented Stage Progress Bar -->
    <ol class="flex w-full max-w-xl shrink-0 list-none gap-1 pt-2" aria-label="Stage Progress">
      {#each Array(state.totalSteps) as _, i}
        <li class="flex-1">
          <button
            type="button"
            class="h-1 w-full cursor-pointer rounded-full border-0 p-0 transition-colors duration-300 {i <= state.currentStepIndex ? 'bg-[#39C5BB]' : 'bg-white/20'}"
            onclick={() => controller.seek(i)}
            aria-label="Step {i + 1}"
          ></button>
        </li>
      {/each}
    </ol>

    <!-- Controls -->
    <div class="flex items-center gap-4 py-1">
      <button
        type="button"
        class="flex size-10 cursor-pointer items-center justify-center rounded-full border border-white/10 bg-white/5 text-stone-300 hover:bg-white/15 hover:text-white disabled:opacity-30"
        disabled={state.currentStepIndex === 0}
        onclick={() => controller.previous()}
      >
        ⏮
      </button>

      <button
        type="button"
        class="flex size-12 cursor-pointer items-center justify-center rounded-full border border-[#39C5BB]/40 bg-[#39C5BB]/30 text-lg text-[#39C5BB] hover:bg-[#39C5BB]/50 hover:text-white"
        onclick={() => controller.togglePlay()}
      >
        {state.isPlaying ? "⏸" : "▶"}
      </button>

      <button
        type="button"
        class="flex size-10 cursor-pointer items-center justify-center rounded-full border border-white/10 bg-white/5 text-stone-300 hover:bg-white/15 hover:text-white disabled:opacity-30"
        disabled={state.currentStepIndex === state.totalSteps - 1}
        onclick={() => controller.next()}
      >
        ⏭
      </button>
    </div>
  </footer>
</div>
