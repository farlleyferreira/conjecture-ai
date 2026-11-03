package cjtr

import "testing"

func TestReadObservation_RealExample(t *testing.T) {
	src := `
observation:
  operation: the shipping fee is calculated
  interface: HTTP POST /orders/shipping
`
	root := loadNode(t, src)

	obs, err := ReadObservation(root)
	if err != nil {
		t.Fatalf("ReadObservation returned unexpected error: %v", err)
	}
	if obs == nil {
		t.Fatal("ReadObservation = nil, want a non-nil *Observation")
	}
	if obs.Operation != "the shipping fee is calculated" {
		t.Errorf("obs.Operation = %q", obs.Operation)
	}
	if obs.Interface != "HTTP POST /orders/shipping" {
		t.Errorf("obs.Interface = %q", obs.Interface)
	}
}

func TestReadObservation_NoKey(t *testing.T) {
	root := loadNode(t, "conjecture: 1.0.0\n")

	obs, err := ReadObservation(root)
	if err != nil {
		t.Fatalf("ReadObservation returned unexpected error: %v", err)
	}
	if obs != nil {
		t.Errorf("ReadObservation with no \"observation\" key = %+v, want nil", obs)
	}
}

func TestReadObservation_NotAMapping(t *testing.T) {
	root := loadNode(t, "observation: a plain string\n")

	_, err := ReadObservation(root)
	if err == nil {
		t.Fatal("ReadObservation with \"observation\" as a scalar = nil error, want an error")
	}
}
