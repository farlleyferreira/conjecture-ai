package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

func hiddenRunFor(f loadedFicha, code, family string, cases int) evidenceledger.Run {
	return evidenceledger.Run{
		HypothesisID: hiddenFixtureID, SpecVersion: fmt.Sprintf("%x", sha256.Sum256(f.raw)), CodeVersion: code,
		Kind: evidenceledger.RunHidden, Cases: cases, Conformant: true, ProducerFamily: family,
	}
}

func TestMeasureTheory_UnsignedAndUnreproducedSaysExactlyWhatIsMissing(t *testing.T) {
	f, _ := loadHiddenFixture(t, hiddenFixture)
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := evidenceledger.AppendRun(db, hiddenRunFor(f, "code-1", "anthropic", 100)); err != nil {
		t.Fatal(err)
	}

	m, err := measureTheory(db, f)
	if err != nil {
		t.Fatalf("measureTheory returned unexpected error: %v", err)
	}
	if m.mechanismConfirmed || m.differentFamilyReproduced {
		t.Errorf("measureTheory = %+v, want neither condition met", m)
	}
	detail := strings.Join(m.detail, "\n")
	if !strings.Contains(detail, "not signed with --confirmed") || !strings.Contains(detail, "qualified so far: anthropic") {
		t.Errorf("detail should name both shortfalls, got:\n%s", detail)
	}
}

func TestMeasureTheory_TwoDeclaredFamiliesEachMeetingTheCountReproduce(t *testing.T) {
	f, _ := loadHiddenFixture(t, hiddenFixture)
	f.signed = &cjtr.Signed{Confirmed: true}
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatal(err)
	}
	for _, r := range []evidenceledger.Run{
		hiddenRunFor(f, "code-1", "anthropic", 100),
		hiddenRunFor(f, "code-2", "openai", 60),
	} {
		if err := evidenceledger.AppendRun(db, r); err != nil {
			t.Fatal(err)
		}
	}

	m, err := measureTheory(db, f)
	if err != nil {
		t.Fatalf("measureTheory returned unexpected error: %v", err)
	}
	if !m.mechanismConfirmed {
		t.Error("mechanismConfirmed = false for a specification signed with --confirmed")
	}
	if m.differentFamilyReproduced {
		t.Error("differentFamilyReproduced = true while the second family has only 60 of 100 cases")
	}

	if err := evidenceledger.AppendRun(db, hiddenRunFor(f, "code-2", "openai", 40)); err != nil {
		t.Fatal(err)
	}
	m, err = measureTheory(db, f)
	if err != nil {
		t.Fatalf("measureTheory returned unexpected error: %v", err)
	}
	if !m.differentFamilyReproduced {
		t.Errorf("measureTheory = %+v, want reproduction met", m)
	}
	// What is still missing is the one thing nothing above recorded:
	// an adversarial attack that found no violation.
	if len(m.detail) != 1 || !strings.Contains(m.detail[0], "no adversarial attack on record") {
		t.Errorf("detail = %q, want only the missing adversarial attack", m.detail)
	}
}

func TestMeasureLaw_NeedsTheoryAndTwoQualifiedCodeVersions(t *testing.T) {
	f, _ := loadHiddenFixture(t, hiddenFixture)
	db := openCheckTestStore(t)
	if err := evidenceledger.EnsureRunSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := evidenceledger.AppendRun(db, hiddenRunFor(f, "code-1", "", 100)); err != nil {
		t.Fatal(err)
	}

	m, err := measureLaw(db, f, hiddenFixtureID)
	if err != nil {
		t.Fatalf("measureLaw returned unexpected error: %v", err)
	}
	if m.atTheory || !m.hiddenSamplesConformant || m.newCodeVersionsConformant {
		t.Errorf("measureLaw = %+v, want not at Theory, hidden samples met, new code versions not met", m)
	}

	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{HypothesisID: hiddenFixtureID, Stage: evidenceledger.Theory, ValidityHash: []byte("v")}); err != nil {
		t.Fatal(err)
	}
	if err := evidenceledger.AppendRun(db, hiddenRunFor(f, "code-2", "", 100)); err != nil {
		t.Fatal(err)
	}
	m, err = measureLaw(db, f, hiddenFixtureID)
	if err != nil {
		t.Fatalf("measureLaw returned unexpected error: %v", err)
	}
	if !m.atTheory || !m.hiddenSamplesConformant || !m.newCodeVersionsConformant || len(m.detail) != 0 {
		t.Errorf("measureLaw = %+v, want every condition met", m)
	}
}

func TestRunPromote_RejectsTheAttestationFlagsItUsedToTake(t *testing.T) {
	cases := [][]string{
		{"theory", "file.cjtr", "--mechanism-confirmed"},
		{"theory", "file.cjtr", "--different-family-reproduced"},
		{"law", "file.cjtr", "some-id", "--hidden-samples-conformant"},
		{"law", "file.cjtr", "some-id", "--new-code-versions-conformant"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		if code := runPromote(args, &stdout, &stderr); code != 1 {
			t.Errorf("runPromote(%v): exit code = %d, want 1", args, code)
		}
		if !strings.Contains(stderr.String(), "no longer accepted") || !strings.Contains(stderr.String(), "measured from the ledger") {
			t.Errorf("runPromote(%v): stderr should explain the flag is now measured, got %q", args, stderr.String())
		}
	}
}
