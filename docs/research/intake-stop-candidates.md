---
title: "Intake stop candidates and adversarial experiments"
status: "proposed"
date: "2026-07-18"
---

# Intake stop candidates and adversarial experiments

This note extends the unified intake contract in ADR 0022 and the CLI compiler
contract. It tests the proposed offline flow against messy source material,
without turning guessed places into authored stories.

## Working model

```text
GPX / Dawarich tracks + visits / Immich or local photos
    -> canonical observations
    -> derived visits and stop candidates
    -> review decisions
    -> memento candidates
    -> authored mementos
    -> static publication
```

The important boundaries are:

- A **visit** is derived evidence about a stay at a place. Dawarich provides
  visits when connected; a local GPX fallback derives the same shape.
- A **stop candidate** is a reviewable draft grouping of route and media
  evidence. It is not a public `place` table and it is not automatically a
  memento.
- A **memento candidate** proposes a possible story and template kind.
- A **memento** is authored content and may be published only after review.

One stop may produce zero, one, or several mementos. A five-day Kansai trip may
therefore contain Osaka, Kobe, and Nara stop candidates while only publishing a
train ticket, a Nara Park story, and a receipt.

## User stories

### US-01 — Plan a local journey without mutation

As an author, I can provide a GPX file and photo directory and receive a
read-only plan before Felicia writes to the database or public site.

Acceptance criteria:

- The plan reports route dates, bounds, segments, point counts, and photo counts.
- It identifies the capabilities used: Dawarich visits, GPX clustering, Immich
  metadata, filesystem metadata, or sidecar data.
- Running the same inputs twice produces the same candidate IDs and plan hash.
- Invalid inputs produce diagnostics and do not create partial authored records.

### US-02 — Review evidence-backed stop candidates

As an author, I can inspect why Felicia proposed Osaka, Kobe, or Nara as stops
before deciding whether they matter to my journey.

Acceptance criteria:

- Each candidate shows its time window, coordinate, confidence, and evidence.
- Evidence can include route intervals, dwell duration, and matched photo IDs.
- I can keep, ignore, rename, merge, split, or reorder candidates.
- An ignored candidate remains ignored after a re-import of the same source.

### US-03 — Handle incomplete or ambiguous media honestly

As an author, I can import photos that lack EXIF or timestamps without Felicia
pretending to know where or when they belong.

Acceptance criteria:

- Photos are classified as confidently matched, weakly matched, unattached, or
  invalid.
- Filesystem timestamps are marked as low-confidence evidence.
- A JSONL sidecar can assign a timestamp, coordinate, stop, or memento kind.
- Unattached photos remain reviewable and are never silently published.

### US-04 — Construct one or more mementos from a stop

As an author, I can turn a kept stop into a draft memento, or create multiple
mementos from one stop without making every stop into a story.

Acceptance criteria:

- A kept stop can produce zero, one, or several mementos.
- Each memento has an explicit kind and selected media.
- A memento retains its stop and source evidence links.
- A draft can be saved with incomplete prose and completed later in an editor or
  the web authoring UI.

### US-05 — Use an agent without surrendering authorship

As an author, I can ask my local agent to suggest titles, kinds, photo groups,
and structured fields while retaining explicit control over every change.

Acceptance criteria:

- Agent output is a suggestion or patch, never an implicit database mutation.
- Suggestions include their evidence, confidence, provider, and model/version
  when applicable.
- I can inspect a diff, accept selected fields, reject suggestions, or edit them.
- Authored fields are protected from later source re-imports.

### US-06 — Publish only the reviewed story

As an author, I can preview and compile a public journey without exposing raw
tracks, original EXIF, private candidates, or unattached media.

Acceptance criteria:

- Only explicitly selected mementos and media enter the static artifact.
- The public route is sanitized according to the privacy policy.
- The build report lists included and excluded records and media.
- A failed privacy or validation check blocks publication.

## Adversarial cases

