# Real-trip walkthrough of the CLI entry path

Walking the documented local-authoring route with a real multi-day trip —
recorded on a phone and in Google Timeline — instead of the hand-written
fixtures every existing test uses. The trip's own data is not reproduced here.

**Inputs:** a Google Timeline export (61 segments: 32 visits, 29 activities,
spanning six days) and 117 files from the phone (96 HEIC, 14 PNG screenshots,
7 MOV).

**Outcome:** the path did not run at all. Nine defects sat between
`make journey-local` and a packaged journey; a tenth was found in the GUI
afterwards. All nine are fixed and covered by
`tests/test_local_journey_raw_intake.py`. The tenth is a design seam and is
recorded below, not fixed.

## Why nothing caught this

Every local-journey fixture in the repository is a **hand-written workspace** —
`journey.json`, `stops.json`, `mementos.json` written by a person. The producer
that is supposed to generate them (`felicia-cli journey plan`, driven by
`make journey-local`) had no test at all. The one test that validated a plan
document hand-wrote that too, in the same stale field casing as the schema, so
both sides agreed with each other while neither agreed with the producer.

The result: the CLI entry path was broken continuously for some time, while a
green `make validate` reported otherwise.

## The nine defects

| #   | Defect                                                                                      | Class                |
| --- | ------------------------------------------------------------------------------------------- | -------------------- |
| 1   | `plan.json` rejected: `date_start`/`date_end` not in the schema (added by `6a0ccf9`)        | contract drift       |
| 2   | `mementos.json` rejected: blank `occurred_tz` — `.get(key, default)` over an existing key   | empty-value handling |
| 3   | `plan.json` rejected: a candidate's `kind` is deliberately unset until promote time         | contract semantics   |
| 4   | `kind_data: null` — a nil Go map                                                            | nil marshalling      |
| 5   | `memory_links: null` in candidates and in media assets — nil Go slices                      | nil marshalling      |
| 6   | The whole `stop_candidate` definition still in Go PascalCase after the type gained tags     | contract drift       |
| 7   | Derived stops carried blank provenance while their evidence named a source (ADR-0010)       | **data quality**     |
| 8   | Every photo silently dropped: `uri`/`title` read from an asset serialising as `URI`/`Title` | **silent data loss** |
| 9   | Journey dated the day of import, discarding the bounds the planner derived                  | **data quality**     |

Defects 7–9 are the ones to worry about. The rest fail loudly; these three
produce a plausible, publishable, wrong result — an unattributable observation,
a site with no photos, and a travel journal dated today.

Defect 9 only appeared with real data: a synthetic track written for today's
date hides it.

## What the pipeline produced

After the fixes, from the real inputs:

```text
1599 track points → 20 stops → 11 mementos → 45 photos → 26 MB artifact
journey dates derived from the track, not from the day of import
```

## Capability gaps this exposed

None of these are defects; they are things felicia does not do. Each one had to
be done outside the tool before the pipeline could start.

### 1. Google Timeline cannot be imported

The export carries `visit` and `activity` segments and **no track points at
all**. The local route source reads GPX points, so the trip had to be converted
by synthesising points: a visit became a stationary sample every five minutes,
an activity a straight line between its endpoints.

The conversion is lossy in the direction that matters. Timeline already provides
the **visit** layer that felicia derives by clustering points — so visits were
flattened into points only for felicia to cluster them back, losing the place
names, `placeID`s, and semantic types on the way. All 20 derived stops came out
unnamed (`stop_label_missing` ×20), and every activity became a straight line
that does not follow the route actually travelled.

Timeline maps onto `VisitSource` far better than onto `RouteSource`.

### 2. HEIC is unsupported end to end

96 of the trip's photos were HEIC, the common phone default. The public media boundary
accepts JPEG, PNG, and WebP, and nothing in the pipeline converts. They were
converted with `sips` outside felicia. EXIF survived conversion intact.

### 3. Local photos have no timestamps

EXIF is only decoded on the Immich path, so a photo folder is timestamp-less and
cannot be joined to a track. `scripts/photo_sidecar.py` (added alongside this
walkthrough) reads DateTimeOriginal and GPS out of JPEG and HEIC and writes the
JSONL sidecar; it recovered all 96 timestamps and 89 GPS points. The 14
screenshots have no capture time and are reported rather than dropped.

This is tooling-level cover for a provider-level gap.

## The tenth finding: the CLI and GUI paths do not meet

The package carried 20 stops. After importing it, the admin GUI shows:

```text
Intake inbox — No stop candidates yet. Run "Plan intake" to generate some.
```

Stop candidates are produced by the planner against a journey's _sources_ and
are not persisted by package import, which brings in journeys and mementos only.
`Plan intake` re-runs the planner against Dawarich and Immich — not against the
GPX the trip came from.

So the two paths each hold half of what the other needs:

```text
CLI  local files → mementos ✓    stop candidates ✗
GUI  stop candidates ✓           local files ✗
```

The consequence for an author is concrete: the 20 stops exist only in the CLI
workspace's `stops.json`, so the surface built for naming, merging, and
discarding stops cannot be used on a trip that entered through the CLI. Naming
them means hand-editing JSON.

This matters for [FELICIA-ADMIN-02 M4](../roadmap/admin-gui-v2-epic.md): adding
file upload to the GUI closes the _entry_ gap but leaves the two paths
unconnected. Either package import must write stop candidates, or intake
planning must be able to run over local sources.

## Reproducing

The fixture track in `tests/fixtures/local-journey-raw/` reproduces defects 1–9
in miniature. For the real-data walkthrough the conversion steps are not in the
repository; they were a Timeline-to-GPX converter and `sips`.

```bash
uv run python scripts/photo_sidecar.py ~/trip/photos --tz Asia/Tokyo --out ~/trip/photos.jsonl
make journey-local GPX=~/trip/route.gpx PHOTOS=~/trip/photos SIDECAR=~/trip/photos.jsonl
```

Check the counts in `stops.json` afterwards: a track with no 20-minute dwell
inside 250 m yields no stops, and `preprocess` reports success either way.
