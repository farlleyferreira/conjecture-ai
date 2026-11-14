package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

// Promotion past Thesis used to take the conditions the specification
// names as command-line flags: --mechanism-confirmed,
// --different-family-reproduced, --hidden-samples-conformant,
// --new-code-versions-conformant. Whoever ran the command asserted
// them and the command believed it. That made Theory and Law
// attestations wearing the vocabulary of evidence.
//
// This file computes each of them from what is on record instead:
//
//   - mechanism confirmed: a human signed the specification with
//     --confirmed (the one condition that is, by its nature, a
//     person's judgment - so the measurement is that the judgment
//     was recorded, by whom, on this exact text).
//   - different-family reproduction: the rule met its weight's
//     required count of failure-free hidden cases on implementations
//     declared as written by at least two different model families.
//   - conformant against hidden samples: the rule met that count on
//     at least one code version.
//   - conformant across new code versions: it met it on at least two.
//
// One thing stays declared rather than verified, and is named as such
// wherever it is shown: which model family wrote an implementation
// ("conjecture check --produced-by"). Nothing in a source tree proves
// its author.

// removedPromoteFlags maps each flag promotion no longer accepts to
// what now stands in its place.
var removedPromoteFlags = map[string]string{
	"--mechanism-confirmed":          "sign the specification with \"conjecture sign --confirmed\"",
	"--different-family-reproduced":  "run \"conjecture check --produced-by <family>\" on implementations from two model families until each meets the required hidden cases",
	"--hidden-samples-conformant":    "run \"conjecture check\" until the rule meets its required hidden cases",
	"--new-code-versions-conformant": "run \"conjecture check\" against a second version of the code until it meets the required hidden cases too",
}

// theoryMeasure is what the ledger says about a phenomenon's
// readiness for Theory.
type theoryMeasure struct {
	mechanismConfirmed        bool
	differentFamilyReproduced bool
	// detail explains, one line per shortfall, what is missing.
	detail []string
}

// measureTheory computes [theoryMeasure] for every hypothesis and law
// in f.
func measureTheory(db *sql.DB, f loadedFicha) (theoryMeasure, error) {
	var m theoryMeasure
	m.mechanismConfirmed = f.signed != nil && f.signed.Confirmed
	if !m.mechanismConfirmed {
		m.detail = append(m.detail, "the specification is not signed with --confirmed")
	}

	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	m.differentFamilyReproduced = true
	lawIDs := lawCompositeIDs(f)
	for _, id := range allHypothesisIDs(f) {
		weight, err := hypothesisWeight(db, id, lawIDs[id])
		if err != nil {
			return theoryMeasure{}, err
		}
		runs, err := evidenceledger.Runs(db, id)
		if err != nil {
			return theoryMeasure{}, err
		}
		families := evidenceledger.Measure(runs, specVersion).QualifiedFamilies(weight)
		if len(families) < 2 {
			m.differentFamilyReproduced = false
			have := "none"
			if len(families) > 0 {
				have = strings.Join(families, ", ")
			}
			m.detail = append(m.detail, fmt.Sprintf("%s: %d failure-free hidden cases needed on implementations from 2 declared model families; qualified so far: %s",
				id, weight.RequiredCases(), have))
		}
	}
	for _, id := range allHypothesisIDs(f) {
		robust, err := evidenceledger.SwarmAttackRobust(db, id, adversarialSLMProducedBy)
		if err != nil {
			return theoryMeasure{}, err
		}
		if !robust {
			m.detail = append(m.detail, fmt.Sprintf("%s: no adversarial attack on record that found no violation (run \"conjecture attack\" or \"conjecture swarm\")", id))
		}
	}
	return m, nil
}

// lawMeasure is what the ledger says about one hypothesis's readiness
// for Law.
type lawMeasure struct {
	atTheory                  bool
	hiddenSamplesConformant   bool
	newCodeVersionsConformant bool
	detail                    []string
}

// measureLaw computes [lawMeasure] for compositeID.
func measureLaw(db *sql.DB, f loadedFicha, compositeID string) (lawMeasure, error) {
	var m lawMeasure

	latest, found, err := evidenceledger.Latest(db, compositeID)
	if err != nil {
		return lawMeasure{}, err
	}
	m.atTheory = found && latest.Stage == evidenceledger.Theory
	if !m.atTheory {
		stage := "never checked"
		if found {
			stage = latest.Stage.String()
		}
		m.detail = append(m.detail, fmt.Sprintf("it is at %s, not Theory", stage))
	}

	weight, err := hypothesisWeight(db, compositeID, lawCompositeIDs(f)[compositeID])
	if err != nil {
		return lawMeasure{}, err
	}
	runs, err := evidenceledger.Runs(db, compositeID)
	if err != nil {
		return lawMeasure{}, err
	}
	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	qualified := evidenceledger.Measure(runs, specVersion).QualifiedCodeVersions(weight)

	m.hiddenSamplesConformant = len(qualified) >= 1
	m.newCodeVersionsConformant = len(qualified) >= 2
	if !m.newCodeVersionsConformant {
		m.detail = append(m.detail, fmt.Sprintf("%d failure-free hidden cases needed on 2 versions of the code; versions that met it so far: %d",
			weight.RequiredCases(), len(qualified)))
	}
	return m, nil
}

// lawCompositeIDs returns the composite id of every user-defined law
// in f - the ids whose weight is the gated law weight rather than the
// one recorded in the context graph.
func lawCompositeIDs(f loadedFicha) map[string]bool {
	out := map[string]bool{}
	for _, id := range allHypothesisIDs(f) {
		hypothesisID := id[strings.LastIndex(id, "::")+2:]
		if _, _, _, isLaw, found := findHypothesisByID(f, hypothesisID); found && isLaw {
			out[id] = true
		}
	}
	return out
}
