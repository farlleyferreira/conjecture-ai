package cjtr

import "testing"

func TestReadSimplifications_RealExample(t *testing.T) {
	root := loadNode(t, "simplifications: [domestic shipping only, one delivery per order]\n")

	got, err := ReadSimplifications(root)
	if err != nil {
		t.Fatalf("ReadSimplifications returned unexpected error: %v", err)
	}
	want := []string{"domestic shipping only", "one delivery per order"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ReadSimplifications = %v, want %v", got, want)
	}
}

func TestReadSimplifications_NoKey(t *testing.T) {
	root := loadNode(t, "conjecture: 1.0.0\n")

	got, err := ReadSimplifications(root)
	if err != nil {
		t.Fatalf("ReadSimplifications returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("ReadSimplifications with no \"simplifications\" key = %v, want nil", got)
	}
}

func TestReadSimplifications_NotAList(t *testing.T) {
	root := loadNode(t, "simplifications: a plain string\n")

	_, err := ReadSimplifications(root)
	if err == nil {
		t.Fatal("ReadSimplifications with \"simplifications\" as a scalar = nil error, want an error")
	}
}
