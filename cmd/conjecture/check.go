package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/contextengine"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/omissiondetection"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
	"github.com/farlleyferreira/conjecture-ai/internal/specjudge"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

const checkUsage = "usage: conjecture check <file.cjtr> <docker-image> <command...>"

// lawContestedThreshold is recordEvidence's own default for
// [userdefinedlaws.MeetsContestedThreshold]'s "configurable threshold"
// - the specification's own wording ("Empirical counterevidence SHALL
// accumulate toward a configurable threshold") names no number, so
// this is this project's own chosen default (not exposed as a flag
// yet, the same way queue.go's queueAgingRate/queueAgingUnit are
// fixed CLI-level defaults rather than per-run parameters), not a
// value the specification gives.
const lawContestedThreshold = 3

// runCheck implements "conjecture check" - the first command wiring
// spec-judge, conformance-cycle, and evidence-ledger together into one
// real loop: for every hypothesis and user-defined law in file, it
// judges falsifiability (spec-judge's own Finding/Judge), and - only
// for an Accepted verdict - runs a real conformance check at the
// strongest rigor level it actually qualifies for
// (conformancecycle.CheckHypothesisWithSelectedRigor) against
// implementation code running in image via command, recording one
// evidence-ledger record per judged hypothesis that declares an "id".
//
// This CLI shape ("conjecture check <file.cjtr> <docker-image>
// <command...>") is this project's own invention: no specification or
// product-proposal text names a verb or argument order for running the
// conformance cycle for real, unlike context-composition's or
// cli-mcp-integration's commands, which were grounded in the product
// proposal's own worked-example tables. It is kept deliberately close
// to [sandbox.Options]'s own shape (Image, Command) since that is what
// it is actually configuring.
//
// Honesty note: this still only runs one round of the selected rigor
// level against one already-written implementation. It does not
// implement the outer judge-to-implementer round loop
// ("Round and token limits") where a failing round would feed its
// counterexample back to an LLM that revises the implementation - that
// needs its own design (a target-language decision, a new LLM prompt/
// role), left for a following change.
func runCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) < 3 {
		_, _ = fmt.Fprintln(stderr, checkUsage)
		return 1
	}
	path, image, command := args[0], args[1], args[2:]

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := evidenceledger.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := workqueue.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := contextengine.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	sb := &sandbox.Sandbox{}
	ctx := context.Background()
	exitCode := 0

	// specVersion identifies the exact frozen .cjtr content this run
	// checked against - [evidenceledger.ValidityHash]'s own
	// "specification... version it was produced against" dependency.
	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))

	// runGroup checks every one of hyps. isLaw is passed through to
	// [checkHypothesisConformance] so that, and only that, [recordEvidence]
	// also updates user-defined-laws' own status lifecycle - a plain
	// hypothesis has no such status to update.
	runGroup := func(hyps []cjtr.Hypothesis, compositeID string, isLaw bool) {
		for _, h := range hyps {
			if failed := checkHypothesisConformance(ctx, db, sb, image, command, h, compositeID, specVersion, isLaw, f.concepts, f.tables, stdout); failed {
				exitCode = 1
			}
		}
	}
	runGroup(f.fileLaws, f.phenomenonID, true)

	for _, rule := range f.rules {
		hyps, err := cjtr.Hypotheses(rule.Body, f.concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %v\n", rule.CompositeID, err)
			exitCode = 1
			continue
		}
		laws, err := cjtr.UserDefinedLaws(rule.Body, f.concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %v\n", rule.CompositeID, err)
			exitCode = 1
			continue
		}
		runGroup(hyps, rule.CompositeID, false)
		runGroup(laws, rule.CompositeID, true)
	}

	return exitCode
}

