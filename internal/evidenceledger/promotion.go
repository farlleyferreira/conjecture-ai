package evidenceledger

import "database/sql"

// PromoteToTheory reports whether a phenomenon's theses qualify for
// promotion to Theory - "Thesis promoted to Theory": "WHEN every
// thesis in a phenomenon is confirmed, the mechanism is confirmed, a
// second round with a different-family model reproduces the result,
// and swarm/attack robustness checks find no violation THEN the
// system promotes the phenomenon's theses to Theory."
//
// allThesesConfirmed and swarmAttackRobust are meant to be derived by
// this package's own callers from real recorded evidence (see
// [AllAtStage] and [SwarmAttackRobust], both in this same file).
// mechanismConfirmed and differentFamilyReproduced are not derived
// anywhere: no specification text in this codebase defines what "the
// mechanism" is as a checkable fact, and nothing tracks which model
// family generated a given implementation (the same gap adversarial-
// slm's own SelectSLM already names for "the code-generating model's
// family," and leaves to its own caller). Both are accepted here as
// the caller's own attested facts, the same "caller supplies the
// fact" posture this codebase already takes for context-engine's own
// isUserDefinedLaw and omission-detection's own addressedCategories -
// this function does not guess at either.
func PromoteToTheory(allThesesConfirmed, mechanismConfirmed, differentFamilyReproduced, swarmAttackRobust bool) bool {
	return allThesesConfirmed && mechanismConfirmed && differentFamilyReproduced && swarmAttackRobust
}

// PromoteToLaw reports whether a universally quantified hypothesis
// qualifies for promotion to Law - "Theory promoted to Law": "WHEN a
// universally quantified (always) hypothesis remains conformant
// against hidden samples and across new code versions THEN the
// system promotes it to Law."
//
// isUniversal is the only one of the three facts this codebase can
// derive on its own ([cjtr.Hypothesis.IsUniversal]). Neither
// conformantAgainstHiddenSamples nor conformantAcrossNewCodeVersions
// is: this codebase has no held-out sample set distinct from the ones
// conformance-cycle already generates from a hypothesis's own
// declared domain, and no per-evidence-record field for a code
// version distinct from [ValidityHash]'s own one-way hash of it -
// both are the caller's own attested facts, the same posture
// [PromoteToTheory] already takes for its own two undefined
// conditions.
func PromoteToLaw(isUniversal, conformantAgainstHiddenSamples, conformantAcrossNewCodeVersions bool) bool {
	return isUniversal && conformantAgainstHiddenSamples && conformantAcrossNewCodeVersions
}

// AllAtStage reports whether every one of hypothesisIDs' own latest
// evidence record currently stands at exactly stage - "every thesis
// in a phenomenon is confirmed," read as "currently standing at
// Thesis," the only concrete meaning this codebase's own stage-
// tracking already gives "confirmed." A hypothesis with no recorded
// evidence at all is not considered confirmed at any stage, so it
// fails this check rather than being silently skipped over.
func AllAtStage(db *sql.DB, hypothesisIDs []string, stage Stage) (bool, error) {
	for _, id := range hypothesisIDs {
		latest, found, err := Latest(db, id)
		if err != nil {
			return false, err
		}
		if !found || latest.Stage != stage {
			return false, nil
		}
	}
	return true, nil
}

// SwarmAttackRobust reports whether every adversarial-SLM attack ever
// recorded against hypothesisID found no violation - "swarm/attack
// robustness checks find no violation." producedBy identifies which
// of hypothesisID's own evidence records came from an adversarial-SLM
// attack, since [Evidence.ProducedBy] is free text shared across
// every evidence producer this codebase has; this function does not
// hardcode any one producer's own label, so it stays correct however
// a caller (e.g. cmd/conjecture's own "conjecture attack") happens to
// spell it.
//
// A hypothesis no attack was ever recorded against is not robust by
// omission - robustness means something was actually tried and
// survived, not that nothing was tried - so this reports false for
// that case too, the same "no evidence is not confirmed" reading
// [AllAtStage] already takes.
func SwarmAttackRobust(db *sql.DB, hypothesisID, producedBy string) (bool, error) {
	history, err := History(db, hypothesisID)
	if err != nil {
		return false, err
	}
	attacked := false
	for _, e := range history {
		if e.ProducedBy != producedBy {
			continue
		}
		attacked = true
		if e.Survived == "" {
			return false, nil
		}
	}
	return attacked, nil
}
