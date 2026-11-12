// Package attackerswarm implements pieces of the attacker-swarm
// capability (openspec/specs/attacker-swarm/spec.md): the mode/family
// validation, the multi-armed bandit budget allocator, and
// deterministic finding aggregation. It does not run any attacking
// agent itself - that is cmd/conjecture's own concern ("conjecture
// swarm attack", swarm.go), dispatching through the same local-model
// runners [adversarialslm] already provides. "No communication
// between attacking agents" is an orchestration property of how that
// caller actually runs agents (never threading one round's own
// finding into another's input within the same run), not something
// this package's own pure functions enforce on their own.
package attackerswarm

import (
	"fmt"
	"math"
)

// MinStrategies is the specification's own floor: "The swarm SHALL
// support at least five distinct attack strategies."
const MinStrategies = 5

// Mode is one of the swarm's three operating modes. Its zero value is
// Off, matching "the attacker swarm SHALL be disabled by default."
type Mode int

const (
	Off Mode = iota
	AttackersOnly
	AttackersPlusAuditors
)

func (m Mode) String() string {
	switch m {
	case Off:
		return "off"
	case AttackersOnly:
		return "attackers-only"
	case AttackersPlusAuditors:
		return "attackers-plus-auditors"
	default:
		return "unknown mode"
	}
}

// ValidateMode checks mode's own family-independence requirement
// ("auditor mode requires a different model family"). Off and
// AttackersOnly have no auditor family to check, so they always pass;
// attackerFamily/auditorFamily are ignored for them.
func ValidateMode(mode Mode, attackerFamily, auditorFamily string) error {
	if mode != AttackersPlusAuditors {
		return nil
	}
	if attackerFamily == auditorFamily {
		return fmt.Errorf("attackerswarm: attackers-plus-auditors mode requires a different auditor family (got %q for both)", attackerFamily)
	}
	return nil
}

// Strategy is one attack strategy's running bandit statistics: how many
// times it has been allocated budget (Pulls) and how many of those
// allocations found a violation (Successes).
type Strategy struct {
	Name      string
	Pulls     int
	Successes int
}

// AllocateNext selects, via the UCB1 multi-armed bandit algorithm,
// which strategy (by index into strategies) should receive the next
// unit of the remaining attack budget ("The swarm SHALL... use a
// multi-armed bandit algorithm to allocate a configured attack budget
// across strategies"). It requires at least [MinStrategies] entries.
//
// A strategy never yet pulled is always chosen first (UCB1's own
// bootstrap step, needed before its mean success rate means anything),
// lowest index first among ties; this keeps AllocateNext a pure,
// deterministic function of strategies rather than reaching for
// randomness to break a tie.
func AllocateNext(strategies []Strategy) (int, error) {
	if len(strategies) < MinStrategies {
		return 0, fmt.Errorf("attackerswarm: %d strategies given, want at least %d", len(strategies), MinStrategies)
	}

	for i, s := range strategies {
		if s.Pulls == 0 {
			return i, nil
		}
	}

	total := 0
	for _, s := range strategies {
		total += s.Pulls
	}

	best, bestScore := -1, math.Inf(-1)
	for i, s := range strategies {
		mean := float64(s.Successes) / float64(s.Pulls)
		bonus := math.Sqrt(2 * math.Log(float64(total)) / float64(s.Pulls))
		if score := mean + bonus; score > bestScore {
			best, bestScore = i, score
		}
	}
	return best, nil
}

// Finding is one attacking or auditing agent's reported result for a
// single attack run.
type Finding struct {
	AgentFamily string
	Violated    bool
}

// AggregateResult is [Aggregate]'s deterministic summary of a set of
// findings. CrossFamilyAgreement is reported as a signal only - per
// "cross-family agreement SHALL be treated as a signal, never as
// evidence by itself" - callers must not treat it as proof on its own;
// the underlying evidence (e.g. via evidenceledger) still has to stand
// on its own.
type AggregateResult struct {
	ViolationReported    bool
	CrossFamilyAgreement bool
}

// Aggregate deterministically combines findings ("The system SHALL
// aggregate swarm results deterministically from the individual
// agents' findings") - the result depends only on findings' contents,
// never their order. ViolationReported is true if any finding reports a
// violation. CrossFamilyAgreement is true if two or more distinct
// agent families agree with each other, either that a violation
// occurred or that it did not.
func Aggregate(findings []Finding) AggregateResult {
	violatingFamilies := map[string]bool{}
	cleanFamilies := map[string]bool{}
	anyViolation := false

	for _, f := range findings {
		if f.Violated {
			violatingFamilies[f.AgentFamily] = true
			anyViolation = true
		} else {
			cleanFamilies[f.AgentFamily] = true
		}
	}

	return AggregateResult{
		ViolationReported:    anyViolation,
		CrossFamilyAgreement: len(violatingFamilies) >= 2 || len(cleanFamilies) >= 2,
	}
}
