# Archive

These docs were written when felicia drifted into a **spec-freeze / pre-TDD** framing
(locked decisions, failing-test plans, a gap register marked LOCKED/PROPOSED). On
2026-06-12 we pulled back: **felicia is still in the research stage**, deliberately
unhurried, and that machinery was premature burden.

Nothing here is wrong or deleted — it's parked. Much of it (the data model sketch, the
A+E loop, the device walkthroughs, the importer pipeline shape) is good thinking we'll
draw on when we *do* move to spec. Read it for detail; don't treat it as binding.

The living research-stage docs are one level up:

- [`../direction.md`](../direction.md) — current north star + the personal-now /
  product-ready direction.
- `../research/` — exploration trail (workflows, liuaaron teardown, product-vs-personal
  analysis).

| File | Was | Why parked |
| --- | --- | --- |
| `design.md` | "current source of truth" design | premature lock-in; superseded by `direction.md` for now |
| `importer-spec.md` | CLI contract + first-failing-test plan | spec/TDD artifact — too early |
| `spec-gaps.md` | gap register, LOCKED/PROPOSED, freeze checklist | the freeze apparatus we're stepping back from |
| `device-walkthroughs.md` | per-kit rituals, "decisions baked in" | useful, but asserts locked decisions |
| `plan.md` / `todo.md` | milestone plan + task list | planning ahead of the research stage |
| `architecture.md` / `setup.md` | infra/setup notes | tied to the locked self-hosted design |
