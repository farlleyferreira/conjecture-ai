package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

// fakeImplPath is a fresh build of internal/conformancecycle's own
// testdata/fakeimpl fixture, shared by every test in this file - the
// same real "docker + implementation" stand-in that package's own
// tests use, reused here instead of maintaining an independent copy
// of it.
//
// fakeImplementPath is this package's own testdata/fakeimplement
// fixture (see its own doc comment), used only by implement_test.go -
// built here too since a package's test binary may only have one
// TestMain.
//
// fakeZ3Path is a fresh build of internal/expressioncore's own
// testdata/fakez3 fixture, used only by consistency_test.go - the
// same real "z3" stand-in that package's own tests use, reused here
// rather than maintaining an independent copy of it.
var fakeImplPath, fakeImplementPath, fakeZ3Path string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "checktest-fakeimpl")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	exeSuffix := ""
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	fakeImplPath = filepath.Join(dir, "fakeimpl"+exeSuffix)
	fakeImplementPath = filepath.Join(dir, "fakeimplement"+exeSuffix)
	fakeZ3Path = filepath.Join(dir, "fakez3"+exeSuffix)

	// Located via PATH, not runtime.GOROOT() (deprecated since Go
	// 1.24 - its own value reflects where this test binary was
	// compiled, not necessarily a real toolchain install): go test
	// itself only ever runs via the go tool, so "go" being
	// unresolvable here would mean a broken environment, not
	// something worth a fallback for.
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, build := range []struct{ out, pkg string }{
		{fakeImplPath, "../../internal/conformancecycle/testdata/fakeimpl"},
		{fakeImplementPath, "./testdata/fakeimplement"},
		{fakeZ3Path, "../../internal/expressioncore/testdata/fakez3"},
	} {
		buildCmd := exec.Command(goBin, "build", "-o", build.out, build.pkg)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "building %s: %v\n%s", build.pkg, err, out)
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}

// checkFixture is a minimal, self-contained .cjtr document for this
// file's own tests: one hypothesis with a declared "id", one input
// concept with a closed domain (so DirectedSamplingInputs has
// something to vary), and one worked example to use as its baseline.
const checkFixture = `
conjecture: 1.0.0

phenomenon: demo

concepts:
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal

rules:
  shipping fee rule:
    id: shipping-rule
    hypotheses:
      fee matches:
        id: fee-matches
        always: the shipping fee is 7.50
        examples:
          - {order type: regular, shipping fee: 7.50}
`

func writeCheckFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(checkFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func openCheckTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := evidenceledger.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if err := workqueue.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestRunCheck_TooFewArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCheck([]string{"file.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runCheck with too few arguments = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), checkUsage) {
		t.Errorf("stderr = %q, want it to contain the usage string", stderr.String())
	}
}

func fixtureHypothesis(t *testing.T) (cjtr.Hypothesis, map[string]cjtr.Concept) {
	t.Helper()
	path := writeCheckFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	if len(f.rules) != 1 {
		t.Fatalf("len(f.rules) = %d, want 1", len(f.rules))
	}
	hyps, err := cjtr.Hypotheses(f.rules[0].Body, f.concepts)
	if err != nil {
		t.Fatalf("cjtr.Hypotheses returned unexpected error: %v", err)
	}
	if len(hyps) != 1 {
		t.Fatalf("len(hyps) = %d, want 1", len(hyps))
	}
	return hyps[0], f.concepts
}

func TestCheckHypothesisConformance_AcceptedAndConformantRecordsEvidence(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	failed := checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{"output.shipping_fee":"7.50"}`},
		h, "demo-area::demo::shipping-rule", "test-spec-version", false, concepts, nil, &bytes.Buffer{},
	)
	if failed {
		t.Error("checkHypothesisConformance = true (failed), want false (conformant)")
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].Survived == "" {
		t.Error("recorded evidence has empty Survived, want a description of the passing run")
	}
}

func TestCheckHypothesisConformance_NonConformantFailsAndRecordsCounterexample(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	// Every generated case will see this mismatching output, so the
	// hypothesis is non-conformant for all of them.
	failed := checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{"output.shipping_fee":"99.00"}`},
		h, "demo-area::demo::shipping-rule", "test-spec-version", false, concepts, nil, &bytes.Buffer{},
	)
	if !failed {
		t.Error("checkHypothesisConformance = false, want true (non-conformant)")
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if !strings.Contains(history[0].Attacked, "counterexample") {
		t.Errorf("recorded evidence.Attacked = %q, want it to mention the counterexample", history[0].Attacked)
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("workqueue.List returned unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "demo-area::demo::shipping-rule::fee-matches" || items[0].Tier != workqueue.Blocker {
		t.Errorf("queued items = %+v, want one blocker-tier item for the failing hypothesis", items)
	}
}

func TestCheckHypothesisConformance_IsLawConformantSetsLawStatusConformant(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{"output.shipping_fee":"7.50"}`},
		h, "demo-area::demo::shipping-rule", "test-spec-version", true, concepts, nil, &bytes.Buffer{},
	)

	status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Conformant {
		t.Errorf("law status = %v, want Conformant", status)
	}
}

func TestCheckHypothesisConformance_IsLawNonConformantSetsLawStatusViolated(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{"output.shipping_fee":"99.00"}`},
		h, "demo-area::demo::shipping-rule", "test-spec-version", true, concepts, nil, &bytes.Buffer{},
	)

	status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Violated {
		t.Errorf("law status = %v, want Violated", status)
	}
}

