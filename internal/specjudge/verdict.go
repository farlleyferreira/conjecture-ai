// Package specjudge implements pieces of the spec-judge capability
// (openspec/specs/spec-judge/spec.md): how a .cjtr specification is
// judged for soundness before any implementation code exists. The
// falsifiability check itself - the one check this codebase can run
// without an SMT solver or a decision-structure engine - already lives
// in internal/cjtr (Falsifiable), tied closely to cjtr.Statement/
// Concept; this package holds spec-judge's own standalone concept
// instead: the overall verdict a judging run produces.
package specjudge

import "fmt"

// Verdict is the three-way outcome spec-judge's own run produces on a
// specification ("Judge verdict on the specification": "The judge
// SHALL produce one of three verdicts on a .cjtr specification:
// accepted, pending, or rejected").
type Verdict int

const (
	Accepted Verdict = iota
	Pending
	Rejected
)

func (v Verdict) String() string {
	switch v {
	case Accepted:
		return "accepted"
	case Pending:
		return "pending"
	case Rejected:
		return "rejected"
	default:
		return "unknown verdict"
	}
}

// ParseVerdict parses s's [Verdict.String] form back into a Verdict.
func ParseVerdict(s string) (Verdict, error) {
	switch s {
	case "accepted":
		return Accepted, nil
	case "pending":
		return Pending, nil
	case "rejected":
		return Rejected, nil
	default:
		return 0, fmt.Errorf("specjudge: %q is not a recognized verdict", s)
	}
}

// Severity distinguishes a [Finding] that forces a Rejected verdict by
// itself from one that only makes the verdict Pending.
//
// The specification's own checks are not uniform on this: the
// falsifiability check says the judge "SHALL reject" a hypothesis
// outright, while every other check it names (the solver-based
// consistency check, the decision-structure completeness check, the
// boundary/side-effect check) is worded as "reports" or "flags" an
// issue, without that same "reject" word. This package follows that
// distinction literally rather than inventing its own severity scheme:
// only a check whose own requirement says "reject" produces a
// Rejecting finding; everything else produces a Blocking one, which
// can only ever push the verdict to Pending.
type Severity int

const (
	Blocking Severity = iota
	Rejecting
)

// Finding is one specific issue found while judging a specification -
// "SHALL report the specific findings that led to a pending or
// rejected verdict."
type Finding struct {
	Severity Severity
	Message  string
}

// Judge computes the overall verdict for a specification from every
// Finding collected while judging it: Rejected if any finding is
// Rejecting, Pending if there is at least one Finding but none of them
// are, Accepted if there are none at all - "WHEN the judge finds at
// least one unresolved issue but no outright rejection THEN it returns
// a pending verdict together with the list of unresolved issues."
func Judge(findings []Finding) Verdict {
	if len(findings) == 0 {
		return Accepted
	}
	for _, f := range findings {
		if f.Severity == Rejecting {
			return Rejected
		}
	}
	return Pending
}
