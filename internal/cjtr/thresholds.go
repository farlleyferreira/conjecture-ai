package cjtr

import (
	"math/big"
	"sort"
	"strings"
)

// ThresholdValues returns, for each input concept h's own "given"
// conditions compare against a number, the values right at that
// number: the number itself, one step below it and one step above.
//
// [BoundaryValues] only knows a concept's declared range, so on its
// own directed sampling tested the edges of the domain ("0 hours",
// "720 hours") and never the edge the rule is actually about ("24
// hours") - exactly where the commonest mistake of all lives, ">"
// written for ">=". That was found for real: an implementation
// charging the fee at exactly 24.00 hours passed directed sampling
// and 40 random cases alike, a random draw landing on one value in
// 72,001 being what it is.
//
// The step is the concept's declared places when it has them, or
// otherwise the finest precision the threshold and the concept's
// range are themselves written at. Values falling outside a declared
// range are dropped: the rule says nothing about them.
//
// The result maps concept name to its threshold values, each list
// sorted and free of duplicates.
func ThresholdValues(h Hypothesis, concepts map[string]Concept) map[string][]string {
	found := map[string]map[string]bool{}
	for _, stmt := range h.Given {
		term, ok := concepts[stmt.Term]
		if !ok || term.Kind != VariableConcept || !strings.HasPrefix(term.Variable, "input.") {
			continue
		}
		for _, operand := range []struct {
			value     string
			isConcept bool
		}{{stmt.Value, stmt.ValueIsConcept}, {stmt.Value2, stmt.Value2IsConcept}} {
			literal := operand.value
			if operand.isConcept {
				constant, ok := concepts[operand.value]
				if !ok || constant.Kind != ConstantConcept {
					continue
				}
				literal = constant.Value
			}
			if literal == "" {
				continue
			}
			threshold, err := parseExactDecimal(literal)
			if err != nil {
				continue // a text comparison: no neighbouring values exist
			}

			places := term.Places
			if !term.HasPlaces {
				places = decimalPlaces(literal)
				for _, bound := range []string{term.Min, term.Max} {
					if p := decimalPlaces(bound); term.HasRange && p > places {
						places = p
					}
				}
			}
			step := unitStep(places)

			for _, candidate := range []*big.Rat{
				new(big.Rat).Sub(threshold, step), threshold, new(big.Rat).Add(threshold, step),
			} {
				if term.HasRange {
					min, errMin := parseExactDecimal(term.Min)
					max, errMax := parseExactDecimal(term.Max)
					if errMin != nil || errMax != nil || candidate.Cmp(min) < 0 || candidate.Cmp(max) > 0 {
						continue
					}
				}
				if found[stmt.Term] == nil {
					found[stmt.Term] = map[string]bool{}
				}
				found[stmt.Term][formatDecimal(candidate, places)] = true
			}
		}
	}

	out := make(map[string][]string, len(found))
	for name, set := range found {
		values := make([]string, 0, len(set))
		for v := range set {
			values = append(values, v)
		}
		sort.Strings(values)
		out[name] = values
	}
	return out
}
