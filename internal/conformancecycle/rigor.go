package conformancecycle

import "github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"

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
