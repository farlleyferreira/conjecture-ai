package cjtr

import "testing"

func TestReadScope_RealExample(t *testing.T) {
	src := `
scope:
  inside: [cart, coupon, shipping calculation]
  later: [store pickup]
  outside: [international shipping]
`
	root := loadNode(t, src)

	scope, err := ReadScope(root)
	if err != nil {
		t.Fatalf("ReadScope returned unexpected error: %v", err)
	}
	if scope == nil {
		t.Fatal("ReadScope = nil, want a non-nil *Scope")
	}

	wantInside := []string{"cart", "coupon", "shipping calculation"}
	if len(scope.Inside) != 3 || scope.Inside[0] != wantInside[0] || scope.Inside[2] != wantInside[2] {
		t.Errorf("scope.Inside = %v, want %v", scope.Inside, wantInside)
	}
	if len(scope.Later) != 1 || scope.Later[0] != "store pickup" {
		t.Errorf("scope.Later = %v, want [store pickup]", scope.Later)
	}
	if len(scope.Outside) != 1 || scope.Outside[0] != "international shipping" {
		t.Errorf("scope.Outside = %v, want [international shipping]", scope.Outside)
	}
}

func TestReadScope_NoKey(t *testing.T) {
	root := loadNode(t, "conjecture: 1.0.0\n")

	scope, err := ReadScope(root)
	if err != nil {
		t.Fatalf("ReadScope returned unexpected error: %v", err)
	}
	if scope != nil {
		t.Errorf("ReadScope with no \"scope\" key = %+v, want nil", scope)
	}
}

func TestReadScope_PartialFields(t *testing.T) {
	root := loadNode(t, "scope:\n  inside: [cart]\n")

	scope, err := ReadScope(root)
	if err != nil {
		t.Fatalf("ReadScope returned unexpected error: %v", err)
	}
	if len(scope.Inside) != 1 || scope.Inside[0] != "cart" {
		t.Errorf("scope.Inside = %v, want [cart]", scope.Inside)
	}
	if scope.Later != nil || scope.Outside != nil {
		t.Errorf("scope.Later/Outside = %v/%v, want nil/nil (not declared)", scope.Later, scope.Outside)
	}
}

func TestReadScope_NotAMapping(t *testing.T) {
	root := loadNode(t, "scope: a plain string\n")

	_, err := ReadScope(root)
	if err == nil {
		t.Fatal("ReadScope with \"scope\" as a scalar = nil error, want an error")
	}
}

func TestReadScope_FieldNotAList(t *testing.T) {
	root := loadNode(t, "scope:\n  inside: cart\n")

	_, err := ReadScope(root)
	if err == nil {
		t.Fatal("ReadScope with \"inside\" as a scalar = nil error, want an error")
	}
}
