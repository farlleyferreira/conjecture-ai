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
