package cjtr

import (
	"math/big"
	"testing"
)

// fixedPick returns a pick function that always answers k (reduced
// into range), so a test can ask for "the first", "the last" or "some
// middle" value deterministically.
func fixedPick(k int64) func(*big.Int) *big.Int {
	return func(n *big.Int) *big.Int {
		v := big.NewInt(k)
		if v.Sign() < 0 || v.Cmp(n) >= 0 {
			return new(big.Int).Sub(n, big.NewInt(1))
		}
		return v
	}
}

func TestRandomValue_OneOf(t *testing.T) {
	c := Concept{Name: "fare class", OneOf: []string{"economy-basic", "economy-flex"}}
	for k, want := range map[int64]string{0: "economy-basic", 1: "economy-flex"} {
		got, ok, err := RandomValue(c, fixedPick(k))
		if err != nil || !ok || got != want {
			t.Errorf("RandomValue(one_of, pick %d) = %q, %v, %v, want %q, true, nil", k, got, ok, err, want)
		}
	}
}

func TestRandomValue_RangeStaysInsideAndKeepsPlaces(t *testing.T) {
	c := Concept{Name: "hours", HasRange: true, Min: "0.00", Max: "720.00"}
	cases := map[int64]string{0: "0.00", 2400: "24.00", -1: "720.00"}
	for k, want := range cases {
		got, ok, err := RandomValue(c, fixedPick(k))
		if err != nil || !ok || got != want {
			t.Errorf("RandomValue(range, pick %d) = %q, %v, %v, want %q, true, nil", k, got, ok, err, want)
		}
	}
}

func TestRandomValue_DeclaredPlacesWin(t *testing.T) {
	c := Concept{Name: "qty", HasRange: true, Min: "1", Max: "3", HasPlaces: true, Places: 1}
	got, ok, err := RandomValue(c, fixedPick(5))
	if err != nil || !ok || got != "1.5" {
		t.Errorf("RandomValue = %q, %v, %v, want \"1.5\", true, nil", got, ok, err)
	}
}

func TestRandomValue_SinglePointRange(t *testing.T) {
	c := Concept{Name: "fixed", HasRange: true, Min: "7", Max: "7"}
	got, ok, err := RandomValue(c, fixedPick(0))
	if err != nil || !ok || got != "7" {
		t.Errorf("RandomValue = %q, %v, %v, want \"7\", true, nil", got, ok, err)
	}
}

func TestRandomValue_NoDeclaredDomainIsNotSampleable(t *testing.T) {
	for _, c := range []Concept{{Name: "free text", Type: "text"}, {Name: "unbounded", Type: "decimal"}} {
		if _, ok, err := RandomValue(c, fixedPick(0)); ok || err != nil {
			t.Errorf("RandomValue(%q) = ok %v, err %v, want not sampleable and no error", c.Name, ok, err)
		}
	}
}

func TestRandomValue_InvertedRangeIsAnError(t *testing.T) {
	c := Concept{Name: "bad", HasRange: true, Min: "10", Max: "1"}
	if _, _, err := RandomValue(c, fixedPick(0)); err == nil {
		t.Error("RandomValue with min above max = nil error, want an error")
	}
}
