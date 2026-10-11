<script lang="ts">
  import type { Journey, Lang, Memento } from "@felicia/model"

  let {
    journey,
    lang = "zh",
    onSelectPhoto,
  }: {
    journey: Journey
    lang: Lang
    onSelectPhoto: (memento: Memento, photoIndex: number) => void
  } = $props()

  // Group mementos by date
  const mementosByDate = $derived.by(() => {
    const groups: { date: string; dayIndex: number; mementos: Memento[] }[] = []
    for (const m of journey.mementos) {
      const d = (m.date.en || m.date.zh || m.date.ja || "").slice(0, 10)
      const last = groups.at(-1)
      if (last && last.date === d) {
        last.mementos.push(m)
      } else {
        groups.push({ date: d, dayIndex: groups.length + 1, mementos: [m] })
      }
    }
    return groups
  })
</script>

<div class="pointer-events-auto mx-auto max-w-3xl px-4 pt-24 pb-32 text-stone-100">
  <!-- Journey Header -->
  <header class="mb-12 text-center">
    <p class="text-xs font-medium tracking-widest text-orange-400 uppercase">
      {journey.place[lang] || journey.place.zh || journey.place.ja}
    </p>
    <h1 class="mt-2 text-3xl font-extrabold tracking-tight max-[48rem]:text-2xl">
      {journey.title[lang] || journey.title.zh || journey.title.ja}
    </h1>
    <p class="mt-2 text-xs text-stone-400">
      {journey.dates[lang] || journey.dates.zh || journey.dates.ja}
    </p>
  </header>

  <!-- Chronological Days Stream -->
  <div class="space-y-16">
    {#each mementosByDate as group}
      {@const weather = journey.weather?.[group.dayIndex] ?? journey.weather?.[group.date]}
      <section class="relative border-l border-white/10 pl-6 max-[48rem]:pl-4">
        <!-- Day Dot Milestone -->
        <div class="absolute -left-2 top-0 size-4 rounded-full border-2 border-stone-900 bg-orange-400"></div>

        <!-- Day Header -->
        <header class="mb-6 flex flex-wrap items-center gap-3">
          <h2 class="text-lg font-bold text-white">
            第 {group.dayIndex} 天 · {group.date}
          </h2>
          {#if weather}
            <span class="inline-flex items-center gap-1.5 rounded-full bg-white/10 px-2.5 py-0.5 text-xs text-stone-300">
              <span>☁️</span>
              <span>{weather.label}</span>
            </span>
          {/if}
        </header>

        <!-- Mementos of this day -->
        <div class="space-y-10">
          {#each group.mementos as memento}
            <article class="rounded-2xl border border-white/10 bg-white/5 p-5 shadow-lg backdrop-blur-md">
              <div class="flex items-center justify-between text-xs text-stone-400">
                <span class="font-medium text-orange-400">{memento.place[lang] || memento.place.zh}</span>
                <span>{memento.date[lang] || memento.date.zh}</span>
              </div>

              <h3 class="mt-1 text-base font-bold text-white">
                {memento.title[lang] || memento.title.zh}
              </h3>

              {#if memento.essay[lang] || memento.essay.zh}
                <p class="mt-3 text-xs leading-relaxed text-stone-300 whitespace-pre-line">
                  {memento.essay[lang] || memento.essay.zh}
                </p>
              {/if}

              <!-- Photo Gallery Grid -->
              {#if memento.photos.length > 0}
                <div class="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-3">
                  {#each memento.photos as photo, pIdx}
                    <button
                      type="button"
                      class="group relative aspect-[4/3] cursor-zoom-in overflow-hidden rounded-xl border border-white/10 bg-stone-900 p-0 text-left transition hover:scale-[1.02]"
                      onclick={() => onSelectPhoto(memento, pIdx)}
                    >
                      <img
                        src={photo.src}
                        alt=""
                        loading="lazy"
                        class="size-full object-cover transition duration-300 group-hover:opacity-90"
                      />
                      {#if photo.video}
                        <span class="absolute top-2 left-2 flex items-center gap-1 rounded-full bg-black/60 px-2 py-0.5 text-[0.625rem] font-medium text-white backdrop-blur">
                          <span class="size-1.5 rounded-full bg-orange-400"></span> LIVE
                        </span>
                      {/if}
                    </button>
                  {/each}
                </div>
              {/if}
            </article>
          {/each}
        </div>
      </section>
    {/each}
  </div>
</div>
