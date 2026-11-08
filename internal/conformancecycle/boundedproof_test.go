package conformancecycle

import (
	"context"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// narrowBoundedInputHypothesis is boundedInputHypothesis's own concept,
// narrowed to a 4-value domain (0.00, 0.01, 0.02, 0.03) so the full
// exhaustive case count is small and exact to assert against.
func narrowBoundedInputHypothesis(t *testing.T) (cjtr.Hypothesis, map[string]cjtr.Concept) {
	t.Helper()
	concepts := map[string]cjtr.Concept{
		"input value": {
			Name: "input value", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal",
			HasRange: true, Min: "0.00", Max: "0.03", HasPlaces: true, Places: 2,
		},
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the output value is 42.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"input value": "0.02", "output value": "42.00"}},
	}
	return h, concepts
}

func TestBoundedProofInputs_ExhaustsTheWholeDomainFromBaseline(t *testing.T) {
	h, concepts := narrowBoundedInputHypothesis(t)

	cases, found, err := BoundedProofInputs(h, concepts)
	if err != nil {
		t.Fatalf("BoundedProofInputs returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true (input value has a boundable range+places domain)")
	}
	if len(cases) != 4 {
		t.Fatalf("len(cases) = %d, want 4 (0.00, 0.01, 0.02, 0.03)", len(cases))
	}
	want := []string{"0.00", "0.01", "0.02", "0.03"}
	for i, w := range want {
		if cases[i]["input.x"] != w {
			t.Errorf("cases[%d][\"input.x\"] = %q, want %q", i, cases[i]["input.x"], w)
		}
		if cases[i]["output.result"] != "42.00" {
			t.Errorf("case %d: output.result = %q, want the baseline's %q untouched", i, cases[i]["output.result"], "42.00")
		}
	}
}

func TestBoundedProofInputs_NoExamplesIsError(t *testing.T) {
	h, concepts := narrowBoundedInputHypothesis(t)
	h.Examples = nil

	_, _, err := BoundedProofInputs(h, concepts)
	if err == nil {
		t.Fatal("BoundedProofInputs with no worked examples = nil error, want one")
	}
}

func TestBoundedProofInputs_NoBoundableConceptIsNotFound(t *testing.T) {
	// boundedInputHypothesis's own wide range has no declared places,
	// so it is not boundable - mirrors
	// TestDirectedSamplingInputs_ConceptWithNoDomainContributesNoCases,
	// but bounded proof must additionally report found=false, since an
	// empty case list here is not a passing check.
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

	cases, found, err := BoundedProofInputs(h, concepts)
	if err != nil {
		t.Fatalf("BoundedProofInputs returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true with cases %v, want false (no declared places to step by)", cases)
	}
	if len(cases) != 0 {
		t.Errorf("len(cases) = %d, want 0", len(cases))
	}
}

func TestCheckHypothesisBoundedProof_AllConformant(t *testing.T) {
	h, concepts := narrowBoundedInputHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	verdict, counterexample, found, err := CheckHypothesisBoundedProof(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisBoundedProof returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if verdict != Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
	if counterexample != nil {
		t.Errorf("counterexample = %+v, want nil", counterexample)
	}
}

func TestCheckHypothesisBoundedProof_ReturnsMinimalCounterexampleAmongFailures(t *testing.T) {
	h, concepts := narrowBoundedInputHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"99.00"}`)

	verdict, counterexample, found, err := CheckHypothesisBoundedProof(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisBoundedProof returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
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
	if bindings["input.x"] != "0.00" {
		t.Errorf("counterexample input.x = %q, want %q (the first-found smallest)", bindings["input.x"], "0.00")
	}
}

func TestCheckHypothesisBoundedProof_InconclusiveWhenEveryRoundTimesOut(t *testing.T) {
	h, concepts := narrowBoundedInputHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "SLEEP_FOREVER")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	verdict, counterexample, found, err := CheckHypothesisBoundedProof(ctx, sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisBoundedProof returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if verdict != Inconclusive {
		t.Errorf("verdict = %v, want Inconclusive", verdict)
	}
	if counterexample != nil {
		t.Errorf("counterexample = %+v, want nil", counterexample)
	}
}

func TestCheckHypothesisBoundedProof_NotFoundWhenNoConceptIsBoundable(t *testing.T) {
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
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	verdict, counterexample, found, err := CheckHypothesisBoundedProof(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisBoundedProof returned unexpected error: %v", err)
	}
	if found {
		t.Error("found = true, want false (no boundable concept, so no round ever ran)")
	}
	if verdict != 0 || counterexample != nil {
		t.Errorf("verdict = %v, counterexample = %+v, want the zero Verdict and nil (meaningless when found is false)", verdict, counterexample)
	}
}
