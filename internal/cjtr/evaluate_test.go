package cjtr

import "testing"

// evaluateExampleConcepts is parsed from the same shape as the worked
// .cjtr example's "concepts" section (its table-lookup concept omitted:
// these tests exercise plain variable/constant resolution, which needs no
// loaded [Tables]; table resolution has its own tests in table_test.go).
const evaluateExampleConcepts = `
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

func loadEvaluateConcepts(t *testing.T) map[string]Concept {
	t.Helper()
	root := loadNode(t, evaluateExampleConcepts)
	concepts, err := Concepts(root)
	if err != nil {
		t.Fatalf("Concepts returned unexpected error: %v", err)
	}
	return concepts
}

func TestEvaluate_RealExampleScenarios(t *testing.T) {
	concepts := loadEvaluateConcepts(t)

	tests := []struct {
		name      string
		statement string
		bindings  Bindings
		want      bool
	}{
		{
			"shipping fee at least zero: exactly zero",
			"the shipping fee is at least 0.00",
			Bindings{"output.shipping_fee": "0.00"},
			true,
		},
		{
			"shipping fee at least zero: negative",
			"the shipping fee is at least 0.00",
			Bindings{"output.shipping_fee": "-0.01"},
			false,
		},
		{
			"discounted subtotal above threshold: well above",
			"the discounted subtotal is greater than the free shipping threshold",
			Bindings{"input.discounted_subtotal": "150.00"},
			true,
		},
		{
			"discounted subtotal above threshold: exactly at threshold (not strictly greater)",
			"the discounted subtotal is greater than the free shipping threshold",
			Bindings{"input.discounted_subtotal": "100.00"},
			false,
		},
		{
			"discounted subtotal above threshold: below",
			"the discounted subtotal is greater than the free shipping threshold",
			Bindings{"input.discounted_subtotal": "99.99"},
			false,
		},
		{
			"order type is not subscription: regular",
			"the order type is not subscription",
			Bindings{"input.order_type": "regular"},
			true,
		},
		{
			"order type is not subscription: subscription",
			"the order type is not subscription",
			Bindings{"input.order_type": "subscription"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := ParseComparison(tt.statement, concepts)
			if err != nil {
				t.Fatalf("ParseComparison(%q) returned unexpected error: %v", tt.statement, err)
			}

			got, err := Evaluate(stmt, concepts, nil, tt.bindings)
			if err != nil {
				t.Fatalf("Evaluate returned unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.statement, got, tt.want)
			}
		})
	}
}

func TestEvaluate_ExactDecimalEquality(t *testing.T) {
	concepts := loadEvaluateConcepts(t)

	// "100.00" and "100.0" differ as text but are the same exact decimal
	// value - equality on a decimal concept must compare numerically, not
	// as strings, or this would wrongly report false.
	stmt, err := ParseComparison("the discounted subtotal is 100.0", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}

	got, err := Evaluate(stmt, concepts, nil, Bindings{"input.discounted_subtotal": "100.00"})
	if err != nil {
		t.Fatalf("Evaluate returned unexpected error: %v", err)
	}
	if !got {
		t.Error("Evaluate(\"100.0\" == \"100.00\") = false, want true (same exact decimal value)")
	}
}

func TestEvaluate_UnboundVariable(t *testing.T) {
	concepts := loadEvaluateConcepts(t)

	stmt, err := ParseComparison("the shipping fee is at least 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}

	_, err = Evaluate(stmt, concepts, nil, Bindings{})
	if err == nil {
		t.Fatal("Evaluate with no binding for the term's variable = nil error, want an error")
	}
}

func TestEvaluate_InvalidDecimal(t *testing.T) {
	concepts := loadEvaluateConcepts(t)

	stmt, err := ParseComparison("the shipping fee is at least 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}

	_, err = Evaluate(stmt, concepts, nil, Bindings{"output.shipping_fee": "not a number"})
	if err == nil {
		t.Fatal("Evaluate with an unparseable decimal binding = nil error, want an error")
	}
}
