# Proposal

## Why

conformance-cycle's own "Three levels of universal ('for all') rigor"
requirement names directed sampling, bounded proof, and complete proof,
selected by "the rule's assigned weight and available evidence." Only
directed sampling exists in the codebase; bounded proof and complete proof
are unimplemented `Verdict`-adjacent labels with no code behind them. This
leaves the two higher rigor levels - the ones a critical-weight rule (a
user-defined law, per `contextengine.GatedWeight`'s own "isUserDefinedLaw's
rule wins outright") is meant to get - entirely unavailable, so every
hypothesis and law in this codebase is checked at the same, lowest rigor
level regardless of its own weight.

## What Changes

- Add `cjtr.ExhaustiveValues`: every value of a concept's own declared
  finite domain (a closed `one_of` list, or a decimal range with a declared
  precision), not just the boundary sample `cjtr.BoundaryValues` already
  returns, capped at a disclosed maximum count.
- Add `conformancecycle.BoundedProofInputs`/`CheckHypothesisBoundedProof`:
  exhaustively covers one input concept's own finite domain at a time
  (the same "vary one, hold the rest at baseline" shape
  `DirectedSamplingInputs` already uses), rather than only its boundary.
- Add `conformancecycle.CompleteProofInputs`/`CheckHypothesisCompleteProof`:
  the full combinatorial cross-product across every boundable input concept
  at once, capped at a disclosed maximum total case count.
- Add rigor-level selection in `cmd/conjecture`: for a user-defined law
  (the only kind of hypothesis with a real, grounded weight today), try
  complete proof, then bounded proof, then fall back to directed sampling,
  in that order, depending on which level's own input domain is actually
  boundable; a plain hypothesis (no declared weight) always uses directed
  sampling, unchanged.
- Record which rigor level actually ran in the evidence record, instead of
  the current hardcoded "directed sampling" text.

## Capabilities

### Modified Capabilities

- `conformance-cycle`: no requirement text changes - the "Three levels of
  universal rigor" requirement already exists and is already implemented
  for directed sampling; this change implements the two levels its own
  text already names but the codebase never built. See `.openspec.yaml`
  (`skip_specs: true`): no spec delta, since the requirement text itself
  does not change.

## Impact

- Affected paths: `internal/cjtr/sampling.go`,
  `internal/conformancecycle/` (two new files), `cmd/conjecture/check.go`,
  `cmd/conjecture/implement.go`.
- No new dependency.
- A user-defined law whose input concepts declare a finite domain
  (`one_of`, or a decimal range with a declared precision) will now run
  through more sandbox invocations per check than before (one per
  enumerated value instead of per boundary value) - bounded by this
  change's own disclosed count caps, not unbounded.
