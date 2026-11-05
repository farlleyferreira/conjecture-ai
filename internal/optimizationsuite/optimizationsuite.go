// Package optimizationsuite implements the pure-logic slice of the
// optimization-suite capability (openspec/specs/optimization-suite/
// spec.md) that does not depend on real execution: the approval gate
// and floor-clamp a multi-objective calibration proposal goes through
// before it can take effect.
//
// "Bandit-based swarm budget allocation": the specification requires
// reusing attacker-swarm's own bandit mechanism verbatim ("it uses the
// same bandit mechanism the attacker swarm capability defines"), so
// this package adds no allocator of its own - a caller uses
// attackerswarm.AllocateNext directly, exactly as attacker-swarm itself
// does.
//
// Not attempted here: the evolutionary input generator (a genetic
// algorithm with its own mutation/crossover design this package does
// not yet have grounds to invent a shape for) and "calibration runs
// offline" (a scheduling property, not a decision rule a pure function
// can check).
package optimizationsuite

import "github.com/farlleyferreira/conjecture-ai/internal/teamconfig"

// CalibrationProposal is one configuration field a multi-objective
// calibration run proposes changing.
type CalibrationProposal struct {
	Field string
	Value float64
}

// ApplyProposal decides what current becomes after proposal, given
// whether a human has approved it and a fixed floor that direction
// orders.
//
// "Multi-objective calibration proposes, never applies": when approved
// is false, ApplyProposal returns current unchanged - a pending
// proposal has no effect at all, not even a floor-clamped preview of
// one.
//
// "Fixed floors never crossed by calibration": when approved, the
// result is still never allowed to cross floor - it is clamped there
// instead of being set to proposal.Value outright. floor and direction
// reuse teamconfig's own floor/direction primitive rather than
// duplicating it: a calibration floor and a team-config floor are the
// same never-weaker-than-this rule, just applied to a different pair of
// values.
func ApplyProposal(current float64, proposal CalibrationProposal, approved bool, floor float64, direction teamconfig.Direction) float64 {
	if !approved {
		return current
	}
	if teamconfig.MeetsFloor(proposal.Value, floor, direction) {
		return proposal.Value
	}
	return floor
}
