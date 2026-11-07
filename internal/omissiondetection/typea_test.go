package omissiondetection

import "testing"

func TestCheckTypeA_ZeroCasesIsOmission(t *testing.T) {
	omission, found := CheckTypeA("area::phenomenon::rule::hyp", 0)
	if !found {
		t.Fatal("CheckTypeA with 0 exercised cases = not found, want an omission")
	}
	if omission.HypothesisID != "area::phenomenon::rule::hyp" {
		t.Errorf("omission.HypothesisID = %q, want %q", omission.HypothesisID, "area::phenomenon::rule::hyp")
	}
}

func TestCheckTypeA_AnyCasesIsNotAnOmission(t *testing.T) {
	for _, n := range []int{1, 2, 100} {
		if _, found := CheckTypeA("x", n); found {
			t.Errorf("CheckTypeA with %d exercised cases = found an omission, want none", n)
		}
	}
}
