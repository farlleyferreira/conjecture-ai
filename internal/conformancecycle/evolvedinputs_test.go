package conformancecycle

import (
	"strconv"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
)

// unboundedThresholdHypothesis returns a hypothesis whose one input
// concept ("discounted subtotal", input.x) declares neither a range
// nor a one_of - [cjtr.BoundaryValues]/[cjtr.ExhaustiveValues] both
// contribute nothing for it - with a baseline far enough from the
// given condition's own literal threshold (100.00) that
// EvolveForBehaviorChange can cross it well within
// evolveMaxGenerations at evolveStepSize.
func unboundedThresholdHypothesis(t *testing.T) (cjtr.Hypothesis, map[string]cjtr.Concept) {
	t.Helper()
	concepts := map[string]cjtr.Concept{
		"discounted subtotal": {
			Name: "discounted subtotal", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal",
		},
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	given, err := cjtr.ParseComparison("the discounted subtotal is greater than 100.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	then, err := cjtr.ParseComparison("the shipping fee is 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", Given: []cjtr.Statement{given}, Then: then,
		Examples: []map[string]string{{"discounted subtotal": "95.00", "shipping fee": "0.00"}},
	}
	return h, concepts
}

func TestEvolvedInputs_FindsCrossingWhenLiteralBoundaryKnown(t *testing.T) {
	h, concepts := unboundedThresholdHypothesis(t)

	cases, found, err := EvolvedInputs(h, concepts, nil)
	if err != nil {
		t.Fatalf("EvolvedInputs returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true (the search should cross the literal threshold of 100.00)")
	}
	if len(cases) != 1 {
		t.Fatalf("len(cases) = %d, want 1", len(cases))
	}
	x, err := strconv.ParseFloat(cases[0]["input.x"], 64)
	if err != nil {
		t.Fatalf("cases[0][\"input.x\"] = %q is not a number: %v", cases[0]["input.x"], err)
	}
	if x <= 100.00 {
		t.Errorf("cases[0][\"input.x\"] = %v, want a value that crosses \"greater than 100.00\"", x)
	}
	if cases[0]["output.result"] != "0.00" {
		t.Errorf("case output.result = %q, want the baseline's %q untouched", cases[0]["output.result"], "0.00")
	}

	// Confirm the crossing is real: evaluate the given condition at
	// both the baseline and the evolved value.
	baselineGiven, err := cjtr.Evaluate(h.Given[0], concepts, nil, cjtr.Bindings{"input.x": "95.00"})
	if err != nil {
		t.Fatalf("Evaluate returned unexpected error: %v", err)
	}
	evolvedGiven, err := cjtr.Evaluate(h.Given[0], concepts, nil, cjtr.Bindings{"input.x": cases[0]["input.x"]})
	if err != nil {
		t.Fatalf("Evaluate returned unexpected error: %v", err)
	}
	if baselineGiven == evolvedGiven {
		t.Errorf("given-satisfied at baseline (%v) and evolved value (%v) are the same, want different", baselineGiven, evolvedGiven)
	}
}

func TestEvolvedInputs_UniversalHypothesisContributesNothing(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"discounted subtotal": {Name: "discounted subtotal", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the discounted subtotal is at least 0", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"discounted subtotal": "5"}},
	}

	cases, found, err := EvolvedInputs(h, concepts, nil)
	if err != nil {
		t.Fatalf("EvolvedInputs returned unexpected error: %v", err)
	}
	if found || cases != nil {
		t.Errorf("found = %v, cases = %v, want false, nil (a universal hypothesis has no given-condition branch to search for)", found, cases)
	}
}

func TestEvolvedInputs_ConceptAlreadyBoundedIsSkipped(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"discounted subtotal": {
			Name: "discounted subtotal", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal",
			HasRange: true, Min: "0.00", Max: "200.00", HasPlaces: true, Places: 2,
		},
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	given, err := cjtr.ParseComparison("the discounted subtotal is greater than 100.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	then, err := cjtr.ParseComparison("the shipping fee is 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", Given: []cjtr.Statement{given}, Then: then,
		Examples: []map[string]string{{"discounted subtotal": "95.00", "shipping fee": "0.00"}},
	}

	cases, found, err := EvolvedInputs(h, concepts, nil)
	if err != nil {
		t.Fatalf("EvolvedInputs returned unexpected error: %v", err)
	}
	if found || cases != nil {
		t.Errorf("found = %v, cases = %v, want false, nil (a concept with a declared range is already handled by BoundaryValues/ExhaustiveValues)", found, cases)
	}
}

func TestEvolvedInputs_ConceptNotMentionedInGivenContributesNothing(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"discounted subtotal": {Name: "discounted subtotal", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal"},
		"order type":          {Name: "order type", Kind: cjtr.VariableConcept, Variable: "input.y", OneOf: []string{"regular", "subscription"}},
		"shipping fee":        {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	given, err := cjtr.ParseComparison("the order type is not subscription", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	then, err := cjtr.ParseComparison("the shipping fee is 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", Given: []cjtr.Statement{given}, Then: then,
		Examples: []map[string]string{{"discounted subtotal": "95.00", "order type": "regular", "shipping fee": "0.00"}},
	}

	cases, found, err := EvolvedInputs(h, concepts, nil)
	if err != nil {
		t.Fatalf("EvolvedInputs returned unexpected error: %v", err)
	}
	if found || cases != nil {
		t.Errorf("found = %v, cases = %v, want false, nil (discounted subtotal never appears in \"given\", so no value of it can flip anything)", found, cases)
	}
}

func TestEvolvedInputs_TextConceptIsSkipped(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"order type":   {Name: "order type", Kind: cjtr.VariableConcept, Variable: "input.y"}, // no Type, no range, no one_of
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	given, err := cjtr.ParseComparison("the shipping fee is at least 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	then, err := cjtr.ParseComparison("the shipping fee is 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", Given: []cjtr.Statement{given}, Then: then,
		Examples: []map[string]string{{"order type": "regular", "shipping fee": "0.00"}},
	}

	cases, found, err := EvolvedInputs(h, concepts, nil)
	if err != nil {
		t.Fatalf("EvolvedInputs returned unexpected error: %v", err)
	}
	if found || cases != nil {
		t.Errorf("found = %v, cases = %v, want false, nil (a non-decimal concept has no continuous space to search)", found, cases)
	}
}

func TestEvolvedInputs_NoExamplesIsError(t *testing.T) {
	h, concepts := unboundedThresholdHypothesis(t)
	h.Examples = nil

	_, _, err := EvolvedInputs(h, concepts, nil)
	if err == nil {
		t.Fatal("EvolvedInputs with no worked examples = nil error, want one")
	}
	if !strings.Contains(err.Error(), "baseline") {
		t.Errorf("error = %q, want it to mention the missing baseline", err.Error())
	}
}
