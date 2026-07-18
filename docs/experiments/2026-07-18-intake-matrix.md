---
title: "Intake experiment matrix"
status: "review"
date: "2026-07-18"
---

# Intake experiment matrix

This report records the first executable pass over the intake user stories.
The harness is [`scripts/run_intake_experiments.py`](../../scripts/run_intake_experiments.py)
and runs the real `felicia-cli` against temporary synthetic inputs. Generated
raw inputs and JSON reports stay under `.felicia/` or the process temp folder.

Run it with:

```text
make experiment-intake
```

## Results

| Case                        | Result             | Evidence                                                                                                                                             |
| --------------------------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| US-01 plan without mutation | pass               | Raw GPX/local-photo plan executes, returns a versioned plan, and repeated output hashes match.                                                       |
| US-02 review stops          | partial            | Persistence and review APIs exist, but this harness does not invent author decisions or exercise an HTTP review session.                             |
| US-03 missing metadata      | partial            | Files remain discoverable; a JSONL sidecar now promotes timestamp/location metadata, while EXIF extraction and confidence classes remain incomplete. |
| US-04 multiple mementos     | partial            | The current planner emits at most one generic memento candidate per matched stop.                                                                    |
| US-05 agent suggestions     | not run            | Suggestion schema/store is intentionally not implemented yet.                                                                                        |
| US-06 safe publish          | pass-prepared-only | Existing prepared-package publication path validates; raw-intake publication is not wired.                                                           |
| Evil: invalid GPX           | pass               | Invalid coordinates fail before a plan is returned.                                                                                                  |
| Evil: large GPX             | baseline           | 20,000 points are measured; the current parser materializes the XML document, so memory is not yet bounded.                                          |

## Pushback and implementation gaps

The experiment confirms the main architecture, but it also prevents us from
calling the whole workflow complete:

1. Local media needs an explicit sidecar contract and confidence classification
   before timestamp-less photos can be promoted into matches.
2. One stop must support multiple authored mementos rather than only one generic
   candidate.
3. Agent suggestions need a reviewable patch/suggestion record with provider and
   model version; they must remain non-mutating until accepted.
4. Very large GPX files need streaming or bounded parsing and per-segment
   diagnostics. The current adapter is correct for small files but does not yet
   satisfy the scale requirement.
5. Raw-intake publication needs a privacy gate that excludes unattached media,
   private originals, and unsanitized route data.

Task 6 therefore validates the current seams and identifies the next product
work; it does not claim that all six stories pass.
