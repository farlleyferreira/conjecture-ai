package expressioncore

import (
	"strings"
	"testing"
)

func TestEncodeComparatorConstraint_EveryComparator(t *testing.T) {
	cases := []struct {
		comparator string
		bound1     string
		bound2     string // "" when not a two-value comparator
		want       string
	}{
		{"is", "50.00", "", "(= x 50)"},
		{"equals", "50.00", "", "(= x 50)"},
		{"is not", "50.00", "", "(not (= x 50))"},
		{"does not equal", "50.00", "", "(not (= x 50))"},
		{"is greater than", "50.00", "", "(> x 50)"},
		{"is less than", "50.00", "", "(< x 50)"},
		{"is at least", "50.00", "", "(>= x 50)"},
		{"is at most", "50.00", "", "(<= x 50)"},
		{"is between", "50.00", "150.00", "(and (>= x 50) (<= x 150))"},
		{"is strictly between", "50.00", "150.00", "(and (> x 50) (< x 150))"},
	}

	for _, c := range cases {
		bound1 := mustDecimal(t, c.bound1)
		var bound2 Decimal
		if c.bound2 != "" {
			bound2 = mustDecimal(t, c.bound2)
		}
		got, err := EncodeComparatorConstraint("x", c.comparator, bound1, bound2)
		if err != nil {
			t.Fatalf("EncodeComparatorConstraint(%q) returned unexpected error: %v", c.comparator, err)
		}
		if got != c.want {
			t.Errorf("EncodeComparatorConstraint(%q) = %q, want %q", c.comparator, got, c.want)
		}
	}
}

func TestEncodeComparatorConstraint_UnrecognizedComparator(t *testing.T) {
	bound := mustDecimal(t, "1.00")
	if _, err := EncodeComparatorConstraint("x", "is one of", bound, Decimal{}); err == nil {
		t.Error(`EncodeComparatorConstraint("is one of") = nil error, want one`)
	}
}

func TestSMTRealLiteral_FractionalValue(t *testing.T) {
	d := mustDecimal(t, "7.50")
	got, err := EncodeComparatorConstraint("x", "is", d, Decimal{})
	if err != nil {
		t.Fatalf("EncodeComparatorConstraint returned unexpected error: %v", err)
	}
	// 7.50 = 15/2 exactly - rendered via its own num/denom, not "7.5",
	// so there is no ambiguity with SMT-LIB2's own decimal-literal
	// grammar.
	want := "(= x (/ 15 2))"
	if got != want {
		t.Errorf("EncodeComparatorConstraint with a fractional bound = %q, want %q", got, want)
	}
}

func TestSMTRealLiteral_NegativeValue(t *testing.T) {
	d := mustDecimal(t, "-5.00")
	got, err := EncodeComparatorConstraint("x", "is at least", d, Decimal{})
	if err != nil {
		t.Fatalf("EncodeComparatorConstraint returned unexpected error: %v", err)
	}
	want := "(>= x (- 5))"
	if got != want {
		t.Errorf("EncodeComparatorConstraint with a negative bound = %q, want %q", got, want)
	}
}

func TestSMTRealLiteral_NegativeFractionalValue(t *testing.T) {
	d := mustDecimal(t, "-0.25")
	got, err := EncodeComparatorConstraint("x", "is", d, Decimal{})
	if err != nil {
		t.Fatalf("EncodeComparatorConstraint returned unexpected error: %v", err)
	}
	want := "(= x (/ (- 1) 4))"
	if got != want {
		t.Errorf("EncodeComparatorConstraint with a negative fractional bound = %q, want %q", got, want)
	}
}

func TestBuildSatisfiabilityQuery_AssemblesDeclareAssertCheckSat(t *testing.T) {
	c1, err := EncodeComparatorConstraint("shipping_fee", "is at least", mustDecimal(t, "50.00"), Decimal{})
	if err != nil {
		t.Fatalf("EncodeComparatorConstraint returned unexpected error: %v", err)
	}
	c2, err := EncodeComparatorConstraint("shipping_fee", "is at most", mustDecimal(t, "10.00"), Decimal{})
	if err != nil {
		t.Fatalf("EncodeComparatorConstraint returned unexpected error: %v", err)
	}

	query := BuildSatisfiabilityQuery("shipping_fee", []string{c1, c2})

	for _, want := range []string{
		"(declare-const shipping_fee Real)",
		"(assert (>= shipping_fee 50))",
		"(assert (<= shipping_fee 10))",
		"(check-sat)",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("BuildSatisfiabilityQuery output = %q, want it to contain %q", query, want)
		}
	}
}
