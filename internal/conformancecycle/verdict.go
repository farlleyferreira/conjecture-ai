// Package conformancecycle implements pieces of the conformance-cycle
// capability (openspec/specs/conformance-cycle/spec.md) that do not
// depend on actually running implementation code - that capability's
// input-generation half already lives in internal/cjtr (BoundaryValues,
// serving its default "directed sampling" rigor level), tied closely to
// cjtr.Concept; this package holds the capability's own standalone
// concepts instead, starting with its verdict type.
package conformancecycle

import "fmt"

// Verdict is the three-way outcome a conformance check produces
// ("Three-way conformance verdict": "The system SHALL produce one of
// three verdicts for a rule: conformant, non-conformant, or
// inconclusive").
type Verdict int

const (
	Conformant Verdict = iota
	NonConformant
	Inconclusive
)

func (v Verdict) String() string {
	switch v {
	case Conformant:
		return "conformant"
	case NonConformant:
		return "non-conformant"
	case Inconclusive:
		return "inconclusive"
	default:
		return "unknown verdict"
	}
}

// ParseVerdict parses s's [Verdict.String] form back into a Verdict.
func ParseVerdict(s string) (Verdict, error) {
	switch s {
	case "conformant":
		return Conformant, nil
	case "non-conformant":
		return NonConformant, nil
	case "inconclusive":
		return Inconclusive, nil
	default:
		return 0, fmt.Errorf("conformancecycle: %q is not a recognized verdict", s)
	}
}

// VerdictFromOutcome reports the verdict for one check's outcome:
// Inconclusive whenever the check timed out, regardless of what it
// would otherwise have concluded - "When a check exceeds its allotted
// time on constrained hardware, the system SHALL report the result as
// inconclusive, never as a rule violation" (hardware-scheduler's own
// "Timeout never counts as a violation" requirement states the
// identical rule; this one function satisfies both, rather than this
// codebase ending up with two separate, possibly diverging
// implementations of the same sentence). Otherwise, Conformant when
// passed, NonConformant when not.
func VerdictFromOutcome(timedOut, passed bool) Verdict {
	if timedOut {
		return Inconclusive
	}
	if passed {
		return Conformant
	}
	return NonConformant
}