// judgeFalsifiability applies specjudge's own falsifiability check to
// h, printing the same "judge: <verdict>" line (and, if rejected, each
// finding) that checkHypothesisConformance always has - shared with
// runResults (results.go), which judges a hypothesis the same way
// before trusting an externally-reported result for it, rather than
// recording evidence for one the judge would have rejected outright.
//
// ok is false only when checking falsifiability itself errored (and
// that error has already been printed to stdout) - the caller should
// treat that the same as a rejecting verdict.
func judgeFalsifiability(compositeID string, h cjtr.Hypothesis, concepts map[string]cjtr.Concept, stdout io.Writer) (verdict specjudge.Verdict, ok bool) {
	var findings []specjudge.Finding
	for _, stmt := range unfalsifiabilityCandidates(h) {
		falsifiable, err := cjtr.Falsifiable(stmt, concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR checking falsifiability: %v\n", compositeID, h.Name, err)
			return 0, false
		}
		if !falsifiable {
			findings = append(findings, specjudge.Finding{
				Severity: specjudge.Rejecting,
				Message:  fmt.Sprintf("unfalsifiable: %q holds for every value %q's own declared domain allows", formatStatement(stmt), stmt.Term),
			})
		}
	}

	verdict = specjudge.Judge(findings)
	_, _ = fmt.Fprintf(stdout, "%s: %s: judge: %s\n", compositeID, h.Name, verdict)
	if verdict != specjudge.Accepted {
		for _, finding := range findings {
			_, _ = fmt.Fprintf(stdout, "%s: %s:   - %s\n", compositeID, h.Name, finding.Message)
		}
	}
	return verdict, true
}

// checkHypothesisConformance judges and, if accepted, conformance-
// checks h, returning whether this hypothesis's result should fail
// runCheck's own exit code (a rejecting judge verdict or a
// NonConformant conformance verdict - never Inconclusive or Pending,
// neither of which is a definitive failure).
func checkHypothesisConformance(
	ctx context.Context,
	db *sql.DB,
	sb *sandbox.Sandbox,
	image string,
	command []string,
	h cjtr.Hypothesis,
	compositeID string,
	specVersion string,
	isLaw bool,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	stdout io.Writer,
) bool {
	verdict, ok := judgeFalsifiability(compositeID, h, concepts, stdout)
	if !ok {
		return true
	}
	if verdict != specjudge.Accepted {
		return true
	}

	// omission-detection's "Type A - asserted but not implemented":
	// "WHEN a statement asserts a behavior but no case or observation
	// exercises it THEN the system flags it as a Type A omission."
	// Checked before running anything, not after: with zero generated
	// cases, CheckHypothesisDirectedSampling would otherwise return
	// Conformant vacuously (no failure was found because nothing was
	// ever tried), silently reporting a hypothesis as conformant when
	// it was never actually exercised - exactly the dishonest outcome
	// Type A exists to catch instead.
	var cases []cjtr.Bindings
	if len(h.Examples) > 0 {
		var err error
		cases, err = conformancecycle.DirectedSamplingInputs(h, concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR: %v\n", compositeID, h.Name, err)
			return true
		}
	}
	if _, omitted := omissiondetection.CheckTypeA(compositeID+"::"+h.ID, len(cases)); omitted {
		_, _ = fmt.Fprintf(stdout, "%s: %s: TYPE A OMISSION: asserted but no case or observation exercises it - no worked example, or no input concept with a declared domain to sample\n", compositeID, h.Name)
		return false
	}

	dir, err := os.MkdirTemp("", "conjecture-check")
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR: %v\n", compositeID, h.Name, err)
		return true
	}
	defer func() { _ = os.RemoveAll(dir) }()
	opts := sandbox.Options{Image: image, Dir: dir, Command: command}

	level, cvVerdict, counterexample, caseCount, err := conformancecycle.CheckHypothesisWithSelectedRigor(ctx, sb, opts, h, concepts, tables, isLaw)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR running conformance check: %v\n", compositeID, h.Name, err)
		return true
	}
	_, _ = fmt.Fprintf(stdout, "%s: %s: conformance (%s): %s\n", compositeID, h.Name, level, cvVerdict)
	if counterexample != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s:   counterexample: %v\n", compositeID, h.Name, counterexample.Value)
	}

	recordEvidence(db, compositeID, h, level, cvVerdict, counterexample, image, specVersion, caseCount, isLaw, stdout)

	return cvVerdict == conformancecycle.NonConformant
}

