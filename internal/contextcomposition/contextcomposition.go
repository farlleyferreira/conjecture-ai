// Package contextcomposition implements the grounded pieces of the
// context-composition capability (openspec/specs/context-composition/
// spec.md): the archive-not-delete lifecycle state, the four
// composition modes' closed vocabulary, a conflict-precedence order
// over those four modes (see [PrecedenceOrder]'s own doc comment for
// where that order comes from - the specification itself gives none),
// and composition-by-reference itself (see reference.go) - the one
// composition mode the specification gives a concrete scenario for
// ("does not copy content").
//
// Deliberately not attempted here: the lifecycle commands themselves
// (list/resume/open/origin/diff/restore need real storage - see
// internal/persistence - this package only models the state those
// commands would act on), the other three composition modes
// (embedding, inheritance, view - the specification names them but
// gives no scenario describing how any of their actual semantics
// differ, unlike reference's own concrete one, so none is guessed at
// here), and resolving a conflict between two sources composed by the
// *same* mode (see [ResolveConflict]'s own doc comment - this is a
// separate, still-open question PrecedenceOrder does not answer).
package contextcomposition

import (
	"fmt"
	"time"
)

// LifecycleState is whether a context is still active or has been
// superseded by a newer one.
type LifecycleState int

const (
	Active LifecycleState = iota
	Archived
)

func (s LifecycleState) String() string {
	switch s {
	case Active:
		return "Active"
	case Archived:
		return "Archived"
	default:
		return "unknown lifecycle state"
	}
}

// ParseLifecycleState parses s's [LifecycleState.String] form back into
// a LifecycleState.
func ParseLifecycleState(s string) (LifecycleState, error) {
	switch s {
	case "Active":
		return Active, nil
	case "Archived":
		return Archived, nil
	default:
		return 0, fmt.Errorf("contextcomposition: %q is not a recognized lifecycle state", s)
	}
}

// Context is one versioned, composable verification context.
type Context struct {
	ID    string
	State LifecycleState
	// SupersededBy is the ID of the context that superseded this one,
	// set only once State is Archived.
	SupersededBy string
	// Area is the context's own declared area, for "conjecture
	// context list"'s own area filter - "" when the context has none
	// (not every context is tied to one .cjtr file's own area/
	// phenomenon path). This is the caller's own fact, the same
	// "caller supplies the fact" posture this codebase already takes
	// everywhere a value cannot be derived from the cjtr-language
	// grammar alone - see [CreateContext]'s own doc comment for the
	// one real caller that already has a grounded area to supply
	// (cmd/conjecture's own "conjecture check", reusing the exact
	// area/phenomenon path convention loadFicha already derives a
	// .cjtr file's own area from).
	Area string
	// CreatedAt is set only for a context loaded from the store (see
	// store.go) - the zero value for one built in memory.
	CreatedAt time.Time
}

// Supersede returns ctx archived and recording supersededBy -
// "contexts are archived, never deleted... a superseded context still
// diffable" (see the specification's own scenario) - rather than
// removing ctx. Supersede does not itself enforce that supersededBy
// names a real context; a caller already holds that fact.
func Supersede(ctx Context, supersededBy string) Context {
	ctx.State = Archived
	ctx.SupersededBy = supersededBy
	return ctx
}

// CompositionMode is one of the four ways the specification names for
// composing contexts.
type CompositionMode int

const (
	ByReference CompositionMode = iota
	ByEmbedding
	ByInheritance
	ByView
)

func (m CompositionMode) String() string {
	switch m {
	case ByReference:
		return "ByReference"
	case ByEmbedding:
		return "ByEmbedding"
	case ByInheritance:
		return "ByInheritance"
	case ByView:
		return "ByView"
	default:
		return "unknown composition mode"
	}
}

// ParseCompositionMode parses s's [CompositionMode.String] form back
// into a CompositionMode. The specification names exactly these four
// modes - a closed set, so an unrecognized value is an error rather
// than a silently-accepted extension.
func ParseCompositionMode(s string) (CompositionMode, error) {
	switch s {
	case "ByReference":
		return ByReference, nil
	case "ByEmbedding":
		return ByEmbedding, nil
	case "ByInheritance":
		return ByInheritance, nil
	case "ByView":
		return ByView, nil
	default:
		return 0, fmt.Errorf("contextcomposition: %q is not a recognized composition mode", s)
	}
}

// PrecedenceOrder ranks the four composition modes from highest to
// lowest authority when two composed contexts conflict over the same
// value - "explicit conflict precedence... not an unspecified or
// arbitrary rule." Neither the specification, the archived bootstrap
// change's design notes, nor the original product proposal document
// gives a concrete ordering anywhere; this one was constructed at the
// user's explicit direction, by analogy to how scientific methodology
// ranks a structure's authority when accounts of the same fact
// conflict:
//
//   - ByReference ranks highest: it points straight at the source's
//     own current content, unmediated - the analogue of a direct
//     empirical observation, the least-interpreted form a value can
//     take.
//   - ByEmbedding ranks next: a copy taken at composition time - the
//     analogue of a recorded measurement, one step removed from the
//     live source but still a direct capture of it, not a derivation.
//   - ByInheritance ranks next: a context built from a base with its
//     own overrides layered on top - the analogue of a conclusion
//     resting on an inherited, less-scrutinized assumption, since an
//     override introduces content of its own over the base rather than
//     carrying the base's own value forward unchanged.
//   - ByView ranks lowest: a filtered or projected read of other
//     content - the analogue of an interpretive lens, which asserts
//     nothing new on its own and only reframes what is already there.
//
// This resolves a conflict between two *different* modes only. It does
// not resolve one between two sources composed by the *same* mode
// (e.g. two ByReference sources disagreeing) - see [ResolveConflict]'s
// own doc comment for that still-open case.
var PrecedenceOrder = []CompositionMode{ByReference, ByEmbedding, ByInheritance, ByView}

// rank returns m's index into [PrecedenceOrder] (lower is higher
// authority), or len(PrecedenceOrder) for a value PrecedenceOrder does
// not name - which Outranks then treats as never winning, rather than
// panicking on an invalid mode.
func rank(m CompositionMode) int {
	for i, candidate := range PrecedenceOrder {
		if candidate == m {
			return i
		}
	}
	return len(PrecedenceOrder)
}

// Outranks reports whether a's value should win over b's when they
// conflict over the same node, per [PrecedenceOrder].
func Outranks(a, b CompositionMode) bool {
	return rank(a) < rank(b)
}

// ResolveConflict picks which of two conflicting values to keep by
// [PrecedenceOrder], reporting the winning mode and whether the
// outcome was actually decided.
//
// decided is false when a and b are the *same* mode: PrecedenceOrder
// ranks the four modes against each other, not two sources composed
// the same way, so it gives no basis to prefer one over the other. This
// is a genuinely open question the specification does not answer
// either - a caller must not treat ResolveConflict's returned winner as
// meaningful when decided is false.
func ResolveConflict(a, b CompositionMode) (winner CompositionMode, decided bool) {
	if a == b {
		return a, false
	}
	if Outranks(a, b) {
		return a, true
	}
	return b, true
}
