package userdefinedlaws

import "testing"

func TestStatus_StringAndParseRoundTrip(t *testing.T) {
	statuses := []Status{Unverified, Conformant, ConformantUnderGeneralization, Violated, Contested, Invalidated}
	for _, s := range statuses {
		parsed, err := ParseStatus(s.String())
		if err != nil {
			t.Errorf("ParseStatus(%q) returned unexpected error: %v", s.String(), err)
			continue
		}
		if parsed != s {
			t.Errorf("ParseStatus(%q) = %v, want %v", s.String(), parsed, s)
		}
	}
}

func TestParseStatus_UnrecognizedIsError(t *testing.T) {
	_, err := ParseStatus("not-a-status")
	if err == nil {
		t.Fatal(`ParseStatus("not-a-status") = nil error, want one`)
	}
}

func TestStatus_ZeroValueIsUnverified(t *testing.T) {
	var s Status
	if s != Unverified {
		t.Errorf("zero value Status = %v, want Unverified", s)
	}
}

func TestResolveConflict_FormalProofAlwaysInvalidatesRegardlessOfAuthority(t *testing.T) {
	cases := []bool{true, false}
	for _, equalAuthority := range cases {
		got := ResolveConflict(true, equalAuthority)
		if got != AutoInvalidated {
			t.Errorf("ResolveConflict(true, %v) = %v, want AutoInvalidated", equalAuthority, got)
		}
	}
}

func TestResolveConflict_EqualAuthorityWithNoProofRaisesQuestion(t *testing.T) {
	got := ResolveConflict(false, true)
	if got != RaiseQuestion {
		t.Errorf("ResolveConflict(false, true) = %v, want RaiseQuestion", got)
	}
}

func TestResolveConflict_UnequalAuthorityWithNoProofLawPrevails(t *testing.T) {
	// Models "Precedence over promoted hypotheses": a hypothesis is
	// never of equal authority to a user-defined law.
	got := ResolveConflict(false, false)
	if got != LawPrevails {
		t.Errorf("ResolveConflict(false, false) = %v, want LawPrevails", got)
	}
}

func TestMeetsContestedThreshold(t *testing.T) {
	cases := []struct {
		count, threshold int
		want             bool
	}{
		{count: 4, threshold: 5, want: false},
		{count: 5, threshold: 5, want: true},
		{count: 6, threshold: 5, want: true},
		{count: 0, threshold: 1, want: false},
	}
	for _, c := range cases {
		got := MeetsContestedThreshold(c.count, c.threshold)
		if got != c.want {
			t.Errorf("MeetsContestedThreshold(%d, %d) = %v, want %v", c.count, c.threshold, got, c.want)
		}
	}
}
