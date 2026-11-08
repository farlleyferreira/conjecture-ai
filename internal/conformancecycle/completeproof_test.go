package conformancecycle

import (
	"context"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// twoBoundableInputsHypothesis returns a universal hypothesis with two
// boundable input concepts - a 2-value one_of and a 3-value decimal
// range - so the full cross-product is small and exact (6 cases) to
// assert against, for CompleteProofInputs/CheckHypothesisCompleteProof's
// own tests.
func twoBoundableInputsHypothesis(t *testing.T) (cjtr.Hypothesis, map[string]cjtr.Concept) {
	t.Helper()
	concepts := map[string]cjtr.Concept{
		"order type": {
			Name: "order type", Kind: cjtr.VariableConcept, Variable: "input.type",
			OneOf: []string{"regular", "subscription"},
		},
		"quantity": {
			Name: "quantity", Kind: cjtr.VariableConcept, Variable: "input.qty", Type: "decimal",
			HasRange: true, Min: "0.00", Max: "0.02", HasPlaces: true, Places: 2,
		},
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the output value is 42.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"order type": "regular", "quantity": "0.01", "output value": "42.00"}},
	}
	return h, concepts
}

func TestCompleteProofInputs_FullCrossProductFromBaseline(t *testing.T) {
	h, concepts := twoBoundableInputsHypothesis(t)

	cases, found, err := CompleteProofInputs(h, concepts)
	if err != nil {
		t.Fatalf("CompleteProofInputs returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true (both input concepts are boundable)")
	}
	// 2 order-type values x 3 quantity values = 6 cases.
	if len(cases) != 6 {
		t.Fatalf("len(cases) = %d, want 6", len(cases))
	}
	seen := make(map[string]bool, len(cases))
	for _, c := range cases {
		if c["output.result"] != "42.00" {
			t.Errorf("case %+v: output.result = %q, want the baseline's %q untouched", c, c["output.result"], "42.00")
		}
		seen[c["input.type"]+"|"+c["input.qty"]] = true
	}
	for _, typ := range []string{"regular", "subscription"} {
		for _, qty := range []string{"0.00", "0.01", "0.02"} {
			key := typ + "|" + qty
			if !seen[key] {
				t.Errorf("missing combination %s", key)
			}
		}
	}
}

func TestCompleteProofInputs_NoExamplesIsError(t *testing.T) {
	h, concepts := twoBoundableInputsHypothesis(t)
	h.Examples = nil

	_, _, err := CompleteProofInputs(h, concepts)
	if err == nil {
		t.Fatal("CompleteProofInputs with no worked examples = nil error, want one")
	}
}

func TestCompleteProofInputs_FallsBackWhenOneConceptIsNotBoundable(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"order type": {
			Name: "order type", Kind: cjtr.VariableConcept, Variable: "input.type",
			OneOf: []string{"regular", "subscription"},
		},
		"unbounded":    {Name: "unbounded", Kind: cjtr.VariableConcept, Variable: "input.y", Type: "decimal"},
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the output value is 42.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"order type": "regular", "unbounded": "5", "output value": "42.00"}},
	}

	cases, found, err := CompleteProofInputs(h, concepts)
	if err != nil {
		t.Fatalf("CompleteProofInputs returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true with cases %v, want false ('unbounded' has no declared domain)", cases)
	}
	if cases != nil {
		t.Errorf("cases = %v, want nil", cases)
	}
}

func TestCompleteProofInputs_FallsBackWhenCrossProductExceedsCap(t *testing.T) {
	// 50 x 50 = 2500 > maxCompleteProofTotalCases (2000).
	concepts := map[string]cjtr.Concept{
		"a": {
			Name: "a", Kind: cjtr.VariableConcept, Variable: "input.a", Type: "decimal",
			HasRange: true, Min: "0.00", Max: "0.49", HasPlaces: true, Places: 2,
		},
		"b": {
			Name: "b", Kind: cjtr.VariableConcept, Variable: "input.b", Type: "decimal",
			HasRange: true, Min: "0.00", Max: "0.49", HasPlaces: true, Places: 2,
		},
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the output value is 42.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"a": "0.00", "b": "0.00", "output value": "42.00"}},
	}

	cases, found, err := CompleteProofInputs(h, concepts)
	if err != nil {
		t.Fatalf("CompleteProofInputs returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true with %d cases, want false (50x50 = 2500 exceeds the cap of %d)", len(cases), maxCompleteProofTotalCases)
	}
}

func TestCompleteProofInputs_NoInputConceptsIsNotFound(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the output value is 42.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"output value": "42.00"}},
	}

	cases, found, err := CompleteProofInputs(h, concepts)
	if err != nil {
		t.Fatalf("CompleteProofInputs returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true with cases %v, want false (no input concept to cross at all)", cases)
	}
}

func TestCheckHypothesisCompleteProof_AllConformant(t *testing.T) {
	h, concepts := twoBoundableInputsHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	verdict, counterexample, found, err := CheckHypothesisCompleteProof(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisCompleteProof returned unexpected error: %v", err)
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

func TestCheckHypothesisCompleteProof_ReturnsMinimalCounterexampleAmongFailures(t *testing.T) {
	h, concepts := twoBoundableInputsHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"99.00"}`)

	verdict, counterexample, found, err := CheckHypothesisCompleteProof(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisCompleteProof returned unexpected error: %v", err)
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
}

func TestCheckHypothesisCompleteProof_InconclusiveWhenEveryRoundTimesOut(t *testing.T) {
	h, concepts := twoBoundableInputsHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "SLEEP_FOREVER")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	verdict, counterexample, found, err := CheckHypothesisCompleteProof(ctx, sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisCompleteProof returned unexpected error: %v", err)
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

func TestCheckHypothesisCompleteProof_NotFoundWhenNotEveryConceptIsBoundable(t *testing.T) {
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

	verdict, counterexample, found, err := CheckHypothesisCompleteProof(context.Background(), sb, opts, h, concepts, nil)
	if err != nil {
		t.Fatalf("CheckHypothesisCompleteProof returned unexpected error: %v", err)
	}
	if found {
		t.Error("found = true, want false (no boundable concept, so no round ever ran)")
	}
	if verdict != 0 || counterexample != nil {
		t.Errorf("verdict = %v, counterexample = %+v, want the zero Verdict and nil (meaningless when found is false)", verdict, counterexample)
	}
}
