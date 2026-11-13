package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// implementFixture declares exactly one input concept with a
// single-value domain ("one_of: [x]"), so DirectedSamplingInputs
// generates exactly one case per conformance check - one sandbox
// invocation per outer implement-loop round, which is what lets
// fakeimplement's own attempt counter track rounds one-to-one.
const implementFixture = `
conjecture: 1.0.0

phenomenon: demo implement

concepts:
  flag:
    variable: input.flag
    one_of: [x]
  result:
    variable: output.result
    type: decimal

rules:
  result rule:
    id: result-rule
    hypotheses:
      result matches:
        id: result-matches
        always: the result is 7.50
        examples:
          - {flag: x, result: 7.50}
`

// implementFixtureWithLaw adds a "user_defined_laws" entry (with its
// own declared "id", unlike verify_test.go's fixtureCjtrWithLaw, which
// needs none since runVerify never persists anything by id) to
// confirm [findHypothesis] reports a law as isLaw=true rather than
// treating it as a plain hypothesis.
const implementFixtureWithLaw = `
conjecture: 1.0.0

phenomenon: demo implement

concepts:
  flag:
    variable: input.flag
    one_of: [x]
  result:
    variable: output.result
    type: decimal

rules:
  result rule:
    id: result-rule
    user_defined_laws:
      result law:
        id: result-law
        always: the result is 7.50
        examples:
          - {flag: x, result: 7.50}
`

