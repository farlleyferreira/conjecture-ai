// Package contextcomposition implements the grounded pieces of the
// context-composition capability (openspec/specs/context-composition/
// spec.md): the archive-not-delete lifecycle state and the four
// composition modes' closed vocabulary.
//
// Deliberately not attempted here: the lifecycle commands themselves
// (list/resume/open/origin/diff/restore need real storage - see
// internal/persistence - this package only models the state those
// commands would act on), composition-by-reference's own "does not
// copy content" rule (a storage-layer property no pure function can
// check without a real store to point into), and the conflict
// precedence order ("explicit, documented precedence order" - the
// specification names that one must exist, not what it is; neither
// this file nor the product proposal document gives a concrete
// ordering anywhere, so inventing one here would not be grounded in
// anything the specification actually says).
package contextcomposition

import "fmt"

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
