# Research — tabi design reset

> 2026-08-20. `tabi` (旅) — a cartoon/game cat that runs a journey's route and cuddles at each
> memento stop — went through a full build (D3 animation, MapLibre basemap, cuddle patterns,
> a return leg) reactively: implement a guess, screenshot it, get told it's wrong, implement
> the next guess. The mechanics work and are documented (`docs/tutorials/d3/` and
> `docs/tutorials/maplibre/` in harus-workstation). The _design_ never got decided on paper —
> it got discovered by trial. This note is the stop: the code is parked at
> `packages/felicia-shared/src/theme-ui/_incubating/tabi/`, unregistered, and this is the
> diagnosis plus the open questions to settle before it's touched again.

## The actual goal (learned after the first draft of this note)

The point of `tabi` isn't "ship a cute default cat" — it's to become capable of **3D modeling
it yourself**, with the animation engine driving a model _you_ authored rather than one Claude
picked. That reframes the character question entirely: it's not "which reference does Claude
like," it's "what does a route-driven character pipeline need to look like so a
self-authored 3D model can be dropped in and animated by it." This is the strongest argument
for the 3D/Three.js fork below — 2D pixel-sprite art doesn't give you anything to learn
_modeling_ on. Settled: starting from zero (no prior 3D-modeling background), and modeling
itself is a taught skill here, the same tutorial-lesson practice `docs/tutorials/d3/` already
uses -- see `docs/tutorials/blender/` in harus-workstation (scaffolded, first lesson lands
once that track actually starts). Tool not yet picked, but Blender is the obvious
free/standard default worth confirming rather than assuming.

## The diagnosis (what the reactive pass got wrong)

1. **"Cartoon" was implemented before it was defined.** The first attempt at a cartoon
   basemap was a CSS `saturate()`/`contrast()` filter — a surface-level knob reached for
   because it was the nearest available lever, not because anyone had established what
   "cartoon" needed structurally. It produced no visible change. The real technique (strip
   real MapLibre style layers, flatten fills, add ink-line outlines) only got found after
   being asked directly and admitting the first attempt didn't understand the word.
2. **The character design moved through three unrelated languages with no throughline**:
   an emoji placeholder → a soft flat-vector SVG cat (rounded shapes, white outline) → a
   sharp-cornered pixel-sprite rebuild (dark outline, `curveStepAfter` tail), the last change
   only happening after an actual reference (`SUPER NYANCO RUN`) was found and looked at.
   Nothing before that reference was grounded in anything concrete.
3. **The UI chrome drifted from the rest of the theme and nobody noticed until asked to
   audit it.** The map and cat moved to a bold pixel-sprite look; the play button, journey
   picker, and filter toggle stayed on the original soft-glass pill styling three redesigns
   later. A design language that isn't checked against every surface isn't actually a
   language yet, it's a pattern applied to whatever was touched most recently.
4. **Palette and pacing constants were each tuned in isolation** (the establishing/homecoming
   legs ran on a flat 6-second budget while local hops scaled with distance; the map's
   land/water/park colors were picked per-surface, not from one stated system) rather than
   set once from a single, stated direction.

None of this is a claim that the _mechanics_ are wrong — `d3-timer`/`d3-interpolate`/
`d3-ease` sequencing, `d3-geo` for real distance, the MapLibre `setPaintProperty`/
`setLayoutProperty` runtime-restyle technique are all correct and stay correct regardless of
which visual direction wins. What's missing is a design decided before code, not one
discovered by shipping guesses and waiting for correction.

## References gathered (concrete, not vibes)

