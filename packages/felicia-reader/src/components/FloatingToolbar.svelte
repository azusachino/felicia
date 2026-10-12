<script lang="ts">
  import type { Journey, Lang, Theme } from "@felicia/model"

  let {
    journeys,
    activeJourneyId,
    mode = "map",
    lang = "en",
    theme = "dark",
    onSelectJourney,
    onModeChange,
    onToggleTheme,
  }: {
    journeys: Journey[]
    activeJourneyId: string | null
    mode: "map" | "journal"
    lang: Lang
    theme?: Theme
    onSelectJourney: (id: string | null) => void
    onModeChange: (mode: "map" | "journal") => void
    onToggleTheme?: () => void
  } = $props()

  const labels = $derived(
    {
      zh: { allPlaces: "全部地点", trips: "旅程回放", map: "地图", journal: "手记", lightTheme: "浅色模式", darkTheme: "深色模式" },
      ja: { allPlaces: "すべての場所", trips: "旅の再生", map: "地図", journal: "手記", lightTheme: "ライトモード", darkTheme: "ダークモード" },
      en: { allPlaces: "All Places", trips: "Journeys", map: "Map", journal: "Journal", lightTheme: "Light Mode", darkTheme: "Dark Mode" },
    }[lang]
  )
</script>

<header class="calendar-map-toolbar pointer-events-none absolute inset-x-2 top-3 z-20 flex items-center justify-between gap-1.5 sm:inset-x-4 sm:top-4 sm:gap-2 sm:px-28">
  <!-- Left: Brand / Home Pill -->
  <div class="pointer-events-auto flex shrink-0 items-center">
    <a
      href="#top"
      class="inline-flex min-h-9 items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium shadow-lg backdrop-blur-xl transition sm:min-h-10 sm:gap-2 sm:px-3.5 {theme === 'light' ? 'border-black/10 bg-white/80 text-stone-900 hover:bg-white/95' : 'border-white/20 bg-black/60 text-white hover:bg-black/75'}"
    >
      <img src="/felicia-mark.svg" alt="" class="size-4 shrink-0 rounded-sm" />
      <span class="tracking-wide max-[30rem]:hidden">Felicia</span>
    </a>
  </div>

  <!-- Center: Journey Selector Pill -->
  <div class="pointer-events-auto relative flex min-w-0 max-w-[14rem] flex-1 items-center rounded-full border text-xs shadow-lg backdrop-blur-xl transition sm:max-w-[16rem] {theme === 'light' ? 'border-black/10 bg-white/80 text-stone-900 hover:bg-white/95' : 'border-white/20 bg-black/60 text-white hover:bg-black/75'}">
    <select
      class="min-h-9 w-full cursor-pointer appearance-none truncate rounded-full bg-transparent pr-7 pl-3 text-xs outline-none sm:min-h-10 sm:pr-8 sm:pl-3.5 {theme === 'light' ? 'text-stone-900' : 'text-white'}"
      value={activeJourneyId ?? ""}
      onchange={(e) => {
        const val = (e.target as HTMLSelectElement).value
        onSelectJourney(val ? val : null)
      }}
    >
      <option value="" class={theme === 'light' ? 'bg-white text-stone-900' : 'bg-stone-900 text-white'}>{labels.allPlaces}</option>
      {#each journeys as j}
        <option value={j.id} class={theme === 'light' ? 'bg-white text-stone-900' : 'bg-stone-900 text-white'}>
          {j.title[lang] || j.title.zh || j.title.ja}
        </option>
      {/each}
    </select>
    <span class="pointer-events-none absolute right-2.5 text-[0.65rem] sm:right-3 sm:text-xs {theme === 'light' ? 'text-stone-500' : 'text-stone-400'}">▼</span>
  </div>

  <!-- Right: Dual-Mode Switch Pill & Theme Toggle -->
  <div class="pointer-events-auto flex shrink-0 items-center gap-1.5 sm:gap-2">
    <nav class="relative isolate inline-grid grid-cols-2 items-center rounded-full border p-0.5 text-xs shadow-lg backdrop-blur-xl sm:p-1 {theme === 'light' ? 'border-black/10 bg-white/80 text-stone-600' : 'border-white/20 bg-black/60 text-stone-300'}">
      <span
        class="pointer-events-none absolute inset-y-0.5 left-0.5 z-0 w-[calc(50%-2px)] rounded-full shadow-sm transition-transform duration-300 ease-out sm:inset-y-1 sm:left-1 sm:w-[calc(50%-4px)] {theme === 'light' ? 'bg-stone-900/10' : 'bg-white/25'}"
        style:transform={mode === "journal" ? "translateX(100%)" : "translateX(0%)"}
      ></span>
      <button
        type="button"
        class="relative z-10 flex min-h-8 min-w-[3.2rem] cursor-pointer items-center justify-center rounded-full border-0 bg-transparent px-2 text-xs font-medium transition sm:min-w-[4.2rem] sm:px-3 {theme === 'light' ? 'text-stone-800 hover:text-black' : 'text-white hover:text-white'}"
        class:font-semibold={mode === "map"}
        onclick={() => onModeChange("map")}
      >
        {labels.map}
      </button>
      <button
        type="button"
        class="relative z-10 flex min-h-8 min-w-[3.2rem] cursor-pointer items-center justify-center rounded-full border-0 bg-transparent px-2 text-xs font-medium transition sm:min-w-[4.2rem] sm:px-3 {theme === 'light' ? 'text-stone-800 hover:text-black' : 'text-white hover:text-white'}"
        class:font-semibold={mode === "journal"}
        onclick={() => onModeChange("journal")}
      >
        {labels.journal}
      </button>
    </nav>

    {#if onToggleTheme}
      <button
        type="button"
        class="inline-flex size-9 items-center justify-center rounded-full border shadow-lg backdrop-blur-xl transition sm:size-10 {theme === 'light' ? 'border-black/10 bg-white/80 text-stone-800 hover:bg-white hover:text-stone-900' : 'border-white/20 bg-black/60 text-stone-200 hover:bg-black/75 hover:text-white'}"
        title={theme === "light" ? labels.darkTheme : labels.lightTheme}
        aria-label={theme === "light" ? labels.darkTheme : labels.lightTheme}
        onclick={onToggleTheme}
      >
        {#if theme === "light"}
          <!-- Moon icon for switching to dark -->
          <svg class="size-4 text-stone-700" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
          </svg>
          <!-- Sun icon for switching to light -->
          <svg class="size-4 text-[#39C5BB]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="5"></circle>
            <line x1="12" y1="1" x2="12" y2="3"></line>
            <line x1="12" y1="21" x2="12" y2="23"></line>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
            <line x1="1" y1="12" x2="3" y2="12"></line>
            <line x1="21" y1="12" x2="23" y2="12"></line>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
          </svg>
        {/if}
      </button>
    {/if}
  </div>
</header>
