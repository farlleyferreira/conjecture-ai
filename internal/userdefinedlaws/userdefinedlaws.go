// Package userdefinedlaws implements the user-defined-laws capability's
// decision rules (openspec/specs/user-defined-laws/spec.md): a law's
// explicit status lifecycle, and its invalidation/precedence pathway
// when it conflicts with another statement.
//
// This package only implements those decision rules themselves, as
// pure functions over facts the caller supplies (whether a formal proof
// contradicts a law, whether two conflicting statements carry equal
// declared authority, how much counterevidence has accumulated) - it
// does not parse a .cjtr file's own "user_defined_laws" section (that
// already lives in internal/cjtr, see UserDefinedLaws), run a
// conformance cycle, or produce a formal proof itself.
package userdefinedlaws

import "fmt"

// Status is one of a user-defined law's explicit lifecycle states
// ("A user-defined law SHALL carry one of the following statuses:
// unverified, conformant, conformant-under-generalization, violated,
// contested, or invalidated"). The zero value, Unverified, is a law's
// starting status before any conformance evidence exists.
type Status int

const (
	Unverified Status = iota
	Conformant
	ConformantUnderGeneralization
	Violated
	Contested
	Invalidated
)

func (s Status) String() string {
	switch s {
	case Unverified:
		return "unverified"
	case Conformant:
		return "conformant"
	case ConformantUnderGeneralization:
		return "conformant-under-generalization"
	case Violated:
		return "violated"
	case Contested:
		return "contested"
	case Invalidated:
		return "invalidated"
	default:
		return "unknown status"
	}
}

// ParseStatus parses s's [Status.String] form back into a Status.
func ParseStatus(s string) (Status, error) {
	switch s {
	case "unverified":
		return Unverified, nil
	case "conformant":
		return Conformant, nil
	case "conformant-under-generalization":
		return ConformantUnderGeneralization, nil
	case "violated":
		return Violated, nil
	case "contested":
		return Contested, nil
	case "invalidated":
		return Invalidated, nil
	default:
		return 0, fmt.Errorf("userdefinedlaws: %q is not a recognized status", s)
	}
}

// ConflictOutcome is the result of applying the invalidation pathway to
// one conflict between a user-defined law and another statement.
type ConflictOutcome int

const (
	// LawPrevails: the law is not invalidated and nothing is raised to
	// a human - either there is no formal proof against it and the
	// other side does not carry equal authority (the specification's
	// fixed "Precedence over promoted hypotheses" rule, since a
	// promoted hypothesis is never of equal authority to a law).
	LawPrevails ConflictOutcome = iota
	// RaiseQuestion: no formal proof settles the conflict, and both
	// sides carry equal declared authority - the system refuses to
	// decide unilaterally.
	RaiseQuestion
	// AutoInvalidated: a formal proof contradicts the law - invalidated
	// automatically, regardless of either side's authority.
	AutoInvalidated
)

func (o ConflictOutcome) String() string {
	switch o {
	case LawPrevails:
		return "law prevails"
	case RaiseQuestion:
		return "raise question"
	case AutoInvalidated:
		return "auto-invalidated"
	default:
		return "unknown conflict outcome"
	}
}

// ResolveConflict decides the outcome of a conflict between a user-
// defined law and another statement (a promoted hypothesis, or another
// law of its own declared authority), per user-defined-laws'
// "Invalidation pathway" and "Precedence over promoted hypotheses"
// requirements.
//
// A formal proof contradicting the law always wins, regardless of
// authority ("the system SHALL invalidate a user-defined law
// automatically when a formal proof contradicts it"). Absent one, an
// equal-authority conflict is never decided unilaterally - it raises a
// question instead ("When two statements of equal authority conflict
// without a formal proof, the system SHALL raise a question rather
// than deciding unilaterally"). This same branch also covers
// "Precedence over promoted hypotheses": a promoted hypothesis is
// never of equal authority to a user-defined law, so a caller passes
// equalAuthority=false for that case, and the law prevails - the
// specification's own fixed precedence rule, not a runtime judgment
// call this function makes.
func ResolveConflict(hasContradictingFormalProof, equalAuthority bool) ConflictOutcome {
	if hasContradictingFormalProof {
		return AutoInvalidated
	}
	if equalAuthority {
		return RaiseQuestion
	}
	return LawPrevails
}

// MeetsContestedThreshold reports whether counterevidenceCount pieces
// of accumulated empirical counterevidence against a user-defined law
// have crossed threshold ("Empirical counterevidence SHALL accumulate
// toward a configurable threshold before the law is marked contested").
// threshold is a caller-supplied parameter, never a default this
// package invents, since the specification's own wording makes it
// explicitly configurable per law rather than naming one fixed number.
func MeetsContestedThreshold(counterevidenceCount, threshold int) bool {
	return counterevidenceCount >= threshold
}