// recordEvidence appends one evidence-ledger record for h's
// conformance result, if h declares an "id" - [cjtr.Hypothesis.ID]'s
// own doc comment: "this package does not guess at a slugification
// rule for the ones that omit it," so neither does this: a hypothesis
// without one is still judged and checked, but its result is not
// persisted, rather than inventing an ID for it here.
//
// When isLaw is true, this also updates user-defined-laws' own status
// lifecycle (internal/userdefinedlaws): a Conformant verdict sets the
// law's status to [userdefinedlaws.Conformant] - exactly the scenario
// the specification names ("Violation changes status to violated" for
// its NonConformant counterpart below). A NonConformant verdict first
// records one more piece of accumulated empirical counterevidence
// ([userdefinedlaws.RecordCounterevidence]) and sets status to
// [userdefinedlaws.Contested] once that running total crosses
// lawContestedThreshold (via [userdefinedlaws.MeetsContestedThreshold]
// - "Empirical counterevidence SHALL accumulate toward a configurable
// threshold before the law is marked contested"), or to
// [userdefinedlaws.Violated] otherwise. An Inconclusive verdict leaves
// the status unchanged: the specification names no transition for it,
// so none is invented here. The remaining two statuses
// ("conformant-under-generalization", "invalidated") stay unreachable
// from this path - they need generalization detection and formal-
// proof/[userdefinedlaws.ResolveConflict] wiring respectively, neither
// of which this change builds; each is a real, separate, disclosed
// gap, not an oversight.
//
// "Single-counterexample demotion" is applied for real: a
// NonConformant verdict demotes the hypothesis one stage below
// whatever its most recent evidence record left it at (via
// [evidenceledger.Demote]), with the counterexample that caused it
// recorded alongside. A Conformant or Inconclusive verdict carries the
// stage forward unchanged for a plain hypothesis - promoting it
// upward needs the weight-dependent accumulated-case-count machinery
// ([evidenceledger.Weight.MeetsPromotionThreshold]), which this
// codebase still cannot apply to a plain hypothesis: nothing in the
// .cjtr grammar lets one declare its own risk weight, so there is no
// weight to look the threshold up by. That remains a separate,
// disclosed gap, not something guessed at here.
//
// A law is different: [contextengine.GatedWeight]'s own
// "isUserDefinedLaw's rule wins outright" makes every law's weight
// [evidenceledger.Critical], regardless of any other signal - a real,
// already-grounded weight this package can use. So, when isLaw is
// true and the verdict is Conformant at Hypothesis stage, caseCount
// (the number of cases the selected rigor level actually exercised,
// supplied by the caller - see check.go's/implement.go's own
// CheckHypothesisWithSelectedRigor call, or 1 per externally-reported
// passing result in results.go) is added to a running failure-free-case total
// ([userdefinedlaws.AddFailureFreeCases]), and the law is promoted to
// Thesis ("Hypothesis promoted to Thesis... found conformant by the
// conformance cycle with repeatable results") the moment that total
// meets critical weight's own required-case count
// (MeetsPromotionThreshold). "Repeatable results" is read here as this
// running total itself: it only grows across repeated Conformant runs
// of the same law, and a NonConformant verdict resets it back to zero
// ([userdefinedlaws.ResetFailureFreeCases]) at the same time it
// demotes the stage - a single counterexample undoes the "repeatable"
// claim the same way it undoes everything else. This still never
// promotes past Thesis: Thesis-to-Theory and Theory-to-Law each need
// their own distinct, unbuilt criteria (phenomenon-level confirmation
// and swarm robustness; hidden-sample testing across code versions),
// so this only ever fires the one promotion edge gated purely by case
// count.
//
// This also wires "Evidence flagged stale after a dependency
// changes": when a prior record exists for the same hypothesis,
// [evidenceledger.Stale] compares its own validity hash against this
// run's, and a stdout warning is printed when they differ - the stage
// still carries forward as recorded (the specification says flagged,
// not discarded or reset), so this is a visibility fix, not a
// behavior change to anything else this function does.
//
// level names which rigor level actually produced verdict - recorded
// in evidence.Attacked/evidence.Open instead of the hardcoded
// "directed sampling" text this function used before bounded and
// complete proof existed (conformance-cycle-rigor-levels' design
// decision 6). results.go, whose own path never runs any of
// conformancecycle's three rigor levels (it maps an externally-
// reported test result instead), keeps passing
// [conformancecycle.DirectedSampling] here exactly as before - this
// is unrelated to its own, separately disclosed gap, not fixed by
// this parameter.
// unrunRigorLevelsNote is [recordEvidence]'s own disclosure of which
// of the three rigor levels did not produce level's own verdict -
// honest about why, not just that, matching
// [conformancecycle.CheckHypothesisWithSelectedRigor]'s own fallback
// order (complete proof, then bounded proof, then directed sampling).
func unrunRigorLevelsNote(level conformancecycle.RigorLevel, isLaw bool) string {
	switch level {
	case conformancecycle.CompleteProof:
		return ""
	case conformancecycle.BoundedProof:
		return "complete proof not run: not every input concept has a boundable domain within this package's disclosed cap"
	default: // conformancecycle.DirectedSampling
		if !isLaw {
			return "bounded proof and complete proof rigor levels not run: a plain hypothesis has no declared weight to select them by"
		}
		return "bounded proof and complete proof rigor levels not run: no input concept has a boundable domain within this package's disclosed cap"
	}
}

