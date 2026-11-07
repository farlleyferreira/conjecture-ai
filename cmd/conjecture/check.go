package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
	"github.com/farlleyferreira/conjecture-ai/internal/specjudge"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
)

const checkUsage = "usage: conjecture check <file.cjtr> <docker-image> <command...>"

// runCheck implements "conjecture check" - the first command wiring
// spec-judge, conformance-cycle, and evidence-ledger together into one
// real loop: for every hypothesis and user-defined law in file, it
// judges falsifiability (spec-judge's own Finding/Judge), and - only
// for an Accepted verdict - runs a real directed-sampling conformance
// check (conformancecycle.CheckHypothesisDirectedSampling) against
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
// Honesty note: this still only runs one round of one rigor level
// (directed sampling) against one already-written implementation. It
// does not implement the outer judge-to-implementer round loop
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

	sb := &sandbox.Sandbox{}
	ctx := context.Background()
	exitCode := 0

	// specVersion identifies the exact frozen .cjtr content this run
	// checked against - [evidenceledger.ValidityHash]'s own
	// "specification... version it was produced against" dependency.
	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))

	runGroup := func(hyps []cjtr.Hypothesis, compositeID string) {
		for _, h := range hyps {
			if failed := checkHypothesisConformance(ctx, db, sb, image, command, h, compositeID, specVersion, f.concepts, f.tables, stdout); failed {
				exitCode = 1
			}
		}
	}
	runGroup(f.fileLaws, f.phenomenonID)

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
		runGroup(append(hyps, laws...), rule.CompositeID)
	}

	return exitCode
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
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	stdout io.Writer,
) bool {
	var findings []specjudge.Finding
	for _, stmt := range unfalsifiabilityCandidates(h) {
		ok, err := cjtr.Falsifiable(stmt, concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR checking falsifiability: %v\n", compositeID, h.Name, err)
			return true
		}
		if !ok {
			findings = append(findings, specjudge.Finding{
				Severity: specjudge.Rejecting,
				Message:  fmt.Sprintf("unfalsifiable: %q holds for every value %q's own declared domain allows", formatStatement(stmt), stmt.Term),
			})
		}
	}

	verdict := specjudge.Judge(findings)
	_, _ = fmt.Fprintf(stdout, "%s: %s: judge: %s\n", compositeID, h.Name, verdict)
	if verdict != specjudge.Accepted {
		for _, finding := range findings {
			_, _ = fmt.Fprintf(stdout, "%s: %s:   - %s\n", compositeID, h.Name, finding.Message)
		}
		return true
	}

	if len(h.Examples) == 0 {
		_, _ = fmt.Fprintf(stdout, "%s: %s: no worked example to use as a directed-sampling baseline, skipping conformance check\n", compositeID, h.Name)
		return false
	}

	dir, err := os.MkdirTemp("", "conjecture-check")
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR: %v\n", compositeID, h.Name, err)
		return true
	}
	defer func() { _ = os.RemoveAll(dir) }()
	opts := sandbox.Options{Image: image, Dir: dir, Command: command}

	cvVerdict, counterexample, err := conformancecycle.CheckHypothesisDirectedSampling(ctx, sb, opts, h, concepts, tables)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR running conformance check: %v\n", compositeID, h.Name, err)
		return true
	}
	_, _ = fmt.Fprintf(stdout, "%s: %s: conformance: %s\n", compositeID, h.Name, cvVerdict)
	if counterexample != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s:   counterexample: %v\n", compositeID, h.Name, counterexample.Value)
	}

	recordEvidence(db, compositeID, h, cvVerdict, counterexample, image, specVersion, stdout)

	return cvVerdict == conformancecycle.NonConformant
}

// recordEvidence appends one evidence-ledger record for h's
// conformance result, if h declares an "id" - [cjtr.Hypothesis.ID]'s
// own doc comment: "this package does not guess at a slugification
// rule for the ones that omit it," so neither does this: a hypothesis
// without one is still judged and checked, but its result is not
// persisted, rather than inventing an ID for it here.
//
// "Single-counterexample demotion" is applied for real: a
// NonConformant verdict demotes the hypothesis one stage below
// whatever its most recent evidence record left it at (via
// [evidenceledger.Demote]), with the counterexample that caused it
// recorded alongside. A Conformant or Inconclusive verdict carries the
// stage forward unchanged - promoting it upward needs the weight-
// dependent accumulated-case-count machinery
// ([evidenceledger.Weight.MeetsPromotionThreshold]), which this
// codebase cannot apply yet: nothing in the .cjtr grammar lets a
// hypothesis declare its own risk weight, so there is no weight to
// look the threshold up by. That is a separate, disclosed gap, not
// something guessed at here.
func recordEvidence(db *sql.DB, compositeID string, h cjtr.Hypothesis, verdict conformancecycle.Verdict, counterexample *conformancecycle.Counterexample, image, specVersion string, stdout io.Writer) {
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

	stage := evidenceledger.Hypothesis
	if previous, found, err := evidenceledger.Latest(db, hypothesisID); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR reading prior evidence: %v\n", compositeID, h.Name, err)
		return
	} else if found {
		stage = previous.Stage
	}
	if verdict == conformancecycle.NonConformant {
		stage = evidenceledger.Demote(stage)
	}

	evidence := evidenceledger.Evidence{
		HypothesisID: hypothesisID,
		Stage:        stage,
		ProducedBy:   "conjecture check " + version.String(),
		Unchecked:    "bounded proof and complete proof rigor levels not run",
		ValidityHash: evidenceledger.ValidityHash(specVersion, image, version.String(), "", ""),
	}
	switch verdict {
	case conformancecycle.Conformant:
		evidence.Attacked = "directed sampling"
		evidence.Survived = "every generated boundary case conformed"
	case conformancecycle.NonConformant:
		evidence.Attacked = "directed sampling"
		if counterexample != nil {
			evidence.Attacked += fmt.Sprintf("; found counterexample: %v", counterexample.Value)
		}
	case conformancecycle.Inconclusive:
		evidence.Open = "directed sampling: at least one round timed out with no definitive violation found"
	}

	if _, err := evidenceledger.Append(db, evidence); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording evidence: %v\n", compositeID, h.Name, err)
	}
}
