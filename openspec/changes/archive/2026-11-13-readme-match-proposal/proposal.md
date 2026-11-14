# Proposal

## Why

The user supplied the original 91-page product proposal (the real source
of truth this codebase's `openspec/specs/*/spec.md` Purpose statements
were built from) and asked whether the MVP actually reflects it. A
grounded, line-by-line comparison (reading the whole proposal, grepping
the real source for every concrete claim) found the README was
undeselling real, verified mechanics the MVP already has: the context
engine's three independent axes, the specific multi-condition criteria
for promoting to Law, the swarm's five-strategy/bandit-allocated design,
and knowledge bases' four authority levels. One proposal item
(`materialization.cjtr` generation) is a genuine, acknowledged gap,
scheduled for next week as an internal, non-blocking artifact; explicit
direction was not to cite it in the README. Every other apparent gap
found during the comparison (observed-assertion extraction, a pinned
default SLM model, statistical rule-of-three promotion thresholds, a
`conjecture review`/`report`/`why` command set) was confirmed to be a
conscious design decision, not an oversight - also not something to
frame as a "gap" in the README.

## What Changes

- `README.md`/`README.en.md`'s "Why `conjecture`" section sharpened with
  four real, verified mechanics the proposal names and the real code
  implements, none of which the README previously mentioned at all:
  - Swarm attack strategy count (≥5) and bandit budget allocation
    (`internal/attackerswarm`).
  - The maturity ladder's real, distinct per-step criteria
    (`internal/evidenceledger/promotion.go`): Thesis→Theory needs
    cross-family reproduction + whole-swarm robustness; Theory→Law needs
    hidden-sample conformance + conformance across new code versions -
    replacing a vaguer "evidence accumulates" line, and correctly
    attributing which condition belongs to which step (an earlier draft
    of this bullet conflated the two steps' criteria).
  - Context engine's three independent axes (weight/relevance/confidence,
    never multiplied; weight never auto-tuned - `internal/contextengine`).
  - Knowledge bases' four authority levels (norm/definition/reference/
    historical - `internal/knowledgebases`).
- Removed the "Materialization content generation... out of scope" bullet
  from the "Scope and limits" collapsed section in both files, per
  explicit direction not to cite it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - documentation only, no behavior change. `skip_specs: true`.

## Follow-up: still didn't land after the first pass

Feedback on the first version of this change: still not good enough -
wrong on structure, tone, and still not showing enough real depth. Full
rewrite of both files' opening, adding two new sections grounded in the
proposal's own real architecture:

- **"The problem this actually solves"** - opens with the proposal's own
  problem framing (trust that's binary and invisible, whoever writes the
  code grading it, intent never directly checked, rigor scaling with
  size instead of risk) condensed to the four sharpest causes, each one
  a cause `conjecture`'s own real mechanics directly answer.
- **"Two loops, so no model ever grades its own work"** - names the
  product's real two-loop architecture (spec loop: you + judge refine
  until signed; conformance loop: LLM + judge iterate until the frozen
  spec holds) explicitly, instead of leaving it implicit across
  scattered bullets. Grounded in the real trust-boundary mechanics
  (`llm-orchestration`'s separate-session rule, already verified this
  session) and in `conjecture never writes the application code itself`
  (the proposal's own claim, matching the real CLI: `draft`/`implement`
  ask an LLM to write it, `conjecture` only judges and produces
  mutations/cases/verdicts).
- Trimmed the first two "Why conjecture" bullets (frozen spec, trust
  boundary) since the new "Two loops" section already covers them more
  completely - avoids saying the same thing twice in two different
  shapes.

## Impact

- Affected paths: `README.md`, `README.en.md`.
- No code or test changes; `go build`/`go vet` unaffected.
- Every new bullet traced to real source: `internal/attackerswarm`
  (strategy count + bandit allocation), `internal/evidenceledger/
  promotion.go` (Law criteria), `internal/contextengine` (three axes,
  no auto-tuning), `internal/knowledgebases` (four authority levels) -
  all confirmed present in the real codebase before being added to the
  README, not copied from the proposal without checking.
