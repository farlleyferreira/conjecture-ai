package cjtr

import (
	"fmt"
	"math/big"

	"github.com/farlleyferreira/conjecture-ai/internal/expressioncore"
)

// Bindings maps a variable path exactly as declared by a VariableConcept's
// "variable" field (e.g. "input.discounted_subtotal") to its observed
// value, as raw text - the same exact-decimal-preserving text every other
// part of this package passes around, never a parsed float.
type Bindings map[string]string

// Evaluate resolves a parsed [Statement]'s term and value against a
// file's declared concepts and a set of observed bindings, and reports
// whether the comparison holds.
//
// Concept/table resolution (constant, variable, table lookup) stays
// this package's own job - resolveValue/resolveStatementValue below,
// unchanged. The final comparison step itself (compare, below)
// delegates to internal/expressioncore: a decimal concept's values are
// compared via [expressioncore.DecimalType] (exact, via [math/big.Rat]
// underneath - never through a lossy float64 conversion, per the
// exact-decimal-arithmetic requirement that runs through the whole
// specification) and anything else via CEL's own native string
// equality, so this package no longer evaluates a comparison by hand -
// it compiles to CEL, the specification's own chosen expression
// engine, like every other statement is meant to.
func Evaluate(stmt Statement, concepts map[string]Concept, tables Tables, bindings Bindings) (bool, error) {
	termConcept, ok := concepts[stmt.Term]
	if !ok {
		return false, fmt.Errorf("cjtr: statement's term %q is not a declared concept", stmt.Term)
	}

	termValue, err := resolveValue(termConcept, concepts, tables, bindings)
	if err != nil {
		return false, fmt.Errorf("cjtr: resolving term %q: %w", stmt.Term, err)
	}

	valueValue, err := resolveStatementValue(stmt.Value, stmt.ValueIsConcept, concepts, tables, bindings)
	if err != nil {
		return false, fmt.Errorf("cjtr: resolving value %q: %w", stmt.Value, err)
	}

	if !twoValueComparators[stmt.Comparator] {
		return compare(stmt.Comparator, termConcept, termValue, valueValue, "")
	}

	value2Value, err := resolveStatementValue(stmt.Value2, stmt.Value2IsConcept, concepts, tables, bindings)
	if err != nil {
		return false, fmt.Errorf("cjtr: resolving value %q: %w", stmt.Value2, err)
	}
	return compare(stmt.Comparator, termConcept, termValue, valueValue, value2Value)
}

// resolveStatementValue returns a statement side's value as raw text: the
// literal text itself, or (when isConcept) that concept's resolved value.
func resolveStatementValue(text string, isConcept bool, concepts map[string]Concept, tables Tables, bindings Bindings) (string, error) {
	if !isConcept {
		return text, nil
	}
	concept, ok := concepts[text]
	if !ok {
		return "", fmt.Errorf("%q is not a declared concept", text)
	}
	return resolveValue(concept, concepts, tables, bindings)
}

// resolveValue returns a concept's current value as raw text: a constant
// concept's declared value, a variable concept's bound value, or a table
// concept's looked-up value (see [Table.Lookup]).
func resolveValue(c Concept, concepts map[string]Concept, tables Tables, bindings Bindings) (string, error) {
	switch c.Kind {
	case ConstantConcept:
		return c.Value, nil
	case VariableConcept:
		value, ok := bindings[c.Variable]
		if !ok {
			return "", fmt.Errorf("no binding for variable %q", c.Variable)
		}
		return value, nil
	case TableConcept:
		table, ok := tables[c.Name]
		if !ok {
			return "", fmt.Errorf("no loaded table for concept %q", c.Name)
		}
		keys := make(map[string]string, len(c.ByKeys))
		for _, keyName := range c.ByKeys {
			keyConcept, ok := concepts[keyName]
			if !ok {
				return "", fmt.Errorf("table concept %q: by-key %q is not a declared concept", c.Name, keyName)
			}
			v, err := resolveValue(keyConcept, concepts, tables, bindings)
			if err != nil {
				return "", fmt.Errorf("table concept %q: resolving by-key %q: %w", c.Name, keyName, err)
			}
			keys[keyName] = v
		}
		return table.Lookup(keys, concepts)
	default:
		return "", fmt.Errorf("resolveValue: concept %q has an unknown kind", c.Name)
	}
}

// compare applies comparator to termValue and valueValue via
// internal/expressioncore - a CEL program, compiled once per
// (comparator, operand type) pair and cached there, not re-compiled
// on every call. value2Value is only used by the two-value
// comparators ("is between"/"is strictly between"), where it is the
// upper bound (Y) and valueValue is the lower bound (X); an empty
// value2Value is expressioncore's own "not supplied" convention for
// every other comparator, matching how [Evaluate] itself only ever
// resolves a real Value2 for those two comparators.
//
// Which operand type to compile against is this function's own
// decision, not expressioncore's: an ordering comparator ("is greater
// than" and the rest) always parses its operands as exact decimals
// regardless of the term concept's declared type - this package's own
// prior, hand-written compare() did exactly this unconditionally via
// numericCompare, so this preserves that rule rather than changing
// it - and only an equality comparator ever compares plain text, when
// termConcept is not itself a decimal concept.
func compare(comparator string, termConcept Concept, termValue, valueValue, value2Value string) (bool, error) {
	decimal := true
	switch comparator {
	case "is", "equals", "is not", "does not equal":
		decimal = termConcept.Type == "decimal"
	}

	if decimal {
		return compareDecimal(comparator, termValue, valueValue, value2Value)
	}
	return compareText(comparator, termValue, valueValue, value2Value)
}

func compareDecimal(comparator, termValue, valueValue, value2Value string) (bool, error) {
	prg, err := expressioncore.CompileComparison(comparator, expressioncore.DecimalType)
	if err != nil {
		return false, fmt.Errorf("evaluate: %w", err)
	}
	term, err := expressioncore.NewDecimal(termValue)
	if err != nil {
		return false, err
	}
	value, err := expressioncore.NewDecimal(valueValue)
	if err != nil {
		return false, err
	}
	var value2 any = expressioncore.Decimal{}
	if value2Value != "" {
		if value2, err = expressioncore.NewDecimal(value2Value); err != nil {
			return false, err
		}
	}
	return expressioncore.EvalComparison(prg, term, value, value2)
}

func compareText(comparator, termValue, valueValue, value2Value string) (bool, error) {
	prg, err := expressioncore.CompileComparison(comparator, expressioncore.StringType)
	if err != nil {
		return false, fmt.Errorf("evaluate: %w", err)
	}
	return expressioncore.EvalComparison(prg, termValue, valueValue, value2Value)
}

// valuesEqual compares two values for a decimal concept exactly (never via
// float), or as plain text for any other concept type.
func valuesEqual(termConcept Concept, a, b string) (bool, error) {
	if termConcept.Type == "decimal" {
		ra, err := parseExactDecimal(a)
		if err != nil {
			return false, err
		}
		rb, err := parseExactDecimal(b)
		if err != nil {
			return false, err
		}
		return ra.Cmp(rb) == 0, nil
	}
	return a == b, nil
}

// parseExactDecimal parses s as an exact decimal number. big.Rat parses
// decimal text exactly (e.g. "100.00" becomes precisely 100, not a
// binary-float approximation), which is the whole reason this package
// never converts a monetary or other exact-decimal value through float64.
func parseExactDecimal(s string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("%q is not a valid exact decimal number", s)
	}
	return r, nil
}
