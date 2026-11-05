// Package contextengine implements pieces of the context-engine
// capability (openspec/specs/context-engine/spec.md): the fixed
// weight-gating rules and the single-level relevance-propagation rule.
// It does not implement the context graph's storage, its confidence
// axis, or anything that reads a rule's own statements to decide
// whether it "involves money, identity, permissions, or personal data"
// - the cjtr-language grammar this codebase parses has no field for
// that, so this package takes it as a caller-supplied fact, the same
// way omission-detection takes its addressed-categories map as given.
package contextengine

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

// Weight is context-engine's node weight axis - the same Low/Medium/
// High/Critical scale evidence-ledger already defines for a
// hypothesis's risk weight (see [evidenceledger.Weight]), reused rather
// than duplicated: the specification's own wording ("weighted critical",
// "weighted high or above") is the same vocabulary evidence-ledger
// already uses for risk-proportional rigor, and context-engine's own
// specification text gives this axis no separate scale of its own.
type Weight = evidenceledger.Weight

const (
	Low      = evidenceledger.Low
	Medium   = evidenceledger.Medium
	High     = evidenceledger.High
	Critical = evidenceledger.Critical
)

// Category is one of the four kinds of rule the context-engine
// specification names as always warranting at least a High weight ("a
// rule involving money, identity, permissions, or personal data SHALL
// always be weighted high or above"). This is a closed list - the
// specification names exactly these four, unlike omission-detection's
// open, configurable catalog - so GatedWeight treats any other value as
// a caller error rather than silently ignoring it.
type Category string

const (
	Money        Category = "money"
	Identity     Category = "identity"
	Permissions  Category = "permissions"
	PersonalData Category = "personal_data"
)

func validCategory(c Category) bool {
	switch c {
	case Money, Identity, Permissions, PersonalData:
		return true
	default:
		return false
	}
}

// GatedWeight computes a node's weight from the specification's fixed
// gating rules, applied as a floor over base (the caller's own starting
// weight signal). Neither rule ever lowers base; with no categories and
// isUserDefinedLaw false, base passes through unchanged.
//
// isUserDefinedLaw's rule wins outright ("weighted critical regardless
// of other signals") - the specification gives nothing that overrides
// it back down. The category rule only raises base up to High when it
// is lower: "high or above" is read as a floor, not as pinning every
// sensitive rule to exactly High, since a rule already at Critical from
// some other signal must stay there.
//
// GatedWeight is the only function this package exposes that produces a
// Weight - there is deliberately no "optimize" or "auto-adjust" entry
// point, per "Weight is never auto-optimized." The specification's only
// other sanctioned path to change a weight is an approved calibration
// proposal (optimization-suite), which this codebase has not built.
func GatedWeight(base Weight, isUserDefinedLaw bool, categories []Category) (Weight, error) {
	for _, c := range categories {
		if !validCategory(c) {
			return 0, fmt.Errorf("contextengine: %q is not a recognized category", c)
		}
	}

	w := base
	if len(categories) > 0 && w < High {
		w = High
	}
	if isUserDefinedLaw {
		w = Critical
	}
	return w, nil
}

// Propagate reports a relevance signal's effect one level out from
// origin: every node neighbors directly lists for origin, and no
// further ("the system SHALL propagate a relevance signal exactly one
// level through the graph... SHALL NOT propagate it transitively beyond
// that"). The specification names no representation for a "relevance
// signal" itself, so this is generic over T - whatever scale or type
// the caller uses passes through unexamined; this package only decides
// which nodes receive it.
//
// origin itself is never a key of the result: Propagate reports the
// signal's spread to other nodes, not origin's own assignment, which is
// the caller's concern before propagation runs.
func Propagate[T any](neighbors map[string][]string, origin string, signal T) map[string]T {
	affected := make(map[string]T, len(neighbors[origin]))
	for _, n := range neighbors[origin] {
		affected[n] = signal
	}
	return affected
}
