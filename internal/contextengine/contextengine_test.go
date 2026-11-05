package contextengine

import "testing"

func TestGatedWeight_NoSignalsPassesBaseThrough(t *testing.T) {
	got, err := GatedWeight(Medium, false, nil)
	if err != nil {
		t.Fatalf("GatedWeight returned unexpected error: %v", err)
	}
	if got != Medium {
		t.Errorf("GatedWeight = %v, want Medium (base unchanged)", got)
	}
}

func TestGatedWeight_UserDefinedLawIsAlwaysCritical(t *testing.T) {
	got, err := GatedWeight(Low, true, nil)
	if err != nil {
		t.Fatalf("GatedWeight returned unexpected error: %v", err)
	}
	if got != Critical {
		t.Errorf("GatedWeight = %v, want Critical", got)
	}
}

func TestGatedWeight_UserDefinedLawOverridesEvenAnEmptyCategoryList(t *testing.T) {
	got, err := GatedWeight(Critical, false, []Category{}) // empty, not nil - should behave the same
	if err != nil {
		t.Fatalf("GatedWeight returned unexpected error: %v", err)
	}
	if got != Critical {
		t.Errorf("GatedWeight = %v, want Critical (already there; empty categories changes nothing)", got)
	}
}

func TestGatedWeight_SensitiveCategoryRaisesLowToHigh(t *testing.T) {
	got, err := GatedWeight(Low, false, []Category{Money})
	if err != nil {
		t.Fatalf("GatedWeight returned unexpected error: %v", err)
	}
	if got != High {
		t.Errorf("GatedWeight = %v, want High", got)
	}
}

func TestGatedWeight_SensitiveCategoryNeverLowersAnAlreadyHigherWeight(t *testing.T) {
	got, err := GatedWeight(Critical, false, []Category{Identity})
	if err != nil {
		t.Fatalf("GatedWeight returned unexpected error: %v", err)
	}
	if got != Critical {
		t.Errorf("GatedWeight = %v, want Critical (a category floor never lowers an already-higher weight)", got)
	}
}

func TestGatedWeight_EachNamedCategoryTriggersTheFloor(t *testing.T) {
	for _, c := range []Category{Money, Identity, Permissions, PersonalData} {
		got, err := GatedWeight(Low, false, []Category{c})
		if err != nil {
			t.Fatalf("GatedWeight(%q) returned unexpected error: %v", c, err)
		}
		if got != High {
			t.Errorf("GatedWeight(%q) = %v, want High", c, got)
		}
	}
}

func TestGatedWeight_UnrecognizedCategoryIsAnError(t *testing.T) {
	_, err := GatedWeight(Low, false, []Category{"security"})
	if err == nil {
		t.Fatal("GatedWeight with an unrecognized category = nil error, want one (this is a closed list, not an open catalog)")
	}
}

func TestPropagate_AffectsOnlyImmediateNeighbors(t *testing.T) {
	// Chain A -> B -> C -> D. Applying the signal at B must reach A and
	// C (its direct neighbors) but never D, which is two hops away.
	neighbors := map[string][]string{
		"A": {"B"},
		"B": {"A", "C"},
		"C": {"B", "D"},
		"D": {"C"},
	}

	got := Propagate(neighbors, "B", "relevant")

	want := map[string]string{"A": "relevant", "C": "relevant"}
	if len(got) != len(want) {
		t.Fatalf("Propagate returned %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Propagate[%q] = %q, want %q", k, got[k], v)
		}
	}
	if _, ok := got["D"]; ok {
		t.Error("Propagate reached \"D\", two hops from the origin - propagation must stop at one level")
	}
	if _, ok := got["B"]; ok {
		t.Error("Propagate included the origin node itself in its result")
	}
}

func TestPropagate_OriginWithNoNeighborsIsEmpty(t *testing.T) {
	got := Propagate(map[string][]string{"A": nil}, "A", 42)
	if len(got) != 0 {
		t.Errorf("Propagate = %v, want empty", got)
	}
}

func TestPropagate_UnknownOriginIsEmpty(t *testing.T) {
	got := Propagate(map[string][]string{"A": {"B"}}, "nonexistent", true)
	if len(got) != 0 {
		t.Errorf("Propagate = %v, want empty", got)
	}
}
