package optimizationsuite

import (
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/teamconfig"
)

func TestApplyProposal_UnapprovedLeavesCurrentUnchanged(t *testing.T) {
	got := ApplyProposal(5.0, CalibrationProposal{Field: "max_rounds", Value: 50.0}, false, 1.0, teamconfig.Higher)
	if got != 5.0 {
		t.Errorf("ApplyProposal (unapproved) = %v, want 5.0 (current unchanged)", got)
	}
}

func TestApplyProposal_ApprovedWithinFloorApplies(t *testing.T) {
	// Higher direction: a bigger value is stricter, so a proposal at or
	// above the floor is fine as-is.
	got := ApplyProposal(5.0, CalibrationProposal{Field: "min_cases", Value: 500.0}, true, 100.0, teamconfig.Higher)
	if got != 500.0 {
		t.Errorf("ApplyProposal (approved, within floor) = %v, want 500.0", got)
	}
}

func TestApplyProposal_ApprovedBelowHigherFloorIsClamped(t *testing.T) {
	got := ApplyProposal(5.0, CalibrationProposal{Field: "min_cases", Value: 50.0}, true, 100.0, teamconfig.Higher)
	if got != 100.0 {
		t.Errorf("ApplyProposal (approved, below the Higher floor) = %v, want 100.0 (clamped at the floor, not applied as-is)", got)
	}
}

func TestApplyProposal_ApprovedAboveLowerFloorIsClamped(t *testing.T) {
	// Lower direction: a smaller value is stricter (e.g. a maximum
	// tolerated violation rate), so a proposal above the floor must be
	// clamped down to it.
	got := ApplyProposal(0.01, CalibrationProposal{Field: "max_violation_percent", Value: 5.0}, true, 1.0, teamconfig.Lower)
	if got != 1.0 {
		t.Errorf("ApplyProposal (approved, above the Lower floor) = %v, want 1.0 (clamped)", got)
	}
}

func TestApplyProposal_ApprovedExactlyAtFloorApplies(t *testing.T) {
	got := ApplyProposal(5.0, CalibrationProposal{Field: "min_cases", Value: 100.0}, true, 100.0, teamconfig.Higher)
	if got != 100.0 {
		t.Errorf("ApplyProposal (approved, exactly at the floor) = %v, want 100.0", got)
	}
}
