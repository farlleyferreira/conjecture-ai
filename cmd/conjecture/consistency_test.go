package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/expressioncore"
)

func universalHypothesis(name, term, comparator, value string) cjtr.Hypothesis {
	return cjtr.Hypothesis{
		Name:        name,
		IsUniversal: true,
		Then:        cjtr.Statement{Term: term, Comparator: comparator, Value: value},
	}
}

func conditionalHypothesis(name, givenTerm, givenValue, term, comparator, value string) cjtr.Hypothesis {
	return cjtr.Hypothesis{
		Name:  name,
		Given: []cjtr.Statement{{Term: givenTerm, Comparator: "is", Value: givenValue}},
		Then:  cjtr.Statement{Term: term, Comparator: comparator, Value: value},
	}
}

func TestCheckConsistency_ContradictoryUniversalStatementsAreRejected(t *testing.T) {
	t.Setenv("FAKEZ3_RESULT", "unsat")
	concepts := map[string]cjtr.Concept{
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
	}
	hyps := []cjtr.Hypothesis{
		universalHypothesis("at-least-fifty", "shipping fee", "is at least", "50.00"),
		universalHypothesis("at-most-ten", "shipping fee", "is at most", "10.00"),
	}

	var stdout bytes.Buffer
	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: fakeZ3Path}, &stdout)

	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	if !strings.Contains(stdout.String(), "CONTRADICTION") {
		t.Errorf("stdout = %q, want it to report the contradiction", stdout.String())
	}
}

func TestCheckConsistency_SatisfiableUniversalStatementsAreNotRejected(t *testing.T) {
	// FAKEZ3_RESULT left unset: fakez3's own default is "sat".
	concepts := map[string]cjtr.Concept{
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
	}
	hyps := []cjtr.Hypothesis{
		universalHypothesis("at-least-zero", "shipping fee", "is at least", "0.00"),
		universalHypothesis("at-most-fifty", "shipping fee", "is at most", "50.00"),
	}

	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: fakeZ3Path}, &bytes.Buffer{})
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none (satisfiable)", findings)
	}
}

func TestCheckConsistency_ConditionalHypothesesNeverChecked(t *testing.T) {
	// Even with the solver forced to report unsat, two conditional
	// (Given/Then) hypotheses asserting different things about the
	// same term under different Givens must never be reported as
	// contradictory - they are never simultaneously asserted, unlike
	// two IsUniversal hypotheses' own Then statements.
	t.Setenv("FAKEZ3_RESULT", "unsat")
	concepts := map[string]cjtr.Concept{
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
		"order type":   {Name: "order type", Kind: cjtr.VariableConcept, Variable: "input.order_type", OneOf: []string{"regular", "subscription"}},
	}
	hyps := []cjtr.Hypothesis{
		conditionalHypothesis("subscription-fee", "order type", "subscription", "shipping fee", "is", "7.50"),
		conditionalHypothesis("regular-fee", "order type", "regular", "shipping fee", "is", "0.00"),
	}

	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: fakeZ3Path}, &bytes.Buffer{})
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none (conditional hypotheses are never checked against each other)", findings)
	}
}

func TestCheckConsistency_SingleStatementNeverCallsSolver(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
	}
	hyps := []cjtr.Hypothesis{
		universalHypothesis("only-one", "shipping fee", "is at least", "0.00"),
	}

	// A nonexistent solver binary would surface as a stdout "skipped"
	// message if the solver were ever actually invoked - asserting its
	// absence proves checkConsistency never called it for a single
	// statement.
	var stdout bytes.Buffer
	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: "this-binary-does-not-exist"}, &stdout)
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none", findings)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty (solver never invoked for a single statement)", stdout.String())
	}
}

func TestCheckConsistency_NonDecimalConceptSkipped(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"order type": {Name: "order type", Kind: cjtr.VariableConcept, Variable: "input.order_type", OneOf: []string{"regular", "subscription"}},
	}
	hyps := []cjtr.Hypothesis{
		universalHypothesis("a", "order type", "is", "regular"),
		universalHypothesis("b", "order type", "is", "subscription"),
	}

	var stdout bytes.Buffer
	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: "this-binary-does-not-exist"}, &stdout)
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none (non-decimal concepts are not checked)", findings)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty (solver never invoked for a non-decimal concept)", stdout.String())
	}
}

func TestCheckConsistency_VariableConceptBoundSkipped(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"shipping fee":   {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
		"minimum charge": {Name: "minimum charge", Kind: cjtr.VariableConcept, Variable: "input.minimum_charge", Type: "decimal"},
	}
	hyps := []cjtr.Hypothesis{
		universalHypothesis("at-least-zero", "shipping fee", "is at least", "0.00"),
		{
			Name:        "at-least-minimum",
			IsUniversal: true,
			Then:        cjtr.Statement{Term: "shipping fee", Comparator: "is at least", Value: "minimum charge", ValueIsConcept: true},
		},
	}

	// "minimum charge" is a VariableConcept, not resolvable without a
	// binding - this leaves only one usable constraint, so the solver
	// must never be called.
	var stdout bytes.Buffer
	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: "this-binary-does-not-exist"}, &stdout)
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none", findings)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty (solver never invoked)", stdout.String())
	}
}

func TestCheckConsistency_ConstantConceptBoundResolved(t *testing.T) {
	t.Setenv("FAKEZ3_RESULT", "unsat")
	concepts := map[string]cjtr.Concept{
		"shipping fee":      {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
		"minimum threshold": {Name: "minimum threshold", Kind: cjtr.ConstantConcept, Value: "50.00"},
	}
	hyps := []cjtr.Hypothesis{
		{
			Name:        "at-least-minimum",
			IsUniversal: true,
			Then:        cjtr.Statement{Term: "shipping fee", Comparator: "is at least", Value: "minimum threshold", ValueIsConcept: true},
		},
		universalHypothesis("at-most-ten", "shipping fee", "is at most", "10.00"),
	}

	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: fakeZ3Path}, &bytes.Buffer{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1 (a ConstantConcept bound should resolve and detect the contradiction)", len(findings))
	}
}

func TestCheckConsistency_SolverErrorIsNonFatalAndSkipped(t *testing.T) {
	t.Setenv("FAKEZ3_RESULT", "error")
	concepts := map[string]cjtr.Concept{
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
	}
	hyps := []cjtr.Hypothesis{
		universalHypothesis("at-least-fifty", "shipping fee", "is at least", "50.00"),
		universalHypothesis("at-most-ten", "shipping fee", "is at most", "10.00"),
	}

	var stdout bytes.Buffer
	findings := checkConsistency(hyps, concepts, expressioncore.Solver{Z3Binary: fakeZ3Path}, &stdout)
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none (a solver error is skipped, not a finding)", findings)
	}
	if !strings.Contains(stdout.String(), "skipped") {
		t.Errorf("stdout = %q, want it to report the check was skipped", stdout.String())
	}
}
