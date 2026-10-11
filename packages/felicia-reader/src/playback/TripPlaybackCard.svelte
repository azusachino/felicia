<script lang="ts">
  import type { Lang } from "@felicia/model"
  import type { PlaybackController, PlaybackStep } from "./engine"

  let {
    controller,
    currentStep,
    lang = "zh",
    onOpenStage,
    onOpenExport,
    onOpenDetail,
    onClose,
  }: {
    controller: PlaybackController
    currentStep: PlaybackStep
    lang: Lang
    onOpenStage?: () => void
    onOpenExport?: () => void
    onOpenDetail?: (mementoId: string) => void
    onClose?: () => void
  } = $props()

  let state = $state(controller.getState())

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
        dayPrefix: "第",
        daySuffix: "天",
        records: "条记录",
        places: "个地点",
        photos: "张照片",
        km: "公里",
        hours: "小时车程",
        present: "放映",
        export: "导出视频",
        prev: "上一站",
        next: "下一站",
        play: "播放",
        pause: "暂停",
        close: "退出回放",
      },
      ja: {
        intro: "旅の再生",
        summary: "旅の振り返り",
        dayPrefix: "第",
        daySuffix: "日",
        records: "件の記録",
        places: "ヶ所",
        photos: "枚の写真",
        km: "キロ",
        hours: "時間の行程",
        present: "上映",
        export: "書き出し",
        prev: "前へ",
        next: "次へ",
        play: "再生",
        pause: "一時停止",
        close: "閉じる",
      },
      en: {
        intro: "Trip Playback",
        summary: "Trip Summary",
        dayPrefix: "Day",
        daySuffix: "",
        records: "records",
        places: "places",
        photos: "photos",
        km: "km",
        hours: "driving hrs",
        present: "Present",
        export: "Export",
        prev: "Previous",
        next: "Next",
        play: "Play",
        pause: "Pause",
        close: "Close",
      },
    }[lang]
  )
</script>

