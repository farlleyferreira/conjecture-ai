package cjtr

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Rule is one rule read from a .cjtr file's top-level "rules" mapping.
//
// This only reads a rule's name, declared id, and "unchanged" - not its
// "when" or "user_defined_laws" (those need more of the vocabulary/grammar
// work to interpret correctly, and are not implemented yet).
type Rule struct {
	// Name is the rule's natural-language key as written, e.g.
	// "shipping is free above the threshold".
	Name string
	// ID is the rule's declared slug (its required "id" field), e.g.
	// "free-above".
	ID string
	// CompositeID is area::phenomenon::ID.
	CompositeID string
	// Unchanged lists the declared concept names a rule-level "nothing
	// else changes" clause names (grammar table: "unchanged" -> "list of
	// terms" -> "clauses \"nothing else changes\""; the specification's
	// worked example declares it directly under the rule, as a sibling of
	// "hypotheses" - not under an individual hypothesis - and its
	// "inheritance" rule says a rule's declarations apply to everything
	// inside it). Parsing only validates that every name is a declared
	// concept; it does not yet enforce the clause itself, since that
	// means comparing a term's value across a pair of executions
	// ("remains"/"does not change", in the comparator vocabulary), and
	// this package has no cross-execution state to compare against (that
	// is the still-unbuilt persistence/evidence-ledger capability, not
	// this one).
	Unchanged []string
	// Body is the rule's own mapping node, for reading fields this
	// package parses separately (e.g. [Hypotheses]) without re-walking
	// "rules" from the top-level document.
	Body *yamlstrict.Node
}

// Rules reads the top-level "rules" mapping of a strictly-loaded .cjtr
// document and returns each rule with its composite identifier.
//
// area and phenomenon are supplied by the caller, not read from the
// document: per the cjtr-language spec, a file's area is its directory
// (conjecture/phenomena/<area>/) and its phenomenon slug is its filename,
// neither of which this function has access to from the parsed content
// alone. concepts (as returned by [Concepts]) is used only to validate a
// rule's "unchanged" clause.
func Rules(root *yamlstrict.Node, area, phenomenon string, concepts map[string]Concept) ([]Rule, error) {
	rulesNode, ok := root.Get("rules")
	if !ok {
		return nil, fmt.Errorf("cjtr: missing required top-level \"rules\" key")
	}
	if rulesNode.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: \"rules\" must be a mapping of rule name to rule body, got %v", rulesNode.Kind)
	}

	rules := make([]Rule, 0, len(rulesNode.Pairs))
	for _, p := range rulesNode.Pairs {
		name := p.Key
		body := p.Value

		if body.Kind != yamlstrict.MappingKind {
			return nil, fmt.Errorf("cjtr: rule %q must be a mapping, got %v", name, body.Kind)
		}

		idNode, ok := body.Get("id")
		if !ok || idNode.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("cjtr: rule %q is missing its required \"id\" field", name)
		}

		composite, err := ComposeID(area, phenomenon, idNode.Value)
		if err != nil {
			return nil, fmt.Errorf("cjtr: rule %q: %w", name, err)
		}

		var unchanged []string
		if unchangedNode, ok := body.Get("unchanged"); ok {
			unchanged, err = parseUnchanged(unchangedNode, concepts, name)
			if err != nil {
				return nil, err
			}
		}

		rules = append(rules, Rule{Name: name, ID: idNode.Value, CompositeID: composite, Unchanged: unchanged, Body: body})
	}

	return rules, nil
}

// parseUnchanged reads "unchanged" (grammar table: a list of terms) and
// validates every entry names a declared concept.
func parseUnchanged(node *yamlstrict.Node, concepts map[string]Concept, ruleName string) ([]string, error) {
	if node.Kind != yamlstrict.SequenceKind {
		return nil, fmt.Errorf("cjtr: rule %q: \"unchanged\" must be a list, got %v", ruleName, node.Kind)
	}

	terms := make([]string, 0, len(node.Items))
	for _, item := range node.Items {
		if item.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("cjtr: rule %q: every \"unchanged\" entry must be a scalar term name", ruleName)
		}
		if _, ok := concepts[item.Value]; !ok {
			return nil, fmt.Errorf("cjtr: rule %q: \"unchanged\" names undeclared concept %q", ruleName, item.Value)
		}
		terms = append(terms, item.Value)
	}
	return terms, nil
}
