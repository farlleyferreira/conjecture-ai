package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/adversarialslm"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

func TestRunPromote_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPromote(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runPromote with no subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunPromote_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPromote([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runPromote with an unknown subcommand: exit code = %d, want 1", code)
	}
}

func TestRunPromoteTheory_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPromote([]string{"theory"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("promote theory with no path: exit code = %d, want 1", code)
	}
}

func TestRunPromoteLaw_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPromote([]string{"law", "file.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("promote law with no hypothesis id: exit code = %d, want 1", code)
	}
}

func TestAllHypothesisIDs_CollectsEveryDeclaredID(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	ids := allHypothesisIDs(f)
	if len(ids) != 1 || ids[0] != "demo-area::demo::shipping-rule::fee-matches" {
		t.Errorf("allHypothesisIDs = %v, want [\"demo-area::demo::shipping-rule::fee-matches\"]", ids)
	}
}

func TestPromoteTheory_AllConditionsMetPromotesEveryHypothesis(t *testing.T) {
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
	// Recorded through the real attackHypothesis, not a hand-built
	// Evidence literal, so this also exercises attackHypothesis's own
	// stage-carry-forward fix: without it, this attack record would
	// silently reset the hypothesis back to Hypothesis stage, and the
	// promotion below would wrongly see "not confirmed at Thesis."
	fakeRunner := &fakeAttackRunner{output: adversarialslm.AttackOutput{Outcome: adversarialslm.NoViolationFound}}
	var attackStdout, attackStderr bytes.Buffer
	if code := attackHypothesis(context.Background(), db, f, "fee-matches", fakeRunner, "test-model", &attackStdout, &attackStderr); code != 0 {
		t.Fatalf("attackHypothesis returned exit code %d, want 0; stderr:\n%s", code, attackStderr.String())
	}

	var stdout, stderr bytes.Buffer
	code := promoteTheory(db, f, true, true, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("promoteTheory returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "promoted") {
		t.Errorf("stdout should report the promotion, got %q", stdout.String())
	}

	latest, found, err := evidenceledger.Latest(db, id)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if !found || latest.Stage != evidenceledger.Theory {
		t.Errorf("latest stage = %v (found=%v), want Theory", latest.Stage, found)
	}
}

func TestPromoteTheory_ThesisNotConfirmedDoesNotPromote(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	// No evidence recorded at all - never reached Thesis.

	var stdout, stderr bytes.Buffer
	code := promoteTheory(db, f, true, true, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("promoteTheory returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "not promoted") {
		t.Errorf("stdout should report it was not promoted, got %q", stdout.String())
	}

	const id = "demo-area::demo::shipping-rule::fee-matches"
	_, found, err := evidenceledger.Latest(db, id)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if found {
		t.Error("Latest found = true, want false (nothing should have been recorded)")
	}
}

func TestPromoteTheory_MechanismNotConfirmedDoesNotPromote(t *testing.T) {
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
	fakeRunner := &fakeAttackRunner{output: adversarialslm.AttackOutput{Outcome: adversarialslm.NoViolationFound}}
	var attackStdout, attackStderr bytes.Buffer
	if code := attackHypothesis(context.Background(), db, f, "fee-matches", fakeRunner, "test-model", &attackStdout, &attackStderr); code != 0 {
		t.Fatalf("attackHypothesis returned exit code %d, want 0; stderr:\n%s", code, attackStderr.String())
	}

	var stdout, stderr bytes.Buffer
	// mechanismConfirmed = false this time.
	code := promoteTheory(db, f, false, true, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("promoteTheory returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "not promoted") {
		t.Errorf("stdout should report it was not promoted, got %q", stdout.String())
	}
}

func TestPromoteLaw_UniversalAndBothFlagsPromotes(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)

	var stdout, stderr bytes.Buffer
	code := promoteLaw(db, f, "fee-matches", true, true, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("promoteLaw returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "promoted to Law") {
		t.Errorf("stdout should report the promotion, got %q", stdout.String())
	}

	const id = "demo-area::demo::shipping-rule::fee-matches"
	latest, found, err := evidenceledger.Latest(db, id)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if !found || latest.Stage != evidenceledger.Law {
		t.Errorf("latest stage = %v (found=%v), want Law", latest.Stage, found)
	}
}

func TestPromoteLaw_MissingFlagDoesNotPromote(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)

	var stdout, stderr bytes.Buffer
	code := promoteLaw(db, f, "fee-matches", true, false, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("promoteLaw returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "not promoted") {
		t.Errorf("stdout should report it was not promoted, got %q", stdout.String())
	}
}

func TestPromoteLaw_UnknownHypothesisIDIsError(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)

	var stdout, stderr bytes.Buffer
	code := promoteLaw(db, f, "no-such-id", true, true, &stdout, &stderr)
	if code != 1 {
		t.Errorf("promoteLaw with an unknown id: exit code = %d, want 1", code)
	}
}