<aside class="calendar-map-trip pointer-events-auto absolute bottom-4 left-4 z-20 flex max-h-[calc(100vh-6rem)] w-[min(23rem,calc(100vw-2rem))] flex-col overflow-hidden rounded-2xl border border-white/20 bg-black/75 text-white shadow-2xl backdrop-blur-2xl transition max-[48rem]:inset-x-3 max-[48rem]:bottom-4 max-[48rem]:w-auto">
  <!-- Top Segmented Progress Bar -->
  <ol class="flex shrink-0 list-none gap-1 px-4 pt-3.5" aria-label="Trip Progress">
    {#each Array(state.totalSteps) as _, i}
      <li class="flex-1">
        <button
          type="button"
          class="h-1 w-full cursor-pointer rounded-full border-0 p-0 transition-colors duration-300 {i <= state.currentStepIndex ? 'bg-orange-400' : 'bg-white/20'}"
          onclick={() => controller.seek(i)}
          aria-label="Step {i + 1}"
        ></button>
      </li>
    {/each}
  </ol>

  <!-- Step Content -->
  <div class="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 pt-2.5 pb-3">
    {#if currentStep.type === "intro"}
      <!-- INTRO CARD -->
      <div class="py-1">
        <p class="text-[0.7rem] font-medium tracking-wider text-orange-400 uppercase">{i18n.intro}</p>
        <h2 class="mt-0.5 truncate text-lg font-bold">{currentStep.title}</h2>
        <p class="text-xs text-stone-400">{currentStep.dateRange} · {currentStep.dayCount} {i18n.daySuffix} · {currentStep.stopCount} {i18n.records}</p>

        <!-- 3 Tilted Polaroid Cover Fan -->
        {#if currentStep.covers.length > 0}
          <div class="relative my-4 flex h-28 items-center justify-center">
            {#each currentStep.covers.slice(0, 3) as cover, idx}
              <div
                class="absolute size-24 overflow-hidden rounded-lg border-2 border-white/90 bg-stone-900 shadow-xl transition-transform duration-300"
                style:transform={idx === 0 ? "translateX(-2.2rem) rotate(-7deg)" : idx === 2 ? "translateX(2.2rem) rotate(7deg)" : "rotate(0deg) scale(1.05)"}
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
              <div class="rounded-lg bg-white/10 p-2 text-center">
                <div class="font-mono text-base font-semibold">{currentStep.distanceKm.toLocaleString()}</div>
                <div class="text-[0.65rem] text-stone-400">{i18n.km}</div>
              </div>
            {/if}
            {#if currentStep.drivingHours}
              <div class="rounded-lg bg-white/10 p-2 text-center">
                <div class="font-mono text-base font-semibold">{currentStep.drivingHours}</div>
                <div class="text-[0.65rem] text-stone-400">{i18n.hours}</div>
              </div>
            {/if}
          </div>
        {/if}
      </div>

    {:else if currentStep.type === "day"}
      <!-- DAY BANNER CARD -->
      <div class="py-3">
        <p class="text-[0.7rem] font-medium tracking-wider text-orange-400 uppercase">
          {i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}
        </p>
        <h2 class="mt-0.5 text-lg font-bold">{currentStep.date}</h2>

        {#if currentStep.weather}
          <div class="mt-2 inline-flex items-center gap-1.5 rounded-full bg-white/10 px-2.5 py-1 text-xs text-stone-300">
            <span>☁️</span>
            <span>{currentStep.weather.label}</span>
          </div>
        {/if}

        {#if currentStep.places.length > 0}
          <div class="mt-3 text-xs text-stone-400">
            <span class="font-medium text-stone-300">{currentStep.places.join(" → ")}</span>
          </div>
        {/if}
      </div>

    {:else if currentStep.type === "stop"}
      <!-- STOP CARD -->
      <div class="py-1">
        <div class="flex items-center justify-between text-xs text-stone-400">
          <span class="text-orange-400">{i18n.dayPrefix} {currentStep.day} {i18n.daySuffix}</span>
          <span class="rounded bg-white/10 px-1.5 py-0.5">{currentStep.memento.place[lang] || currentStep.memento.place.zh}</span>
        </div>
        <h2 class="mt-1 text-base font-bold">
          {currentStep.memento.title[lang] || currentStep.memento.title.zh}
        </h2>

        <!-- Photo / Live Photo Polaroid Frame -->
        {#if currentStep.memento.photos.length > 0}
          {@const photo = currentStep.memento.photos[0]}
          <div class="relative my-3 overflow-hidden rounded-xl border border-white/20 bg-stone-900 shadow-md">
            {#if photo.video}
              <video
                src={photo.video}
                poster={photo.src}
                autoplay
                loop
                muted
                playsinline
                class="aspect-[4/3] w-full object-cover"
              ></video>
              <span class="absolute top-2 left-2 flex items-center gap-1 rounded-full bg-black/60 px-2 py-0.5 text-[0.625rem] font-medium text-white backdrop-blur">
                <span class="size-2 rounded-full bg-orange-400"></span> LIVE
              </span>
            {:else}
              <img src={photo.src} alt="" class="aspect-[4/3] w-full object-cover" />
            {/if}
          </div>
        {/if}

        {#if currentStep.memento.essay[lang] || currentStep.memento.essay.zh}
          <p class="mt-1 line-clamp-3 text-xs leading-relaxed text-stone-300">
            {currentStep.memento.essay[lang] || currentStep.memento.essay.zh}
          </p>
        {/if}

        {#if onOpenDetail}
          <button
            type="button"
            class="mt-2 text-xs text-orange-400 underline underline-offset-4 hover:text-orange-300"
            onclick={() => onOpenDetail?.(currentStep.memento.id)}
          >
            阅读完整手记 →
          </button>
        {/if}
      </div>

    {:else if currentStep.type === "summary"}
      <!-- SUMMARY SCORECARD -->
      <div class="py-2">
        <p class="text-[0.7rem] font-medium tracking-wider text-orange-400 uppercase">{i18n.summary}</p>
        <h2 class="mt-0.5 text-lg font-bold">{currentStep.title}</h2>

        <div class="mt-4 grid grid-cols-2 gap-2 text-xs">
          <div class="rounded-lg bg-white/10 p-2.5 text-center">
            <div class="font-mono text-lg font-bold">{currentStep.dayCount}</div>
            <div class="text-[0.65rem] text-stone-400">{i18n.daySuffix}</div>
          </div>
          <div class="rounded-lg bg-white/10 p-2.5 text-center">
            <div class="font-mono text-lg font-bold">{currentStep.stopCount}</div>
            <div class="text-[0.65rem] text-stone-400">{i18n.records}</div>
          </div>
          <div class="rounded-lg bg-white/10 p-2.5 text-center">
            <div class="font-mono text-lg font-bold">{currentStep.placeCount}</div>
            <div class="text-[0.65rem] text-stone-400">{i18n.places}</div>
          </div>
          <div class="rounded-lg bg-white/10 p-2.5 text-center">
            <div class="font-mono text-lg font-bold">{currentStep.photoCount}</div>
            <div class="text-[0.65rem] text-stone-400">{i18n.photos}</div>
          </div>
          {#if currentStep.distanceKm}
            <div class="col-span-2 rounded-lg bg-white/10 p-2 text-center">
              <span class="font-mono text-base font-bold">{currentStep.distanceKm.toLocaleString()}</span>
              <span class="text-xs text-stone-400"> {i18n.km}</span>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </div>

  <!-- Bottom Playback Controls Bar -->
  <footer class="flex items-center justify-between border-t border-white/10 px-3 py-2 text-xs">
    <!-- Prev / Play / Next -->
    <div class="flex items-center gap-1">
      <button
        type="button"
        class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent text-stone-300 hover:bg-white/15 hover:text-white disabled:opacity-30"
        disabled={state.currentStepIndex === 0}
        onclick={() => controller.previous()}
        aria-label={i18n.prev}
      >
        ⏮
      </button>

      <button
        type="button"
        class="flex size-9 cursor-pointer items-center justify-center rounded-full border border-orange-400/40 bg-orange-500/20 text-orange-300 hover:bg-orange-500/30 hover:text-white"
        onclick={() => controller.togglePlay()}
        aria-label={state.isPlaying ? i18n.pause : i18n.play}
      >
        {state.isPlaying ? "⏸" : "▶"}
      </button>

      <button
        type="button"
        class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent text-stone-300 hover:bg-white/15 hover:text-white disabled:opacity-30"
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
        class="rounded-full border border-white/15 px-2 py-1 font-mono text-[0.7rem] text-stone-300 hover:bg-white/10 hover:text-white"
        onclick={nextSpeed}
      >
        {state.speed}×
      </button>

      {#if onOpenStage}
        <button
          type="button"
          class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent text-stone-300 hover:bg-white/15 hover:text-white"
          onclick={onOpenStage}
          title={i18n.present}
        >
          🎬
        </button>
      {/if}

      {#if onOpenExport}
        <button
          type="button"
          class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent text-stone-300 hover:bg-white/15 hover:text-white"
          onclick={onOpenExport}
          title={i18n.export}
        >
          📼
        </button>
      {/if}

      {#if onClose}
        <button
          type="button"
          class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent text-stone-400 hover:bg-white/15 hover:text-white"
          onclick={onClose}
          title={i18n.close}
        >
          ✕
        </button>
      {/if}
    </div>
  </footer>
</aside>
