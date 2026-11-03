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
	// TableConcept is resolved by looking a row up in an external CSV file
	// (declares "table" and "by", e.g. the specification's "table shipping
	// fee" concept). See [Table.Lookup] for how a row is matched and which
	// column holds the value - the specification's worked example never
	// names that column, so this package picks the only reading that does
	// not invent a naming convention it doesn't declare: whichever single
	// column isn't one of the "by" keys.
	TableConcept
)

// Concept is one entry of a .cjtr file's top-level "concepts" mapping: a
// named, typed quantity that statements refer to by its key phrase.
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

	// TablePath is the table's CSV path, exactly as written (TableConcept
	// only), e.g. "bases/shipping-table.csv". It is relative to the
	// conjecture tree's root (the parent of "phenomena/"), matching the
	// path comment at the top of the specification's own worked example
	// file and the "bases/" directory that path implies alongside
	// "phenomena/" - resolving and loading it is the caller's job (see
	// [LoadTables]), not [Concepts]'s, since this package's parsing never
	// touches disk.
	TablePath string
	// ByKeys names the table's lookup-key concepts, in declaration order
	// (TableConcept only). Each name is also a regular entry of the same
	// map [Concepts] returns, as an ordinary VariableConcept - a by-key is
	// a concept in every sense except where it's declared.
	ByKeys []string
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
		c, err := parseConcept(p.Key, p.Value, concepts)
		if err != nil {
			return nil, err
		}
		if err := addConcept(concepts, p.Key, c); err != nil {
			return nil, err
		}
	}

	return concepts, nil
}

// addConcept inserts a concept by name, rejecting a collision with one
// already present - including a by-key a table concept added earlier,
// since those share the same namespace as every other concept.
func addConcept(concepts map[string]Concept, name string, c Concept) error {
	if _, exists := concepts[name]; exists {
		return fmt.Errorf("cjtr: concept %q is declared more than once (possibly as another table concept's by-key)", name)
	}
	concepts[name] = c
	return nil
}

func parseConcept(name string, body *yamlstrict.Node, concepts map[string]Concept) (Concept, error) {
	if body.Kind != yamlstrict.MappingKind {
		return Concept{}, fmt.Errorf("cjtr: concept %q must be a mapping, got %v", name, body.Kind)
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

	tablePath, hasTable := scalarField(body, name, "table")
	variable, hasVariable := scalarField(body, name, "variable")
	value, hasValue := scalarField(body, name, "value")

	switch {
	case hasTable && (hasVariable || hasValue):
		return Concept{}, fmt.Errorf("cjtr: concept %q declares \"table\" together with \"variable\" or \"value\"", name)
	case hasVariable && hasValue:
		return Concept{}, fmt.Errorf("cjtr: concept %q declares both \"variable\" and \"value\"", name)
	case hasTable:
		byNode, ok := body.Get("by")
		if !ok {
			return Concept{}, fmt.Errorf("cjtr: concept %q declares \"table\" but is missing \"by\"", name)
		}
		if byNode.Kind != yamlstrict.MappingKind {
			return Concept{}, fmt.Errorf("cjtr: concept %q: \"by\" must be a mapping of key name to concept, got %v", name, byNode.Kind)
		}
		c.Kind = TableConcept
		c.TablePath = tablePath
		for _, p := range byNode.Pairs {
			keyConcept, err := parseByKey(name, p.Key, p.Value)
			if err != nil {
				return Concept{}, err
			}
			if err := addConcept(concepts, p.Key, keyConcept); err != nil {
				return Concept{}, err
			}
			c.ByKeys = append(c.ByKeys, p.Key)
		}
	case hasVariable:
		c.Kind = VariableConcept
		c.Variable = variable
	case hasValue:
		c.Kind = ConstantConcept
		c.Value = value
	default:
		return Concept{}, fmt.Errorf("cjtr: concept %q must declare either \"variable\", \"value\", or \"table\"", name)
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

// parseByKey parses one entry of a table concept's "by" mapping - a
// nested concept declaring "label", "variable", "type", and "unit" (the
// specification's worked example never gives a by-key a "value", "range",
// "one_of", or "places", so this does not parse those either).
func parseByKey(tableName, keyName string, body *yamlstrict.Node) (Concept, error) {
	if body.Kind != yamlstrict.MappingKind {
		return Concept{}, fmt.Errorf("cjtr: concept %q: by-key %q must be a mapping, got %v", tableName, keyName, body.Kind)
	}

	variable, ok := scalarField(body, keyName, "variable")
	if !ok {
		return Concept{}, fmt.Errorf("cjtr: concept %q: by-key %q must declare \"variable\"", tableName, keyName)
	}

	c := Concept{Name: keyName, Kind: VariableConcept, Variable: variable}
	if label, ok := scalarField(body, keyName, "label"); ok {
		c.Label = label
	}
	if unit, ok := scalarField(body, keyName, "unit"); ok {
		c.Unit = unit
	}
	if typ, ok := scalarField(body, keyName, "type"); ok {
		c.Type = typ
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
