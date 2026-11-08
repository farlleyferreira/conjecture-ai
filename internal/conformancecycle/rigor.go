package conformancecycle

import (
	"context"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// RigorLevel is one of the three levels of "for all" rigor the
// specification names for a universally quantified rule.
type RigorLevel int

const (
	DirectedSampling RigorLevel = iota
	BoundedProof
	CompleteProof
)

func (l RigorLevel) String() string {
	switch l {
	case DirectedSampling:
		return "directed sampling"
	case BoundedProof:
		return "bounded proof"
	case CompleteProof:
		return "complete proof"
	default:
		return "unknown rigor level"
	}
}

// UsesDirectedSampling reports whether weight and hasStrongerEvidence
// match the one rigor-level rule the specification gives concretely:
// "WHEN a low-weight rule is checked with no stronger evidence
// available THEN the conformance cycle uses directed sampling rather
// than bounded or complete proof."
//
// "The system SHALL select the level according to the rule's assigned
// weight and available evidence" implies a fuller decision table
// exists across all four weights and both evidence states, but the
// specification gives no further concrete rule - what level a higher
// weight selects, or what level additional evidence selects once it
// is available - so this package does not invent the rest of that
// table. A caller needing to classify every other combination still
// has to supply that decision itself.
func UsesDirectedSampling(weight evidenceledger.Weight, hasStrongerEvidence bool) bool {
	return weight == evidenceledger.Low && !hasStrongerEvidence
}

// CheckHypothesisWithSelectedRigor runs the strongest rigor level h
// actually qualifies for against a real implementation: for a law
// (isLaw), complete proof, then bounded proof, then directed
// sampling, trying each weaker level only when the stronger one's own
// input domain was not boundable (conformance-cycle-rigor-levels'
// design decision 5). A plain hypothesis has no declared weight
// anywhere in the .cjtr grammar - [contextengine.GatedWeight]'s own
// "isUserDefinedLaw's rule wins outright" is the only grounded weight
// this codebase has - so it always uses directed sampling, exactly as
// before bounded/complete proof existed.
//
// The returned RigorLevel names whichever level actually ran, and the
// returned case count is the number of cases that level exercised -
// both needed by a caller that records which level produced the
// returned verdict, or tracks accumulated case counts (such as law
// promotion), instead of assuming directed sampling always ran.
func CheckHypothesisWithSelectedRigor(
	ctx context.Context,
	sb *sandbox.Sandbox,
	opts sandbox.Options,
	h cjtr.Hypothesis,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	isLaw bool,
) (RigorLevel, Verdict, *Counterexample, int, error) {
	if isLaw {
		cases, found, err := CompleteProofInputs(h, concepts)
		if err != nil {
			return 0, 0, nil, 0, err
		}
		if found {
			verdict, counterexample, _, err := CheckHypothesisCompleteProof(ctx, sb, opts, h, concepts, tables)
			if err != nil {
				return 0, 0, nil, 0, err
			}
			return CompleteProof, verdict, counterexample, len(cases), nil
		}

		cases, found, err = BoundedProofInputs(h, concepts)
		if err != nil {
			return 0, 0, nil, 0, err
		}
		if found {
			verdict, counterexample, _, err := CheckHypothesisBoundedProof(ctx, sb, opts, h, concepts, tables)
			if err != nil {
				return 0, 0, nil, 0, err
			}
			return BoundedProof, verdict, counterexample, len(cases), nil
		}
	}

	cases, err := DirectedSamplingInputs(h, concepts)
	if err != nil {
		return 0, 0, nil, 0, err
	}
	verdict, counterexample, err := CheckHypothesisDirectedSampling(ctx, sb, opts, h, concepts, tables)
	if err != nil {
		return 0, 0, nil, 0, err
	}
	return DirectedSampling, verdict, counterexample, len(cases), nil
}
