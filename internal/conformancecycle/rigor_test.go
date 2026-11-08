package conformancecycle

import (
	"context"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

func TestUsesDirectedSampling_LowWeightNoStrongerEvidence(t *testing.T) {
	if !UsesDirectedSampling(evidenceledger.Low, false) {
		t.Error("UsesDirectedSampling(Low, false) = false, want true (the specification's own named scenario)")
	}
}

func TestUsesDirectedSampling_LowWeightWithStrongerEvidenceIsFalse(t *testing.T) {
	if UsesDirectedSampling(evidenceledger.Low, true) {
		t.Error("UsesDirectedSampling(Low, true) = true, want false (stronger evidence is available)")
	}
}

func TestUsesDirectedSampling_HigherWeightsAreFalse(t *testing.T) {
	for _, w := range []evidenceledger.Weight{evidenceledger.Medium, evidenceledger.High, evidenceledger.Critical} {
		if UsesDirectedSampling(w, false) {
			t.Errorf("UsesDirectedSampling(%v, false) = true, want false (the specification only names Low)", w)
		}
	}
}

func TestCheckHypothesisWithSelectedRigor_LawWithFullyBoundableDomainUsesCompleteProof(t *testing.T) {
	h, concepts := twoBoundableInputsHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	level, verdict, counterexample, caseCount, err := CheckHypothesisWithSelectedRigor(context.Background(), sb, opts, h, concepts, nil, true)
	if err != nil {
		t.Fatalf("CheckHypothesisWithSelectedRigor returned unexpected error: %v", err)
	}
	if level != CompleteProof {
		t.Errorf("level = %v, want CompleteProof (every input concept is boundable)", level)
	}
	if verdict != Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
	if counterexample != nil {
		t.Errorf("counterexample = %+v, want nil", counterexample)
	}
	if caseCount != 6 {
		t.Errorf("caseCount = %d, want 6 (the full 2x3 cross-product)", caseCount)
	}
}

func TestCheckHypothesisWithSelectedRigor_LawWithOnlyOneBoundableConceptUsesBoundedProof(t *testing.T) {
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
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	level, verdict, _, caseCount, err := CheckHypothesisWithSelectedRigor(context.Background(), sb, opts, h, concepts, nil, true)
	if err != nil {
		t.Fatalf("CheckHypothesisWithSelectedRigor returned unexpected error: %v", err)
	}
	if level != BoundedProof {
		t.Errorf("level = %v, want BoundedProof ('unbounded' blocks complete proof, but 'order type' alone still qualifies)", level)
	}
	if verdict != Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
	if caseCount != 2 {
		t.Errorf("caseCount = %d, want 2 (order type's own two one_of values)", caseCount)
	}
}

func TestCheckHypothesisWithSelectedRigor_LawWithNoBoundableConceptFallsBackToDirectedSampling(t *testing.T) {
	// boundedInputHypothesis's own input concept declares range+places,
	// but its step count (10001) exceeds maxBoundedProofValuesPerConcept
	// - boundable in principle, but over this package's own disclosed
	// cap, so both bounded and complete proof fall back.
	h, concepts := boundedInputHypothesis(t)
	wantCases, err := DirectedSamplingInputs(h, concepts)
	if err != nil {
		t.Fatalf("DirectedSamplingInputs returned unexpected error: %v", err)
	}
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	level, verdict, _, caseCount, err := CheckHypothesisWithSelectedRigor(context.Background(), sb, opts, h, concepts, nil, true)
	if err != nil {
		t.Fatalf("CheckHypothesisWithSelectedRigor returned unexpected error: %v", err)
	}
	if level != DirectedSampling {
		t.Errorf("level = %v, want DirectedSampling (the only boundable concept exceeds the per-concept cap)", level)
	}
	if verdict != Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
	if caseCount != len(wantCases) {
		t.Errorf("caseCount = %d, want %d", caseCount, len(wantCases))
	}
}

func TestCheckHypothesisWithSelectedRigor_PlainHypothesisAlwaysUsesDirectedSampling(t *testing.T) {
	// Every input concept here is boundable, but isLaw is false - a
	// plain hypothesis has no declared weight anywhere in the .cjtr
	// grammar, so it must keep using directed sampling regardless.
	h, concepts := twoBoundableInputsHypothesis(t)
	wantCases, err := DirectedSamplingInputs(h, concepts)
	if err != nil {
		t.Fatalf("DirectedSamplingInputs returned unexpected error: %v", err)
	}
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	level, verdict, _, caseCount, err := CheckHypothesisWithSelectedRigor(context.Background(), sb, opts, h, concepts, nil, false)
	if err != nil {
		t.Fatalf("CheckHypothesisWithSelectedRigor returned unexpected error: %v", err)
	}
	if level != DirectedSampling {
		t.Errorf("level = %v, want DirectedSampling (isLaw is false)", level)
	}
	if verdict != Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
	if caseCount != len(wantCases) {
		t.Errorf("caseCount = %d, want %d", caseCount, len(wantCases))
	}
}

func TestRigorLevel_String(t *testing.T) {
	cases := map[RigorLevel]string{
		DirectedSampling: "directed sampling",
		BoundedProof:     "bounded proof",
		CompleteProof:    "complete proof",
	}
	for level, want := range cases {
		if got := level.String(); got != want {
			t.Errorf("%v.String() = %q, want %q", level, got, want)
		}
	}
}
