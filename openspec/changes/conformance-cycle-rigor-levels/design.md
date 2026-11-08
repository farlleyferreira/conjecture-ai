# Design

## Context

conformance-cycle's own spec text names three rigor levels - directed
sampling, bounded proof, complete proof - selected "according to the rule's
assigned weight and available evidence," but gives no further definition
of what "bounded proof" or "complete proof" concretely mean as an
operation this codebase can run. Only directed sampling (boundary-value
sampling against a running implementation, already built) exists today.

This project's own architecture constrains the answer: `language-agnostic-
contracts`' "Observation only through declared contracts" requirement
forbids parsing or interpreting target source code to determine behavior -
the system observes a target only through its declared command/interface
and standard output formats. This rules out the conventional meaning of
"proof" in formal-methods tooling (symbolic execution or static analysis
of the implementation's own source), since that would require reading and
reasoning about the target's code directly, which this project has never
done and is specifically forbidden from doing for behavior determination.

## Goals / Non-Goals

**Goals:**
- Give "bounded proof" and "complete proof" a concrete, buildable meaning
  that stays entirely within black-box, contract-based observation (run
  the sandboxed implementation, read its declared output) - the same
  mechanism directed sampling already uses, just with different, stronger
  input coverage.
- Make the three levels a real hierarchy of increasing coverage, not three
  unrelated techniques: each stronger level is a strict superset of what
  the weaker one checks.
- Apply weight-based selection only where a weight is already real and
  grounded in this codebase: a user-defined law
  (`contextengine.GatedWeight`'s "isUserDefinedLaw's rule wins outright" -
  always Critical). A plain hypothesis still has no declared weight
  anywhere in the `.cjtr` grammar - the same disclosed gap named since
  `evidence-ledger`'s own promotion threshold work - so it is out of scope
  here, not silently guessed at.

**Non-Goals:**
- Symbolic execution, static analysis, or any technique that reads the
  target implementation's own source - forbidden by `language-agnostic-
  contracts`, not merely out of scope.
- A real weight axis for a plain hypothesis - that needs a `.cjtr` grammar
  extension this change does not attempt.
- SMT-based proof of the specification itself - that is
  `spec-judge`/`expression-core`'s own "Consistency check via solver",
  already built separately (a different requirement, checking the spec
  text for contradictions, not checking a running implementation's
  conformance to it).

## Decisions

1. **"Bounded proof" = exhaustive enumeration of one input concept's own
   finite domain, one concept at a time.**
   `cjtr.BoundaryValues` already samples a concept's edges (min, max, and
   the values just inside each). Bounded proof runs the *entire* domain
   when it is finite - every `one_of` value, or every value of a decimal
   range at its own declared `places` precision - using the exact same
   "vary this one concept, hold every other at the worked example's
   baseline" shape `DirectedSamplingInputs` already uses. This is a
   genuine strengthening of the existing mechanism (complete coverage of
   one dimension instead of a boundary sample of it), not a new kind of
   check, and needs no new infrastructure.

2. **"Complete proof" = the same exhaustion, across every boundable input
   concept at once (their full combinatorial cross-product).**
   This is what actually distinguishes it from bounded proof: bounded
   proof varies one concept while holding the rest fixed; complete proof
   covers every *combination*. A hypothesis only qualifies for complete
   proof when every one of its input concepts has a finite, enumerable
   domain - if even one does not, the input space is not actually fully
   covered, so this falls back to bounded proof (or further, to directed
   sampling) rather than claiming a completeness it did not achieve.

3. **A domain only counts as "boundable" when the grammar already
   declares enough to enumerate it exactly - nothing is invented.**
   A closed `one_of` list is already finite. A decimal concept is
   boundable only when it declares both a range (`Min`/`Max`) *and* a
   precision (`Places`): together they give an exact step count
   (`(Max-Min)/10^-Places + 1`) already determined by the specification
   author, not a step size this package would otherwise have to invent.
   A decimal range without a declared `Places` stays unbounded - there is
   no grounded way to enumerate a continuous domain without inventing a
   precision the specification never gave it.

4. **A disclosed maximum enumeration count, not an unbounded one.**
   Exhaustive enumeration of a wide decimal range at a fine precision can
   still mean an enormous case count, each one a real sandboxed run. A
   domain (or, for complete proof, a combination) whose count exceeds a
   fixed, disclosed cap is treated as not boundable for this purpose -
   the same "stronger level, but only within what is actually practical to
   run" posture as everything else "available evidence" gates, rather
   than letting one wide domain block a cycle indefinitely or spend
   unbounded sandbox time.

5. **Weight-based level selection is real only for a law; a plain
   hypothesis is unaffected.**
   "The system SHALL select the level according to the rule's assigned
   weight" needs a weight to select by. `contextengine.GatedWeight`
   already makes every user-defined law's weight Critical, unconditionally
   - the same lever this session already used for evidence-ledger
   promotion and the work-queue's own failure-free-case tracking. A law
   tries complete proof, then bounded proof, then directed sampling, in
   that order, stopping at the first level whose own input domain is
   actually boundable. A plain hypothesis, having no real weight
   anywhere in the grammar, keeps using directed sampling exactly as
   before - this change does not guess at a weight for it.

6. **Which level actually ran is recorded in the evidence, not hidden.**
   `recordEvidence`'s own `Attacked` text currently hardcodes "directed
   sampling" regardless of what actually ran. Now that three different
   levels are real, the evidence record names whichever one the
   selection logic actually used - otherwise a law's own evidence trail
   would silently misreport a complete-proof pass as a mere sample.

## Risks / Trade-offs

- **More sandbox invocations per check for a law with a boundable
  domain.** Bounded/complete proof trade sandbox time for coverage -
  accepted, since it only ever applies to a Critical-weight law (the
  specification's own category meant to receive the strongest check
  available), and is capped (decision 4) rather than unbounded.
- **The maximum-count cap is this project's own chosen number, not one
  the specification gives.** Disclosed in code, following the same
  pattern as every other disclosed-default value this session has
  already introduced (queue aging rate, contested-status threshold,
  work-loop retry cap).
- **Complete proof's combinatorial growth means even two or three
  boundable concepts can exceed the cap quickly**, falling back to
  bounded proof. This is the correct, honest behavior for this
  requirement - a claimed "complete proof" over a space the cap could
  not actually cover would be dishonest - not a bug to work around with
  a larger cap.
