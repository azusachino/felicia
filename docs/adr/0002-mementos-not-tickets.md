# ADR 0002: Mementos as the Core Unit

* **Status:** Accepted
* **Date:** 2026-06-14
* **Decisions:** `felicia:decision:memento-not-ticket`

## Context
Initial design drafts centered on scanning and displaying physical transit/admission tickets (a "Ticket-Era" model). However, physical tickets are rapidly disappearing in Japan and globally, replaced by IC cards, digital QR codes, and headless apps. Furthermore, travel memories are often anchored to items that are not tickets, such as shop receipts, local stamps (*eki* stamps, *goshuin*), character goods, or general souvenirs.

## Decision
We decided to generalize the core unit of the map journal from `Ticket` to `Memento`. A memento represents any physical or digital object anchoring a memory.

Rules for mementos:
1. **Template-first stub rendering:** Rather than scanning a physical ticket, the visual card (the "stub") is rendered dynamically on the frontend using structured data (`kind_data` in JSON).
2. **Kinds classification:** Mementos are classified by `kind` (e.g. `transit`, `goods`, `stamp`, `receipt`, `souvenir`, `ticket`).
3. **Structured click targets:** Clicking a memento stub opens a detailed essay (Markdown) and a curated photo gallery.
4. **Physical Photos as secondary:** If a physical photo of a souvenir exists, it is displayed in the gallery, but the primary map-marker/click-target is the structured memento stub itself.

## Consequences
* The UI gains high customizability; we can render highly thematic, authentic-looking Japanese train tickets, stamp books, or receipt layouts dynamically from simple JSON properties.
* Reduces authoring friction: the user doesn't need to scan/upload pristine ticket images; they only input simple fields (e.g. operators, stations, price) into a form.
* The schema must support a single, flexible table that can hold diverse memento kinds without table-join sprawl (see ADR 0006).
