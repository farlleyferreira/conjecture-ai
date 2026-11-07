package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/junit"
	"github.com/farlleyferreira/conjecture-ai/internal/tap"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

func TestRunResults_TooFewArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runResults([]string{"junit", "file.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runResults with too few arguments = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), resultsUsage) {
		t.Errorf("stderr = %q, want it to contain the usage string", stderr.String())
	}
}

func TestRunResults_UnrecognizedFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runResults([]string{"xunit", "file.cjtr", "results.xml"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runResults with an unrecognized format = %d, want 1", code)
	}
}

func TestJUnitOutcome(t *testing.T) {
	cases := []struct {
		name string
		tc   junit.TestCase
		want conformancecycle.Verdict
	}{
		{"passed", junit.TestCase{Name: "x"}, conformancecycle.Conformant},
		{"skipped", junit.TestCase{Name: "x", Skipped: true}, conformancecycle.Inconclusive},
		{"failed", junit.TestCase{Name: "x", Failure: &junit.Failure{Message: "boom"}}, conformancecycle.NonConformant},
		{"errored", junit.TestCase{Name: "x", Error: &junit.Failure{Message: "boom"}}, conformancecycle.NonConformant},
	}
	for _, c := range cases {
		if got := junitOutcome(c.tc).verdict; got != c.want {
			t.Errorf("%s: junitOutcome verdict = %v, want %v", c.name, got, c.want)
		}
	}
	if got := junitOutcome(junit.TestCase{Failure: &junit.Failure{Message: "boom"}}).failureMessage; got != "boom" {
		t.Errorf("failureMessage = %q, want %q", got, "boom")
	}
}

func TestTAPOutcome(t *testing.T) {
	cases := []struct {
		name string
		line tap.TestLine
		want conformancecycle.Verdict
	}{
		{"ok", tap.TestLine{OK: true, Description: "x"}, conformancecycle.Conformant},
		{"not ok", tap.TestLine{OK: false, Description: "x"}, conformancecycle.NonConformant},
		{"skip", tap.TestLine{OK: false, Description: "x", Directive: "SKIP"}, conformancecycle.Inconclusive},
	}
	for _, c := range cases {
		if got := tapOutcome(c.line).verdict; got != c.want {
			t.Errorf("%s: tapOutcome verdict = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestParseResultOutcomes_JUnit(t *testing.T) {
	const doc = `<testsuite name="s" tests="1" failures="0" errors="0" skipped="0" time="0">
  <testcase classname="c" name="fee-matches" time="0"/>
</testsuite>`
	outcomes, err := parseResultOutcomes("junit", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("parseResultOutcomes returned unexpected error: %v", err)
	}
	outcome, found := outcomes["fee-matches"]
	if !found {
		t.Fatalf("outcomes = %+v, want an entry for \"fee-matches\"", outcomes)
	}
	if outcome.verdict != conformancecycle.Conformant {
		t.Errorf("verdict = %v, want Conformant", outcome.verdict)
	}
}

func TestParseResultOutcomes_TAP(t *testing.T) {
	const doc = "1..1\nok 1 - fee-matches\n"
	outcomes, err := parseResultOutcomes("tap", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("parseResultOutcomes returned unexpected error: %v", err)
	}
	outcome, found := outcomes["fee-matches"]
	if !found {
		t.Fatalf("outcomes = %+v, want an entry for \"fee-matches\"", outcomes)
	}
	if outcome.verdict != conformancecycle.Conformant {
		t.Errorf("verdict = %v, want Conformant", outcome.verdict)
	}
}

func TestRecordResultOutcome_ConformantRecordsEvidence(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	outcomes := map[string]resultOutcome{"fee-matches": {verdict: conformancecycle.Conformant}}

	failed := recordResultOutcome(db, h, "demo-area::demo::shipping-rule", "test-spec-version", "results.xml", false, concepts, outcomes, &bytes.Buffer{})
	if failed {
		t.Error("recordResultOutcome = true (failed), want false (conformant)")
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].Survived == "" {
		t.Error("recorded evidence has empty Survived, want a description of the passing result")
	}
}

func TestRecordResultOutcome_NonConformantFailsRecordsAndQueues(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	outcomes := map[string]resultOutcome{"fee-matches": {verdict: conformancecycle.NonConformant, failureMessage: "expected 7.50, got 9.99"}}

	failed := recordResultOutcome(db, h, "demo-area::demo::shipping-rule", "test-spec-version", "results.xml", false, concepts, outcomes, &bytes.Buffer{})
	if !failed {
		t.Error("recordResultOutcome = false, want true (non-conformant)")
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 || !strings.Contains(history[0].Attacked, "counterexample") {
		t.Errorf("history = %+v, want one record mentioning the counterexample", history)
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("workqueue.List returned unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "demo-area::demo::shipping-rule::fee-matches" {
		t.Errorf("queued items = %+v, want one blocker item for the failing hypothesis", items)
	}
}

func TestRecordResultOutcome_NoMatchingResultIsTypeAOmission(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	outcomes := map[string]resultOutcome{} // no result for "fee-matches"

	var stdout bytes.Buffer
	failed := recordResultOutcome(db, h, "demo-area::demo::shipping-rule", "test-spec-version", "results.xml", false, concepts, outcomes, &stdout)
	if failed {
		t.Error("recordResultOutcome with no matching result = true, want false (omission, not a failure)")
	}
	if !strings.Contains(stdout.String(), "TYPE A OMISSION") {
		t.Errorf("stdout = %q, want it to report a Type A omission", stdout.String())
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("len(history) = %d, want 0 (nothing was actually observed, so nothing is recorded)", len(history))
	}
}

func TestRecordResultOutcome_IsLawSetsLawStatus(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	outcomes := map[string]resultOutcome{"fee-matches": {verdict: conformancecycle.NonConformant, failureMessage: "boom"}}

	recordResultOutcome(db, h, "demo-area::demo::shipping-rule", "test-spec-version", "results.xml", true, concepts, outcomes, &bytes.Buffer{})

	status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Violated {
		t.Errorf("law status = %v, want Violated", status)
	}
}
