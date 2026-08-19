<script lang="ts">
  import { curveBumpX, line } from "d3-shape"
  import type { Lang, Memento } from "../data"
  import { ticketVariantFor, type TicketVariant } from "./tickets/registry"

  let { memento, lang }: { memento: Memento; lang: Lang } = $props()

  const variantLabels: Record<TicketVariant, { ja: string; en: string; zh: string }> = {
    rail: { ja: "鉄道", en: "Rail", zh: "铁路" },
    metro: { ja: "地下鉄", en: "Metro", zh: "地铁" },
    live: { ja: "ライブ", en: "Live", zh: "演出" },
    mountain: { ja: "山岳交通", en: "Mountain", zh: "山岳交通" },
    admission: { ja: "入場券", en: "Admission", zh: "入场券" },
  }
  const routePoints: Record<TicketVariant, [number, number][]> = {
    rail: [
      [12, 54],
      [88, 14],
      [160, 44],
      [236, 12],
      [308, 42],
    ],
    metro: [
      [12, 16],
      [76, 52],
      [150, 18],
      [226, 48],
      [308, 22],
    ],
    live: [
      [12, 44],
      [84, 44],
      [152, 18],
      [228, 44],
      [308, 44],
    ],
    mountain: [
      [12, 54],
      [78, 48],
      [150, 20],
      [224, 26],
      [308, 10],
    ],
    admission: [
      [12, 42],
      [88, 22],
      [160, 44],
      [236, 22],
      [308, 42],
    ],
  }

  const t = (value: { ja: string; en: string; zh: string }) => value[lang]
  const variant = $derived(ticketVariantFor(memento))
  const routePath = $derived(line<[number, number]>().curve(curveBumpX)(routePoints[variant]))

  function text(key: string): string {
    const value = memento.kindData?.[key]
    return typeof value === "string" ? value : ""
  }

  function name(key: string): string {
    const value = memento.kindData?.[key]
    if (typeof value === "string") return value
    if (typeof value === "object" && value !== null && !Array.isArray(value)) {
      const valueName = (value as Record<string, unknown>).name
      return typeof valueName === "string" ? valueName : ""
    }
    return ""
  }

  function displayPrice(): string {
    if (memento.price) return memento.price.replace(/^JPY\s+/, "￥")
    const value = memento.kindData?.price
    if (typeof value !== "object" || value === null || Array.isArray(value)) return ""
    const amount = (value as Record<string, unknown>).amount
    const currency = (value as Record<string, unknown>).currency
    if (typeof amount !== "number") return ""
    if (currency === "JPY") return `￥${amount.toLocaleString("en-US")}`
    return `${typeof currency === "string" ? currency : ""} ${amount.toLocaleString("en-US")}`.trim()
  }

  function stationName(station: { name: string; ja: string }): string {
    return lang === "en" ? station.name : station.ja
  }
</script>

