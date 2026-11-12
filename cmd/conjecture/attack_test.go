package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/adversarialslm"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

type fakeAttackRunner struct {
	output adversarialslm.AttackOutput
	err    error
}

func (f *fakeAttackRunner) Attack(context.Context, adversarialslm.AttackInput) (adversarialslm.AttackOutput, error) {
	return f.output, f.err
}

func TestRunAttack_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runAttack([]string{"only-one-arg"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runAttack with one argument: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunAttack_UnrecognizedRuntimeIsError(t *testing.T) {
	// An unrecognized --runtime fails before ever touching the real
	// per-OS store or a real model server - safe to call directly.
	var stdout, stderr bytes.Buffer
	code := runAttack([]string{"file.cjtr", "some-id", "--runtime", "bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runAttack with an unrecognized runtime: exit code = %d, want 1", code)
	}
}

func TestRunAttack_OllamaRuntimeWithoutModelIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runAttack([]string{"file.cjtr", "some-id", "--runtime", "ollama"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runAttack with --runtime ollama and no --model: exit code = %d, want 1", code)
	}
}

func TestBuildAttackRunner_OllamaDefaultsToOllamaRuntime(t *testing.T) {
	r, err := buildAttackRunner("", "qwen2.5:0.5b", "")
	if err != nil {
		t.Fatalf("buildAttackRunner returned unexpected error: %v", err)
	}
	ollama, ok := r.(*adversarialslm.OllamaRunner)
	if !ok {
		t.Fatalf("buildAttackRunner returned %T, want *adversarialslm.OllamaRunner", r)
	}
	if ollama.Model != "qwen2.5:0.5b" {
		t.Errorf("Model = %q, want %q", ollama.Model, "qwen2.5:0.5b")
	}
}

func TestBuildAttackRunner_OllamaWithoutModelIsError(t *testing.T) {
	if _, err := buildAttackRunner("ollama", "", ""); err == nil {
		t.Fatal("buildAttackRunner(\"ollama\", \"\", \"\") = nil error, want one")
	}
}

func TestBuildAttackRunner_LlamaCppNeedsNoModel(t *testing.T) {
	r, err := buildAttackRunner("llamacpp", "", "")
	if err != nil {
		t.Fatalf("buildAttackRunner returned unexpected error: %v", err)
	}
	if _, ok := r.(*adversarialslm.LlamaCppRunner); !ok {
		t.Fatalf("buildAttackRunner returned %T, want *adversarialslm.LlamaCppRunner", r)
	}
}

func TestBuildAttackRunner_BaseURLOverride(t *testing.T) {
	r, err := buildAttackRunner("llamacpp", "", "http://127.0.0.1:9999")
	if err != nil {
		t.Fatalf("buildAttackRunner returned unexpected error: %v", err)
	}
	llamaCpp := r.(*adversarialslm.LlamaCppRunner)
	if llamaCpp.BaseURL != "http://127.0.0.1:9999" {
		t.Errorf("BaseURL = %q, want the override", llamaCpp.BaseURL)
	}
}

func TestBuildAttackRunner_UnrecognizedRuntimeIsError(t *testing.T) {
	if _, err := buildAttackRunner("bogus", "", ""); err == nil {
		t.Fatal("buildAttackRunner(\"bogus\", ...) = nil error, want one")
	}
}

func TestFindHypothesisByID_FindsRuleScopedHypothesis(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	h, compositeID, interfaceText, found := findHypothesisByID(f, "fee-matches")
	if !found {
		t.Fatal("findHypothesisByID found = false, want true")
	}
	if h.Name != "fee matches" {
		t.Errorf("h.Name = %q, want %q", h.Name, "fee matches")
	}
	if compositeID != "demo-area::demo::shipping-rule::fee-matches" {
		t.Errorf("compositeID = %q, want %q", compositeID, "demo-area::demo::shipping-rule::fee-matches")
	}
	if interfaceText != "" {
		t.Errorf("interfaceText = %q, want empty (the fixture's rule declares no \"when\")", interfaceText)
	}
}

func TestFindHypothesisByID_UnknownIDNotFound(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	_, _, _, found := findHypothesisByID(f, "no-such-id")
	if found {
		t.Error("findHypothesisByID found = true for an unknown id, want false")
	}
}

func TestBuildAttackInput_UniversalHypothesisAssertion(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	h, _, interfaceText, found := findHypothesisByID(f, "fee-matches")
	if !found {
		t.Fatal("findHypothesisByID found = false, want true")
	}

	input := buildAttackInput(h, interfaceText, nil)
	if !strings.Contains(input.Assertion, "always") {
		t.Errorf("Assertion = %q, want it to mention \"always\" (a universal hypothesis)", input.Assertion)
	}
	if !strings.Contains(input.Assertion, "shipping fee") {
		t.Errorf("Assertion = %q, want it to mention the Then statement", input.Assertion)
	}
	if len(input.PriorResults) != 0 {
		t.Errorf("PriorResults = %+v, want none (no history given)", input.PriorResults)
	}
}

func TestBuildAttackInput_PriorResultsOnlyFromAdversarialSLMEvidence(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	h, _, _, found := findHypothesisByID(f, "fee-matches")
	if !found {
		t.Fatal("findHypothesisByID found = false, want true")
	}

	history := []evidenceledger.Evidence{
		{ProducedBy: "conformance cycle (directed sampling)", Attacked: "directed sampling", Survived: "every generated case conformed"},
		{ProducedBy: adversarialSLMProducedBy, Attacked: adversarialSLMProducedBy + " attack", Survived: "no violation found by adversarial SLM"},
		{ProducedBy: adversarialSLMProducedBy, Attacked: violationCounterexamplePrefix + "x = -1"},
	}

	input := buildAttackInput(h, "", history)
	if len(input.PriorResults) != 2 {
		t.Fatalf("len(PriorResults) = %d, want 2 (only the adversarial-SLM-produced records)", len(input.PriorResults))
	}
	if input.PriorResults[0].Outcome != adversarialslm.NoViolationFound {
		t.Errorf("PriorResults[0].Outcome = %v, want NoViolationFound", input.PriorResults[0].Outcome)
	}
	if input.PriorResults[1].Outcome != adversarialslm.ViolationFound || input.PriorResults[1].Counterexample != "x = -1" {
		t.Errorf("PriorResults[1] = %+v, want ViolationFound with counterexample \"x = -1\"", input.PriorResults[1])
	}
}

func TestAttackHypothesis_NoViolationFoundRecordsEvidence(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	runner := &fakeAttackRunner{output: adversarialslm.AttackOutput{Outcome: adversarialslm.NoViolationFound}}

	var stdout, stderr bytes.Buffer
	code := attackHypothesis(context.Background(), db, f, "fee-matches", runner, "test-model", &stdout, &stderr)
	if code != 0 {
		t.Fatalf("attackHypothesis returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no_violation_found") {
		t.Errorf("stdout should report no_violation_found, got %q", stdout.String())
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].ProducedBy != adversarialSLMProducedBy {
		t.Errorf("history[0].ProducedBy = %q, want %q", history[0].ProducedBy, adversarialSLMProducedBy)
	}
	if history[0].Survived == "" {
		t.Error("history[0].Survived is empty, want it to report no violation was found")
	}
}

func TestAttackHypothesis_ViolationFoundRecordsEvidence(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	runner := &fakeAttackRunner{output: adversarialslm.AttackOutput{Outcome: adversarialslm.ViolationFound, Counterexample: "shipping fee = 0.00"}}

	var stdout, stderr bytes.Buffer
	code := attackHypothesis(context.Background(), db, f, "fee-matches", runner, "test-model", &stdout, &stderr)
	if code != 0 {
		t.Fatalf("attackHypothesis returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "violation_found") {
		t.Errorf("stdout should report violation_found, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "shipping fee = 0.00") {
		t.Errorf("stdout should report the counterexample, got %q", stdout.String())
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].Survived != "" {
		t.Errorf("history[0].Survived = %q, want empty (a violation was found)", history[0].Survived)
	}
	if !strings.Contains(history[0].Attacked, "shipping fee = 0.00") {
		t.Errorf("history[0].Attacked = %q, want it to mention the counterexample", history[0].Attacked)
	}
}

func TestAttackHypothesis_UnknownHypothesisIDIsError(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	runner := &fakeAttackRunner{output: adversarialslm.AttackOutput{Outcome: adversarialslm.NoViolationFound}}

	var stdout, stderr bytes.Buffer
	code := attackHypothesis(context.Background(), db, f, "no-such-id", runner, "test-model", &stdout, &stderr)
	if code != 1 {
		t.Errorf("attackHypothesis with an unknown id: exit code = %d, want 1", code)
	}
}

func TestAttackHypothesis_RunnerErrorIsError(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	runner := &fakeAttackRunner{err: errors.New("connection refused")}

	var stdout, stderr bytes.Buffer
	code := attackHypothesis(context.Background(), db, f, "fee-matches", runner, "test-model", &stdout, &stderr)
	if code != 1 {
		t.Errorf("attackHypothesis with a runner error: exit code = %d, want 1", code)
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("len(history) = %d, want 0 (nothing should be recorded when the attack itself failed)", len(history))
	}
}

func TestAttackHypothesis_CarriesForwardTheCurrentStage(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	const id = "demo-area::demo::shipping-rule::fee-matches"
	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: id, Stage: evidenceledger.Thesis, ValidityHash: []byte("v"),
	}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	runner := &fakeAttackRunner{output: adversarialslm.AttackOutput{Outcome: adversarialslm.NoViolationFound}}

	var stdout, stderr bytes.Buffer
	code := attackHypothesis(context.Background(), db, f, "fee-matches", runner, "test-model", &stdout, &stderr)
	if code != 0 {
		t.Fatalf("attackHypothesis returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}

	latest, found, err := evidenceledger.Latest(db, id)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if !found || latest.Stage != evidenceledger.Thesis {
		t.Errorf("latest stage = %v (found=%v), want Thesis carried forward, not reset to Hypothesis", latest.Stage, found)
	}
}
