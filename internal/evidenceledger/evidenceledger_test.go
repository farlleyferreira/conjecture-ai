package evidenceledger

import "testing"

func TestWeight_RequiredCases(t *testing.T) {
	tests := []struct {
		w    Weight
		want int
	}{
		{Low, 100},
		{Medium, 300},
		{High, 1000},
		{Critical, 3000},
	}
	for _, tt := range tests {
		if got := tt.w.RequiredCases(); got != tt.want {
			t.Errorf("Weight(%v).RequiredCases() = %d, want %d", tt.w, got, tt.want)
		}
	}
}

func TestWeight_MaxViolationPercent(t *testing.T) {
	tests := []struct {
		w    Weight
		want float64
	}{
		{Low, 3},
		{Medium, 1},
		{High, 0.3},
		{Critical, 0.1},
	}
	for _, tt := range tests {
		if got := tt.w.MaxViolationPercent(); got != tt.want {
			t.Errorf("Weight(%v).MaxViolationPercent() = %v, want %v", tt.w, got, tt.want)
		}
	}
}

func TestWeight_MeetsPromotionThreshold(t *testing.T) {
	if Critical.MeetsPromotionThreshold(2999) {
		t.Error("Critical.MeetsPromotionThreshold(2999) = true, want false (one short of 3000)")
	}
	if !Critical.MeetsPromotionThreshold(3000) {
		t.Error("Critical.MeetsPromotionThreshold(3000) = false, want true")
	}
	if !Low.MeetsPromotionThreshold(150) {
		t.Error("Low.MeetsPromotionThreshold(150) = false, want true (above the 100 required)")
	}
}

func TestDemote(t *testing.T) {
	tests := []struct {
		from, want Stage
	}{
		{Law, Theory},
		{Theory, Thesis},
		{Thesis, Hypothesis},
		{Hypothesis, Hypothesis},
	}
	for _, tt := range tests {
		if got := Demote(tt.from); got != tt.want {
			t.Errorf("Demote(%v) = %v, want %v", tt.from, got, tt.want)
		}
	}
}

func TestStage_String(t *testing.T) {
	tests := []struct {
		s    Stage
		want string
	}{
		{Hypothesis, "Hypothesis"},
		{Thesis, "Thesis"},
		{Theory, "Theory"},
		{Law, "Law"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Stage(%v).String() = %q, want %q", int(tt.s), got, tt.want)
		}
	}
}

func TestValidityHash_Deterministic(t *testing.T) {
	h1 := ValidityHash("spec-1.0", "code-abc", "harness-1", "slm-x", "operator-y")
	h2 := ValidityHash("spec-1.0", "code-abc", "harness-1", "slm-x", "operator-y")
	if Stale(h1, h2) {
		t.Error("ValidityHash gave different results for identical inputs")
	}
}

func TestValidityHash_DiffersOnAnyChangedDependency(t *testing.T) {
	base := ValidityHash("spec-1.0", "code-abc", "harness-1", "slm-x", "operator-y")

	changed := ValidityHash("spec-1.1", "code-abc", "harness-1", "slm-x", "operator-y")
	if !Stale(base, changed) {
		t.Error("changing only specVersion produced the same validity hash")
	}

	changedCode := ValidityHash("spec-1.0", "code-def", "harness-1", "slm-x", "operator-y")
	if !Stale(base, changedCode) {
		t.Error("changing only codeVersion produced the same validity hash")
	}
}

func TestValidityHash_NoFieldConcatenationCollision(t *testing.T) {
	// ("a", "bc", ...) must not hash the same as ("ab", "c", ...).
	h1 := ValidityHash("a", "bc", "x", "y", "z")
	h2 := ValidityHash("ab", "c", "x", "y", "z")
	if !Stale(h1, h2) {
		t.Error("ValidityHash(\"a\",\"bc\",...) collided with ValidityHash(\"ab\",\"c\",...)")
	}
}

func TestStale(t *testing.T) {
	current := ValidityHash("spec-1.0", "code-abc", "harness-1", "slm-x", "operator-y")

	if Stale(current, current) {
		t.Error("Stale(current, current) = true, want false")
	}

	recorded := ValidityHash("spec-0.9", "code-abc", "harness-1", "slm-x", "operator-y")
	if !Stale(recorded, current) {
		t.Error("Stale(recorded-against-an-older-spec, current) = false, want true")
	}
}
