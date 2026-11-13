# Design

## Context

See proposal.md - Why. This follows directly from the conjecture-workflow
skill (block 4): a Socratic back-and-forth on whether `implement`/`work`
could be a private LLM the same way `attack`/`swarm` already are, settled
by reading `attack`'s own real code (it trusts the SLM's claim with no
mechanical check, so `implement`'s real-execution grounding is already
stronger) and landing on the real, groundable difference instead: domain
coverage, not model provenance.

## Goals / Non-Goals

**Goals:**
- Make the refusal a real, code-level guarantee - reachable from the CLI
  directly, not only documented in the skill an agent could ignore.
- Reuse the one classifier this codebase already has for "is this
  hypothesis's domain boundable" rather than inventing a new complexity
  metric.

**Non-Goals:**
- A general-purpose "implementation complexity score." Domain
  boundedness is the one concrete, already-grounded signal this
  codebase computes; a broader notion of complexity (number of
  concepts, cross-rule interactions, …) is not attempted, since nothing
  in this codebase measures those today and inventing a new metric
  for this change would be guessing.
- Changing which rigor level `check`/`implement` actually *runs*
  against an already-accepted implementation.
  `CheckHypothesisWithSelectedRigor`'s own isLaw-gated selection is
  unchanged; this is a separate, new classification applied earlier,
  before any code is even generated, to decide whether to generate it
  at all.
- Retrying a refused work-queue item differently from any other
  unresolvable item. It is dropped the same way a malformed id already
  is (see Impact in proposal.md); building a distinct "permanently
  unimplementable by this oracle" queue state is more machinery than
  this change needs - the agent driving conjecture discovers the
  violation again via `status`/`check` regardless.

## Decisions

1. **The gate runs once per call, before round 0, not per round.** The
   domain's own boundedness never changes between rounds of the same
   hypothesis - computing it once and refusing immediately is both
   correct and cheaper than checking it on every iteration.
2. **A plain hypothesis is gated the same way a law is**, even though
   `CheckHypothesisWithSelectedRigor`'s own rigor-level selection only
   ever tries complete/bounded proof for a law (a plain hypothesis
   "always uses directed sampling" there, for an unrelated reason - it
   has no declared weight to select a stronger level by). This gate is
   not about which rigor level *runs the check*; it is about whether
   *this command's own disposable oracle* should be trusted to write
   the implementation in the first place, which depends only on the
   hypothesis's own declared domain, not on whether it is a law.
3. **A sentinel error (`ErrDomainTooComplexForOracle`), not a new
   `Verdict` value.** Refusing before ever generating code is not a
   conformance verdict (nothing was checked); it is a precondition
   failure, the same category every other "cannot process this item"
   case in `runImplement`/`workLoop` already is, and both already
   handle a returned error the same uniform way - no new plumbing
   needed in either caller.

## Risks / Trade-offs

- **A work-queue item stuck behind this refusal is silently dropped
  from `work`'s own automated loop** (disclosed in proposal.md -
  Impact). The agent driving conjecture still finds it via `status` or
  a direct `check`; this is not a lost signal, just not one `work`
  surfaces proactively today. A future, separate change could restore
  it to the queue tagged "needs the driving agent," not attempted here.
- **Boundedness is a proxy for "safe to trust the oracle," not a proof
  of it.** A hypothesis with a small bounded domain can still be
  subtly wrong in a way complete/bounded proof's own case generation
  does not probe (a logic error that happens to hold across every
  enumerated case). This gate narrows the risk this project's own
  usability discussion identified; it does not eliminate every way an
  oracle's "Conformant" verdict could be optimistic.
