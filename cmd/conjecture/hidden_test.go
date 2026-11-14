package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// hiddenFixture is the rule testdata/fakerigor actually implements
// (free shipping at 100.00 and above, never for a subscription), with
// every input concept given a domain to draw from.
const hiddenFixture = `
conjecture: 1.0.0

phenomenon: hidden sampling demo

concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
    range: {min: 0.00, max: 500.00}
    places: 2
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
      regular orders at the threshold ship free:
        id: free-shipping
        given:
          - the order type is regular
          - the discounted subtotal is at least 100.00
        then: the shipping fee is 0.00
        otherwise: the shipping fee is 7.50
        examples:
          - {discounted subtotal: 150.00, order type: regular, shipping fee: 0.00}
`

const hiddenFixtureID = "demo-area::hidden-sampling-demo::shipping-rule::free-shipping"

func loadHiddenFixture(t *testing.T, content string) (loadedFicha, cjtr.Hypothesis) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "hidden-sampling-demo.cjtr")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	hyps, err := cjtr.Hypotheses(f.rules[0].Body, f.concepts)
	if err != nil || len(hyps) != 1 {
		t.Fatalf("Hypotheses = %d, %v, want exactly one", len(hyps), err)
	}
	if got := f.rules[0].CompositeID + "::" + hyps[0].ID; got != hiddenFixtureID {
		t.Fatalf("fixture composite id = %q, want %q", got, hiddenFixtureID)
	}
	return f, hyps[0]
}

func TestSampleHiddenCases_AccumulatesAndPromotesToThesisAtTheRequiredCount(t *testing.T) {
	f, h := loadHiddenFixture(t, hiddenFixture)
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatalf("EnsureRunSchema returned unexpected error: %v", err)
	}
	sb := &sandbox.Sandbox{DockerBinary: fakeRigorPath}
	run := hiddenRun{samples: 50, specVersion: "spec-1", codeVersion: "code-1", producerFamily: "anthropic"}

	var first bytes.Buffer
	if failed := sampleHiddenCases(context.Background(), db, sb, "unused", []string{"compute"}, h, f.rules[0].CompositeID, run, false, f.concepts, f.tables, &first); failed {
		t.Fatalf("sampleHiddenCases = failed, want conformant - stdout:\n%s", first.String())
	}
	if !strings.Contains(first.String(), "50/100 failure-free hidden cases") {
		t.Errorf("stdout should report the running count, got:\n%s", first.String())
	}
	if strings.Contains(first.String(), "promoted to Thesis") {
		t.Errorf("promoted to Thesis at 50 cases, want the required 100 first:\n%s", first.String())
	}

	var second bytes.Buffer
	if failed := sampleHiddenCases(context.Background(), db, sb, "unused", []string{"compute"}, h, f.rules[0].CompositeID, run, false, f.concepts, f.tables, &second); failed {
		t.Fatalf("second sampleHiddenCases = failed - stdout:\n%s", second.String())
	}
	if !strings.Contains(second.String(), "promoted to Thesis") {
		t.Errorf("stdout should report the promotion at 100 cases, got:\n%s", second.String())
	}

	latest, found, err := evidenceledger.Latest(db, hiddenFixtureID)
	if err != nil || !found || latest.Stage != evidenceledger.Thesis {
		t.Errorf("latest stage = %v (found %v, err %v), want Thesis", latest.Stage, found, err)
	}
	runs, err := evidenceledger.Runs(db, hiddenFixtureID)
	if err != nil || len(runs) != 2 {
		t.Fatalf("Runs = %d, %v, want 2 recorded runs", len(runs), err)
	}
	for _, r := range runs {
		if r.Kind != evidenceledger.RunHidden || r.Cases != 50 || !r.Conformant || r.ProducerFamily != "anthropic" || r.Seed == 0 {
			t.Errorf("recorded run = %+v, want a conformant 50-case hidden run by anthropic with its seed", r)
		}
	}
}

func TestSampleHiddenCases_NewCodeVersionStartsItsCountAgain(t *testing.T) {
	f, h := loadHiddenFixture(t, hiddenFixture)
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatalf("EnsureRunSchema returned unexpected error: %v", err)
	}
	sb := &sandbox.Sandbox{DockerBinary: fakeRigorPath}

	for _, code := range []string{"code-1", "code-2"} {
		var stdout bytes.Buffer
		run := hiddenRun{samples: 30, specVersion: "spec-1", codeVersion: code}
		sampleHiddenCases(context.Background(), db, sb, "unused", []string{"compute"}, h, f.rules[0].CompositeID, run, false, f.concepts, f.tables, &stdout)
		if !strings.Contains(stdout.String(), "30/100 failure-free hidden cases") {
			t.Errorf("%s: stdout should report 30/100 for this code alone, got:\n%s", code, stdout.String())
		}
	}
}