func recordEvidence(db *sql.DB, compositeID string, h cjtr.Hypothesis, level conformancecycle.RigorLevel, verdict conformancecycle.Verdict, counterexample *conformancecycle.Counterexample, image, specVersion string, caseCount int, isLaw bool, stdout io.Writer) {
	if h.ID == "" {
		_, _ = fmt.Fprintf(stdout, "%s: %s: no declared \"id\", evidence not recorded\n", compositeID, h.Name)
		return
	}
	if !cjtr.ValidSegment(h.ID) {
		_, _ = fmt.Fprintf(stdout, "%s: %s: \"id\" %q is not a valid identifier segment, evidence not recorded\n", compositeID, h.Name, h.ID)
		return
	}
	// "::" is the exact separator [cjtr.ComposeID]'s own doc comment
	// documents ("shipping::free-shipping::free-above"), not a value
	// invented here - ComposeID itself cannot be reused for this
	// because compositeID is already a joined string, not a single
	// segment, and would fail ComposeID's own per-segment validation.
	hypothesisID := compositeID + "::" + h.ID
	currentValidityHash := evidenceledger.ValidityHash(specVersion, image, version.String(), "", "")

	stage := evidenceledger.Hypothesis
	if previous, found, err := evidenceledger.Latest(db, hypothesisID); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR reading prior evidence: %v\n", compositeID, h.Name, err)
		return
	} else if found {
		stage = previous.Stage
		// "Evidence flagged stale after a dependency changes": the
		// stage still carries forward as recorded (the specification
		// says this is flagged, not that stale evidence is discarded
		// or that the stage resets), but a caller reading stdout is
		// told plainly that the harness/code/spec version this stage
		// was last earned against is no longer the one being used now.
		if evidenceledger.Stale(previous.ValidityHash, currentValidityHash) {
			_, _ = fmt.Fprintf(stdout, "%s: %s: WARNING: prior evidence is stale - the harness, code, or spec version it was recorded against has since changed\n", compositeID, h.Name)
		}
	}
	if verdict == conformancecycle.NonConformant {
		stage = evidenceledger.Demote(stage)
	}

	evidence := evidenceledger.Evidence{
		HypothesisID: hypothesisID,
		Stage:        stage,
		ProducedBy:   "conjecture check " + version.String(),
		Unchecked:    unrunRigorLevelsNote(level, isLaw),
		ValidityHash: currentValidityHash,
	}
	switch verdict {
	case conformancecycle.Conformant:
		evidence.Attacked = level.String()
		evidence.Survived = "every generated case conformed"
	case conformancecycle.NonConformant:
		evidence.Attacked = level.String()
		if counterexample != nil {
			evidence.Attacked += fmt.Sprintf("; found counterexample: %v", counterexample.Value)
		}
	case conformancecycle.Inconclusive:
		evidence.Open = level.String() + ": at least one round timed out with no definitive violation found"
	}

	if _, err := evidenceledger.Append(db, evidence); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording evidence: %v\n", compositeID, h.Name, err)
		return
	}

	if isLaw {
		switch verdict {
		case conformancecycle.Conformant:
			if err := userdefinedlaws.SetStatus(db, hypothesisID, userdefinedlaws.Conformant); err != nil {
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording law status: %v\n", compositeID, h.Name, err)
				break
			}
			lawWeight, err := contextengine.GatedWeight(evidenceledger.Low, true, nil)
			if err != nil {
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR computing law weight: %v\n", compositeID, h.Name, err)
				break
			}
			// Record the gated weight in the context graph too, not
			// only use it in-process for the promotion check below -
			// otherwise "conjecture graph inspect" could never report
			// a law's weight the specification's own "Axes reported
			// separately" scenario asks for.
			if err := contextengine.SetWeight(db, hypothesisID, lawWeight); err != nil {
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording context graph weight: %v\n", compositeID, h.Name, err)
				break
			}
			total, err := userdefinedlaws.AddFailureFreeCases(db, hypothesisID, caseCount)
			if err != nil {
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording failure-free cases: %v\n", compositeID, h.Name, err)
				break
			}
			if stage == evidenceledger.Hypothesis && lawWeight.MeetsPromotionThreshold(total) {
				promotion := evidenceledger.Evidence{
					HypothesisID: hypothesisID,
					Stage:        evidenceledger.Thesis,
					ProducedBy:   evidence.ProducedBy,
					Survived:     fmt.Sprintf("promoted to Thesis: %d failure-free cases at critical weight met the required %d", total, lawWeight.RequiredCases()),
					ValidityHash: evidence.ValidityHash,
				}
				if _, err := evidenceledger.Append(db, promotion); err != nil {
					_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording promotion: %v\n", compositeID, h.Name, err)
				} else {
					_, _ = fmt.Fprintf(stdout, "%s: %s: promoted to Thesis\n", compositeID, h.Name)
				}
			}
		case conformancecycle.NonConformant:
			if err := userdefinedlaws.ResetFailureFreeCases(db, hypothesisID); err != nil {
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR resetting failure-free cases: %v\n", compositeID, h.Name, err)
			}
			count, err := userdefinedlaws.RecordCounterevidence(db, hypothesisID)
			if err != nil {
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording counterevidence: %v\n", compositeID, h.Name, err)
				break
			}
			lawStatus := userdefinedlaws.Violated
			if userdefinedlaws.MeetsContestedThreshold(count, lawContestedThreshold) {
				lawStatus = userdefinedlaws.Contested
			}
			if err := userdefinedlaws.SetStatus(db, hypothesisID, lawStatus); err != nil {
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording law status: %v\n", compositeID, h.Name, err)
			}
		}
	}

	// work-queue's own "single queue... that drives what conjecture
	// works on next": a confirmed violation is exactly the kind of
	// blocker-tier work the specification's own scenario names ("a
	// blocker-tier item... preempts any background-tier item"). Weight
	// 1 and severity 1 are this integration's own chosen defaults -
	// nothing scores a hypothesis's own weight or a violation's own
	// severity anywhere in this codebase yet (the same gap
	// MeetsPromotionThreshold's own missing per-hypothesis weight
	// already names), so a fixed, disclosed default is used rather
	// than inventing a scoring rule. Re-queueing the same already-
	// queued hypothesis is not an error worth reporting: Add's
	// duplicate-ID rejection exists to catch a caller bug, not this
	// expected case of checking the same hypothesis again while its
	// prior failure is still unaddressed.
	if verdict == conformancecycle.NonConformant {
		item := workqueue.Item{ID: hypothesisID, Tier: workqueue.Blocker, Weight: 1, Severity: 1, CreatedAt: time.Now().UTC()}
		_ = workqueue.Add(db, item)
	}
}
