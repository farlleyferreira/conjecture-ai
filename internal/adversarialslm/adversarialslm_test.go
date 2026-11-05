package adversarialslm

import "testing"

func permissive(name, family string) Candidate {
	return Candidate{Name: name, Family: family, License: Permissive}
}

func TestSelectSLM_UsesDefaultWhenNoFamilyCollision(t *testing.T) {
	def := permissive("phi-mini", "phi")
	reserve := permissive("qwen-small", "qwen")

	got, err := SelectSLM(def, reserve, "llama")
	if err != nil {
		t.Fatalf("SelectSLM returned unexpected error: %v", err)
	}
	if got != def {
		t.Errorf("SelectSLM = %+v, want the default %+v", got, def)
	}
}

func TestSelectSLM_SwitchesToReserveOnFamilyCollision(t *testing.T) {
	def := permissive("llama-mini", "llama")
	reserve := permissive("qwen-small", "qwen")

	got, err := SelectSLM(def, reserve, "llama")
	if err != nil {
		t.Fatalf("SelectSLM returned unexpected error: %v", err)
	}
	if got != reserve {
		t.Errorf("SelectSLM = %+v, want the reserve %+v (family collision with the generator)", got, reserve)
	}
}

func TestSelectSLM_DefaultWithNonPermissiveLicenseIsError(t *testing.T) {
	def := Candidate{Name: "research-model", Family: "phi", License: ResearchOnly}
	reserve := permissive("qwen-small", "qwen")

	_, err := SelectSLM(def, reserve, "llama")
	if err == nil {
		t.Fatal("SelectSLM with a research-only default = nil error, want one")
	}
}

func TestSelectSLM_ReserveWithNonPermissiveLicenseIsErrorEvenIfUnused(t *testing.T) {
	// The default is selected (no collision), but the reserve's license
	// must still be checked - the specification forbids a non-permissive
	// license for either role, not just whichever role a given run ends
	// up using.
	def := permissive("phi-mini", "phi")
	reserve := Candidate{Name: "noncommercial-model", Family: "qwen", License: NonCommercial}

	_, err := SelectSLM(def, reserve, "llama")
	if err == nil {
		t.Fatal("SelectSLM with a non-commercial reserve = nil error, want one")
	}
}

func TestParseAttackOutput_ValidViolationFound(t *testing.T) {
	got, err := ParseAttackOutput([]byte(`{"outcome": "violation_found", "counterexample": "x = -1"}`))
	if err != nil {
		t.Fatalf("ParseAttackOutput returned unexpected error: %v", err)
	}
	if got.Outcome != ViolationFound {
		t.Errorf("Outcome = %v, want ViolationFound", got.Outcome)
	}
	if got.Counterexample != "x = -1" {
		t.Errorf("Counterexample = %q, want %q", got.Counterexample, "x = -1")
	}
}

func TestParseAttackOutput_ValidNoViolationFoundWithoutCounterexample(t *testing.T) {
	got, err := ParseAttackOutput([]byte(`{"outcome": "no_violation_found"}`))
	if err != nil {
		t.Fatalf("ParseAttackOutput returned unexpected error: %v", err)
	}
	if got.Outcome != NoViolationFound {
		t.Errorf("Outcome = %v, want NoViolationFound", got.Outcome)
	}
}

func TestParseAttackOutput_UnrecognizedOutcomeValueRejected(t *testing.T) {
	_, err := ParseAttackOutput([]byte(`{"outcome": "maybe"}`))
	if err == nil {
		t.Fatal("ParseAttackOutput with an outcome outside the grammar's enum = nil error, want one")
	}
}

func TestParseAttackOutput_UnknownFieldRejected(t *testing.T) {
	_, err := ParseAttackOutput([]byte(`{"outcome": "violation_found", "extra": "field"}`))
	if err == nil {
		t.Fatal("ParseAttackOutput with an undeclared field = nil error, want one (the grammar is closed)")
	}
}

func TestParseAttackOutput_MissingOutcomeRejected(t *testing.T) {
	_, err := ParseAttackOutput([]byte(`{"counterexample": "x = -1"}`))
	if err == nil {
		t.Fatal("ParseAttackOutput with no outcome field = nil error, want one")
	}
}

func TestParseAttackOutput_FreeFormTextRejected(t *testing.T) {
	_, err := ParseAttackOutput([]byte(`the hypothesis seems to fail when x is negative`))
	if err == nil {
		t.Fatal("ParseAttackOutput with free-form text = nil error, want one (must not be interpreted loosely)")
	}
}

func TestAttackOutput_EvidenceFacts(t *testing.T) {
	cases := []struct {
		outcome      AttackOutcome
		wantAttacked bool
		wantSurvived bool
	}{
		{ViolationFound, true, false},
		{NoViolationFound, true, true},
	}
	for _, c := range cases {
		attacked, survived := AttackOutput{Outcome: c.outcome}.EvidenceFacts()
		if attacked != c.wantAttacked || survived != c.wantSurvived {
			t.Errorf("EvidenceFacts(%v) = (%v, %v), want (%v, %v)", c.outcome, attacked, survived, c.wantAttacked, c.wantSurvived)
		}
	}
}
