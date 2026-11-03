package cjtr

import (
	"strings"
	"testing"
)

// exampleConcepts mirrors the concept names declared in the worked
// .cjtr example (conjecture/phenomena/shipping/free-shipping.cjtr). Only
// the keys matter to ParseComparison; the values' contents are irrelevant
// to this parser.
var exampleConcepts = map[string]Concept{
	"discounted subtotal":     {Name: "discounted subtotal"},
	"order type":              {Name: "order type"},
	"shipping fee":            {Name: "shipping fee"},
	"taxes":                   {Name: "taxes"},
	"free shipping threshold": {Name: "free shipping threshold"},
	"table shipping fee":      {Name: "table shipping fee"},
}

func TestParseComparison_RealExampleStatements(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Statement
	}{
		{
			"given: order type is subscription",
			"the order type is subscription",
			Statement{Term: "order type", Comparator: "is", Value: "subscription", ValueIsConcept: false},
		},
		{
			"then: shipping fee equals table shipping fee",
			"the shipping fee equals the table shipping fee",
			Statement{Term: "shipping fee", Comparator: "equals", Value: "table shipping fee", ValueIsConcept: true},
		},
		{
			"always: shipping fee is at least 0.00",
			"the shipping fee is at least 0.00",
			Statement{Term: "shipping fee", Comparator: "is at least", Value: "0.00", ValueIsConcept: false},
		},
		{
			"given: discounted subtotal is greater than free shipping threshold",
			"the discounted subtotal is greater than the free shipping threshold",
			Statement{Term: "discounted subtotal", Comparator: "is greater than", Value: "free shipping threshold", ValueIsConcept: true},
		},
		{
			"given: order type is not subscription",
			"the order type is not subscription",
			Statement{Term: "order type", Comparator: "is not", Value: "subscription", ValueIsConcept: false},
		},
		{
			"then: shipping fee is 0.00",
			"the shipping fee is 0.00",
			Statement{Term: "shipping fee", Comparator: "is", Value: "0.00", ValueIsConcept: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseComparison(tt.text, exampleConcepts)
			if err != nil {
				t.Fatalf("ParseComparison(%q) returned unexpected error: %v", tt.text, err)
			}
			if got != tt.want {
				t.Errorf("ParseComparison(%q) = %+v, want %+v", tt.text, got, tt.want)
			}
		})
	}
}

func TestParseComparison_IsBetween(t *testing.T) {
	got, err := ParseComparison("the shipping fee is between 5.00 and 10.00", exampleConcepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	want := Statement{Term: "shipping fee", Comparator: "is between", Value: "5.00", Value2: "10.00"}
	if got != want {
		t.Errorf("ParseComparison = %+v, want %+v", got, want)
	}
}

func TestParseComparison_IsStrictlyBetweenWithConceptBound(t *testing.T) {
	got, err := ParseComparison("the shipping fee is strictly between the free shipping threshold and 200.00", exampleConcepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	want := Statement{
		Term: "shipping fee", Comparator: "is strictly between",
		Value: "free shipping threshold", ValueIsConcept: true,
		Value2: "200.00",
	}
	if got != want {
		t.Errorf("ParseComparison = %+v, want %+v", got, want)
	}
}

func TestParseComparison_IsBetweenMissingAnd(t *testing.T) {
	_, err := ParseComparison("the shipping fee is between 5.00", exampleConcepts)
	if err == nil {
		t.Fatal("ParseComparison(\"is between\" with no \" and \") = nil error, want an error")
	}
}

func TestParseComparison_MissingArticle(t *testing.T) {
	_, err := ParseComparison("shipping fee is 0.00", exampleConcepts)
	if err == nil {
		t.Fatal("ParseComparison without a leading \"the \" = nil error, want an error")
	}
}

func TestParseComparison_UnknownTerm(t *testing.T) {
	_, err := ParseComparison("the nonexistent concept is 1", exampleConcepts)
	if err == nil {
		t.Fatal("ParseComparison with an undeclared term = nil error, want an error")
	}
}

func TestParseComparison_MissingValue(t *testing.T) {
	_, err := ParseComparison("the shipping fee is", exampleConcepts)
	if err == nil {
		t.Fatal("ParseComparison with no value = nil error, want an error")
	}
}

func TestParseComparison_UnsupportedComparator(t *testing.T) {
	// "is empty" is a real vocabulary comparator, but not one this parser
	// supports yet - and it is a case where "is" (supported) is itself a
	// prefix, so this also guards against the shorter phrase winning by
	// accident.
	_, err := ParseComparison("the shipping fee is empty", exampleConcepts)
	if err == nil {
		t.Fatal("ParseComparison with \"is empty\" = nil error, want an error")
	}
	got := err.Error()
	if !strings.Contains(got, "is empty") || !strings.Contains(got, "not supported") {
		t.Errorf("error = %q, want it to mention \"is empty\" and \"not supported\"", got)
	}
}

func TestParseComparison_UnrecognizedComparator(t *testing.T) {
	_, err := ParseComparison("the shipping fee sort of resembles 1", exampleConcepts)
	if err == nil {
		t.Fatal("ParseComparison with a made-up comparator = nil error, want an error")
	}
}
