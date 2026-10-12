<script lang="ts">
  import { untrack } from "svelte"
  import type { Lang, Theme } from "@felicia/model"
  import type { PlaybackController, PlaybackControllerState, PlaybackStep } from "./engine"
  import { getDayColor } from "./timeline"

  let {
    controller,
    currentStep,
    lang = "en",
    theme = "dark",
    legPhase = "arrived",
    legDistanceKm = 0,
    onOpenStage,
    onOpenExport,
    onOpenDetail,
    onClose,
  }: {
    controller: PlaybackController
    currentStep: PlaybackStep
    lang: Lang
    theme?: Theme
    legPhase?: "transit" | "arrived"
    legDistanceKm?: number
    onOpenStage?: () => void
    onOpenExport?: () => void
    onOpenDetail?: (mementoId: string) => void
    onClose?: () => void
  } = $props()

  let state = $state<PlaybackControllerState>(untrack(() => controller.getState()))

  $effect(() => {
    return controller.subscribe((s) => {
      state = s
    })
  })
  const speeds = [1, 1.5, 2]

  function nextSpeed() {
    const next = speeds[(speeds.indexOf(state.speed) + 1) % speeds.length]
    controller.setSpeed(next)
  }

  const i18n = $derived(
    {
      zh: {
        intro: "旅程回放",
        summary: "旅程回顾",
        departure: "旅程出发",
        arrival: "归途到达",
        origin: "起点",
        destination: "终点",
        cumDist: "累计行程",
        dayPrefix: "第",
        daySuffix: "天",
        records: "条记录",
        places: "个地点",
        photos: "张照片",
        km: "公里",
        hours: "小时车程",
        present: "放映",
        prev: "上一站",
        next: "下一站",
        play: "播放",
        pause: "暂停",
        close: "退出回放",
        readStory: "阅读完整手记 →",
        departureTime: "出发时间",
        arrivalTime: "到达时间",
        transit: "行进中",
        nextStop: "下一站目的地",
        arrived: "已到达",
        legDist: "区间距离",
        export: "导出视频与章节",
      },
      ja: {
        intro: "旅の再生",
        summary: "旅の振り返り",
        departure: "出発",
        arrival: "到着・帰路",
        origin: "起点",
        destination: "終着点",
        cumDist: "累計行程",
        dayPrefix: "第",
        daySuffix: "日",
        records: "件の記録",
        places: "ヶ所",
        photos: "枚の写真",
        km: "キロ",
        hours: "時間の行程",
        present: "上映",
        prev: "前へ",
        next: "次へ",
        play: "再生",
        pause: "一時停止",
        close: "閉じる",
        readStory: "手記を読む →",
        departureTime: "出発時刻",
        arrivalTime: "到着時刻",
        transit: "移動中",
        nextStop: "次の目的地",
        arrived: "到着",
        legDist: "区間距離",
        export: "動画・チャプター書き出し",
      },
      en: {
        intro: "Trip Playback",
        summary: "Trip Summary",
        departure: "Departure",
        arrival: "Arrival / Return",
        origin: "Origin",
        destination: "Destination",
        cumDist: "Total Distance",
        dayPrefix: "Day",
        daySuffix: "",
        records: "records",
        places: "places",
        photos: "photos",
        km: "km",
        hours: "driving hrs",
        present: "Present",
        prev: "Previous",
        next: "Next",
        play: "Play",
        pause: "Pause",
        close: "Close",
        readStory: "Read full story →",
        departureTime: "Departure Time",
        arrivalTime: "Arrival Time",
        transit: "En route",
        nextStop: "Next Destination",
        arrived: "Arrived",
        legDist: "Leg Distance",
        export: "Export Video & Chapters",
      },
    }[lang]
  )
</script>

