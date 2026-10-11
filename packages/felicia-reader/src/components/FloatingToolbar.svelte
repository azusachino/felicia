<script lang="ts">
  import type { Journey, Lang } from "@felicia/model"

  let {
    journeys,
    activeJourneyId,
    mode = "map",
    lang = "zh",
    onSelectJourney,
    onModeChange,
  }: {
    journeys: Journey[]
    activeJourneyId: string | null
    mode: "map" | "journal"
    lang: Lang
    onSelectJourney: (id: string | null) => void
    onModeChange: (mode: "map" | "journal") => void
  } = $props()

  const labels = $derived(
    {
      zh: { allPlaces: "全部地点", trips: "旅程回放", map: "地图", journal: "手记" },
      ja: { allPlaces: "すべての場所", trips: "旅の再生", map: "地図", journal: "手記" },
      en: { allPlaces: "All Places", trips: "Journeys", map: "Map", journal: "Journal" },
    }[lang]
  )
</script>

<header class="calendar-map-toolbar pointer-events-none absolute inset-x-4 top-4 z-20 flex items-center justify-between gap-2 max-[48rem]:inset-x-2 sm:px-28">
  <!-- Left: Brand / Home Pill -->
  <div class="pointer-events-auto flex items-center">
    <a
      href="#top"
      class="inline-flex min-h-10 items-center gap-2 rounded-full border border-white/20 bg-black/60 px-3.5 py-1 text-xs font-medium text-white shadow-lg backdrop-blur-xl transition hover:bg-black/75"
    >
      <span class="inline-block size-2 rounded-full bg-orange-400"></span>
      <span class="tracking-wide">Felicia</span>
    </a>
  </div>

  <!-- Center: Journey Selector Pill -->
  <div class="pointer-events-auto relative flex max-w-[16rem] items-center rounded-full border border-white/20 bg-black/60 text-xs text-white shadow-lg backdrop-blur-xl transition hover:bg-black/75">
    <select
      class="min-h-10 w-full cursor-pointer appearance-none truncate rounded-full bg-transparent pr-8 pl-3.5 text-xs text-white outline-none"
      value={activeJourneyId ?? ""}
      onchange={(e) => {
        const val = (e.target as HTMLSelectElement).value
        onSelectJourney(val ? val : null)
      }}
    >
      <option value="" class="bg-stone-900 text-white">{labels.allPlaces}</option>
      {#if journeys.length > 0}
        <optgroup label={labels.trips} class="bg-stone-900 text-white">
          {#each journeys as j}
            <option value={j.id} class="bg-stone-900 text-white">
              {j.title[lang] || j.title.zh || j.title.ja}
            </option>
          {/each}
        </optgroup>
      {/if}
    </select>
    <span class="pointer-events-none absolute right-3 text-xs text-stone-400">▼</span>
  </div>

  <!-- Right: Dual-Mode Switch Pill (Map vs Journal) -->
  <nav class="pointer-events-auto relative isolate inline-grid grid-cols-2 items-center rounded-full border border-white/20 bg-black/60 p-1 text-xs text-stone-300 shadow-lg backdrop-blur-xl">
    <span
      class="pointer-events-none absolute inset-y-1 left-1 z-0 w-[calc(50%-4px)] rounded-full bg-white/25 shadow-sm transition-transform duration-300 ease-out"
      style:transform={mode === "journal" ? "translateX(100%)" : "translateX(0%)"}
    ></span>
    <button
      type="button"
      class="relative z-10 flex min-h-8 min-w-[4.2rem] cursor-pointer items-center justify-center rounded-full border-0 bg-transparent px-3 text-xs font-medium text-white transition hover:text-white"
      class:font-semibold={mode === "map"}
      onclick={() => onModeChange("map")}
    >
      {labels.map}
    </button>
    <button
      type="button"
      class="relative z-10 flex min-h-8 min-w-[4.2rem] cursor-pointer items-center justify-center rounded-full border-0 bg-transparent px-3 text-xs font-medium text-white transition hover:text-white"
      class:font-semibold={mode === "journal"}
      onclick={() => onModeChange("journal")}
    >
      {labels.journal}
    </button>
  </nav>
</header>