func TestSampleHiddenCases_ACounterexampleFailsDemotesAndResetsTheCount(t *testing.T) {
	// The same rule with the threshold moved: fakerigor still ships
	// free from 100.00, so every regular order between 100.00 and
	// 300.00 now breaks the rule - far too many to miss in 60 draws.
	f, h := loadHiddenFixture(t, strings.ReplaceAll(hiddenFixture, "is at least 100.00", "is at least 300.00"))
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatalf("EnsureRunSchema returned unexpected error: %v", err)
	}
	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{HypothesisID: hiddenFixtureID, Stage: evidenceledger.Thesis, ValidityHash: []byte("v")}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	sb := &sandbox.Sandbox{DockerBinary: fakeRigorPath}
	run := hiddenRun{samples: 60, specVersion: "spec-1", codeVersion: "code-1"}

	var stdout bytes.Buffer
	if failed := sampleHiddenCases(context.Background(), db, sb, "unused", []string{"compute"}, h, f.rules[0].CompositeID, run, false, f.concepts, f.tables, &stdout); !failed {
		t.Fatalf("sampleHiddenCases = conformant, want a counterexample - stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "counterexample: ") || !strings.Contains(stdout.String(), "-> shipping_fee=0.00") {
		t.Errorf("stdout should show the counterexample with the implementation's real output, got:\n%s", stdout.String())
	}

	latest, _, err := evidenceledger.Latest(db, hiddenFixtureID)
	if err != nil || latest.Stage != evidenceledger.Hypothesis {
		t.Errorf("latest stage = %v (err %v), want it demoted from Thesis to Hypothesis", latest.Stage, err)
	}
	runs, _ := evidenceledger.Runs(db, hiddenFixtureID)
	if got := evidenceledger.Measure(runs, "spec-1").FailureFree("code-1"); got != 0 {
		t.Errorf("failure-free count after a counterexample = %d, want 0", got)
	}
}

func TestSampleHiddenCases_NoDeclaredDomainIsSkippedAndSaysWhy(t *testing.T) {
	withoutRange := strings.Replace(hiddenFixture, "    range: {min: 0.00, max: 500.00}\n    places: 2\n", "", 1)
	f, h := loadHiddenFixture(t, withoutRange)
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatalf("EnsureRunSchema returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	run := hiddenRun{samples: 20, specVersion: "spec-1", codeVersion: "code-1"}
	failed := sampleHiddenCases(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeRigorPath}, "unused", []string{"compute"}, h, f.rules[0].CompositeID, run, false, f.concepts, f.tables, &stdout)
	if failed {
		t.Errorf("sampleHiddenCases = failed, want a skip (an unsampleable rule is not a violation)")
	}
	if !strings.Contains(stdout.String(), "hidden sampling skipped") || !strings.Contains(stdout.String(), "discounted subtotal") {
		t.Errorf("stdout should say sampling was skipped and name the concept, got:\n%s", stdout.String())
	}
	if runs, _ := evidenceledger.Runs(db, hiddenFixtureID); len(runs) != 0 {
		t.Errorf("recorded %d runs for a skipped sampling, want none", len(runs))
	}
}

func TestSampleHiddenCases_ZeroSamplesDoesNothing(t *testing.T) {
	f, h := loadHiddenFixture(t, hiddenFixture)
	db := openCheckTestStore(t)
	var stdout bytes.Buffer
	failed := sampleHiddenCases(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeRigorPath}, "unused", []string{"compute"}, h, f.rules[0].CompositeID, hiddenRun{}, false, f.concepts, f.tables, &stdout)
	if failed || stdout.Len() != 0 {
		t.Errorf("sampleHiddenCases with 0 samples = failed %v, stdout %q, want nothing at all", failed, stdout.String())
	}
}

func TestSourceTreeVersion_FollowsTheCodeAndIgnoresSpecsAndVCS(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	version := func() string {
		t.Helper()
		v, err := sourceTreeVersion(root)
		if err != nil {
			t.Fatalf("sourceTreeVersion returned unexpected error: %v", err)
		}
		return v
	}

	write("pricing/fees.py", "fee = 1\n")
	before := version()
	if !strings.HasPrefix(before, "src:") {
		t.Errorf("sourceTreeVersion = %q, want a src: hash", before)
	}

	write("conjecture/phenomena/x/rule.cjtr", "conjecture: 1.0.0\n")
	write(".git/HEAD", "ref: refs/heads/main\n")
	write("node_modules/dep/index.js", "module.exports = 1\n")
	if got := version(); got != before {
		t.Errorf("sourceTreeVersion changed after touching only specs, .git and node_modules")
	}

	write("pricing/fees.py", "fee = 2\n")
	if got := version(); got == before {
		t.Errorf("sourceTreeVersion did not change after the code changed")
	}
}

func TestSampleHiddenCases_ARuleWithNoOtherwiseCountsOnlyTheCasesItSpeaksTo(t *testing.T) {
	// One band of the rule, with no "otherwise": about four in five
	// random subtotals fall outside it, and a regular order is one in
	// two. Before this was handled, the first such case aborted the
	// whole check with an error.
	band := strings.Replace(hiddenFixture, "        otherwise: the shipping fee is 7.50\n", "", 1)
	if band == hiddenFixture {
		t.Fatal("the fixture no longer has the \"otherwise\" line this test removes")
	}
	f, h := loadHiddenFixture(t, band)
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	run := hiddenRun{samples: 200, specVersion: "spec-1", codeVersion: "code-1"}
	failed := sampleHiddenCases(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeRigorPath}, "unused", []string{"compute"}, h, f.rules[0].CompositeID, run, false, f.concepts, f.tables, &stdout)
	if failed {
		t.Fatalf("sampleHiddenCases = failed, want conformant - stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "inside this rule's \"given\"") {
		t.Errorf("stdout should say how many cases the rule applied to, got:\n%s", stdout.String())
	}
	runs, err := evidenceledger.Runs(db, hiddenFixtureID)
	if err != nil || len(runs) != 1 {
		t.Fatalf("Runs = %d, %v, want 1", len(runs), err)
	}
	if runs[0].Cases <= 0 || runs[0].Cases >= 200 {
		t.Errorf("recorded cases = %d, want only the applicable ones (more than 0, fewer than the 200 drawn)", runs[0].Cases)
	}
}
