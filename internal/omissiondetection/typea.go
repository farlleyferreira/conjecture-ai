package omissiondetection

// TypeAOmission is one hypothesis asserting behavior that no case or
// observation ever exercised.
type TypeAOmission struct {
	HypothesisID string
}

// CheckTypeA reports a Type A omission for hypothesisID when
// exercisedCaseCount is zero - "Asserted behavior with no evidence
// flagged": "WHEN a statement asserts a behavior but no case or
// observation exercises it THEN the system flags it as a Type A
// omission."
//
// exercisedCaseCount is supplied by the caller, not determined here:
// conformance-cycle's own directed-sampling run already counts how
// many real cases it generated and checked against running code (see
// conformancecycle.DirectedSamplingInputs) - the "observation and
// generated cases against the running system" the requirement's own
// wording names - so this package reuses that count rather than
// generating cases of its own.
func CheckTypeA(hypothesisID string, exercisedCaseCount int) (TypeAOmission, bool) {
	if exercisedCaseCount > 0 {
		return TypeAOmission{}, false
	}
	return TypeAOmission{HypothesisID: hypothesisID}, true
}
