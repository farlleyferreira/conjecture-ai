package conformancecycle

import "testing"

func TestBudget_NotExhaustedBelowBothLimits(t *testing.T) {
	b := Budget{MaxRounds: 5, MaxTokens: 1000}
	if b.Exhausted(4, 999) {
		t.Error("Exhausted(4, 999) = true, want false (below both limits)")
	}
}

func TestBudget_ExhaustedWhenRoundReachesMax(t *testing.T) {
	b := Budget{MaxRounds: 5, MaxTokens: 1000}
	if !b.Exhausted(5, 0) {
		t.Error("Exhausted(5, 0) = false, want true (round reached MaxRounds)")
	}
}

func TestBudget_ExhaustedWhenTokensReachMax(t *testing.T) {
	b := Budget{MaxRounds: 5, MaxTokens: 1000}
	if !b.Exhausted(0, 1000) {
		t.Error("Exhausted(0, 1000) = false, want true (tokensUsed reached MaxTokens)")
	}
}

func TestVerdictOnBudgetExhausted_IsNonConformantNotInconclusive(t *testing.T) {
	if VerdictOnBudgetExhausted != NonConformant {
		t.Errorf("VerdictOnBudgetExhausted = %v, want NonConformant", VerdictOnBudgetExhausted)
	}
}
