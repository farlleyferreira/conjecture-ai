package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/junit"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/specjudge"
	"github.com/farlleyferreira/conjecture-ai/internal/tap"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

const resultsUsage = "usage: conjecture results <junit|tap> <file.cjtr> <results-file>"

// runResults implements "conjecture results" - language-agnostic-
// contracts' "Standard result formats" requirement ("the system
// parses that standard format to determine pass/fail outcomes").
// Instead of running conformancecycle's own directed-sampling harness
// inside a sandbox the way "conjecture check" does, this command
// accepts a target system's own already-produced JUnit XML or TAP
// test results directly as evidence.
//
// Honesty note: this command's shape, and the correlation it draws
// between a declared hypothesis/law "id" and a result's own name, are
// this project's own inventions - no specification names a verb for
// this or a convention for matching a .cjtr id to an externally-
// produced test's own name. The convention used here is the most
// literal one available: a JUnit <testcase name="..."> or a TAP test
// line's own description must equal the hypothesis's declared "id"
// exactly (ClassName and test numbering are ignored). A hypothesis
// whose id has no exactly-matching result is reported as a Type A
// omission - the same "nothing exercised it" outcome "conjecture
// check" reports when directed sampling generates zero cases. A TAP
// "TODO" directive is not treated specially (TAP's own convention that
// a failing TODO test is not a failure is not applied here): the
// specification does not address it, so this does not invent that
// handling.
//
// Each hypothesis/law is still judged for falsifiability first, the
// same way "conjecture check" always does (judgeFalsifiability, shared
// with check.go) - an externally-reported result is still evidence
// about something, but it should not let an unfalsifiable hypothesis
// get recorded as "conformant" just because some test happened to
// pass.
func runResults(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		_, _ = fmt.Fprintln(stderr, resultsUsage)
		return 1
	}
	format, path, resultsPath := args[0], args[1], args[2]
	if format != "junit" && format != "tap" {
		_, _ = fmt.Fprintf(stderr, "conjecture: unrecognized format %q, want \"junit\" or \"tap\"\n", format)
		return 1
	}

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	resultsFile, err := os.Open(resultsPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	outcomes, err := parseResultOutcomes(format, resultsFile)
	_ = resultsFile.Close()
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

	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	exitCode := 0

	runGroup := func(hyps []cjtr.Hypothesis, compositeID string, isLaw bool) {
		for _, h := range hyps {
			if failed := recordResultOutcome(db, h, compositeID, specVersion, resultsPath, isLaw, f.concepts, outcomes, stdout); failed {
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

// resultOutcome is one externally-reported test's pass/fail state,
// looked up by its own name (runResults' own doc comment names the
// exact-match correlation convention).
type resultOutcome struct {
	verdict        conformancecycle.Verdict
	failureMessage string
}

func parseResultOutcomes(format string, r io.Reader) (map[string]resultOutcome, error) {
	outcomes := make(map[string]resultOutcome)
	switch format {
	case "junit":
		suites, err := junit.Parse(r)
		if err != nil {
			return nil, err
		}
		for _, suite := range suites.Suites {
			for _, tc := range suite.TestCases {
				outcomes[tc.Name] = junitOutcome(tc)
			}
		}
	case "tap":
		result, err := tap.Parse(r)
		if err != nil {
			return nil, err
		}
		for _, line := range result.Tests {
			outcomes[line.Description] = tapOutcome(line)
		}
	}
	return outcomes, nil
}

func junitOutcome(tc junit.TestCase) resultOutcome {
	switch {
	case tc.Passed():
		return resultOutcome{verdict: conformancecycle.Conformant}
	case tc.Skipped:
		return resultOutcome{verdict: conformancecycle.Inconclusive}
	case tc.Failure != nil:
		return resultOutcome{verdict: conformancecycle.NonConformant, failureMessage: tc.Failure.Message}
	default: // tc.Error != nil
		return resultOutcome{verdict: conformancecycle.NonConformant, failureMessage: tc.Error.Message}
	}
}

func tapOutcome(line tap.TestLine) resultOutcome {
	if line.Directive == "SKIP" {
		return resultOutcome{verdict: conformancecycle.Inconclusive}
	}
	if line.OK {
		return resultOutcome{verdict: conformancecycle.Conformant}
	}
	return resultOutcome{verdict: conformancecycle.NonConformant, failureMessage: line.Description}
}

// recordResultOutcome judges h, then (only if accepted) looks up its
// result by id in outcomes and records it via recordEvidence - the
// same shared function "conjecture check" and "conjecture implement"
// both use. resultsPath stands in for recordEvidence's own "code
// version" slot (the same generic role "conjecture check" fills with
// its Docker image, per [evidenceledger.ValidityHash]'s own five named
// dependencies): there is no Docker image here, but resultsPath still
// identifies which externally-produced results this evidence came
// from.
//
// Returns true when this hypothesis's result should fail runResults'
// own exit code (a NonConformant outcome - never Inconclusive or a
// Type A omission, neither a definitive failure).
func recordResultOutcome(db *sql.DB, h cjtr.Hypothesis, compositeID, specVersion, resultsPath string, isLaw bool, concepts map[string]cjtr.Concept, outcomes map[string]resultOutcome, stdout io.Writer) bool {
	verdict, ok := judgeFalsifiability(compositeID, h, concepts, stdout)
	if !ok {
		return true
	}
	if verdict != specjudge.Accepted {
		return true
	}

	if h.ID == "" {
		_, _ = fmt.Fprintf(stdout, "%s: %s: no declared \"id\", no result to match\n", compositeID, h.Name)
		return false
	}

	outcome, found := outcomes[h.ID]
	if !found {
		// omission-detection's own Type A: no case or observation
		// exercises this hypothesis - here, no result in the given file
		// is named like it.
		_, _ = fmt.Fprintf(stdout, "%s: %s: TYPE A OMISSION: no result in the given file matches id %q\n", compositeID, h.Name, h.ID)
		return false
	}

	var counterexample *conformancecycle.Counterexample
	if outcome.failureMessage != "" {
		counterexample = &conformancecycle.Counterexample{Value: outcome.failureMessage}
	}
	_, _ = fmt.Fprintf(stdout, "%s: %s: result: %s\n", compositeID, h.Name, outcome.verdict)

	// One externally-reported result is one observed case - the most
	// literal mapping available here, see recordEvidence's own doc
	// comment on caseCount. This path never actually runs any of
	// conformancecycle's three rigor levels - it maps an externally-
	// reported test result instead - so it passes DirectedSampling
	// unconditionally, exactly the label this call always used before
	// recordEvidence could name any other level (see recordEvidence's
	// own doc comment on the level parameter).
	recordEvidence(db, compositeID, h, conformancecycle.DirectedSampling, outcome.verdict, counterexample, resultsPath, specVersion, 1, isLaw, stdout)

	return outcome.verdict == conformancecycle.NonConformant
}