- **[daniwell / aidn.jp](https://aidn.jp/contents/)** — a curated gallery of Japanese web/game
  animation work. The most relevant piece found is a green rabbit-eared character in a
  3D/voxel-ish style: chunky proportions, flat lit surfaces, real depth. Bold typography and
  high color contrast run through the whole gallery.
- **[SUPER NYANCO RUN](https://aidn.jp/snr/)** — an actual pixel-art cat-runner game found in
  that same gallery. Flat 2D pixel sprite (not 3D), thick black outlines, a bold rounded pixel
  font, solid-white UI buttons with a black border and a hard offset shadow (no blur, no
  transparency). This is the closest existing reference to "a cat running a route."
- **[taste-skill](https://github.com/Leonxlnx/taste-skill)** — an AI-agent design-discipline
  skill (built for React/Tailwind marketing sites, but its process generalizes). The two
  transferable pieces: state a one-line **Design Read** ("reading this as: X for Y, leaning
  toward Z") _before_ generating anything, and once a palette/language is picked, audit every
  surface against it rather than only the one just touched.

These two game references point in genuinely different technical directions (2D flat sprite
vs. 3D voxel character) — see the open fork below.

## Open forks — not yet settled

Unlike a normal research note, these are deliberately left open rather than resolved on paper
here — the point of this reset is to decide them _with_ a human before more code, not have an
agent decide them alone again.

1. **Rendering technology: stay 2D (SVG + D3), or move to 3D (Three.js via a MapLibre
   `CustomLayerInterface`)?** The Nyanco Run reference argues 2D; the daniwell rabbit argues
   3D; the actual goal above (learning to model the character yourself) argues 3D more
   directly than either reference does — there's no "modeling" skill to build in flat SVG
   shapes. These aren't the same amount of work either way — 3D means a new rendering pipeline
   and rethinking most of the D3-overlay lessons for a scene graph instead of SVG attributes,
   not a coat of paint on the current approach. If 3D wins, the architecture worth keeping
   from this pass is the _separation already built_: `catAt`/`mode`/gait-pose are plain data
   (lng/lat, an enum, a few numbers), and `positionCat()` is the only place that turns that
   data into a drawn frame. That seam is what makes "swap the renderer, keep the engine"
   possible — a self-authored glTF/GLB model would consume the same position/pose data through
   a Three.js-flavored version of that one function, not a rewrite of the sequencer, the
   timers, or the `d3-geo` distance math.

   If 3D wins, the _which library_ question still needs its own answer. A first comparison
   pass named Phaser 4 (2D engine), Excalibur.js (2D engine), PixiJS (2D renderer), Babylon.js
   (3D engine, "batteries-included"), PlayCanvas (3D engine + editor), Three.js (3D renderer,
   "you build the engine layer"), and Cocos Creator (2D/3D engine) as options. That comparison
   was generic, though — it doesn't weigh the one constraint this project actually has: the
   character has to render as a supplementary layer _inside MapLibre's own WebGL canvas_
   (`CustomLayerInterface`), not own the page or its own runtime loop. That rules out the
   full-engine options (Phaser, Cocos, PlayCanvas's own editor/runtime) by construction, not
   preference — they're built to own the canvas and the game loop themselves. Three.js and
   Babylon.js both work embedded as a library inside someone else's render loop; between those
   two, "batteries-included" cuts both ways for a from-zero learner — more built-in tooling to
   lean on, but also more to learn before the first triangle renders. Not decided here.

2. **Palette.** Not locked. Whatever the map/cat/chrome share, it needs to be stated once and
   checked everywhere, not picked per-component.
3. **Character design.** Proportions, pose count (continuous rig vs. discrete sprite frames
   like a real pixel-art runner would use), and outline treatment are all open.
4. **Map treatment.** The _technique_ (strip real style layers, flat recolor, add outline
   layers) is proven and worth keeping regardless of which palette wins — see
   `docs/tutorials/maplibre/00-a-real-map-isnt-the-only-map-a-vector-style-can-draw.md` in
   harus-workstation. The specific colors applied through it are not locked.
5. **UI chrome language.** The pixel-bordered/hard-shadow treatment built in the reactive pass
   is a candidate, not a decision — it should be judged against whatever palette/character
   direction gets picked, not kept by default because it's already written.

## Process boundaries, going forward

- **Always** state a one-line Design Read before implementing a visual decision — what this
  is, for whom, leaning toward which concrete reference — the same discipline taste-skill
  names for its own (different) domain.
- **Always** verify a visual change by actually looking at it (`agent-browser` screenshot),
  not by describing what it should look like.
- **Ask first** before locking a palette, character design, or chrome language — present it as
  a proposal grounded in a named reference, not a shipped fait accompli waiting for correction.
- **Never** implement a shallow stand-in (a CSS filter, a generic pastel default) for a design
  word that hasn't actually been defined yet. If the word is ambiguous, find a concrete
  reference or ask, before writing the code that guesses at it.

## Next step

Not implementation. The next session on `tabi` should settle the open forks above — starting
with the 2D/3D technology fork, since it decides how much of the rest is even reusable — before
any file under `_incubating/tabi/` gets touched or re-registered.
