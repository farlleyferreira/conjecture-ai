package cjtr

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Hypothesis is one hypothesis read from a rule's "hypotheses" mapping.
//
// A universal hypothesis declares "always" instead of "given"/"then": it
// has no Given and Otherwise, and Then holds the "always" predicate.
type Hypothesis struct {
	// Name is the hypothesis's natural-language key as written.
	Name string
	// ID is the hypothesis's declared slug, if any ("" otherwise - unlike
	// Rule, a hypothesis is not required to declare one; the
	// specification's own worked example has hypotheses both with and
	// without an explicit "id", and this package does not guess at a
	// slugification rule for the ones that omit it).
	ID string

	IsUniversal bool
	// Given is empty when IsUniversal is true. When not empty, every
	// statement must hold (there are no parentheses in the grammar: "and"
	// across given phrases is the only combination allowed).
	Given     []Statement
	Then      Statement
	Otherwise *Statement

	// Examples are concrete rows: concept name -> value as written, as a
	// literal - not yet as another concept's name (a reference, as
	// [Statement] values can be). The specification's own worked example
	// uses such a reference as a placeholder for "whatever the table
	// concept resolves to" in rows that don't supply a table concept's
	// by-keys; this package does not resolve that placeholder, since
	// nothing in the specification says what it should mean for a row
	// that genuinely cannot supply a concrete value (skip the check?
	// require it to still resolve with no by-key bindings and fail?).
	Examples []map[string]string
}

// Hypotheses reads a rule body's "hypotheses" mapping, if present. A rule
// with no "hypotheses" key returns (nil, nil): not every rule needs one
// (e.g. one that only states "unchanged" clauses).
func Hypotheses(ruleBody *yamlstrict.Node, concepts map[string]Concept) ([]Hypothesis, error) {
	hypsNode, ok := ruleBody.Get("hypotheses")
	if !ok {
		return nil, nil
	}
	if hypsNode.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: \"hypotheses\" must be a mapping of hypothesis name to body, got %v", hypsNode.Kind)
	}

	hyps := make([]Hypothesis, 0, len(hypsNode.Pairs))
	for _, p := range hypsNode.Pairs {
		h, err := parseHypothesis(p.Key, p.Value, concepts)
		if err != nil {
			return nil, err
		}
		hyps = append(hyps, h)
	}
	return hyps, nil
}

func parseHypothesis(name string, body *yamlstrict.Node, concepts map[string]Concept) (Hypothesis, error) {
	if body.Kind != yamlstrict.MappingKind {
		return Hypothesis{}, fmt.Errorf("cjtr: hypothesis %q must be a mapping, got %v", name, body.Kind)
	}

	h := Hypothesis{Name: name}

	if idNode, ok := body.Get("id"); ok {
		if idNode.Kind != yamlstrict.ScalarKind {
			return Hypothesis{}, fmt.Errorf("cjtr: hypothesis %q: \"id\" must be a scalar", name)
		}
		h.ID = idNode.Value
	}

	alwaysNode, hasAlways := body.Get("always")
	givenNode, hasGiven := body.Get("given")
	thenNode, hasThen := body.Get("then")

	switch {
	case hasAlways && (hasGiven || hasThen):
		return Hypothesis{}, fmt.Errorf("cjtr: hypothesis %q declares both \"always\" and \"given\"/\"then\"", name)
	case hasAlways:
		stmt, err := parseStatementField(alwaysNode, concepts, name, "always")
		if err != nil {
			return Hypothesis{}, err
		}
		h.IsUniversal = true
		h.Then = stmt
	case hasGiven && hasThen:
		given, err := parseGiven(givenNode, concepts, name)
		if err != nil {
			return Hypothesis{}, err
		}
		stmt, err := parseStatementField(thenNode, concepts, name, "then")
		if err != nil {
			return Hypothesis{}, err
		}
		h.Given = given
		h.Then = stmt
	default:
		return Hypothesis{}, fmt.Errorf("cjtr: hypothesis %q must declare either \"always\" or both \"given\" and \"then\"", name)
	}

	if otherwiseNode, ok := body.Get("otherwise"); ok {
		stmt, err := parseStatementField(otherwiseNode, concepts, name, "otherwise")
		if err != nil {
			return Hypothesis{}, err
		}
		h.Otherwise = &stmt
	}

	if examplesNode, ok := body.Get("examples"); ok {
		examples, err := parseExamples(examplesNode, name)
		if err != nil {
			return Hypothesis{}, err
		}
		h.Examples = examples
	}

	return h, nil
}

