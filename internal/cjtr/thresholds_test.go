package cjtr

import (
	"strings"
	"testing"
)

func thresholdConcepts() map[string]Concept {
	return map[string]Concept{
		"hours":  {Name: "hours", Kind: VariableConcept, Variable: "input.hours", Type: "decimal"},
		"class":  {Name: "class", Kind: VariableConcept, Variable: "input.class", Type: "text", OneOf: []string{"basic", "flex"}},
		"fee":    {Name: "fee", Kind: VariableConcept, Variable: "output.fee", Type: "decimal"},
		"window": {Name: "window", Kind: ConstantConcept, Value: "24.00"},
	}
}

func TestThresholdValues_AroundAConstantConcept(t *testing.T) {
	h := Hypothesis{Given: []Statement{
		{Term: "class", Comparator: "is", Value: "flex"},
		{Term: "hours", Comparator: "is at least", Value: "window", ValueIsConcept: true},
	}}
	got := ThresholdValues(h, thresholdConcepts())
	if joined := strings.Join(got["hours"], ","); joined != "23.99,24.00,24.01" {
		t.Errorf("ThresholdValues[hours] = %q, want \"23.99,24.00,24.01\"", joined)
	}
	if _, ok := got["class"]; ok {
		t.Errorf("ThresholdValues has an entry for a text comparison: %v", got["class"])
	}
}

func TestThresholdValues_LiteralAndDeclaredPlaces(t *testing.T) {
	concepts := thresholdConcepts()
	hours := concepts["hours"]
	hours.HasPlaces, hours.Places = true, 0
	concepts["hours"] = hours

	h := Hypothesis{Given: []Statement{{Term: "hours", Comparator: "is above", Value: "24"}}}
	if joined := strings.Join(ThresholdValues(h, concepts)["hours"], ","); joined != "23,24,25" {
		t.Errorf("ThresholdValues[hours] = %q, want \"23,24,25\"", joined)
	}
}

func TestThresholdValues_ClampedToTheDeclaredRange(t *testing.T) {
	concepts := thresholdConcepts()
	hours := concepts["hours"]
	hours.HasRange, hours.Min, hours.Max = true, "24.00", "720.00"
	concepts["hours"] = hours

	h := Hypothesis{Given: []Statement{{Term: "hours", Comparator: "is at least", Value: "24.00"}}}
	if joined := strings.Join(ThresholdValues(h, concepts)["hours"], ","); joined != "24.00,24.01" {
		t.Errorf("ThresholdValues[hours] = %q, want \"24.00,24.01\" (23.99 is outside the range)", joined)
	}
}

func TestThresholdValues_BothEndsOfABetween(t *testing.T) {
	h := Hypothesis{Given: []Statement{{Term: "hours", Comparator: "is between", Value: "1.0", Value2: "2.0"}}}
	if joined := strings.Join(ThresholdValues(h, thresholdConcepts())["hours"], ","); joined != "0.9,1.0,1.1,1.9,2.0,2.1" {
		t.Errorf("ThresholdValues[hours] = %q, want both thresholds' neighbours", joined)
	}
}

func TestThresholdValues_IgnoresOutputsAndUnknownTerms(t *testing.T) {
	h := Hypothesis{Given: []Statement{
		{Term: "fee", Comparator: "is at least", Value: "10.00"},
		{Term: "nope", Comparator: "is at least", Value: "10.00"},
	}}
	if got := ThresholdValues(h, thresholdConcepts()); len(got) != 0 {
		t.Errorf("ThresholdValues = %v, want nothing for an output or an undeclared term", got)
	}
}
