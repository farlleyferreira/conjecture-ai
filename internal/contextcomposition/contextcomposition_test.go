package contextcomposition

import "testing"

func TestLifecycleState_ZeroValueIsActive(t *testing.T) {
	var s LifecycleState
	if s != Active {
		t.Errorf("LifecycleState's zero value = %v, want Active", s)
	}
}

func TestLifecycleState_RoundTrip(t *testing.T) {
	for _, s := range []LifecycleState{Active, Archived} {
		got, err := ParseLifecycleState(s.String())
		if err != nil {
			t.Fatalf("ParseLifecycleState(%q) returned unexpected error: %v", s.String(), err)
		}
		if got != s {
			t.Errorf("ParseLifecycleState(%q) = %v, want %v", s.String(), got, s)
		}
	}
}

func TestParseLifecycleState_UnrecognizedIsError(t *testing.T) {
	_, err := ParseLifecycleState("Deleted")
	if err == nil {
		t.Fatal("ParseLifecycleState(\"Deleted\") = nil error, want one (contexts are never deleted)")
	}
}

func TestSupersede_ArchivesRatherThanDeletes(t *testing.T) {
	ctx := Context{ID: "ctx-1", State: Active}

	got := Supersede(ctx, "ctx-2")

	if got.ID != "ctx-1" {
		t.Errorf("Supersede changed ID to %q, want it unchanged (\"ctx-1\")", got.ID)
	}
	if got.State != Archived {
		t.Errorf("Supersede's result state = %v, want Archived", got.State)
	}
	if got.SupersededBy != "ctx-2" {
		t.Errorf("SupersededBy = %q, want %q", got.SupersededBy, "ctx-2")
	}
}

func TestCompositionMode_RoundTrip(t *testing.T) {
	for _, m := range []CompositionMode{ByReference, ByEmbedding, ByInheritance, ByView} {
		got, err := ParseCompositionMode(m.String())
		if err != nil {
			t.Fatalf("ParseCompositionMode(%q) returned unexpected error: %v", m.String(), err)
		}
		if got != m {
			t.Errorf("ParseCompositionMode(%q) = %v, want %v", m.String(), got, m)
		}
	}
}

func TestParseCompositionMode_UnrecognizedIsError(t *testing.T) {
	_, err := ParseCompositionMode("ByCopy")
	if err == nil {
		t.Fatal("ParseCompositionMode(\"ByCopy\") = nil error, want one (the specification names exactly four modes)")
	}
}
