package omissiondetection

import (
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
)

// rangedInputConcepts declares one ranged decimal input concept
// (0.00..100.00, 2 places) and one output concept - the same shape
// conformance-cycle's own directed-sampling tests use, so
// [conformancecycle.DirectedSamplingInputs] has real boundary values
// (Min, Max, justAboveMin, justBelowMax) to generate from.
func rangedInputConcepts() map[string]cjtr.Concept {
	return map[string]cjtr.Concept{
		"input value": {
			Name: "input value", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal",
			HasRange: true, Min: "0.00", Max: "100.00", HasPlaces: true, Places: 2,
		},
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
}

func parseComparison(t *testing.T, text string, concepts map[string]cjtr.Concept) cjtr.Statement {
	t.Helper()
	stmt, err := cjtr.ParseComparison(text, concepts)
	if err != nil {
		t.Fatalf("ParseComparison(%q) returned unexpected error: %v", text, err)
	}
	return stmt
}

func TestCheckTypeB_PartialRangeCoverageFlagsTheUncoveredBoundary(t *testing.T) {
	concepts := rangedInputConcepts()
	given := parseComparison(t, "the input value is less than 50.00", concepts)
	then := parseComparison(t, "the output value is 1.00", concepts)
	h := cjtr.Hypothesis{
		Name: "low range", Given: []cjtr.Statement{given}, Then: then,
		Examples: []map[string]string{{"input value": "10.00", "output value": "1.00"}},
	}

	omissions, err := CheckTypeB([]cjtr.Hypothesis{h}, concepts, nil)
	if err != nil {
		t.Fatalf("CheckTypeB returned unexpected error: %v", err)
	}

	// The declared range is 0.00..100.00, but this one hypothesis only
	// covers values below 50.00: its own Max (100.00) and justBelowMax
	// (99.99) boundary candidates fall outside "< 50.00" and nothing
	// else addresses them.
	got := make(map[string]bool, len(omissions))
	for _, o := range omissions {
		got[o.Input["input.x"]] = true
	}
	for _, want := range []string{"100.00", "99.99"} {
		if !got[want] {
			t.Errorf("CheckTypeB did not flag input.x=%s as a Type B omission; omissions: %+v", want, omissions)
		}
	}
	for _, covered := range []string{"0.00", "0.01"} {
		if got[covered] {
			t.Errorf("CheckTypeB flagged input.x=%s, which \"< 50.00\" already covers", covered)
		}
	}
}

func TestCheckTypeB_GivenThenOtherwiseCoversTheWholeDomain(t *testing.T) {
	// A single hypothesis with "given X then Y otherwise Z" defines a
	// result across its own Given's entire complement too, not only
	// within it - this is the exact shape a real .cjtr file (a member-
	// loyalty discount rule) surfaced as a false Type B omission
	// before addressedByAny learned to read Otherwise as coverage.
	concepts := rangedInputConcepts()
	given := parseComparison(t, "the input value is at least 50.00", concepts)
	then := parseComparison(t, "the output value is 1.00", concepts)
	otherwise := parseComparison(t, "the output value is 0.00", concepts)
	h := cjtr.Hypothesis{
		Name: "with otherwise", Given: []cjtr.Statement{given}, Then: then, Otherwise: &otherwise,
		Examples: []map[string]string{{"input value": "75.00", "output value": "1.00"}},
	}

	omissions, err := CheckTypeB([]cjtr.Hypothesis{h}, concepts, nil)
	if err != nil {
		t.Fatalf("CheckTypeB returned unexpected error: %v", err)
	}
	if len(omissions) != 0 {
		t.Errorf("CheckTypeB = %+v, want no omissions (otherwise covers everything below 50.00 too)", omissions)
	}
}

func TestCheckTypeB_FullRangeCoverageAcrossTwoHypothesesFindsNoOmission(t *testing.T) {
	concepts := rangedInputConcepts()
	lowGiven := parseComparison(t, "the input value is less than 50.00", concepts)
	lowThen := parseComparison(t, "the output value is 1.00", concepts)
	low := cjtr.Hypothesis{
		Name: "low range", Given: []cjtr.Statement{lowGiven}, Then: lowThen,
		Examples: []map[string]string{{"input value": "10.00", "output value": "1.00"}},
	}

	highGiven := parseComparison(t, "the input value is at least 50.00", concepts)
	highThen := parseComparison(t, "the output value is 2.00", concepts)
	high := cjtr.Hypothesis{
		Name: "high range", Given: []cjtr.Statement{highGiven}, Then: highThen,
		Examples: []map[string]string{{"input value": "90.00", "output value": "2.00"}},
	}

	omissions, err := CheckTypeB([]cjtr.Hypothesis{low, high}, concepts, nil)
	if err != nil {
		t.Fatalf("CheckTypeB returned unexpected error: %v", err)
	}
	if len(omissions) != 0 {
		t.Errorf("CheckTypeB = %+v, want no omissions (the two hypotheses jointly cover the whole range)", omissions)
	}
}

func TestCheckTypeB_UniversalHypothesisAddressesEveryCase(t *testing.T) {
	concepts := rangedInputConcepts()
	narrowGiven := parseComparison(t, "the input value is less than 50.00", concepts)
	narrowThen := parseComparison(t, "the output value is 1.00", concepts)
	narrow := cjtr.Hypothesis{
		Name: "low range", Given: []cjtr.Statement{narrowGiven}, Then: narrowThen,
		Examples: []map[string]string{{"input value": "10.00", "output value": "1.00"}},
	}

	universalThen := parseComparison(t, "the output value is 3.00", concepts)
	universal := cjtr.Hypothesis{
		Name: "catch-all", IsUniversal: true, Then: universalThen,
		Examples: []map[string]string{{"input value": "50.00", "output value": "3.00"}},
	}

	omissions, err := CheckTypeB([]cjtr.Hypothesis{narrow, universal}, concepts, nil)
	if err != nil {
		t.Fatalf("CheckTypeB returned unexpected error: %v", err)
	}
	if len(omissions) != 0 {
		t.Errorf("CheckTypeB = %+v, want no omissions (the universal hypothesis addresses every case)", omissions)
	}
}

func TestCheckTypeB_HypothesisWithNoExampleIsNotAGenerator(t *testing.T) {
	concepts := rangedInputConcepts()
	given := parseComparison(t, "the input value is less than 50.00", concepts)
	then := parseComparison(t, "the output value is 1.00", concepts)
	h := cjtr.Hypothesis{Name: "no example", Given: []cjtr.Statement{given}, Then: then}

	omissions, err := CheckTypeB([]cjtr.Hypothesis{h}, concepts, nil)
	if err != nil {
		t.Fatalf("CheckTypeB returned unexpected error: %v", err)
	}
	if len(omissions) != 0 {
		t.Errorf("CheckTypeB = %+v, want no omissions (nothing to generate from without a worked example)", omissions)
	}
}

func TestCheckTypeB_DeduplicatesTheSameUncoveredCase(t *testing.T) {
	concepts := rangedInputConcepts()
	given := parseComparison(t, "the input value is less than 50.00", concepts)
	then1 := parseComparison(t, "the output value is 1.00", concepts)
	h1 := cjtr.Hypothesis{
		Name: "low range a", Given: []cjtr.Statement{given}, Then: then1,
		Examples: []map[string]string{{"input value": "10.00", "output value": "1.00"}},
	}
	then2 := parseComparison(t, "the output value is 1.00", concepts)
	h2 := cjtr.Hypothesis{
		Name: "low range b", Given: []cjtr.Statement{given}, Then: then2,
		Examples: []map[string]string{{"input value": "20.00", "output value": "1.00"}},
	}

	omissions, err := CheckTypeB([]cjtr.Hypothesis{h1, h2}, concepts, nil)
	if err != nil {
		t.Fatalf("CheckTypeB returned unexpected error: %v", err)
	}

	seen := make(map[string]int)
	for _, o := range omissions {
		seen[o.Input["input.x"]]++
	}
	for value, count := range seen {
		if count != 1 {
			t.Errorf("input.x=%s reported %d times, want 1 (deduplicated)", value, count)
		}
	}
}
