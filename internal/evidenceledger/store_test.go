package evidenceledger

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

// testValidityHash stands in for a real [ValidityHash] in tests that
// exercise chaining/ordering, not the validity hash's own content.
var testValidityHash = []byte("test-validity-hash")

func openTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestAppend_FirstRecordHasNoPreviousChain(t *testing.T) {
	db := openTestStore(t)

	stored, err := Append(db, Evidence{
		HypothesisID: "shipping::free-shipping::free-above::never-negative",
		Stage:        Hypothesis,
		Attacked:     "boundary values near zero",
		Survived:     "all 100 boundary cases",
		ProducedBy:   "directed-sampling",
		ValidityHash: ValidityHash("spec-1", "code-1", "harness-1", "slm-1", "op-1"),
	})
	if err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if stored.ID == 0 {
		t.Error("stored.ID = 0, want a non-zero assigned id")
	}
	if len(stored.ChainHash) == 0 {
		t.Error("stored.ChainHash is empty, want a computed hash even for the first record")
	}
	if stored.CreatedAt.IsZero() {
		t.Error("stored.CreatedAt is zero, want the time Append recorded it")
	}
}

func TestAppend_ChainsToPreviousRecord(t *testing.T) {
	db := openTestStore(t)
	const hid = "shipping::free-shipping::free-above::never-negative"

	first, err := Append(db, Evidence{HypothesisID: hid, Stage: Hypothesis, ProducedBy: "a", ValidityHash: testValidityHash})
	if err != nil {
		t.Fatalf("Append (first) returned unexpected error: %v", err)
	}
	second, err := Append(db, Evidence{HypothesisID: hid, Stage: Thesis, ProducedBy: "b", ValidityHash: testValidityHash})
	if err != nil {
		t.Fatalf("Append (second) returned unexpected error: %v", err)
	}

	if bytes.Equal(first.ChainHash, second.ChainHash) {
		t.Error("two different records produced the same chain hash")
	}

	// Changing the previous record's chain hash must change the next
	// one's, proving the chain actually depends on it (not just on the
	// new record's own content).
	altered := chainHashOf([]byte("a different previous hash"), Evidence{
		HypothesisID: hid, Stage: Thesis, ProducedBy: "b", ValidityHash: testValidityHash,
	})
	if bytes.Equal(altered, second.ChainHash) {
		t.Error("chain hash did not actually depend on the previous record's chain hash")
	}
}

func TestAppend_SeparateHypothesesHaveIndependentChains(t *testing.T) {
	db := openTestStore(t)

	a, err := Append(db, Evidence{HypothesisID: "a", Stage: Hypothesis, ProducedBy: "x", ValidityHash: testValidityHash})
	if err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	b, err := Append(db, Evidence{HypothesisID: "b", Stage: Hypothesis, ProducedBy: "x", ValidityHash: testValidityHash})
	if err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	// Both must chain from "no previous," independently of each other -
	// each one's stored chain hash should match what chaining fresh
	// from nil would give it.
	if want := chainHashOf(nil, Evidence{HypothesisID: "a", Stage: Hypothesis, ProducedBy: "x", ValidityHash: testValidityHash}); !bytes.Equal(a.ChainHash, want) {
		t.Errorf("a.ChainHash did not chain from \"no previous\"")
	}
	if want := chainHashOf(nil, Evidence{HypothesisID: "b", Stage: Hypothesis, ProducedBy: "x", ValidityHash: testValidityHash}); !bytes.Equal(b.ChainHash, want) {
		t.Errorf("b.ChainHash did not chain from \"no previous\"")
	}
	// The chain hash also covers HypothesisID itself, so the two must
	// still not collide.
	if bytes.Equal(a.ChainHash, b.ChainHash) {
		t.Error("two different hypotheses' first records produced the same chain hash")
	}
}

func TestLatest_NoRecordsYet(t *testing.T) {
	db := openTestStore(t)

	_, ok, err := Latest(db, "nonexistent")
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if ok {
		t.Error("Latest on a hypothesis with no records = ok true, want false")
	}
}

func TestLatest_ReturnsMostRecent(t *testing.T) {
	db := openTestStore(t)
	const hid = "h"

	if _, err := Append(db, Evidence{HypothesisID: hid, Stage: Hypothesis, ProducedBy: "a", ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if _, err := Append(db, Evidence{HypothesisID: hid, Stage: Thesis, ProducedBy: "b", ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	latest, ok, err := Latest(db, hid)
	if err != nil {
		t.Fatalf("Latest returned unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("Latest = ok false, want true")
	}
	if latest.Stage != Thesis || latest.ProducedBy != "b" {
		t.Errorf("Latest = %+v, want the second (most recent) record", latest)
	}
}

func TestHistory_OldestFirst(t *testing.T) {
	db := openTestStore(t)
	const hid = "h"

	for _, producedBy := range []string{"a", "b", "c"} {
		if _, err := Append(db, Evidence{HypothesisID: hid, Stage: Hypothesis, ProducedBy: producedBy, ValidityHash: testValidityHash}); err != nil {
			t.Fatalf("Append returned unexpected error: %v", err)
		}
	}

	history, err := History(db, hid)
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("len(History) = %d, want 3", len(history))
	}
	for i, want := range []string{"a", "b", "c"} {
		if history[i].ProducedBy != want {
			t.Errorf("History[%d].ProducedBy = %q, want %q", i, history[i].ProducedBy, want)
		}
	}
}

func TestEnsureSchema_IdempotentAcrossReopens(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")

	db1, err := persistence.OpenStoreAt(dir)
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	if err := EnsureSchema(db1); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if _, err := Append(db1, Evidence{HypothesisID: "h", Stage: Hypothesis, ProducedBy: "a", ValidityHash: testValidityHash}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("db1.Close returned unexpected error: %v", err)
	}

	db2, err := persistence.OpenStoreAt(dir)
	if err != nil {
		t.Fatalf("OpenStoreAt (reopen) returned unexpected error: %v", err)
	}
	defer func() { _ = db2.Close() }()
	if err := EnsureSchema(db2); err != nil {
		t.Fatalf("EnsureSchema (reopen) returned unexpected error: %v", err)
	}

	history, err := History(db2, "h")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(History) = %d, want 1 (the record from before the reopen)", len(history))
	}
}