func writeImplementFixtureWithLaw(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(implementFixtureWithLaw), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func writeImplementFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(implementFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

// implementFixtureUnboundedDomain declares its one input concept with
// a "range" but no "places" - [cjtr.BoundaryValues] still returns its
// two endpoints (so [conformancecycle.DirectedSamplingInputs] - and so
// the Type A omission check - still has cases to work with), but
// [cjtr.ExhaustiveValues] refuses it (a decimal range needs a declared
// step to enumerate "every value" of, not just its boundary): neither
// [conformancecycle.CompleteProofInputs] nor [conformancecycle.
// BoundedProofInputs] finds a boundable domain here. This is the
// fixture [TestRunImplementLoop_RefusesUnboundedDomain] (and
// work_test.go's own equivalent) uses to exercise
// [ErrDomainTooComplexForOracle].
const implementFixtureUnboundedDomain = `
conjecture: 1.0.0

phenomenon: demo implement unbounded

concepts:
  amount:
    variable: input.amount
    type: decimal
    range: {min: 0.00, max: 1000.00}
  result:
    variable: output.result
    type: decimal

rules:
  result rule:
    id: result-rule
    hypotheses:
      result matches:
        id: result-matches
        always: the result is 7.50
        examples:
          - {amount: 10.00, result: 7.50}
`

func writeImplementFixtureUnboundedDomain(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(implementFixtureUnboundedDomain), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

// fakeImplementerClient stands in for an LLM provider: it never
// actually generates code (fakeimplement's own sandbox binary ignores
// whatever is written, see its doc comment) - it only records every
// prompt it was sent, so a test can assert what the implement loop
// told it on each round.
type fakeImplementerClient struct {
	prompts []string
}

func (c *fakeImplementerClient) Complete(_ context.Context, _, userPrompt string) (string, error) {
	c.prompts = append(c.prompts, userPrompt)
	return "print('placeholder')\n", nil
}

func TestRunImplement_TooFewArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runImplement([]string{"file.cjtr", "rule-id"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runImplement with too few arguments = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), implementUsage) {
		t.Errorf("stderr = %q, want it to contain the usage string", stderr.String())
	}
}

func TestRunImplement_InvalidMaxRounds(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runImplement([]string{"--max-rounds", "0", "file.cjtr", "rule-id", "hyp-id"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runImplement with --max-rounds 0 = %d, want 1", code)
	}
}

func TestRunImplement_UnconstructableProviderFails(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	path := writeImplementFixture(t)

	var stdout, stderr bytes.Buffer
	code := runImplement([]string{path, "result-rule", "result-matches"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runImplement with no provider key = %d, want 1", code)
	}
}

func TestFindHypothesis_Found(t *testing.T) {
	path := writeImplementFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	rule, h, isLaw, err := findHypothesis(f, "result-rule", "result-matches")
	if err != nil {
		t.Fatalf("findHypothesis returned unexpected error: %v", err)
	}
	if rule.ID != "result-rule" || h.ID != "result-matches" {
		t.Errorf("findHypothesis returned rule %+v, hypothesis %+v", rule, h)
	}
	if isLaw {
		t.Error("findHypothesis reported isLaw = true for a plain hypothesis, want false")
	}
}

func TestFindHypothesis_FoundLawReportsIsLaw(t *testing.T) {
	path := writeImplementFixtureWithLaw(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	rule, h, isLaw, err := findHypothesis(f, "result-rule", "result-law")
	if err != nil {
		t.Fatalf("findHypothesis returned unexpected error: %v", err)
	}
	if rule.ID != "result-rule" || h.ID != "result-law" {
		t.Errorf("findHypothesis returned rule %+v, hypothesis %+v", rule, h)
	}
	if !isLaw {
		t.Error("findHypothesis reported isLaw = false for a user-defined law, want true")
	}
}

func TestFindHypothesis_UnknownRule(t *testing.T) {
	path := writeImplementFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	_, _, _, err = findHypothesis(f, "does-not-exist", "result-matches")
	if err == nil {
		t.Fatal("findHypothesis with an unknown rule id = nil error, want one")
	}
}

func TestFindHypothesis_UnknownHypothesis(t *testing.T) {
	path := writeImplementFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	_, _, _, err = findHypothesis(f, "result-rule", "does-not-exist")
	if err == nil {
		t.Fatal("findHypothesis with an unknown hypothesis id = nil error, want one")
	}
}

func TestRunImplementLoop_SucceedsWithinBudgetAndRecordsEvidence(t *testing.T) {
	t.Setenv("FAKEIMPLEMENT_SUCCEED_AFTER", "2")
	path := writeImplementFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	rule, h, isLaw, err := findHypothesis(f, "result-rule", "result-matches")
	if err != nil {
		t.Fatalf("findHypothesis returned unexpected error: %v", err)
	}

	db := openCheckTestStore(t)
	client := &fakeImplementerClient{}
	sb := &sandbox.Sandbox{DockerBinary: fakeImplementPath}

	verdict, err := runImplementLoop(context.Background(), db, sb, client, rule, h, 3, "test-spec-version", isLaw, f.concepts, f.tables, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("runImplementLoop returned unexpected error: %v", err)
	}
	if verdict != conformancecycle.Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
	if len(client.prompts) != 2 {
		t.Fatalf("len(client.prompts) = %d, want 2 (one failing round, one succeeding)", len(client.prompts))
	}
	if strings.Contains(client.prompts[0], "previous attempt") {
		t.Error("round 0's prompt mentions a previous attempt, want none on the first round")
	}
	if !strings.Contains(client.prompts[1], "previous attempt") || !strings.Contains(client.prompts[1], "failing case") {
		t.Errorf("round 1's prompt = %q, want it to reference the previous attempt and the failing case", client.prompts[1])
	}

	history, err := evidenceledger.History(db, rule.CompositeID+"::"+h.ID)
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
}

func TestRunImplementLoop_NoBoundedInputIsTypeAOmission(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"input value": {Name: "input value", Kind: cjtr.VariableConcept, Variable: "input.x", Type: "decimal"},
		"result":      {Name: "result", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the result is 7.50", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	h := cjtr.Hypothesis{
		Name: "no bounded input", ID: "no-bounded-input", IsUniversal: true, Then: then,
		Examples: []map[string]string{{"input value": "1", "result": "7.50"}},
	}
	rule := cjtr.Rule{ID: "result-rule", CompositeID: "demo-area::demo::result-rule"}

	db := openCheckTestStore(t)
	client := &fakeImplementerClient{}
	sb := &sandbox.Sandbox{DockerBinary: fakeImplementPath}

	var stdout bytes.Buffer
	verdict, err := runImplementLoop(context.Background(), db, sb, client, rule, h, 3, "test-spec-version", false, concepts, nil, &stdout)
	if err != nil {
		t.Fatalf("runImplementLoop returned unexpected error: %v", err)
	}
	if verdict != conformancecycle.Inconclusive {
		t.Errorf("verdict = %v, want Inconclusive", verdict)
	}
	if !strings.Contains(stdout.String(), "TYPE A OMISSION") {
		t.Errorf("stdout = %q, want it to report a Type A omission", stdout.String())
	}
	if len(client.prompts) != 0 {
		t.Errorf("len(client.prompts) = %d, want 0 (the implementer should never be asked for code with nothing to test it against)", len(client.prompts))
	}
}

func TestRunImplementLoop_RefusesUnboundedDomain(t *testing.T) {
	path := writeImplementFixtureUnboundedDomain(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	rule, h, isLaw, err := findHypothesis(f, "result-rule", "result-matches")
	if err != nil {
		t.Fatalf("findHypothesis returned unexpected error: %v", err)
	}

	db := openCheckTestStore(t)
	client := &fakeImplementerClient{}
	sb := &sandbox.Sandbox{DockerBinary: fakeImplementPath}

	_, err = runImplementLoop(context.Background(), db, sb, client, rule, h, 3, "test-spec-version", isLaw, f.concepts, f.tables, &bytes.Buffer{})
	if !errors.Is(err, ErrDomainTooComplexForOracle) {
		t.Fatalf("runImplementLoop error = %v, want ErrDomainTooComplexForOracle", err)
	}
	if len(client.prompts) != 0 {
		t.Errorf("len(client.prompts) = %d, want 0 (the oracle must never be asked for code against a domain it cannot be trusted on)", len(client.prompts))
	}

	history, err := evidenceledger.History(db, rule.CompositeID+"::"+h.ID)
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("history = %+v, want no evidence recorded for a refused hypothesis", history)
	}
}

func TestRunImplementLoop_ExhaustsBudgetWithoutConforming(t *testing.T) {
	t.Setenv("FAKEIMPLEMENT_SUCCEED_AFTER", "0") // never succeeds
	path := writeImplementFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	rule, h, isLaw, err := findHypothesis(f, "result-rule", "result-matches")
	if err != nil {
		t.Fatalf("findHypothesis returned unexpected error: %v", err)
	}

	db := openCheckTestStore(t)
	client := &fakeImplementerClient{}
	sb := &sandbox.Sandbox{DockerBinary: fakeImplementPath}

	verdict, err := runImplementLoop(context.Background(), db, sb, client, rule, h, 2, "test-spec-version", isLaw, f.concepts, f.tables, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("runImplementLoop returned unexpected error: %v", err)
	}
	if verdict != conformancecycle.NonConformant {
		t.Errorf("verdict = %v, want NonConformant (VerdictOnBudgetExhausted)", verdict)
	}
	if len(client.prompts) != 2 {
		t.Errorf("len(client.prompts) = %d, want 2 (maxRounds)", len(client.prompts))
	}
}
