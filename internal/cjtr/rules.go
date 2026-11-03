package cjtr

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Rule is one rule read from a .cjtr file's top-level "rules" mapping.
//
// This only reads a rule's name and declared id - not its "when",
// "unchanged", "user_defined_laws", or "hypotheses" (those need the
// vocabulary/grammar work to interpret correctly, and are not implemented
// yet).
type Rule struct {
	// Name is the rule's natural-language key as written, e.g.
	// "shipping is free above the threshold".
	Name string
	// ID is the rule's declared slug (its required "id" field), e.g.
	// "free-above".
	ID string
	// CompositeID is area::phenomenon::ID.
	CompositeID string
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
// alone.
func Rules(root *yamlstrict.Node, area, phenomenon string) ([]Rule, error) {
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

		rules = append(rules, Rule{Name: name, ID: idNode.Value, CompositeID: composite, Body: body})
	}

	return rules, nil
}