<article class="ticket-template ticket-{variant}">
  <header class="ticket-header">
    <span>{t(variantLabels[variant])}</span>
    <span>{t(memento.date)}</span>
  </header>

  <div class="ticket-body">
    {#if memento.kind === "transit" && memento.transit}
      <div class="ticket-copy">
        <span class="ticket-kind">{text("operator") || t(memento.transit.operator)}</span>
        <strong>{text("line") || t(memento.transit.line)}</strong>
        <span>{stationName(memento.transit.from)} → {stationName(memento.transit.to)}</span>
      </div>
      <div class="ticket-price">
        <span>{t(memento.title)}</span>
        <strong>{memento.transit.fare || displayPrice() || "—"}</strong>
      </div>
    {:else if variant === "live"}
      <div class="ticket-copy">
        <span class="ticket-kind">{text("ticket_type") || t(variantLabels[variant])}</span>
        <strong>{text("artist") || name("name") || t(memento.title)}</strong>
        <span>{name("venue") || t(memento.place)}</span>
      </div>
      <div class="ticket-price">
        <span>{text("seat") || text("gate") || t(memento.date)}</span>
        <strong>{displayPrice() || "—"}</strong>
      </div>
    {:else}
      <div class="ticket-copy">
        <span class="ticket-kind">{text("ticket_type") || t(variantLabels[variant])}</span>
        <strong>{name("name") || t(memento.title)}</strong>
        <span>{name("venue") || t(memento.place)}</span>
      </div>
      <div class="ticket-price">
        <span>{t(memento.title)}</span>
        <strong>{displayPrice() || "—"}</strong>
      </div>
    {/if}

    {#if routePath}
      <svg class="ticket-route" viewBox="0 0 320 64" aria-hidden="true">
        <path d={routePath} />
      </svg>
    {/if}
  </div>

  {#if text("source_url")}
    <a class="ticket-source" href={text("source_url")} target="_blank" rel="noopener noreferrer">official source ↗</a>
  {/if}
</article>

<style>
  .ticket-template {
    position: relative;
    overflow: hidden;
    min-height: 13rem;
    padding: 1.35rem 2.1rem 1.35rem 1.35rem;
    border-radius: 0.35rem;
    color: #241203;
    background: linear-gradient(135deg, #f8e7bb, #eeb46a);
    box-shadow: 0 0.8rem 1.5rem rgb(0 0 0 / 22%);
  }

  .ticket-template::after {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: 1.1rem;
    background: radial-gradient(circle at 0.55rem 0.15rem, transparent 0 0.22rem, currentColor 0.24rem 0.28rem, transparent 0.3rem) 0 0 / 1.1rem 0.62rem;
    content: "";
    opacity: 0.32;
    pointer-events: none;
  }

  .ticket-metro {
    background: linear-gradient(135deg, #d9effa, #8bc5d9);
  }
  .ticket-live {
    background: linear-gradient(135deg, #f2d8e8, #d985a8);
  }
  .ticket-mountain {
    background: linear-gradient(135deg, #dce8c6, #9ebb83);
  }
  .ticket-admission {
    background: linear-gradient(135deg, #f8e7bb, #eeb46a);
  }

  .ticket-header,
  .ticket-price {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.68rem;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .ticket-header {
    border-bottom: 1px solid rgb(36 18 3 / 25%);
    padding-bottom: 0.65rem;
  }

  .ticket-body {
    position: relative;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 1rem;
    align-items: end;
    min-height: 8.4rem;
    padding: 1rem 0;
  }

  .ticket-copy {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    max-width: 15rem;
  }

  .ticket-kind {
    font-size: 0.7rem;
    font-weight: 800;
    letter-spacing: 0.06em;
  }
  .ticket-copy strong {
    font-size: clamp(1.35rem, 4vw, 2rem);
    line-height: 1.05;
  }
  .ticket-copy > span:last-child {
    font-size: 0.82rem;
    opacity: 0.72;
  }

  .ticket-route {
    position: absolute;
    right: -0.4rem;
    bottom: 0.5rem;
    width: 74%;
    opacity: 0.6;
  }
  .ticket-route path {
    fill: none;
    stroke: #9f5a22;
    stroke-width: 2.5;
    stroke-dasharray: 8 7;
    animation: ticket-route 2.8s linear infinite;
  }

  .ticket-price {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: end;
    gap: 0.25rem;
    border-left: 1px dashed rgb(36 18 3 / 38%);
    padding-left: 1rem;
    text-align: right;
  }

  .ticket-price span {
    max-width: 8rem;
    font-size: 0.65rem;
    line-height: 1.3;
  }
  .ticket-price strong {
    font-size: 1.25rem;
    white-space: nowrap;
  }
  .ticket-source {
    position: relative;
    z-index: 1;
    color: inherit;
    font-size: 0.68rem;
    font-weight: 700;
  }

  @keyframes ticket-route {
    to {
      stroke-dashoffset: -30;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .ticket-route path {
      animation: none;
    }
  }
</style>
