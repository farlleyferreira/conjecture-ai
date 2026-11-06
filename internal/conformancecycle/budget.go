package conformancecycle

// Budget is a conformance cycle's configured limits ("The conformance
// cycle SHALL enforce a configured maximum number of judge-to-
// implementer rounds and a configured maximum token budget per
// cycle"). The specification gives no default numbers for either -
// both are caller-configured, not something this package invents a
// value for.
type Budget struct {
	MaxRounds int
	MaxTokens int
}

// Exhausted reports whether round or tokensUsed has reached Budget's
// own limit - "Cycle stops at round limit": "WHEN the configured
// maximum number of rounds is reached without the code reaching
// conformance THEN the cycle stops." round/tokensUsed reaching the
// configured maximum (not only exceeding it) counts as exhausted: the
// specification's own wording is "is reached," not "is exceeded."
func (b Budget) Exhausted(round, tokensUsed int) bool {
	return round >= b.MaxRounds || tokensUsed >= b.MaxTokens
}

// VerdictOnBudgetExhausted is the verdict a cycle reports when its
// budget runs out without the code ever reaching conformance - "the
// cycle stops and reports the outcome as not conformant rather than
// continuing indefinitely." This is a different situation from a
// single check's own execution timing out (see [VerdictFromOutcome],
// which reports that as Inconclusive): the specification's own
// wording for budget exhaustion names NonConformant specifically, not
// Inconclusive - running out of the cycle's own round/token budget to
// find a definitive answer does not get the inconclusive benefit of
// the doubt a hardware timeout does.
const VerdictOnBudgetExhausted = NonConformant
