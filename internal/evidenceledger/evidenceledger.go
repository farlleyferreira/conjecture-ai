// Package evidenceledger implements the evidence-ledger capability's
// decision rules (openspec/specs/evidence-ledger/spec.md): the four-
// stage maturity scale a hypothesis climbs, the weight-dependent numeric
// promotion criteria, demotion on a counterexample, and the validity-hash
// mechanism that flags evidence stale when what it was produced against
// changes.
//
// This package only implements the decision rules themselves, as pure
// functions over facts the caller supplies. It does not run a
// conformance cycle, confirm a mechanism, reproduce a result with a
// different model, or run a swarm/attack check - those facts (and the
// facts conformance-cycle, llm-orchestration, and adversarial-slm/
// attacker-swarm would eventually supply) are inputs here, not
// something this package produces; none of those capabilities are built
// yet.
package evidenceledger

import (
	"bytes"
	"crypto/sha256"
)

// Stage is one of the four-stage maturity scale's points (Hypothesis,
// Thesis, Theory, Law). Theorem is not a fifth stage: the specification
// describes it as "available at any stage for a formally proved
// statement," so it is a separate flag a Stage can carry (see
// [Evidence.Theorem]), not a position on this scale.
type Stage int

const (
	Hypothesis Stage = iota
	Thesis
	Theory
	Law
)

func (s Stage) String() string {
	switch s {
	case Hypothesis:
		return "Hypothesis"
	case Thesis:
		return "Thesis"
	case Theory:
		return "Theory"
	case Law:
		return "Law"
	default:
		return "unknown stage"
	}
}

// Weight is a hypothesis's assigned risk weight, which determines how
// much failure-free evidence it needs before promotion.
type Weight int

const (
	Low Weight = iota
	Medium
	High
	Critical
)

// weightCriteria is the specification's own table: the number of
// failure-free cases required before promotion, and the maximum
// tolerated violation rate at 95% confidence each count corresponds to.
var weightCriteria = map[Weight]struct {
	RequiredCases       int
	MaxViolationPercent float64
}{
	Low:      {RequiredCases: 100, MaxViolationPercent: 3},
	Medium:   {RequiredCases: 300, MaxViolationPercent: 1},
	High:     {RequiredCases: 1000, MaxViolationPercent: 0.3},
	Critical: {RequiredCases: 3000, MaxViolationPercent: 0.1},
}

// RequiredCases returns the number of failure-free cases w requires
// before a hypothesis at that weight may be considered for promotion.
func (w Weight) RequiredCases() int {
	return weightCriteria[w].RequiredCases
}

// MaxViolationPercent returns the maximum tolerated violation rate (at
// 95% confidence) w's required case count corresponds to.
func (w Weight) MaxViolationPercent() float64 {
	return weightCriteria[w].MaxViolationPercent
}

// MeetsPromotionThreshold reports whether failureFreeCases failure-free
// cases is enough evidence, at weight w, to consider the Hypothesis-to-
// Thesis promotion the specification's weight-dependent numeric
// criterion gates ("the number of failure-free cases required before
// promotion SHALL depend on the hypothesis's assigned risk weight").
//
// This is only that one numeric gate. The specification's own scenario
// for this promotion also requires the hypothesis to be "accepted by
// the judge and found conformant by the conformance cycle with
// repeatable results" - facts this package takes as given (the caller
// already knows them), not something it can determine, since neither
// a judge nor a conformance cycle exists in this codebase yet.
func (w Weight) MeetsPromotionThreshold(failureFreeCases int) bool {
	return failureFreeCases >= w.RequiredCases()
}

// Demote returns the stage one level below s ("a single confirmed
// counterexample SHALL demote a hypothesis from its current stage").
// The specification names that a demotion happens, and that it is
// recorded with its cause, but not how far a demotion moves a
// hypothesis down the scale; this takes "demote" at its most literal -
// one level - rather than inventing a larger drop. Demote(Hypothesis)
// returns Hypothesis unchanged: the specification gives no stage below
// it, and what becomes of a hypothesis refuted at its own starting
// point is not this package's call to invent.
func Demote(s Stage) Stage {
	if s == Hypothesis {
		return Hypothesis
	}
	return s - 1
}

// ValidityHash computes the hash an evidence record is chained to,
// linking it to the exact specification, code, harness, SLM, and
// operator versions it was produced against - the specification's own
// five named dependencies ("chained by hash to the specification, code,
// harness, SLM, and operator versions it was produced against").
func ValidityHash(specVersion, codeVersion, harnessVersion, slmVersion, operatorVersion string) []byte {
	h := sha256.New()
	for _, v := range []string{specVersion, codeVersion, harnessVersion, slmVersion, operatorVersion} {
		h.Write([]byte(v))
		h.Write([]byte{0}) // separator, so ("a","bc") and ("ab","c") never collide
	}
	return h.Sum(nil)
}

// Stale reports whether an evidence record recorded with
// recordedValidityHash is stale against the dependencies' current
// validity hash currentValidityHash ("a change to any of them is
// visible as evidence debt rather than silently stale evidence").
func Stale(recordedValidityHash, currentValidityHash []byte) bool {
	return !bytes.Equal(recordedValidityHash, currentValidityHash)
}
