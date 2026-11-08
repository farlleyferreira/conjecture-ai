package cjtr

import (
	"reflect"
	"testing"
)

func TestBoundaryValues_RangeWithPlaces(t *testing.T) {
	// The real worked example's own "discounted subtotal" concept:
	// range 0.00..100000.00, places 2.
	c := Concept{
		Name: "discounted subtotal", Kind: VariableConcept, Type: "decimal",
		HasRange: true, Min: "0.00", Max: "100000.00",
		HasPlaces: true, Places: 2,
	}

	got, err := BoundaryValues(c)
	if err != nil {
		t.Fatalf("BoundaryValues returned unexpected error: %v", err)
	}
	want := []string{"0.00", "100000.00", "0.01", "99999.99"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BoundaryValues = %v, want %v", got, want)
	}
}

func TestBoundaryValues_RangeWithoutPlaces(t *testing.T) {
	c := Concept{Name: "x", Kind: VariableConcept, HasRange: true, Min: "0", Max: "100"}

	got, err := BoundaryValues(c)
	if err != nil {
		t.Fatalf("BoundaryValues returned unexpected error: %v", err)
	}
	want := []string{"0", "100"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BoundaryValues = %v, want %v (no \"places\" declared, so no inside-the-boundary values)", got, want)
	}
}

func TestBoundaryValues_OneOf(t *testing.T) {
	// The real worked example's own "order type" concept.
	c := Concept{Name: "order type", Kind: VariableConcept, OneOf: []string{"regular", "subscription"}}

	got, err := BoundaryValues(c)
	if err != nil {
		t.Fatalf("BoundaryValues returned unexpected error: %v", err)
	}
	want := []string{"regular", "subscription"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BoundaryValues = %v, want %v", got, want)
	}
}

func TestBoundaryValues_NeitherRangeNorOneOf(t *testing.T) {
	c := Concept{Name: "shipping fee", Kind: VariableConcept, Type: "decimal"}

	got, err := BoundaryValues(c)
	if err != nil {
		t.Fatalf("BoundaryValues returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("BoundaryValues = %v, want nil (no declared domain to generate from)", got)
	}
}

func TestBoundaryValues_NarrowRangeNoRoomForInsideValues(t *testing.T) {
	// min == max: there is no room for a value strictly between them, so
	// only the two boundary values themselves are returned.
	c := Concept{
		Name: "x", Kind: VariableConcept, HasRange: true, Min: "1.00", Max: "1.00",
		HasPlaces: true, Places: 2,
	}

	got, err := BoundaryValues(c)
	if err != nil {
		t.Fatalf("BoundaryValues returned unexpected error: %v", err)
	}
	want := []string{"1.00", "1.00"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BoundaryValues = %v, want %v", got, want)
	}
}

func TestBoundaryValues_InvalidRangeText(t *testing.T) {
	c := Concept{
		Name: "x", Kind: VariableConcept, HasRange: true, Min: "not a number", Max: "100.00",
		HasPlaces: true, Places: 2,
	}

	_, err := BoundaryValues(c)
	if err == nil {
		t.Fatal("BoundaryValues with an unparseable range.min = nil error, want an error")
	}
}

func TestExhaustiveValues_OneOf(t *testing.T) {
	c := Concept{Name: "order type", Kind: VariableConcept, OneOf: []string{"regular", "subscription"}}

	got, found, err := ExhaustiveValues(c, 10)
	if err != nil {
		t.Fatalf("ExhaustiveValues returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("ExhaustiveValues found = false, want true (a closed one_of list is already finite)")
	}
	want := []string{"regular", "subscription"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExhaustiveValues = %v, want %v", got, want)
	}
}

func TestExhaustiveValues_OneOfExceedsCap(t *testing.T) {
	c := Concept{Name: "order type", Kind: VariableConcept, OneOf: []string{"regular", "subscription", "gift"}}

	got, found, err := ExhaustiveValues(c, 2)
	if err != nil {
		t.Fatalf("ExhaustiveValues returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("ExhaustiveValues found = true with values %v, want false (3 values exceeds the cap of 2)", got)
	}
}

func TestExhaustiveValues_BoundedDecimalRange(t *testing.T) {
	c := Concept{
		Name: "quantity", Kind: VariableConcept, Type: "decimal",
		HasRange: true, Min: "0.00", Max: "0.03",
		HasPlaces: true, Places: 2,
	}

	got, found, err := ExhaustiveValues(c, 10)
	if err != nil {
		t.Fatalf("ExhaustiveValues returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("ExhaustiveValues found = false, want true (range + places give an exact, small step count)")
	}
	want := []string{"0.00", "0.01", "0.02", "0.03"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExhaustiveValues = %v, want %v", got, want)
	}
}

func TestExhaustiveValues_DecimalRangeExceedsCap(t *testing.T) {
	// 100000.00/0.01 + 1 is far more than a cap of 10.
	c := Concept{
		Name: "discounted subtotal", Kind: VariableConcept, Type: "decimal",
		HasRange: true, Min: "0.00", Max: "100000.00",
		HasPlaces: true, Places: 2,
	}

	got, found, err := ExhaustiveValues(c, 10)
	if err != nil {
		t.Fatalf("ExhaustiveValues returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("ExhaustiveValues found = true with %d values, want false (exceeds the cap of 10)", len(got))
	}
}

func TestExhaustiveValues_RangeWithoutPlaces(t *testing.T) {
	// HasRange but no declared Places: no grounded step size, so this
	// domain is not boundable at all.
	c := Concept{Name: "x", Kind: VariableConcept, HasRange: true, Min: "0", Max: "100"}

	got, found, err := ExhaustiveValues(c, 1000)
	if err != nil {
		t.Fatalf("ExhaustiveValues returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("ExhaustiveValues found = true with values %v, want false (no declared places to step by)", got)
	}
}

func TestExhaustiveValues_NeitherRangeNorOneOf(t *testing.T) {
	c := Concept{Name: "shipping fee", Kind: VariableConcept, Type: "decimal"}

	got, found, err := ExhaustiveValues(c, 1000)
	if err != nil {
		t.Fatalf("ExhaustiveValues returned unexpected error: %v", err)
	}
	if found {
		t.Errorf("ExhaustiveValues found = true with values %v, want false (no declared domain at all)", got)
	}
}

func TestExhaustiveValues_RangeNotExactMultipleOfPlaces(t *testing.T) {
	// A step of 0.01 from 0.00 never lands exactly on 0.025.
	c := Concept{
		Name: "x", Kind: VariableConcept, Type: "decimal",
		HasRange: true, Min: "0.00", Max: "0.025",
		HasPlaces: true, Places: 2,
	}

	_, _, err := ExhaustiveValues(c, 1000)
	if err == nil {
		t.Fatal("ExhaustiveValues with a range that is not an exact multiple of its precision = nil error, want an error")
	}
}

func TestExhaustiveValues_InvalidRangeText(t *testing.T) {
	c := Concept{
		Name: "x", Kind: VariableConcept, HasRange: true, Min: "not a number", Max: "100.00",
		HasPlaces: true, Places: 2,
	}

	_, _, err := ExhaustiveValues(c, 1000)
	if err == nil {
		t.Fatal("ExhaustiveValues with an unparseable range.min = nil error, want an error")
	}
}