func TestRecordEvidence_IsLawPromotesToThesisAtCriticalThreshold(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "steady law", ID: "steady-law"}
	hypothesisID := "demo-area::demo::some-rule::steady-law"
	criticalRequiredCases := evidenceledger.Critical.RequiredCases()

	// Below threshold: still Conformant status, still at Hypothesis
	// stage - no promotion yet.
	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "spec-version", criticalRequiredCases-1, true, &bytes.Buffer{})

	latest, found, err := evidenceledger.Latest(db, hypothesisID)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if !found || latest.Stage != evidenceledger.Hypothesis {
		t.Errorf("after %d failure-free cases, latest = %+v, want Stage Hypothesis (not yet promoted)", criticalRequiredCases-1, latest)
	}

	// One more failure-free case crosses the threshold and should
	// promote the law to Thesis.
	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "spec-version", 1, true, &bytes.Buffer{})

	latest, found, err = evidenceledger.Latest(db, hypothesisID)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if !found || latest.Stage != evidenceledger.Thesis {
		t.Errorf("after crossing the threshold, latest = %+v, want Stage Thesis", latest)
	}

	count, err := userdefinedlaws.GetFailureFreeCases(db, hypothesisID)
	if err != nil {
		t.Fatalf("GetFailureFreeCases returned unexpected error: %v", err)
	}
	if count != criticalRequiredCases {
		t.Errorf("GetFailureFreeCases = %d, want %d", count, criticalRequiredCases)
	}
}

func TestRecordEvidence_IsLawNonConformantResetsFailureFreeCases(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "reset law", ID: "reset-law"}
	hypothesisID := "demo-area::demo::some-rule::reset-law"

	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "spec-version", 500, true, &bytes.Buffer{})
	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.NonConformant,
		&conformancecycle.Counterexample{Value: "x"}, "image", "spec-version", 0, true, &bytes.Buffer{})

	count, err := userdefinedlaws.GetFailureFreeCases(db, hypothesisID)
	if err != nil {
		t.Fatalf("GetFailureFreeCases returned unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("GetFailureFreeCases after a NonConformant verdict = %d, want 0 (reset)", count)
	}
}

func TestRecordEvidence_NotIsLawNeverPromotes(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "plain hypothesis", ID: "plain-hypothesis"}
	hypothesisID := "demo-area::demo::some-rule::plain-hypothesis"

	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "spec-version", evidenceledger.Critical.RequiredCases(), false, &bytes.Buffer{})

	latest, found, err := evidenceledger.Latest(db, hypothesisID)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if !found || latest.Stage != evidenceledger.Hypothesis {
		t.Errorf("latest = %+v, want Stage Hypothesis (a plain hypothesis never promotes through this path)", latest)
	}
}

