package cjtr

import (
	"fmt"
	"strconv"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// ConceptKind distinguishes the shapes a .cjtr concept can take.
type ConceptKind int

const (
	// VariableConcept is bound to an input/output variable at observation
	// time (declares "variable", e.g. "input.discounted_subtotal").
	VariableConcept ConceptKind = iota
	// ConstantConcept is a fixed value declared in the file itself
	// (declares "value" instead of "variable").
	ConstantConcept
)

// Concept is one entry of a .cjtr file's top-level "concepts" mapping: a
// named, typed quantity that statements refer to by its key phrase.
//
// Table-lookup concepts (declaring "table" and "by") are not supported yet
// - [Concepts] returns a clear error for one rather than silently
// misreading it.
type Concept struct {
	// Name is the concept's key phrase as written, e.g. "discounted subtotal".
	Name string
	// Label is the concept's Portuguese translation.
	Label string
	Kind  ConceptKind

	// Variable is the dotted input/output path (VariableConcept only),
	// e.g. "input.discounted_subtotal".
	Variable string
	// Value is the constant's raw text (ConstantConcept only), kept as
	// text rather than converted to a float so an exact decimal like
	// "100.00" is never rounded.
	Value string

	// Type is "decimal" or "text", when declared. A concept may instead
	// declare OneOf, which implies a text type with a closed set of
	// allowed values.
	Type string
	Unit string
	// HasPlaces reports whether Places was declared.
	HasPlaces bool
	Places    int
	// HasRange reports whether Min/Max were declared. Both are kept as raw
	// text for the same exact-decimal reason as Value.
	HasRange bool
	Min, Max string
	// OneOf lists the allowed values of a closed-set text concept, when declared.
	OneOf []string
}

// Concepts reads the top-level "concepts" mapping of a strictly-loaded
// .cjtr document.
func Concepts(root *yamlstrict.Node) (map[string]Concept, error) {
	conceptsNode, ok := root.Get("concepts")
	if !ok {
		return nil, fmt.Errorf("cjtr: missing required top-level \"concepts\" key")
	}
	if conceptsNode.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: \"concepts\" must be a mapping of concept name to definition, got %v", conceptsNode.Kind)
	}

	concepts := make(map[string]Concept, len(conceptsNode.Pairs))
	for _, p := range conceptsNode.Pairs {
		c, err := parseConcept(p.Key, p.Value)
		if err != nil {
			return nil, err
		}
		concepts[p.Key] = c
	}

	return concepts, nil
}

func parseConcept(name string, body *yamlstrict.Node) (Concept, error) {
	if body.Kind != yamlstrict.MappingKind {
		return Concept{}, fmt.Errorf("cjtr: concept %q must be a mapping, got %v", name, body.Kind)
	}

	if _, ok := body.Get("table"); ok {
		return Concept{}, fmt.Errorf("cjtr: concept %q: table-lookup concepts are not supported yet", name)
	}

	c := Concept{Name: name}

	if label, ok := scalarField(body, name, "label"); ok {
		c.Label = label
	}
	if unit, ok := scalarField(body, name, "unit"); ok {
		c.Unit = unit
	}
	if typ, ok := scalarField(body, name, "type"); ok {
		c.Type = typ
	}

	variable, hasVariable := scalarField(body, name, "variable")
	value, hasValue := scalarField(body, name, "value")

	switch {
	case hasVariable && hasValue:
		return Concept{}, fmt.Errorf("cjtr: concept %q declares both \"variable\" and \"value\"", name)
	case hasVariable:
		c.Kind = VariableConcept
		c.Variable = variable
	case hasValue:
		c.Kind = ConstantConcept
		c.Value = value
	default:
		return Concept{}, fmt.Errorf("cjtr: concept %q must declare either \"variable\" or \"value\"", name)
	}

	if placesNode, ok := body.Get("places"); ok {
		if placesNode.Kind != yamlstrict.ScalarKind {
			return Concept{}, fmt.Errorf("cjtr: concept %q: \"places\" must be a scalar", name)
		}
		n, err := strconv.Atoi(placesNode.Value)
		if err != nil || n < 0 {
			return Concept{}, fmt.Errorf("cjtr: concept %q: invalid \"places\" %q: want a non-negative integer", name, placesNode.Value)
		}
		c.HasPlaces = true
		c.Places = n
	}

	if rangeNode, ok := body.Get("range"); ok {
		if rangeNode.Kind != yamlstrict.MappingKind {
			return Concept{}, fmt.Errorf("cjtr: concept %q: \"range\" must be a mapping with \"min\" and \"max\"", name)
		}
		min, ok := scalarField(rangeNode, name, "min")
		if !ok {
			return Concept{}, fmt.Errorf("cjtr: concept %q: \"range\" is missing \"min\"", name)
		}
		max, ok := scalarField(rangeNode, name, "max")
		if !ok {
			return Concept{}, fmt.Errorf("cjtr: concept %q: \"range\" is missing \"max\"", name)
		}
		c.HasRange = true
		c.Min, c.Max = min, max
	}

	if oneOfNode, ok := body.Get("one_of"); ok {
		if oneOfNode.Kind != yamlstrict.SequenceKind {
			return Concept{}, fmt.Errorf("cjtr: concept %q: \"one_of\" must be a sequence", name)
		}
		for _, item := range oneOfNode.Items {
			if item.Kind != yamlstrict.ScalarKind {
				return Concept{}, fmt.Errorf("cjtr: concept %q: every \"one_of\" entry must be a scalar", name)
			}
			c.OneOf = append(c.OneOf, item.Value)
		}
	}

	return c, nil
}

// scalarField reads a mapping field expected to be a scalar, reporting
// whether it was present.
func scalarField(body *yamlstrict.Node, conceptName, field string) (string, bool) {
	node, ok := body.Get(field)
	if !ok {
		return "", false
	}
	if node.Kind != yamlstrict.ScalarKind {
		return "", false
	}
	return node.Value, true
}
