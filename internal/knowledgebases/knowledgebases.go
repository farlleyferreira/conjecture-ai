// Package knowledgebases implements the knowledge-bases capability
// (openspec/specs/knowledge-bases/spec.md): external knowledge bases
// (norms, definitions, references, history) attached to a project and
// scoped, so conjecture can cite authority material instead of relying
// only on the .cjtr files themselves.
//
// This package only implements the authority levels, their precedence
// ordering, and the registration record itself (with its norm-authority
// completeness check) - it does not index a base's content ("Local
// indexing for retrieval": textual search plus quantized embeddings,
// infrastructure this codebase does not have), and it does not yet
// expose the "conjecture base add" CLI command the specification names
// (a separate piece: parsing its flags and computing a real file or
// directory's content hash from the filesystem).
package knowledgebases

import "fmt"

// Authority is one of the four levels the specification names, in
// decreasing order of how binding the material is (Norm's own ordinal
// value is the smallest, so a lower Authority value always outranks a
// higher one - see [Authority.Outranks]).
type Authority int

const (
	// Norm is a binding legal or regulatory requirement.
	Norm Authority = iota
	// Definition is an agreed-upon, authoritative definition of a term
	// or concept.
	Definition
	// Reference is general reference material.
	Reference
	// History is historical/background context.
	History
)

func (a Authority) String() string {
	switch a {
	case Norm:
		return "norm"
	case Definition:
		return "definition"
	case Reference:
		return "reference"
	case History:
		return "history"
	default:
		return "unknown authority"
	}
}

// ParseAuthority parses one of the specification's four CLI flag
// values ("norm", "definition", "reference", "history") into an
// Authority.
func ParseAuthority(s string) (Authority, error) {
	switch s {
	case "norm":
		return Norm, nil
	case "definition":
		return Definition, nil
	case "reference":
		return Reference, nil
	case "history":
		return History, nil
	default:
		return 0, fmt.Errorf("knowledgebases: %q is not a recognized authority level (want norm, definition, reference, or history)", s)
	}
}

// Outranks reports whether a takes precedence over b in a conflict
// between two bases.
//
// The specification states exactly one relationship directly: "a norm-
// authority base and a reference-authority base disagree... the system
// resolves the conflict in favor of the norm-authority base." It never
// gives a full order across all four levels. This extends that one
// fact to a total order - Norm > Definition > Reference > History - by
// each level's own plain meaning (a binding requirement, an agreed
// definition, background reference material, and historical context,
// in decreasing bindingness): not an arbitrary ranking invented to fill
// the gap, but the only reading consistent with what each name already
// means and with the one ordering the specification does state.
func (a Authority) Outranks(b Authority) bool {
	return a < b
}