func TestRecordEvidence_IsLawRepeatedViolationsReachContested(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "repeat offender", ID: "repeat-offender"}

	// lawContestedThreshold is 3: the first two NonConformant calls
	// should still leave the law merely Violated.
	for i := 0; i < lawContestedThreshold-1; i++ {
		recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.NonConformant,
			&conformancecycle.Counterexample{Value: "x"}, "image", "spec-version", 0, true, &bytes.Buffer{})

		status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::some-rule::repeat-offender")
		if err != nil {
			t.Fatalf("call %d: GetStatus returned unexpected error: %v", i, err)
		}
		if status != userdefinedlaws.Violated {
			t.Errorf("call %d: law status = %v, want Violated (below threshold)", i, status)
		}
	}

	// The lawContestedThreshold-th NonConformant call crosses the
	// threshold and should escalate to Contested.
	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.NonConformant,
		&conformancecycle.Counterexample{Value: "x"}, "image", "spec-version", 0, true, &bytes.Buffer{})

	status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::some-rule::repeat-offender")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Contested {
		t.Errorf("law status after %d violations = %v, want Contested", lawContestedThreshold, status)
	}

	count, err := userdefinedlaws.GetCounterevidenceCount(db, "demo-area::demo::some-rule::repeat-offender")
	if err != nil {
		t.Fatalf("GetCounterevidenceCount returned unexpected error: %v", err)
	}
	if count != lawContestedThreshold {
		t.Errorf("GetCounterevidenceCount = %d, want %d", count, lawContestedThreshold)
	}
}

func TestCheckHypothesisConformance_NotIsLawNeverSetsLawStatus(t *testing.T) {
	h, concepts := fixtureHypothesis(t)
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{"output.shipping_fee":"99.00"}`},
		h, "demo-area::demo::shipping-rule", "test-spec-version", false, concepts, nil, &bytes.Buffer{},
	)

	status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Unverified {
		t.Errorf("law status for a plain (non-law) hypothesis = %v, want Unverified (never set)", status)
	}
}

func TestCheckHypothesisConformance_UnfalsifiableIsRejectedWithoutRunningConformance(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"order type": {Name: "order type", Kind: cjtr.VariableConcept, Variable: "input.order_type", OneOf: []string{"regular"}},
	}
	then, err := cjtr.ParseComparison("the order type is regular", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "unfalsifiable demo", ID: "unfalsifiable-demo", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"order type": "regular"}},
	}
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	var stdout bytes.Buffer
	failed := checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{}`},
		h, "demo-area::demo::some-rule", "test-spec-version", false, concepts, nil, &stdout,
	)
	if !failed {
		t.Error("checkHypothesisConformance for an unfalsifiable hypothesis = false, want true (rejected)")
	}
	if !strings.Contains(stdout.String(), "unfalsifiable") {
		t.Errorf("stdout = %q, want it to report the hypothesis as unfalsifiable", stdout.String())
	}

	history, err := evidenceledger.History(db, "demo-area::demo::some-rule::unfalsifiable-demo")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("len(history) = %d, want 0 (a rejected hypothesis is never conformance-checked or recorded)", len(history))
	}
}

