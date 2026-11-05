package llmorchestration

import "testing"

func TestRequireSeparateSessions_DistinctCorrectRolesPasses(t *testing.T) {
	err := RequireSeparateSessions(
		Session{ID: "draft-1", Role: Drafter},
		Session{ID: "impl-1", Role: Implementer},
	)
	if err != nil {
		t.Errorf("RequireSeparateSessions returned unexpected error: %v", err)
	}
}

func TestRequireSeparateSessions_SameSessionIDRejected(t *testing.T) {
	err := RequireSeparateSessions(
		Session{ID: "same", Role: Drafter},
		Session{ID: "same", Role: Implementer},
	)
	if err == nil {
		t.Fatal("RequireSeparateSessions with the same session ID = nil error, want one")
	}
}

func TestRequireSeparateSessions_DraftingSessionWithWrongRoleRejected(t *testing.T) {
	err := RequireSeparateSessions(
		Session{ID: "draft-1", Role: Implementer}, // wrong role
		Session{ID: "impl-1", Role: Implementer},
	)
	if err == nil {
		t.Fatal("RequireSeparateSessions with a drafting session not holding Drafter = nil error, want one")
	}
}

func TestRequireSeparateSessions_ImplementingSessionWithWrongRoleRejected(t *testing.T) {
	err := RequireSeparateSessions(
		Session{ID: "draft-1", Role: Drafter},
		Session{ID: "impl-1", Role: Drafter}, // wrong role
	)
	if err == nil {
		t.Fatal("RequireSeparateSessions with an implementing session not holding Implementer = nil error, want one")
	}
}

func TestVisibleCases_WithholdsHiddenCases(t *testing.T) {
	cases := []JudgeCase{
		{Input: "a", Hidden: false},
		{Input: "b", Hidden: true},
		{Input: "c", Hidden: false},
		{Input: "d", Hidden: true},
	}

	got := VisibleCases(cases)

	if len(got) != 2 {
		t.Fatalf("VisibleCases returned %d cases, want 2", len(got))
	}
	for _, c := range got {
		if c.Hidden {
			t.Errorf("VisibleCases included a hidden case: %+v", c)
		}
	}
}

func TestVisibleCases_AllHiddenReturnsNone(t *testing.T) {
	got := VisibleCases([]JudgeCase{{Input: "a", Hidden: true}})
	if len(got) != 0 {
		t.Errorf("VisibleCases = %v, want none", got)
	}
}

func TestAllowed_HumanMayDoAnything(t *testing.T) {
	actions := []Action{
		SubmitProposal, WriteEvidenceRecord, CallAdversarialSLMDirectly,
		SignSpecOnBehalfOfAnotherUser, ChangeConfiguration, ConfirmSpecification,
		PromoteHypothesis, MarkVerdictConformant,
	}
	for _, a := range actions {
		if !Allowed(Human, a) {
			t.Errorf("Allowed(Human, %v) = false, want true", a)
		}
	}
}

func TestAllowed_LLMMayOnlySubmitProposal(t *testing.T) {
	if !Allowed(LLM, SubmitProposal) {
		t.Error("Allowed(LLM, SubmitProposal) = false, want true")
	}

	blocked := []Action{
		WriteEvidenceRecord, CallAdversarialSLMDirectly,
		SignSpecOnBehalfOfAnotherUser, ChangeConfiguration, ConfirmSpecification,
		PromoteHypothesis, MarkVerdictConformant,
	}
	for _, a := range blocked {
		if Allowed(LLM, a) {
			t.Errorf("Allowed(LLM, %v) = true, want false", a)
		}
	}
}
