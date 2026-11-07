package conformancecycle

import (
	"context"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// boundedInputHypothesis returns a universal hypothesis - "the output
// value is 42.00" - with one bounded decimal input concept (range
// 0.00..100.00, places 2) and one baseline worked example, for
// DirectedSamplingInputs/CheckHypothesisDirectedSampling's own tests.
func boundedInputHypothesis(t *testing.T) (cjtr.Hypothesis, map[string]cjtr.Concept) {
	t.Helper()
	concepts := map[string]cjtr.Concept{
		"input value": {
			Name: "input value", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal",
			HasRange: true, Min: "0.00", Max: "100.00", HasPlaces: true, Places: 2,
		},
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the output value is 42.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"input value": "50.00", "output value": "42.00"}},
	}
	return h, concepts
}

func TestDirectedSamplingInputs_VariesOnlyInputConceptsFromBaseline(t *testing.T) {
	h, concepts := boundedInputHypothesis(t)

	cases, err := DirectedSamplingInputs(h, concepts)
	if err != nil {
		t.Fatalf("DirectedSamplingInputs returned unexpected error: %v", err)
	}

	// range + places generates Min, Max, justAboveMin, justBelowMax: 4 cases.
	if len(cases) != 4 {
		t.Fatalf("len(cases) = %d, want 4", len(cases))
	}
	gotValues := make([]string, len(cases))
	for i, c := range cases {
		gotValues[i] = c["input.x"]
		if c["output.result"] != "42.00" {
			t.Errorf("case %d: output.result = %q, want the baseline's %q untouched", i, c["output.result"], "42.00")
		}
	}
	want := []string{"0.00", "100.00", "0.01", "99.99"}
	for i, w := range want {
		if gotValues[i] != w {
			t.Errorf("cases[%d][\"input.x\"] = %q, want %q", i, gotValues[i], w)
		}
	}
}

func TestDirectedSamplingInputs_NoExamplesIsError(t *testing.T) {
	h, concepts := boundedInputHypothesis(t)
	h.Examples = nil

	_, err := DirectedSamplingInputs(h, concepts)
	if err == nil {
		t.Fatal("DirectedSamplingInputs with no worked examples = nil error, want one")
	}
}

func TestDirectedSamplingInputs_ConceptWithNoDomainContributesNoCases(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"unbounded": {Name: "unbounded", Kind: cjtr.VariableConcept, Variable: "input.y", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the unbounded is at least 0", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"unbounded": "5"}},
	}

	cases, err := DirectedSamplingInputs(h, concepts)
	if err != nil {
		t.Fatalf("DirectedSamplingInputs returned unexpected error: %v", err)
	}
	if len(cases) != 0 {
		t.Errorf("len(cases) = %d, want 0 (no range or one_of declared)", len(cases))
	}
}

func TestCheckHypothesisDirectedSampling_AllConformant(t *testing.T) {
	h, concepts := boundedInputHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	verdict, counterexample, err := CheckHypothesisDirectedSampling(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisDirectedSampling returned unexpected error: %v", err)
	}
	if verdict != Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
	if counterexample != nil {
		t.Errorf("counterexample = %+v, want nil", counterexample)
	}
}

func TestCheckHypothesisDirectedSampling_ReturnsMinimalCounterexampleAmongFailures(t *testing.T) {
	h, concepts := boundedInputHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	// Every round's output mismatches "is 42.00", so every one of the 4
	// generated cases fails - proving MinimalCounterexample is actually
	// exercised against more than one real failure, not just a single one.
	opts := testOpts(t, "WRITE", `{"output.result":"99.00"}`)

	verdict, counterexample, err := CheckHypothesisDirectedSampling(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisDirectedSampling returned unexpected error: %v", err)
	}
	if verdict != NonConformant {
		t.Fatalf("verdict = %v, want NonConformant", verdict)
	}
	if counterexample == nil {
		t.Fatal("counterexample = nil, want the minimal failing case")
	}
	bindings, ok := counterexample.Value.(cjtr.Bindings)
	if !ok {
		t.Fatalf("counterexample.Value = %T, want cjtr.Bindings", counterexample.Value)
	}
	// "0.00" and "0.01" tie for the shortest input.x text (both length 4);
	// MinimalCounterexample keeps the first found on a tie, and "0.00" is
	// the first case DirectedSamplingInputs generates (Min before the
	// in-boundary refinements).
	if bindings["input.x"] != "0.00" {
		t.Errorf("counterexample input.x = %q, want %q (the first-found smallest)", bindings["input.x"], "0.00")
	}
}

func TestCheckHypothesisDirectedSampling_InconclusiveWhenEveryRoundTimesOut(t *testing.T) {
	h, concepts := boundedInputHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "SLEEP_FOREVER")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	verdict, counterexample, err := CheckHypothesisDirectedSampling(ctx, sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisDirectedSampling returned unexpected error: %v", err)
	}
	if verdict != Inconclusive {
		t.Errorf("verdict = %v, want Inconclusive", verdict)
	}
	if counterexample != nil {
		t.Errorf("counterexample = %+v, want nil", counterexample)
	}
}