func TestCheckHypothesisConformance_NoExamplesIsTypeAOmission(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the shipping fee is at least 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{Name: "no examples", ID: "no-examples", IsUniversal: true, Then: then}
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	var stdout bytes.Buffer
	failed := checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{}`},
		h, "demo-area::demo::some-rule", "test-spec-version", false, concepts, nil, &stdout,
	)
	if failed {
		t.Error("checkHypothesisConformance with no examples = true, want false (nothing to fail on)")
	}
	if !strings.Contains(stdout.String(), "TYPE A OMISSION") {
		t.Errorf("stdout = %q, want it to report a Type A omission", stdout.String())
	}

	history, err := evidenceledger.History(db, "demo-area::demo::some-rule::no-examples")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("len(history) = %d, want 0 (nothing was actually tested, so nothing is recorded as conformant)", len(history))
	}
}

func TestCheckHypothesisConformance_ExamplesButNoBoundedInputIsTypeAOmission(t *testing.T) {
	// Has a worked example, unlike the no-examples case above, but no
	// input-variable concept declares a range or one_of - so
	// DirectedSamplingInputs still generates zero cases.
	concepts := map[string]cjtr.Concept{
		"shipping fee": {Name: "shipping fee", Kind: cjtr.VariableConcept, Variable: "output.shipping_fee", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the shipping fee is at least 0.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "no bounded input", ID: "no-bounded-input", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"shipping fee": "7.50"}},
	}
	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}

	var stdout bytes.Buffer
	failed := checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"WRITE", `{}`},
		h, "demo-area::demo::some-rule", "test-spec-version", false, concepts, nil, &stdout,
	)
	if failed {
		t.Error("checkHypothesisConformance with no bounded input = true, want false")
	}
	if !strings.Contains(stdout.String(), "TYPE A OMISSION") {
		t.Errorf("stdout = %q, want it to report a Type A omission", stdout.String())
	}
}

func TestRecordEvidence_NoDeclaredIDIsNotRecorded(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "anonymous"} // ID left empty

	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "spec-version", 0, false, &bytes.Buffer{})

	history, err := evidenceledger.History(db, "demo-area::demo::some-rule::anonymous")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("len(history) = %d, want 0 (no id, nothing to record)", len(history))
	}
}

func TestRecordEvidence_InconclusiveIsRecordedAsOpen(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "timed out", ID: "timed-out"}

	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Inconclusive, nil, "image", "spec-version", 0, false, &bytes.Buffer{})

	history, err := evidenceledger.History(db, "demo-area::demo::some-rule::timed-out")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].Open == "" {
		t.Error("recorded evidence has empty Open, want a note that the result was inconclusive")
	}
}

func TestRecordEvidence_NonConformantDemotesFromPriorStage(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "demoted", ID: "demoted"}
	hypothesisID := "demo-area::demo::some-rule::demoted"

	// Seed a prior record at Thesis, as if an earlier run had already
	// promoted it there.
	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Thesis, ValidityHash: []byte("x"),
	}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.NonConformant,
		&conformancecycle.Counterexample{Value: "x = -1"}, "image", "spec-version", 0, false, &bytes.Buffer{})

	history, err := evidenceledger.History(db, hypothesisID)
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if history[1].Stage != evidenceledger.Hypothesis {
		t.Errorf("demoted record's Stage = %v, want %v (one level below Thesis)", history[1].Stage, evidenceledger.Hypothesis)
	}
}

func TestRecordEvidence_WarnsWhenPriorEvidenceIsStale(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "stale check", ID: "stale-check"}
	hypothesisID := "demo-area::demo::some-rule::stale-check"

	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Thesis,
		ValidityHash: evidenceledger.ValidityHash("old-spec-version", "image", "old-code-version", "", ""),
	}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "new-spec-version", 0, false, &stdout)

	if !strings.Contains(stdout.String(), "stale") {
		t.Errorf("stdout = %q, want it to warn that prior evidence is stale", stdout.String())
	}
}

func TestRecordEvidence_NoWarningWhenValidityHashUnchanged(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "fresh check", ID: "fresh-check"}
	hypothesisID := "demo-area::demo::some-rule::fresh-check"

	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Thesis,
		ValidityHash: evidenceledger.ValidityHash("same-spec-version", "image", version.String(), "", ""),
	}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "same-spec-version", 0, false, &stdout)

	if strings.Contains(stdout.String(), "stale") {
		t.Errorf("stdout = %q, should not warn about staleness when the validity hash is unchanged", stdout.String())
	}
}

func TestRecordEvidence_ConformantCarriesPriorStageForward(t *testing.T) {
	db := openCheckTestStore(t)
	h := cjtr.Hypothesis{Name: "steady", ID: "steady"}
	hypothesisID := "demo-area::demo::some-rule::steady"

	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Theory, ValidityHash: []byte("x"),
	}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	recordEvidence(db, "demo-area::demo::some-rule", h, conformancecycle.DirectedSampling, conformancecycle.Conformant, nil, "image", "spec-version", 0, false, &bytes.Buffer{})

	history, err := evidenceledger.History(db, hypothesisID)
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if history[1].Stage != evidenceledger.Theory {
		t.Errorf("Stage = %v, want %v carried forward unchanged (promotion not implemented)", history[1].Stage, evidenceledger.Theory)
	}
}
