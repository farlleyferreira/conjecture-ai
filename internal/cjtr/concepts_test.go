package cjtr

import (
	"reflect"
	"testing"
)

// realConcepts is the "concepts" section of the worked .cjtr example
// (conjecture/phenomena/shipping/free-shipping.cjtr), kept faithful to its
// actual field names and shape.
const realConcepts = `
concepts:
  discounted subtotal:
    label: subtotal com desconto
    variable: input.discounted_subtotal
    type: decimal
    unit: BRL
    places: 2
    range: {min: 0.00, max: 100000.00}
  order type:
    label: tipo de pedido
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    label: frete
    variable: output.shipping_fee
    type: decimal
    unit: BRL
    places: 2
  free shipping threshold:
    label: limite do frete gratis
    value: 100.00
    unit: BRL
`

func TestConcepts_RealExample(t *testing.T) {
	root := loadNode(t, realConcepts)

	concepts, err := Concepts(root)
	if err != nil {
		t.Fatalf("Concepts returned unexpected error: %v", err)
	}
	if len(concepts) != 4 {
		t.Fatalf("Concepts returned %d entries, want 4", len(concepts))
	}

	subtotal, ok := concepts["discounted subtotal"]
	if !ok {
		t.Fatal("\"discounted subtotal\" not found")
	}
	wantSubtotal := Concept{
		Name:      "discounted subtotal",
		Label:     "subtotal com desconto",
		Kind:      VariableConcept,
		Variable:  "input.discounted_subtotal",
		Type:      "decimal",
		Unit:      "BRL",
		HasPlaces: true,
		Places:    2,
		HasRange:  true,
		Min:       "0.00",
		Max:       "100000.00",
	}
	if !reflect.DeepEqual(subtotal, wantSubtotal) {
		t.Errorf("concepts[\"discounted subtotal\"] = %+v, want %+v", subtotal, wantSubtotal)
	}

	orderType, ok := concepts["order type"]
	if !ok {
		t.Fatal("\"order type\" not found")
	}
	if orderType.Kind != VariableConcept || orderType.Variable != "input.order_type" {
		t.Errorf("order type = %+v, want variable input.order_type", orderType)
	}
	if len(orderType.OneOf) != 2 || orderType.OneOf[0] != "regular" || orderType.OneOf[1] != "subscription" {
		t.Errorf("order type.OneOf = %v, want [regular subscription]", orderType.OneOf)
	}

	threshold, ok := concepts["free shipping threshold"]
	if !ok {
		t.Fatal("\"free shipping threshold\" not found")
	}
	if threshold.Kind != ConstantConcept || threshold.Value != "100.00" {
		t.Errorf("free shipping threshold = %+v, want constant 100.00", threshold)
	}
	// The exact decimal must survive as text, never a lossy float conversion.
	if threshold.Value != "100.00" {
		t.Errorf("free shipping threshold.Value = %q, want exact text %q", threshold.Value, "100.00")
	}
}

func TestConcepts_MissingConceptsKey(t *testing.T) {
	root := loadNode(t, "conjecture: 1.0.0\n")

	_, err := Concepts(root)
	if err == nil {
		t.Fatal("Concepts with no \"concepts\" key = nil error, want an error")
	}
}

func TestConcepts_MissingVariableAndValue(t *testing.T) {
	root := loadNode(t, "concepts:\n  x:\n    label: x\n")

	_, err := Concepts(root)
	if err == nil {
		t.Fatal("Concepts with a concept declaring neither variable nor value = nil error, want an error")
	}
}

func TestConcepts_BothVariableAndValue(t *testing.T) {
	root := loadNode(t, "concepts:\n  x:\n    variable: input.x\n    value: 1\n")

	_, err := Concepts(root)
	if err == nil {
		t.Fatal("Concepts with a concept declaring both variable and value = nil error, want an error")
	}
}

func TestConcepts_TableConceptRejected(t *testing.T) {
	src := `
concepts:
  table shipping fee:
    label: frete da tabela
    table: bases/shipping-table.csv
    by:
      region: {label: regiao, variable: input.region, type: text}
`
	root := loadNode(t, src)

	_, err := Concepts(root)
	if err == nil {
		t.Fatal("Concepts with a table concept = nil error, want an error (not supported yet)")
	}
}

func TestConcepts_InvalidPlaces(t *testing.T) {
	root := loadNode(t, "concepts:\n  x:\n    value: 1\n    places: -1\n")

	_, err := Concepts(root)
	if err == nil {
		t.Fatal("Concepts with negative \"places\" = nil error, want an error")
	}
}

func TestConcepts_RangeMissingMax(t *testing.T) {
	root := loadNode(t, "concepts:\n  x:\n    value: 1\n    range: {min: 0}\n")

	_, err := Concepts(root)
	if err == nil {
		t.Fatal("Concepts with a range missing \"max\" = nil error, want an error")
	}
}
