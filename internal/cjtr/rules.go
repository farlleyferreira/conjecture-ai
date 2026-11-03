package cjtr

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Rule is one rule read from a .cjtr file's top-level "rules" mapping.
//
// This only reads a rule's name, declared id, "when", and "unchanged" -
// not "steps" or "user_defined_laws" at the top of a file (for the whole
// phenomenon rather than one rule; see [UserDefinedLaws]'s own doc
// comment for the rule-level case, which this package does implement).
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
	// When names the observation operation this rule attaches to
	// (grammar table: "when" -> "<observation operation>" -> "Chamada
	// pelo contrato"), exactly as declared, if any ("" otherwise - not
	// every rule needs one). See [Observation.Operation]: the
	// specification's worked example has a file-level "observation"
	// mapping naming one operation, and its only rule's "when" repeats
	// that exact text; this package validates that match (see
	// [Rules]'s doc comment) rather than leaving "when" as a bare,
	// unchecked label.
	When string
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
//
// observation (as returned by [ReadObservation], nil if the file declares
// none) validates a rule's "when": the specification gives no other
// relationship between the two, and its worked example's rule "when"
// repeats the file's one declared "observation.operation" exactly, so a
// "when" that names anything else is rejected rather than silently
// accepted as an unrelated label.
func Rules(root *yamlstrict.Node, area, phenomenon string, concepts map[string]Concept, observation *Observation) ([]Rule, error) {
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

		var when string
		if whenNode, ok := body.Get("when"); ok {
			if whenNode.Kind != yamlstrict.ScalarKind {
				return nil, fmt.Errorf("cjtr: rule %q: \"when\" must be a scalar", name)
			}
			when = whenNode.Value
			if observation != nil && when != observation.Operation {
				return nil, fmt.Errorf(
					"cjtr: rule %q: \"when\" (%q) does not match the file's declared observation operation (%q)",
					name, when, observation.Operation,
				)
			}
		}

		rules = append(rules, Rule{Name: name, ID: idNode.Value, CompositeID: composite, Unchanged: unchanged, When: when, Body: body})
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
