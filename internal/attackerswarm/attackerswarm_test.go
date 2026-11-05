package attackerswarm

import "testing"

func TestMode_ZeroValueIsOff(t *testing.T) {
	var m Mode
	if m != Off {
		t.Errorf("Mode's zero value = %v, want Off (disabled by default)", m)
	}
}

func TestValidateMode_OffAndAttackersOnlyIgnoreFamilies(t *testing.T) {
	if err := ValidateMode(Off, "same", "same"); err != nil {
		t.Errorf("ValidateMode(Off, ...) = %v, want nil", err)
	}
	if err := ValidateMode(AttackersOnly, "same", "same"); err != nil {
		t.Errorf("ValidateMode(AttackersOnly, ...) = %v, want nil", err)
	}
}

func TestValidateMode_AuditorsRequireDifferentFamily(t *testing.T) {
	err := ValidateMode(AttackersPlusAuditors, "llama", "llama")
	if err == nil {
		t.Fatal("ValidateMode(AttackersPlusAuditors) with the same family = nil, want an error")
	}
}

func TestValidateMode_AuditorsWithDifferentFamilyPasses(t *testing.T) {
	err := ValidateMode(AttackersPlusAuditors, "llama", "mistral")
	if err != nil {
		t.Errorf("ValidateMode(AttackersPlusAuditors) with different families = %v, want nil", err)
	}
}

func fiveStrategies() []Strategy {
	return []Strategy{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"}}
}

func TestAllocateNext_FewerThanMinStrategiesIsError(t *testing.T) {
	_, err := AllocateNext(fiveStrategies()[:4])
	if err == nil {
		t.Fatal("AllocateNext with 4 strategies = nil error, want one (MinStrategies is 5)")
	}
}

func TestAllocateNext_UnpulledStrategiesGoFirstLowestIndexFirst(t *testing.T) {
	strategies := fiveStrategies()
	strategies[0].Pulls = 3
	strategies[0].Successes = 3 // a's current best possible mean - still must lose to an unpulled strategy

	got, err := AllocateNext(strategies)
	if err != nil {
		t.Fatalf("AllocateNext returned unexpected error: %v", err)
	}
	if got != 1 {
		t.Errorf("AllocateNext = %d, want 1 (the lowest-index never-pulled strategy)", got)
	}
}

func TestAllocateNext_ShiftsTowardMoreProductiveStrategy(t *testing.T) {
	// Strategy index 2 always finds a violation when pulled; the rest
	// never do. Running the allocator/update loop repeatedly must shift
	// a growing share of pulls toward it - the specification's own
	// scenario, "budget shifts toward a more productive strategy."
	strategies := fiveStrategies()

	const rounds = 500
	for round := 0; round < rounds; round++ {
		i, err := AllocateNext(strategies)
		if err != nil {
			t.Fatalf("AllocateNext returned unexpected error at round %d: %v", round, err)
		}
		strategies[i].Pulls++
		if i == 2 {
			strategies[i].Successes++
		}
	}

	for i, s := range strategies {
		if i == 2 {
			continue
		}
		if strategies[2].Pulls <= s.Pulls {
			t.Errorf("strategy 2 (the productive one) got %d pulls, strategy %d got %d - want strategy 2 to receive more", strategies[2].Pulls, i, s.Pulls)
		}
	}
}

func TestAggregate_ViolationReportedIfAnyAgentReportsOne(t *testing.T) {
	got := Aggregate([]Finding{
		{AgentFamily: "llama", Violated: false},
		{AgentFamily: "mistral", Violated: true},
	})
	if !got.ViolationReported {
		t.Error("Aggregate.ViolationReported = false, want true (one agent reported a violation)")
	}
}

func TestAggregate_NoViolationWhenNoneReportOne(t *testing.T) {
	got := Aggregate([]Finding{
		{AgentFamily: "llama", Violated: false},
		{AgentFamily: "mistral", Violated: false},
	})
	if got.ViolationReported {
		t.Error("Aggregate.ViolationReported = true, want false")
	}
}

func TestAggregate_NoCrossFamilyAgreementWithOnlySingleFamily(t *testing.T) {
	got := Aggregate([]Finding{
		{AgentFamily: "llama", Violated: true},
		{AgentFamily: "llama", Violated: true},
	})
	if got.CrossFamilyAgreement {
		t.Error("Aggregate.CrossFamilyAgreement = true, want false (only one family represented)")
	}
}

func TestAggregate_CrossFamilyAgreementOnViolation(t *testing.T) {
	got := Aggregate([]Finding{
		{AgentFamily: "llama", Violated: true},
		{AgentFamily: "mistral", Violated: true},
	})
	if !got.CrossFamilyAgreement {
		t.Error("Aggregate.CrossFamilyAgreement = false, want true (two distinct families both reported a violation)")
	}
}

func TestAggregate_CrossFamilyAgreementOnClean(t *testing.T) {
	got := Aggregate([]Finding{
		{AgentFamily: "llama", Violated: false},
		{AgentFamily: "mistral", Violated: false},
	})
	if !got.CrossFamilyAgreement {
		t.Error("Aggregate.CrossFamilyAgreement = false, want true (two distinct families both reported no violation)")
	}
}

func TestAggregate_ResultDoesNotDependOnFindingOrder(t *testing.T) {
	a := []Finding{
		{AgentFamily: "llama", Violated: true},
		{AgentFamily: "mistral", Violated: false},
		{AgentFamily: "qwen", Violated: true},
	}
	b := []Finding{a[2], a[0], a[1]}

	gotA, gotB := Aggregate(a), Aggregate(b)
	if gotA != gotB {
		t.Errorf("Aggregate depended on finding order: %+v (original) vs %+v (reordered)", gotA, gotB)
	}
}
