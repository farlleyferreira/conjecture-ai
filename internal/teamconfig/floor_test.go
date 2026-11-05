package teamconfig

import (
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

func TestMeetsFloor_HigherIsStricter(t *testing.T) {
	// A minimum required case count: a local override that requires
	// MORE cases than the team floor is stricter and must be accepted;
	// one that requires fewer is weaker and must be rejected.
	const floor = 300

	if !MeetsFloor(300, floor, Higher) {
		t.Error("MeetsFloor(300, floor=300, Higher) = false, want true (equal to the floor)")
	}
	if !MeetsFloor(1000, floor, Higher) {
		t.Error("MeetsFloor(1000, floor=300, Higher) = false, want true (stricter than the floor)")
	}
	if MeetsFloor(100, floor, Higher) {
		t.Error("MeetsFloor(100, floor=300, Higher) = true, want false (weaker than the floor)")
	}
}

func TestMeetsFloor_LowerIsStricter(t *testing.T) {
	// A maximum tolerated violation rate: a local override that
	// tolerates LESS is stricter; one that tolerates more is weaker.
	const floor = 1.0 // percent

	if !MeetsFloor(1.0, floor, Lower) {
		t.Error("MeetsFloor(1.0, floor=1.0, Lower) = false, want true (equal to the floor)")
	}
	if !MeetsFloor(0.3, floor, Lower) {
		t.Error("MeetsFloor(0.3, floor=1.0, Lower) = false, want true (stricter than the floor)")
	}
	if MeetsFloor(3.0, floor, Lower) {
		t.Error("MeetsFloor(3.0, floor=1.0, Lower) = true, want false (weaker than the floor)")
	}
}

func TestMeetsFloor_WithEvidenceLedgerWeight(t *testing.T) {
	// A concrete, already-real setting in this codebase: a team could
	// require at least Medium risk-weight rigor for some category of
	// rule. Weight's own declaration order (Low < Medium < High <
	// Critical) is exactly the ordering this generic comparison needs,
	// with no adapter.
	const floor = evidenceledger.Medium

	if !MeetsFloor(evidenceledger.High, floor, Higher) {
		t.Error("MeetsFloor(High, floor=Medium, Higher) = false, want true")
	}
	if MeetsFloor(evidenceledger.Low, floor, Higher) {
		t.Error("MeetsFloor(Low, floor=Medium, Higher) = true, want false")
	}
}

func TestCheckFloor_Accepted(t *testing.T) {
	if err := CheckFloor("required_cases", 1000, 300, Higher); err != nil {
		t.Errorf("CheckFloor returned unexpected error: %v", err)
	}
}

func TestCheckFloor_Rejected(t *testing.T) {
	err := CheckFloor("required_cases", 100, 300, Higher)
	if err == nil {
		t.Fatal("CheckFloor with a local value weaker than the floor = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "required_cases") {
		t.Errorf("error = %q, want it to name the setting", err.Error())
	}
}