func parseStatementField(node *yamlstrict.Node, concepts map[string]Concept, hypName, field string) (Statement, error) {
	if node.Kind != yamlstrict.ScalarKind {
		return Statement{}, fmt.Errorf("cjtr: hypothesis %q: %q must be a scalar statement, got %v", hypName, field, node.Kind)
	}
	stmt, err := ParseComparison(node.Value, concepts)
	if err != nil {
		return Statement{}, fmt.Errorf("cjtr: hypothesis %q: %q: %w", hypName, field, err)
	}
	return stmt, nil
}

// parseGiven accepts "given" as a single statement or a list of
// statements, per the grammar table ("a phrase or a list of phrases, all
// true").
func parseGiven(node *yamlstrict.Node, concepts map[string]Concept, hypName string) ([]Statement, error) {
	switch node.Kind {
	case yamlstrict.ScalarKind:
		stmt, err := ParseComparison(node.Value, concepts)
		if err != nil {
			return nil, fmt.Errorf("cjtr: hypothesis %q: \"given\": %w", hypName, err)
		}
		return []Statement{stmt}, nil
	case yamlstrict.SequenceKind:
		stmts := make([]Statement, 0, len(node.Items))
		for _, item := range node.Items {
			if item.Kind != yamlstrict.ScalarKind {
				return nil, fmt.Errorf("cjtr: hypothesis %q: every \"given\" entry must be a scalar statement", hypName)
			}
			stmt, err := ParseComparison(item.Value, concepts)
			if err != nil {
				return nil, fmt.Errorf("cjtr: hypothesis %q: \"given\": %w", hypName, err)
			}
			stmts = append(stmts, stmt)
		}
		return stmts, nil
	default:
		return nil, fmt.Errorf("cjtr: hypothesis %q: \"given\" must be a scalar or a list of scalars, got %v", hypName, node.Kind)
	}
}

func parseExamples(node *yamlstrict.Node, hypName string) ([]map[string]string, error) {
	if node.Kind != yamlstrict.SequenceKind {
		return nil, fmt.Errorf("cjtr: hypothesis %q: \"examples\" must be a list, got %v", hypName, node.Kind)
	}

	examples := make([]map[string]string, 0, len(node.Items))
	for _, row := range node.Items {
		if row.Kind != yamlstrict.MappingKind {
			return nil, fmt.Errorf("cjtr: hypothesis %q: each example row must be a mapping, got %v", hypName, row.Kind)
		}
		m := make(map[string]string, len(row.Pairs))
		for _, p := range row.Pairs {
			if p.Value.Kind != yamlstrict.ScalarKind {
				return nil, fmt.Errorf("cjtr: hypothesis %q: example row value for %q must be a scalar", hypName, p.Key)
			}
			m[p.Key] = p.Value.Value
		}
		examples = append(examples, m)
	}
	return examples, nil
}

// CheckExample evaluates one example row against a hypothesis and reports
// whether the row is internally consistent: whichever branch applies to
// it (Then, if Given holds or the hypothesis is universal; Otherwise,
// otherwise) must itself evaluate to true against the row's own stated
// values. A false result means the row contradicts the rule it is
// supposed to illustrate.
func CheckExample(h Hypothesis, concepts map[string]Concept, tables Tables, example map[string]string) (bool, error) {
	bindings, err := bindingsFromExample(example, concepts)
	if err != nil {
		return false, err
	}

	applicable := h.Then
	if !h.IsUniversal {
		holds, err := allHold(h.Given, concepts, tables, bindings)
		if err != nil {
			return false, err
		}
		if !holds {
			if h.Otherwise == nil {
				return false, fmt.Errorf("cjtr: hypothesis %q: \"given\" does not hold for this example and there is no \"otherwise\"", h.Name)
			}
			applicable = *h.Otherwise
		}
	}

	return Evaluate(applicable, concepts, tables, bindings)
}

func allHold(statements []Statement, concepts map[string]Concept, tables Tables, bindings Bindings) (bool, error) {
	for _, s := range statements {
		ok, err := Evaluate(s, concepts, tables, bindings)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// bindingsFromExample turns an example row (concept name -> value as
// written) into [Bindings] (variable path -> value), by looking up each
// named concept's declared "variable" path.
func bindingsFromExample(row map[string]string, concepts map[string]Concept) (Bindings, error) {
	bindings := make(Bindings, len(row))
	for name, value := range row {
		c, ok := concepts[name]
		if !ok {
			return nil, fmt.Errorf("cjtr: example row references undeclared concept %q", name)
		}
		if c.Kind != VariableConcept {
			return nil, fmt.Errorf("cjtr: example row provides a value for %q, which is not a variable concept", name)
		}
		bindings[c.Variable] = value
	}
	return bindings, nil
}
