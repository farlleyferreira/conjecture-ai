package conformancecycle

import "testing"

func TestMinimalCounterexample_ReturnsSmallestBySize(t *testing.T) {
	found := []Counterexample{
		{Size: 10, Value: "big"},
		{Size: 2, Value: "small"},
		{Size: 5, Value: "medium"},
	}

	got, ok := MinimalCounterexample(found)
	if !ok {
		t.Fatal("MinimalCounterexample ok = false, want true")
	}
	if got.Value != "small" {
		t.Errorf("MinimalCounterexample = %+v, want the Size-2 one", got)
	}
}

func TestMinimalCounterexample_TieKeepsTheFirstFound(t *testing.T) {
	found := []Counterexample{
		{Size: 3, Value: "first"},
		{Size: 3, Value: "second"},
	}

	got, ok := MinimalCounterexample(found)
	if !ok {
		t.Fatal("MinimalCounterexample ok = false, want true")
	}
	if got.Value != "first" {
		t.Errorf("MinimalCounterexample = %+v, want the first-found one on a tie", got)
	}
}

func TestMinimalCounterexample_EmptyIsNotOK(t *testing.T) {
	_, ok := MinimalCounterexample(nil)
	if ok {
		t.Error("MinimalCounterexample(nil) ok = true, want false")
	}
}
