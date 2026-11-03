package cjtr

import (
	"fmt"
	"math/big"
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
// Decimal concepts are compared exactly, via [math/big.Rat] - never
// through a lossy float64 conversion, per the exact-decimal-arithmetic
// requirement that runs through the whole specification. This is a direct
// evaluator, not yet a compiler to CEL (the specification's chosen
// expression engine, not started): the observable result is the same
// either way, but nothing here should be mistaken for the real
// expression-core capability.
func Evaluate(stmt Statement, concepts map[string]Concept, tables Tables, bindings Bindings) (bool, error) {
	termConcept, ok := concepts[stmt.Term]
	if !ok {
		return false, fmt.Errorf("cjtr: statement's term %q is not a declared concept", stmt.Term)
	}

	termValue, err := resolveValue(termConcept, concepts, tables, bindings)
	if err != nil {
		return false, fmt.Errorf("cjtr: resolving term %q: %w", stmt.Term, err)
	}

	valueValue := stmt.Value
	if stmt.ValueIsConcept {
		valueConcept, ok := concepts[stmt.Value]
		if !ok {
			return false, fmt.Errorf("cjtr: statement's value %q is not a declared concept", stmt.Value)
		}
		valueValue, err = resolveValue(valueConcept, concepts, tables, bindings)
		if err != nil {
			return false, fmt.Errorf("cjtr: resolving value %q: %w", stmt.Value, err)
		}
	}

	return compare(stmt.Comparator, termConcept, termValue, valueValue)
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

func compare(comparator string, termConcept Concept, termValue, valueValue string) (bool, error) {
	switch comparator {
	case "is", "equals":
		return valuesEqual(termConcept, termValue, valueValue)
	case "is not", "does not equal":
		eq, err := valuesEqual(termConcept, termValue, valueValue)
		return !eq, err
	case "is greater than":
		return numericCompare(termValue, valueValue, func(c int) bool { return c > 0 })
	case "is less than":
		return numericCompare(termValue, valueValue, func(c int) bool { return c < 0 })
	case "is at least":
		return numericCompare(termValue, valueValue, func(c int) bool { return c >= 0 })
	case "is at most":
		return numericCompare(termValue, valueValue, func(c int) bool { return c <= 0 })
	default:
		// ParseComparison only ever produces a Statement with one of the
		// comparators above (see statement.go's supportedComparators), so
		// reaching this means the two files have fallen out of sync.
		return false, fmt.Errorf("evaluate: comparator %q has no evaluation rule", comparator)
	}
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

// numericCompare compares a and b as exact decimals and applies ok to the
// result of [big.Rat.Cmp] (negative, zero, or positive).
func numericCompare(a, b string, ok func(cmp int) bool) (bool, error) {
	ra, err := parseExactDecimal(a)
	if err != nil {
		return false, err
	}
	rb, err := parseExactDecimal(b)
	if err != nil {
		return false, err
	}
	return ok(ra.Cmp(rb)), nil
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
