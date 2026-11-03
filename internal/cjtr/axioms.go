package cjtr

import (
	"errors"
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Axiom is one entry of an "axioms"/"assumes" clause (field-reference
// table: "No topo ou dentro de uma regra, no nível onde valem" -> "Axioma,
// premissa"; grammar table: "Forma do given; premissa não observável fica
// como texto declarado").
//
// An entry whose leading term names a declared concept parses exactly
// like a "given" statement (Statement is set to it); one whose leading
// term does not is an unobservable premise and is kept only as its
// original declared text - the specification's own distinction, not an
// invented fallback. Text is always set, in both cases, to the entry
// exactly as written. Anything else that fails to parse (a declared term
// followed by an unsupported or unrecognized comparator, say) is still a
// hard error, same as "given": that is a vocabulary mistake, not an
// unobservable premise, and this package never silently swallows one.
type Axiom struct {
	Statement *Statement
	Text      string
}

// Axioms reads a rule's or file's "axioms"/"assumes" clause, if present
// (nil, nil otherwise). body may be a rule's own body node or a
// document's root node, matching the specification's "at the top or
// inside a rule" placement - the same pattern [UserDefinedLaws] already
// follows for the same reason.
//
// "axioms" and "assumes" are two names for the same clause (paired in the
// field-reference table, like "is"/"equals" are for the same comparator);
// a body declaring both is rejected rather than silently picking one.
func Axioms(body *yamlstrict.Node, concepts map[string]Concept) ([]Axiom, error) {
	axiomsNode, hasAxioms := body.Get("axioms")
	assumesNode, hasAssumes := body.Get("assumes")
	switch {
	case hasAxioms && hasAssumes:
		return nil, fmt.Errorf(`cjtr: declares both "axioms" and "assumes" - they are the same clause under two names, pick one`)
	case hasAxioms:
		return parseAxioms(axiomsNode, concepts, "axioms")
	case hasAssumes:
		return parseAxioms(assumesNode, concepts, "assumes")
	default:
		return nil, nil
	}
}

// parseAxioms accepts a scalar or a list of scalars, per the same
// "given"-shaped grammar [parseGiven] already follows.
func parseAxioms(node *yamlstrict.Node, concepts map[string]Concept, key string) ([]Axiom, error) {
	switch node.Kind {
	case yamlstrict.ScalarKind:
		a, err := parseAxiom(node.Value, concepts, key)
		if err != nil {
			return nil, err
		}
		return []Axiom{a}, nil
	case yamlstrict.SequenceKind:
		axioms := make([]Axiom, 0, len(node.Items))
		for _, item := range node.Items {
			if item.Kind != yamlstrict.ScalarKind {
				return nil, fmt.Errorf("cjtr: every %q entry must be a scalar", key)
			}
			a, err := parseAxiom(item.Value, concepts, key)
			if err != nil {
				return nil, err
			}
			axioms = append(axioms, a)
		}
		return axioms, nil
	default:
		return nil, fmt.Errorf("cjtr: %q must be a scalar or a list of scalars, got %v", key, node.Kind)
	}
}

func parseAxiom(text string, concepts map[string]Concept, key string) (Axiom, error) {
	stmt, err := ParseComparison(text, concepts)
	if err == nil {
		return Axiom{Statement: &stmt, Text: text}, nil
	}
	if errors.Is(err, ErrNoDeclaredConcept) {
		return Axiom{Text: text}, nil
	}
	return Axiom{}, fmt.Errorf("cjtr: %q: %w", key, err)
}
