# Proposal

## Why

A follow-up to PR #176 ("sharpen Why conjecture with verified real
mechanics"): feedback on that first pass was that the README still
wasn't good enough on structure, tone, and depth. A full restructure was
drafted and committed on that same PR's branch, but PR #176 was merged
before that second commit finished propagating - the merge only picked
up the first commit, so the restructure never landed on `main`. This
change reapplies it.

## What Changes

- Both `README.md` (pt-BR) and `README.en.md` now open with "The problem
  this actually solves" - the proposal's own problem framing (binary/
  invisible trust, whoever writes the code grading it, intent never
  directly checked, rigor scaling with size instead of risk), condensed
  to the four sharpest causes, each one answered by a real mechanic
  `conjecture` already has.
- Added "Two loops, so no model ever grades its own work" - names the
  product's real two-loop architecture explicitly (spec loop: you + judge
  refine until signed; conformance loop: LLM + judge iterate until the
  frozen spec holds), grounded in the real trust-boundary mechanics
  (separate drafting/implementing sessions, hidden judge cases) instead
  of leaving it implicit across scattered bullets.
- Trimmed the first two "Why conjecture" bullets (frozen spec, trust
  boundary) since the new "Two loops" section already covers them.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - documentation only, no behavior change. `skip_specs: true`.

## Impact

- Affected paths: `README.md`, `README.en.md`.
- No code or test changes; `go build`/`go vet` unaffected.
- This is a re-application of content already drafted and reviewed once
  (as part of #176's own second commit) - the content itself is not new,
  only its landing on `main` is.