<aside class="calendar-map-trip pointer-events-auto absolute bottom-3 left-3 z-20 flex h-[15.5rem] max-h-[15.5rem] w-[min(23rem,calc(100vw-1.5rem))] flex-col overflow-hidden overscroll-contain rounded-2xl border shadow-2xl backdrop-blur-2xl transition-colors duration-200 sm:bottom-4 sm:left-4 sm:h-auto sm:max-h-[calc(100vh-6rem)] sm:w-[23rem] {theme === 'light' ? 'border-black/10 bg-white/92 text-stone-900 shadow-stone-400/30' : 'border-white/20 bg-black/80 text-white'}">
  <!-- Top Segmented Progress Bar -->
  <ol class="flex shrink-0 list-none gap-1 px-4 pt-3" aria-label="Trip Progress">
    {#each Array(state.totalSteps) as _, i}
      <li class="flex-1">
        <button
          type="button"
          class="h-1 w-full cursor-pointer rounded-full border-0 p-0 transition-colors duration-300 {i <= state.currentStepIndex ? 'bg-[#39C5BB]' : theme === 'light' ? 'bg-black/10' : 'bg-white/20'}"
          onclick={() => controller.seek(i)}
          aria-label="Step {i + 1}"
        ></button>
      </li>
    {/each}
  </ol>

  <!-- Step Content -->
  <div class="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain px-4 pt-2 pb-2.5 [touch-action:pan-y]">
    {#if currentStep.type === "intro"}
      <!-- INTRO CARD -->
      <div class="py-1">
        <p class="text-[0.7rem] font-semibold tracking-wider text-[#39C5BB] uppercase">{i18n.intro}</p>
        <h2 class="mt-0.5 truncate text-lg font-bold">{currentStep.title}</h2>
        <p class="text-xs {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{currentStep.dateRange} · {currentStep.dayCount} {i18n.daySuffix} · {currentStep.stopCount} {i18n.records}</p>

        <!-- 3 Tilted Polaroid Cover Fan (compact on mobile) -->
        {#if currentStep.covers.length > 0}
          <div class="relative my-1.5 flex h-16 items-center justify-center sm:my-4 sm:h-28">
            {#each currentStep.covers.slice(0, 3) as cover, idx}
              <div
                class="absolute size-14 overflow-hidden rounded-lg border-2 bg-stone-900 shadow-xl transition-transform duration-300 sm:size-24 {theme === 'light' ? 'border-white shadow-stone-400/40' : 'border-white/90'}"
                style:transform={idx === 0 ? "translateX(-1.3rem) rotate(-7deg)" : idx === 2 ? "translateX(1.3rem) rotate(7deg)" : "rotate(0deg) scale(1.05)"}
                style:z-index={idx === 1 ? 3 : 1}
              >
                <img src={cover.src} alt="" class="size-full object-cover" />
              </div>
            {/each}
          </div>
        {/if}

        {#if currentStep.distanceKm || currentStep.drivingHours}
          <div class="mt-2 grid grid-cols-2 gap-2 text-xs">
            {#if currentStep.distanceKm}
              <div class="rounded-lg p-2 text-center {theme === 'light' ? 'bg-black/5 text-stone-800' : 'bg-white/10 text-white'}">
                <div class="font-mono text-base font-semibold">{currentStep.distanceKm.toLocaleString()}</div>
                <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.km}</div>
              </div>
            {/if}
            {#if currentStep.drivingHours}
              <div class="rounded-lg p-2 text-center {theme === 'light' ? 'bg-black/5 text-stone-800' : 'bg-white/10 text-white'}">
                <div class="font-mono text-base font-semibold">{currentStep.drivingHours}</div>
                <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.hours}</div>
              </div>
            {/if}
          </div>
        {/if}
      </div>

    {:else if currentStep.type === "day"}
      <!-- DAY BANNER CARD -->
      <div class="py-3">
        <p class="text-[0.7rem] font-semibold tracking-wider uppercase" style="color: {getDayColor(currentStep.day).color};">
          {i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}
        </p>
        <h2 class="mt-0.5 text-lg font-bold">{currentStep.date}</h2>

        {#if currentStep.weather}
          <div class="mt-2 inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs {theme === 'light' ? 'bg-black/5 text-stone-700' : 'bg-white/10 text-stone-300'}">
            <span>☁️</span>
            <span>{currentStep.weather.label}</span>
          </div>
        {/if}

        {#if currentStep.places.length > 0}
          <div class="mt-3 text-xs {theme === 'light' ? 'text-stone-600' : 'text-stone-400'}">
            <span class="font-medium {theme === 'light' ? 'text-stone-800' : 'text-stone-300'}">{currentStep.places.join(" → ")}</span>
          </div>
        {/if}
      </div>

    {:else if currentStep.type === "waypoint"}
      <!-- WAYPOINT CARD (DEPARTURE OR ARRIVAL) -->
      <div class="py-2">
        <div class="flex items-center justify-between text-xs {theme === 'light' ? 'text-stone-600' : 'text-stone-400'}">
          <span class="inline-flex items-center gap-1.5 font-semibold text-[#39C5BB]">
            {#if currentStep.role === "start"}
              <svg class="size-3.5 fill-current" viewBox="0 0 24 24">
                <polygon points="6 3 20 12 6 21 6 3"></polygon>
              </svg>
              <span>{i18n.departure}</span>
            {:else}
              <svg class="size-3.5 stroke-current" viewBox="0 0 24 24" fill="none" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" fill="currentColor"></path>
                <line x1="4" y1="22" x2="4" y2="15"></line>
              </svg>
              <span>{i18n.arrival}</span>
            {/if}
          </span>
          <span class="rounded px-1.5 py-0.5 font-semibold" style="background-color: {getDayColor(currentStep.day).color}22; color: {getDayColor(currentStep.day).color}; border: 1px solid {getDayColor(currentStep.day).color}44;">{i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}</span>
        </div>
        <h2 class="mt-1 text-base font-bold">{currentStep.title}</h2>
        <p class="mt-0.5 flex items-center gap-1 text-xs {theme === 'light' ? 'text-stone-600' : 'text-stone-300'}">
          <svg class="size-3.5 shrink-0 {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"></path>
            <circle cx="12" cy="10" r="3"></circle>
          </svg>
          <span>{currentStep.place}</span>
        </p>

        <div class="mt-3 rounded-lg border p-2.5 text-xs {theme === 'light' ? 'border-black/10 bg-black/5 text-stone-800' : 'border-white/15 bg-white/5 text-stone-300'}">
          {#if currentStep.time}
            <div class="mb-2 flex items-center justify-between border-b pb-2 {theme === 'light' ? 'border-black/10' : 'border-white/10'}">
              <span class="{theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{currentStep.role === "start" ? i18n.departureTime : i18n.arrivalTime}</span>
              <span class="flex items-center gap-1 font-mono font-semibold {theme === 'light' ? 'text-stone-900' : 'text-white'}">
                <svg class="size-3.5 {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10"></circle>
                  <polyline points="12 6 12 12 16 14"></polyline>
                </svg>
                <span>{currentStep.date ? `${currentStep.date} ` : ""}{currentStep.time}</span>
              </span>
            </div>
          {/if}
          <div class="flex items-center justify-between">
            <span class="{theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{currentStep.role === "start" ? i18n.origin : i18n.destination}</span>
            <span class="font-mono {theme === 'light' ? 'text-stone-700' : 'text-stone-200'}">
              {currentStep.coords[0].toFixed(4)}, {currentStep.coords[1].toFixed(4)}
            </span>
          </div>
          {#if currentStep.distanceKm !== undefined}
            <div class="mt-1.5 flex items-center justify-between border-t pt-1.5 {theme === 'light' ? 'border-black/10' : 'border-white/10'}">
              <span class="{theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.cumDist}</span>
              <span class="font-mono font-semibold {theme === 'light' ? 'text-teal-700' : 'text-[#39C5BB]'}">
                {currentStep.distanceKm.toLocaleString()} {i18n.km}
              </span>
            </div>
          {/if}
        </div>
      </div>

    {:else if currentStep.type === "stop"}
      {#if legPhase === "transit"}
        <!-- TRANSIT EN ROUTE HUD -->
        <div class="py-1">
          <div class="flex items-center justify-between text-xs {theme === 'light' ? 'text-stone-600' : 'text-stone-400'}">
            <span class="font-semibold" style="color: {getDayColor(currentStep.day).color};">
              {i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}
            </span>
            <span class="flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[0.68rem] font-semibold {theme === 'light' ? 'border-[#39C5BB]/40 bg-[#39C5BB]/10 text-teal-700' : 'border-[#39C5BB]/40 bg-[#39C5BB]/20 text-[#39C5BB]'}">
              <span class="size-1.5 rounded-full bg-[#39C5BB] animate-pulse"></span>
              {i18n.transit}
            </span>
          </div>

          <div class="mt-2 text-[0.65rem] font-medium tracking-wider {theme === 'light' ? 'text-stone-500' : 'text-stone-400'} uppercase">{i18n.nextStop}</div>
          <h2 class="mt-0.5 text-base font-bold {theme === 'light' ? 'text-stone-900' : 'text-white'}">
            {currentStep.memento.title[lang] || currentStep.memento.title.ja || currentStep.memento.title.en || currentStep.memento.title.zh}
          </h2>
          <p class="mt-0.5 flex items-center gap-1 text-xs {theme === 'light' ? 'text-stone-600' : 'text-stone-300'}">
            <span class="text-[#39C5BB]">📍</span>
            <span>{currentStep.memento.place[lang] || currentStep.memento.place.ja || currentStep.memento.place.en || currentStep.memento.place.zh}</span>
          </p>

          <!-- Transit Leg Details -->
          <div class="mt-3 rounded-xl border p-3 text-xs {theme === 'light' ? 'border-black/10 bg-black/5' : 'border-white/15 bg-white/5'}">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="flex size-7 items-center justify-center rounded-full bg-[#39C5BB]/20 text-sm">🚗</span>
                <div>
                  <div class="font-semibold {theme === 'light' ? 'text-stone-900' : 'text-white'}">{currentStep.memento.place[lang] || currentStep.memento.place.ja || currentStep.memento.place.en || currentStep.memento.place.zh}</div>
                  <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.transit}</div>
                </div>
              </div>
              {#if currentStep.travelDistanceKm || legDistanceKm}
                <div class="text-right">
                  <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.legDist}</div>
                  <div class="font-mono font-bold {theme === 'light' ? 'text-teal-700' : 'text-[#39C5BB]'}">
                    {currentStep.travelDistanceKm ?? legDistanceKm} {i18n.km}
                  </div>
                </div>
              {/if}
            </div>
          </div>
        </div>
      {:else}
        <!-- ARRIVED MEMENTO CARD -->
        <div class="py-1">
          <div class="flex items-center justify-between text-xs {theme === 'light' ? 'text-stone-600' : 'text-stone-400'}">
            <span class="font-semibold" style="color: {getDayColor(currentStep.day).color};">{i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}</span>
            <div class="flex items-center gap-1.5">
              <span class="flex items-center gap-1 rounded-full border px-2 py-0.5 text-[0.65rem] font-semibold {theme === 'light' ? 'border-emerald-600/30 bg-emerald-600/10 text-emerald-700' : 'border-emerald-500/30 bg-emerald-500/20 text-emerald-300'}">
                <span>✓</span>
                <span>{i18n.arrived}</span>
              </span>
              <span class="rounded px-1.5 py-0.5 {theme === 'light' ? 'bg-black/5 text-stone-700' : 'bg-white/10 text-stone-300'}">{currentStep.memento.place[lang] || currentStep.memento.place.ja || currentStep.memento.place.en || currentStep.memento.place.zh}</span>
            </div>
          </div>
          <h2 class="mt-1 text-base font-bold">
            {currentStep.memento.title[lang] || currentStep.memento.title.ja || currentStep.memento.title.en || currentStep.memento.title.zh}
          </h2>

          <!-- Photo / Live Photo Polaroid Frame -->
          {#if currentStep.memento.photos.length > 0}
            {@const photo = currentStep.memento.photos[0]}
            <div class="relative my-2 overflow-hidden rounded-xl border bg-stone-900 shadow-md sm:my-3 {theme === 'light' ? 'border-black/10' : 'border-white/20'}">
              {#if photo.video}
                <video
                  src={photo.video}
                  poster={photo.src}
                  autoplay
                  loop
                  muted
                  playsinline
                  class="max-h-32 w-full object-cover sm:max-h-none sm:aspect-[4/3]"
                ></video>
                <span class="absolute top-2 left-2 flex items-center gap-1 rounded-full bg-black/60 px-2 py-0.5 text-[0.625rem] font-medium text-white backdrop-blur">
                  <span class="size-2 rounded-full bg-[#FF5E98]"></span> LIVE
                </span>
              {:else}
                <img src={photo.src} alt="" class="max-h-32 w-full object-cover sm:max-h-none sm:aspect-[4/3]" />
              {/if}
            </div>
          {/if}

          {#if currentStep.memento.essay[lang] || currentStep.memento.essay.ja || currentStep.memento.essay.en || currentStep.memento.essay.zh}
            <p class="mt-1 line-clamp-3 text-xs leading-relaxed {theme === 'light' ? 'text-stone-600' : 'text-stone-300'}">
              {currentStep.memento.essay[lang] || currentStep.memento.essay.ja || currentStep.memento.essay.en || currentStep.memento.essay.zh}
            </p>
          {/if}

          {#if onOpenDetail}
            <button
              type="button"
              class="mt-2 text-xs text-[#39C5BB] underline underline-offset-4 hover:text-teal-400"
              onclick={() => onOpenDetail?.(currentStep.memento.id)}
            >
              {i18n.readStory}
            </button>
          {/if}
        </div>
      {/if}

    {:else if currentStep.type === "summary"}
      <!-- SUMMARY SCORECARD -->
      <div class="py-2">
        <p class="text-[0.7rem] font-medium tracking-wider text-[#39C5BB] uppercase">{i18n.summary}</p>
        <h2 class="mt-0.5 text-lg font-bold">{currentStep.title}</h2>

        <div class="mt-4 grid grid-cols-2 gap-2 text-xs">
          <div class="rounded-lg p-2.5 text-center {theme === 'light' ? 'bg-black/5' : 'bg-white/10'}">
            <div class="font-mono text-lg font-bold">{currentStep.dayCount}</div>
            <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.daySuffix}</div>
          </div>
          <div class="rounded-lg p-2.5 text-center {theme === 'light' ? 'bg-black/5' : 'bg-white/10'}">
            <div class="font-mono text-lg font-bold">{currentStep.stopCount}</div>
            <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.records}</div>
          </div>
          <div class="rounded-lg p-2.5 text-center {theme === 'light' ? 'bg-black/5' : 'bg-white/10'}">
            <div class="font-mono text-lg font-bold">{currentStep.placeCount}</div>
            <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.places}</div>
          </div>
          <div class="rounded-lg p-2.5 text-center {theme === 'light' ? 'bg-black/5' : 'bg-white/10'}">
            <div class="font-mono text-lg font-bold">{currentStep.photoCount}</div>
            <div class="text-[0.65rem] {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">{i18n.photos}</div>
          </div>
          {#if currentStep.distanceKm}
            <div class="col-span-2 rounded-lg p-2 text-center {theme === 'light' ? 'bg-black/5' : 'bg-white/10'}">
              <span class="font-mono text-base font-bold">{currentStep.distanceKm.toLocaleString()}</span>
              <span class="text-xs {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}"> {i18n.km}</span>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </div>

  <!-- Bottom Playback Controls Bar -->
  <footer class="flex items-center justify-between border-t px-3 py-2 text-xs {theme === 'light' ? 'border-black/10' : 'border-white/10'}">
    <!-- Prev / Play / Next -->
    <div class="flex items-center gap-1">
      <button
        type="button"
        class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent disabled:opacity-30 {theme === 'light' ? 'text-stone-600 hover:bg-black/5 hover:text-black' : 'text-stone-300 hover:bg-white/15 hover:text-white'}"
        disabled={state.currentStepIndex === 0}
        onclick={() => controller.previous()}
        aria-label={i18n.prev}
      >
        ⏮
      </button>

      <button
        type="button"
        class="flex size-9 cursor-pointer items-center justify-center rounded-full border {theme === 'light' ? 'border-[#39C5BB]/40 bg-[#39C5BB]/15 text-teal-700 hover:bg-[#39C5BB]/25' : 'border-[#39C5BB]/40 bg-[#39C5BB]/20 text-[#39C5BB] hover:bg-[#39C5BB]/30 hover:text-white'}"
        onclick={() => controller.togglePlay()}
        aria-label={state.isPlaying ? i18n.pause : i18n.play}
      >
        {state.isPlaying ? "⏸" : "▶"}
      </button>

      <button
        type="button"
        class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent disabled:opacity-30 {theme === 'light' ? 'text-stone-600 hover:bg-black/5 hover:text-black' : 'text-stone-300 hover:bg-white/15 hover:text-white'}"
        disabled={state.currentStepIndex === state.totalSteps - 1}
        onclick={() => controller.next()}
        aria-label={i18n.next}
      >
        ⏭
      </button>
    </div>

    <!-- Actions: Speed, Stage, Export, Close -->
    <div class="flex items-center gap-1">
      <button
        type="button"
        class="rounded-full border px-2 py-1 font-mono text-[0.7rem] {theme === 'light' ? 'border-black/15 text-stone-700 hover:bg-black/5' : 'border-white/15 text-stone-300 hover:bg-white/10 hover:text-white'}"
        onclick={nextSpeed}
      >
        {state.speed}×
      </button>

      {#if onOpenStage}
        <button
          type="button"
          class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent {theme === 'light' ? 'text-stone-600 hover:bg-black/5 hover:text-black' : 'text-stone-300 hover:bg-white/15 hover:text-white'}"
          onclick={onOpenStage}
          title={i18n.present}
          aria-label={i18n.present}
        >
          🎬
        </button>
      {/if}

      {#if onOpenExport}
        <button
          type="button"
          class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent {theme === 'light' ? 'text-stone-600 hover:bg-black/5 hover:text-black' : 'text-stone-300 hover:bg-white/15 hover:text-white'}"
          onclick={onOpenExport}
          title={i18n.export}
          aria-label={i18n.export}
        >
          📹
        </button>
      {/if}

      {#if onClose}
        <button
          type="button"
          class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent {theme === 'light' ? 'text-stone-500 hover:bg-black/5 hover:text-black' : 'text-stone-400 hover:bg-white/15 hover:text-white'}"
          onclick={onClose}
          title={i18n.close}
        >
          ✕
        </button>
      {/if}
    </div>
  </footer>
</aside>
