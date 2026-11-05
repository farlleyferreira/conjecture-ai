package cjtr

import "testing"

func falsifiabilityTestConcepts(t *testing.T) map[string]Concept {
	t.Helper()
	root := loadNode(t, `
concepts:
  shipping fee:
    variable: output.shipping_fee
    type: decimal
    range: {min: 0.00, max: 100000.00}
  order type:
    variable: input.order_type
    one_of: [regular]
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
  free shipping threshold:
    value: 100.00
`)
	concepts, err := Concepts(root)
	if err != nil {
		t.Fatalf("Concepts returned unexpected error: %v", err)
	}
	return concepts
}

func TestFalsifiable_IsAtLeast(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	tests := []struct {
		name string
		stmt string
		want bool
	}{
		{"below the term's own minimum: unfalsifiable", "the shipping fee is at least -1.00", false},
		{"equal to the term's own minimum: still unfalsifiable (every value satisfies it)", "the shipping fee is at least 0.00", false},
		{"a real, meaningful bound: falsifiable", "the shipping fee is at least 50.00", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := ParseComparison(tt.stmt, concepts)
			if err != nil {
				t.Fatalf("ParseComparison returned unexpected error: %v", err)
			}
			got, err := Falsifiable(stmt, concepts)
			if err != nil {
				t.Fatalf("Falsifiable returned unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Falsifiable(%q) = %v, want %v", tt.stmt, got, tt.want)
			}
		})
	}
}

func TestFalsifiable_IsAtMost(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	stmt, err := ParseComparison("the shipping fee is at most 999999.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if got {
		t.Error("Falsifiable(\"is at most 999999.00\") = true, want false (the term's own max is 100000.00, well below this bound)")
	}
}

func TestFalsifiable_IsGreaterThan(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	// termMin (0.00) is NOT strictly greater than -1.00... wait: the
	// statement itself is "is greater than -1.00", and every value in
	// [0.00, 100000.00] is indeed > -1.00, so it IS a tautology.
	stmt, err := ParseComparison("the shipping fee is greater than -1.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if got {
		t.Error("Falsifiable(\"is greater than -1.00\") = true, want false (every allowed value satisfies it)")
	}
}

func TestFalsifiable_IsGreaterThan_AtTheBoundaryStillFalsifiable(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	// "is greater than 0.00": the term's own minimum (0.00) does NOT
	// satisfy "greater than 0.00", so that boundary value itself would
	// falsify the statement - it is falsifiable.
	stmt, err := ParseComparison("the shipping fee is greater than 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if !got {
		t.Error("Falsifiable(\"is greater than 0.00\") = false, want true (the term's own minimum value would falsify it)")
	}
}

func TestFalsifiable_IsLessThan(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	stmt, err := ParseComparison("the shipping fee is less than 999999.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if got {
		t.Error("Falsifiable(\"is less than 999999.00\") = true, want false (every allowed value satisfies it)")
	}
}

func TestFalsifiable_IsBetween_FullyContained(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	stmt, err := ParseComparison("the shipping fee is between -1.00 and 999999.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if got {
		t.Error("Falsifiable(\"is between -1.00 and 999999.00\") = true, want false (the term's entire range already fits inside these bounds)")
	}
}

func TestFalsifiable_IsBetween_MeaningfulBound(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	stmt, err := ParseComparison("the shipping fee is between 10.00 and 20.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if !got {
		t.Error("Falsifiable(\"is between 10.00 and 20.00\") = false, want true (values outside 10-20 but inside the term's range would falsify it)")
	}
}

func TestFalsifiable_OneOfSingleValue(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	stmt, err := ParseComparison("the order type is regular", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if got {
		t.Error("Falsifiable(\"the order type is regular\") = true, want false (order type's one_of has only this one value)")
	}
}

func TestFalsifiable_ValueIsConceptAlwaysFalsifiable(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	stmt, err := ParseComparison("the discounted subtotal is greater than the free shipping threshold", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if !got {
		t.Error("Falsifiable with a concept-reference value = false, want true (not evaluated statically)")
	}
}

func TestFalsifiable_NoDeclaredRange(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	stmt, err := ParseComparison("the discounted subtotal is at least -1.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	got, err := Falsifiable(stmt, concepts)
	if err != nil {
		t.Fatalf("Falsifiable returned unexpected error: %v", err)
	}
	if !got {
		t.Error("Falsifiable with no declared range on the term = false, want true (nothing to prove a tautology against)")
	}
}

func TestFalsifiable_UndeclaredTerm(t *testing.T) {
	concepts := falsifiabilityTestConcepts(t)

	_, err := Falsifiable(Statement{Term: "nonexistent", Comparator: "is at least", Value: "0"}, concepts)
	if err == nil {
		t.Fatal("Falsifiable with an undeclared term = nil error, want an error")
	}
}