| Case                                           | Expected result                                                              |
| ---------------------------------------------- | ---------------------------------------------------------------------------- |
| Very large GPX                                 | Stream parse; bound memory; simplify per segment; retain raw input privately |
| GPX with no timestamps                         | Use route geometry only; do not time-join photos automatically               |
| GPX with sparse points                         | Preserve a partial route; produce low-confidence or no stops                 |
| Wrong GPX                                      | Show date/bounds/jump diagnostics; require explicit confirmation             |
| Multiple trips in one GPX                      | Detect long time gaps/geographic jumps; propose split boundaries             |
| GPX crossing midnight or DST                   | Preserve source timezone and show inferred-day warnings                      |
| Photos with complete EXIF                      | Match by time and location with confidence and evidence                      |
| Photos with time but no location               | Snap to route/stop only within a temporal threshold                          |
| Photos with no EXIF                            | Keep unattached; support a user sidecar; never invent a location             |
| Filesystem timestamps only                     | Use only as low-confidence fallback and disclose it                          |
| Duplicate imports                              | Reconcile by source identity and content hash                                |
| Dawarich and GPX disagree                      | Prefer configured source; retain conflict evidence; do not merge silently    |
| Immich unavailable                             | Continue from local media or fail only the connected capability              |
| Local media contains unsupported/private files | Report them and exclude them from public output                              |
| Stop has many photos but no dwell              | Do not equate photo density with attraction truth                            |
| Long dwell with no photos                      | Propose a stop, but do not create a memento automatically                    |
| One stop deserves multiple stories             | Permit multiple mementos linked to one stop                                  |
| Candidate is later ignored                     | Preserve the decision so re-import does not resurrect it                     |

## Data and confidence rules

Matching should use this order:

```text
EXIF timestamp + GPS
    -> EXIF timestamp + route/visit snap
    -> user sidecar metadata
    -> filesystem timestamp, low confidence
    -> unattached media
```

Every derived result should retain evidence and warnings. A missing value lowers
confidence; invalid input creates an issue; neither becomes authored content.

The review output should include counts such as:

```text
128 photos discovered
 94 matched confidently
 21 matched by time only
  9 unattached
  4 invalid or unsupported

12 stop candidates
  7 proposed
  3 low-confidence
  2 unresolved
```

## Persistent experiment layout

Raw data and generated artifacts stay outside the committed source tree:

```text
.felicia/experiments/<case>/
├── input/       # GPX, photos, sidecars, or provider snapshots
├── plan.json    # normalized plan and candidate evidence
├── report.json  # warnings, counts, and decisions
└── site/        # optional compiled publication
```

The repository should commit only small synthetic fixtures, fixture metadata,
and expected summaries. Real personal media and large public downloads remain
ignored. Provider experiments should use local mocks first, then a disposable
Podman Compose Dawarich/Immich stack when adapter behavior needs verification.

## Experiment sequence

1. **Synthetic baseline** — one timestamped route, three photo classes, one
   clear stop, and one unattached photo.
2. **No-metadata case** — photos without EXIF, then the same case with a JSONL
   sidecar; verify only the sidecar changes the match.
3. **Bad-track case** — out-of-order timestamps, impossible-speed jumps, sparse
   points, and a second trip; verify diagnostics and bounded output.
4. **Large-track case** — generated high-point GPX; measure peak memory, runtime,
   route simplification, and deterministic output.
5. **Provider parity** — replay equivalent Dawarich, Immich, GPX, and local-photo
   inputs; verify they produce the same canonical route/media/visit shapes.
6. **Re-import case** — change source-owned metadata after authoring title/essay
   and photo order; verify authored fields survive and ignored candidates stay
   ignored.
7. **Publication privacy** — compile a draft containing private originals,
   unattached media, and raw route data; verify only selected derivatives and
   sanitized route output reach `site/`.

Each experiment should produce a machine-readable report with:

```text
input hashes
source capabilities used
candidate counts
warning/error codes
peak memory and elapsed time
deterministic output hash
```

## Design pressure points

The proposal should be reconsidered if any experiment shows that:

- processing requires loading the entire raw GPX or photo set into memory;
- equivalent sources produce materially different canonical semantics;
- re-import cannot preserve review decisions and authored fields;
- users cannot understand why a photo was attached to a stop;
- candidate ranking creates false certainty rather than useful review work;
- the review state requires a permanent public `places` model;
- static compilation cannot reliably exclude private source material.

The first implementation seam should therefore be a pure, deterministic
`Plan` operation with explicit source capabilities and no database mutation.
