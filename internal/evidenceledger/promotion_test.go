package evidenceledger

import "testing"

func TestPromoteToTheory(t *testing.T) {
	tests := []struct {
		name                                                                           string
		allThesesConfirmed, mechanismConfirmed, differentFamilyReproduced, swarmRobust bool
		want                                                                           bool
	}{
		{"all four true", true, true, true, true, true},
		{"theses not confirmed", false, true, true, true, false},
		{"mechanism not confirmed", true, false, true, true, false},
		{"different family not reproduced", true, true, false, true, false},
		{"swarm not robust", true, true, true, false, false},
	}
	for _, tt := range tests {
		got := PromoteToTheory(tt.allThesesConfirmed, tt.mechanismConfirmed, tt.differentFamilyReproduced, tt.swarmRobust)
		if got != tt.want {
			t.Errorf("%s: PromoteToTheory(...) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPromoteToLaw(t *testing.T) {
	tests := []struct {
		name                                                            string
		isUniversal, hiddenSamplesConformant, newCodeVersionsConformant bool
		want                                                            bool
	}{
		{"all three true", true, true, true, true},
		{"not universal", false, true, true, false},
		{"not conformant against hidden samples", true, false, true, false},
		{"not conformant across new code versions", true, true, false, false},
	}
	for _, tt := range tests {
		got := PromoteToLaw(tt.isUniversal, tt.hiddenSamplesConformant, tt.newCodeVersionsConformant)
		if got != tt.want {
			t.Errorf("%s: PromoteToLaw(...) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAllAtStage_EveryHypothesisAtTheGivenStage(t *testing.T) {
	db := openTestStore(t)
	for _, id := range []string{"a::b::c::h1", "a::b::c::h2"} {
		if _, err := Append(db, Evidence{HypothesisID: id, Stage: Thesis, ValidityHash: testValidityHash}); err != nil {
			t.Fatalf("Append returned unexpected error: %v", err)
		}
	}

	got, err := AllAtStage(db, []string{"a::b::c::h1", "a::b::c::h2"}, Thesis)
	if err != nil {
		t.Fatalf("AllAtStage returned unexpected error: %v", err)
	}
	if !got {
		t.Error("AllAtStage = false, want true (both hypotheses are at Thesis)")
	}
}

func TestAllAtStage_OneHypothesisAtADifferentStageFails(t *testing.T) {
	db := openTestStore(t)
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h1", Stage: Thesis, ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h2", Stage: Hypothesis, ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	got, err := AllAtStage(db, []string{"a::b::c::h1", "a::b::c::h2"}, Thesis)
	if err != nil {
		t.Fatalf("AllAtStage returned unexpected error: %v", err)
	}
	if got {
		t.Error("AllAtStage = true, want false (h2 is still at Hypothesis)")
	}
}

func TestAllAtStage_NoEvidenceAtAllFails(t *testing.T) {
	db := openTestStore(t)

	got, err := AllAtStage(db, []string{"a::b::c::never-checked"}, Thesis)
	if err != nil {
		t.Fatalf("AllAtStage returned unexpected error: %v", err)
	}
	if got {
		t.Error("AllAtStage = true, want false (no evidence recorded at all)")
	}
}

func TestAllAtStage_LatestRecordWins(t *testing.T) {
	db := openTestStore(t)
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h1", Stage: Hypothesis, ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h1", Stage: Thesis, ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("second Append returned unexpected error: %v", err)
	}

	got, err := AllAtStage(db, []string{"a::b::c::h1"}, Thesis)
	if err != nil {
		t.Fatalf("AllAtStage returned unexpected error: %v", err)
	}
	if !got {
		t.Error("AllAtStage = false, want true (the latest record is what counts)")
	}
}

func TestSwarmAttackRobust_NoViolationAcrossAllAttacks(t *testing.T) {
	db := openTestStore(t)
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h1", ProducedBy: "adversarial SLM", Attacked: "adversarial SLM attack", Survived: "no violation found by adversarial SLM", ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h1", ProducedBy: "conformance cycle", Attacked: "directed sampling", Survived: "every generated case conformed", ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	got, err := SwarmAttackRobust(db, "a::b::c::h1", "adversarial SLM")
	if err != nil {
		t.Fatalf("SwarmAttackRobust returned unexpected error: %v", err)
	}
	if !got {
		t.Error("SwarmAttackRobust = false, want true (the only adversarial-SLM attack found no violation)")
	}
}

func TestSwarmAttackRobust_AnyViolationFailsIt(t *testing.T) {
	db := openTestStore(t)
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h1", ProducedBy: "adversarial SLM", Attacked: "adversarial SLM attack; found counterexample: x = -1", ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	got, err := SwarmAttackRobust(db, "a::b::c::h1", "adversarial SLM")
	if err != nil {
		t.Fatalf("SwarmAttackRobust returned unexpected error: %v", err)
	}
	if got {
		t.Error("SwarmAttackRobust = true, want false (a violation was recorded)")
	}
}

func TestSwarmAttackRobust_NeverAttackedIsNotRobust(t *testing.T) {
	db := openTestStore(t)
	if _, err := Append(db, Evidence{HypothesisID: "a::b::c::h1", ProducedBy: "conformance cycle", Attacked: "directed sampling", Survived: "every generated case conformed", ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	got, err := SwarmAttackRobust(db, "a::b::c::h1", "adversarial SLM")
	if err != nil {
		t.Fatalf("SwarmAttackRobust returned unexpected error: %v", err)
	}
	if got {
		t.Error("SwarmAttackRobust = true, want false (no adversarial-SLM attack was ever recorded)")
	}
}
