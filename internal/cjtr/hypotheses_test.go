package cjtr

import (
	"testing"
)

// hypothesesExampleConcepts mirrors the real example's concepts, minus
// "table shipping fee" (a table-lookup concept Concepts() does not
// support yet).
const hypothesesExampleConcepts = `
concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal
  free shipping threshold:
    value: 100.00
`

func loadConceptsForHypotheses(t *testing.T) map[string]Concept {
	t.Helper()
	root := loadNode(t, hypothesesExampleConcepts)
	concepts, err := Concepts(root)
	if err != nil {
		t.Fatalf("Concepts returned unexpected error: %v", err)
	}
	return concepts
}

// ruleWithHypotheses adapts the real worked example's rule: same given/
// then/order-type logic, but the "otherwise" branch uses a plain decimal
// threshold ("is at least 5.00") instead of "the shipping fee equals the
// table shipping fee" - that reference needs a table-lookup concept,
// which this codebase does not support yet. The adaptation is confined to
// this test fixture; it is not a claim about what the real file says.
const ruleWithHypotheses = `
rules:
  shipping is free above the threshold:
    id: free-above
    hypotheses:
      the shipping fee is never negative:
        id: never-negative
        always: the shipping fee is at least 0.00

      above the threshold shipping is free:
        given:
          - the discounted subtotal is greater than the free shipping threshold
          - the order type is not subscription
        then: the shipping fee is 0.00
        otherwise: the shipping fee is at least 5.00
        examples:
          - {discounted subtotal: 150.00, order type: regular,      shipping fee: 0.00}
          - {discounted subtotal: 50.00,  order type: regular,      shipping fee: 7.50}
          - {discounted subtotal: 150.00, order type: subscription, shipping fee: 7.50}
`

func TestHypotheses_RealExampleShape(t *testing.T) {
	concepts := loadConceptsForHypotheses(t)
	root := loadNode(t, ruleWithHypotheses)
	rulesNode, _ := root.Get("rules")
	ruleBody := rulesNode.Pairs[0].Value

	hyps, err := Hypotheses(ruleBody, concepts)
	if err != nil {
		t.Fatalf("Hypotheses returned unexpected error: %v", err)
	}
	if len(hyps) != 2 {
		t.Fatalf("Hypotheses returned %d entries, want 2", len(hyps))
	}

	var universal, conditional *Hypothesis
	for i := range hyps {
		if hyps[i].IsUniversal {
			universal = &hyps[i]
		} else {
			conditional = &hyps[i]
		}
	}
	if universal == nil || conditional == nil {
		t.Fatal("expected one universal and one conditional hypothesis")
	}

	if universal.ID != "never-negative" {
		t.Errorf("universal.ID = %q, want %q", universal.ID, "never-negative")
	}
	if universal.Then.Comparator != "is at least" {
		t.Errorf("universal.Then.Comparator = %q, want %q", universal.Then.Comparator, "is at least")
	}

	if len(conditional.Given) != 2 {
		t.Fatalf("conditional.Given has %d statements, want 2", len(conditional.Given))
	}
	if conditional.Otherwise == nil {
		t.Fatal("conditional.Otherwise = nil, want a statement")
	}
	if len(conditional.Examples) != 3 {
		t.Fatalf("conditional.Examples has %d rows, want 3", len(conditional.Examples))
	}
}

func TestCheckExample_RealExampleRows(t *testing.T) {
	concepts := loadConceptsForHypotheses(t)
	root := loadNode(t, ruleWithHypotheses)
	rulesNode, _ := root.Get("rules")
	ruleBody := rulesNode.Pairs[0].Value

	hyps, err := Hypotheses(ruleBody, concepts)
	if err != nil {
		t.Fatalf("Hypotheses returned unexpected error: %v", err)
	}

	var conditional Hypothesis
	for _, h := range hyps {
		if !h.IsUniversal {
			conditional = h
		}
	}

	for i, example := range conditional.Examples {
		ok, err := CheckExample(conditional, concepts, nil, example)
		if err != nil {
			t.Fatalf("CheckExample(row %d = %v) returned unexpected error: %v", i, example, err)
		}
		if !ok {
			t.Errorf("CheckExample(row %d = %v) = false, want true (this row is from the worked example and should be consistent)", i, example)
		}
	}
}

func TestCheckExample_DetectsViolation(t *testing.T) {
	concepts := loadConceptsForHypotheses(t)
	root := loadNode(t, ruleWithHypotheses)
	rulesNode, _ := root.Get("rules")
	ruleBody := rulesNode.Pairs[0].Value

	hyps, err := Hypotheses(ruleBody, concepts)
	if err != nil {
		t.Fatalf("Hypotheses returned unexpected error: %v", err)
	}
	var conditional Hypothesis
	for _, h := range hyps {
		if !h.IsUniversal {
			conditional = h
		}
	}

	// Given holds (150.00 > 100.00, order type is not subscription), so
	// "then" (shipping fee is 0.00) applies - but this row claims 7.50,
	// which contradicts it.
	violating := map[string]string{
		"discounted subtotal": "150.00",
		"order type":          "regular",
		"shipping fee":        "7.50",
	}

	ok, err := CheckExample(conditional, concepts, nil, violating)
	if err != nil {
		t.Fatalf("CheckExample returned unexpected error: %v", err)
	}
	if ok {
		t.Error("CheckExample(violating row) = true, want false")
	}
}

func TestHypotheses_NoHypothesesKey(t *testing.T) {
	root := loadNode(t, "rules:\n  x:\n    id: x\n")
	rulesNode, _ := root.Get("rules")
	ruleBody := rulesNode.Pairs[0].Value

	hyps, err := Hypotheses(ruleBody, map[string]Concept{})
	if err != nil {
		t.Fatalf("Hypotheses returned unexpected error: %v", err)
	}
	if hyps != nil {
		t.Errorf("Hypotheses with no \"hypotheses\" key = %v, want nil", hyps)
	}
}

func TestHypotheses_NeitherAlwaysNorGivenThen(t *testing.T) {
	concepts := loadConceptsForHypotheses(t)
	root := loadNode(t, "rules:\n  x:\n    hypotheses:\n      y:\n        id: y\n")
	rulesNode, _ := root.Get("rules")
	ruleBody := rulesNode.Pairs[0].Value

	_, err := Hypotheses(ruleBody, concepts)
	if err == nil {
		t.Fatal("Hypotheses with neither \"always\" nor \"given\"/\"then\" = nil error, want an error")
	}
}
