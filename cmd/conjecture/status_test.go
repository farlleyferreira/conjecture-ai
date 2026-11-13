package main

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
)

// statusFixture declares one rule-scoped hypothesis ("fee-matches")
// and one file-level user-defined law ("fee-never-negative"), each
// with its own declared "id" - enough to exercise [computeStatus]'s
// two distinct composite-ID scopes (check.go's own convention:
// f.phenomenonID for a file-level law, rule.CompositeID for a
// rule-scoped one), plus a third hypothesis with no declared "id" to
// exercise the Notes path.
const statusFixture = `
conjecture: 1.0.0

phenomenon: demo

concepts:
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal

user_defined_laws:
  fee never negative:
    id: fee-never-negative
    always: the shipping fee is at least 0.00
    examples:
      - {order type: regular, shipping fee: 0.00}

rules:
  shipping fee rule:
    id: shipping-rule
    hypotheses:
      fee matches:
        id: fee-matches
        always: the shipping fee is 7.50
        examples:
          - {order type: regular, shipping fee: 7.50}
      unidentified hypothesis:
        always: the shipping fee is 7.50
        examples:
          - {order type: regular, shipping fee: 7.50}
`

func writeStatusFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(statusFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func openStatusTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := evidenceledger.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestRunStatus_WrongArgCountIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runStatus(nil, &stdout, &stderr); code != 1 {
		t.Errorf("runStatus with no args exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), statusUsage) {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), statusUsage)
	}
}

func TestRunStatus_MissingFileReportsFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStatus([]string{"this-file-does-not-exist.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runStatus exit code = %d, want 1 for a missing file", code)
	}
}

func TestComputeStatus_UnsignedAndNeverCheckedReportsHonestly(t *testing.T) {
	path := writeStatusFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openStatusTestStore(t)

	status, err := computeStatus(db, f)
	if err != nil {
		t.Fatalf("computeStatus returned unexpected error: %v", err)
	}

	if status.Signed {
		t.Error("status.Signed = true, want false for an unsigned fixture")
	}
	if len(status.Entries) != 2 {
		t.Fatalf("status.Entries = %+v, want exactly 2 (the law and the identified hypothesis)", status.Entries)
	}
	for _, e := range status.Entries {
		if e.Checked {
			t.Errorf("entry %q Checked = true, want false (nothing has run check yet)", e.ID)
		}
	}
	var sawLaw bool
	for _, e := range status.Entries {
		if e.IsLaw {
			sawLaw = true
			if e.LawStatus != userdefinedlaws.Unverified.String() {
				t.Errorf("law entry LawStatus = %q, want %q", e.LawStatus, userdefinedlaws.Unverified.String())
			}
		}
	}
	if !sawLaw {
		t.Error("status.Entries has no law entry, want one for \"fee-never-negative\"")
	}

	var foundNote bool
	for _, n := range status.Notes {
		if strings.Contains(n, "no declared") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Errorf("status.Notes = %+v, want a note about the hypothesis with no declared id", status.Notes)
	}
}

func TestComputeStatus_SignedAndCheckedReportsAggregatedState(t *testing.T) {
	path := writeStatusFixture(t)

	storeDir := filepath.Join(t.TempDir(), "decisions")
	previous := openDecisionStore
	openDecisionStore = func() (*sql.DB, error) { return persistence.OpenStoreAt(storeDir) }
	t.Cleanup(func() { openDecisionStore = previous })

	var signStdout, signStderr bytes.Buffer
	if code := runSign([]string{"--user", "alice", "--confirmed", path}, &signStdout, &signStderr); code != 0 {
		t.Fatalf("runSign exit code = %d, want 0; stderr: %s", code, signStderr.String())
	}

	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	if f.signed == nil {
		t.Fatal("f.signed = nil after runSign, want a signed block")
	}

	db := openStatusTestStore(t)
	hypothesisID := f.phenomenonID + "::shipping-rule::fee-matches"
	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Hypothesis,
		ProducedBy: "conjecture check test", Attacked: "directed sampling",
		Survived: "every generated case conformed", ValidityHash: []byte("x"),
	}); err != nil {
		t.Fatalf("evidenceledger.Append returned unexpected error: %v", err)
	}

	lawID := f.phenomenonID + "::fee-never-negative"
	if err := userdefinedlaws.SetStatus(db, lawID, userdefinedlaws.Conformant); err != nil {
		t.Fatalf("userdefinedlaws.SetStatus returned unexpected error: %v", err)
	}

	status, err := computeStatus(db, f)
	if err != nil {
		t.Fatalf("computeStatus returned unexpected error: %v", err)
	}

	if !status.Signed || status.SignedBy != "alice" || !status.Confirmed {
		t.Errorf("status signed fields = %+v, want signed by alice, confirmed", status)
	}

	var sawCheckedHypothesis, sawLawStatus bool
	for _, e := range status.Entries {
		switch e.ID {
		case hypothesisID:
			sawCheckedHypothesis = true
			if !e.Checked || e.Stage != evidenceledger.Hypothesis.String() || e.Attacked != "directed sampling" {
				t.Errorf("hypothesis entry = %+v, want checked at Hypothesis stage with attacked=directed sampling", e)
			}
		case lawID:
			sawLawStatus = true
			if e.LawStatus != userdefinedlaws.Conformant.String() {
				t.Errorf("law entry LawStatus = %q, want %q", e.LawStatus, userdefinedlaws.Conformant.String())
			}
		}
	}
	if !sawCheckedHypothesis {
		t.Errorf("status.Entries = %+v, missing the checked hypothesis %q", status.Entries, hypothesisID)
	}
	if !sawLawStatus {
		t.Errorf("status.Entries = %+v, missing the law %q", status.Entries, lawID)
	}
}
