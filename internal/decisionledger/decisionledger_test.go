package decisionledger

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

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
	got, err := Append(db, Decision{Kind: "promotion", Subject: "area::phenomenon::rule::hyp", Determination: "Theory", DecidedBy: "alice"})
	if err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if got.ID == 0 {
		t.Error("Append did not set a nonzero ID")
	}
	if len(got.ChainHash) == 0 {
		t.Error("Append did not set a ChainHash")
	}
	if got.CreatedAt.IsZero() {
		t.Error("Append did not set CreatedAt")
	}
}

func TestAppend_ChainsToThePreviousRecordRegardlessOfKindOrSubject(t *testing.T) {
	db := openTestStore(t)
	first, err := Append(db, Decision{Kind: "promotion", Subject: "s1", Determination: "Theory", DecidedBy: "alice"})
	if err != nil {
		t.Fatalf("first Append returned unexpected error: %v", err)
	}
	second, err := Append(db, Decision{Kind: "candidate-review", Subject: "s2", Determination: "accepted", DecidedBy: "bob"})
	if err != nil {
		t.Fatalf("second Append returned unexpected error: %v", err)
	}

	payload, err := payloadOf(second)
	if err != nil {
		t.Fatalf("payloadOf returned unexpected error: %v", err)
	}
	want := persistence.ChainHash(first.ChainHash, payload)
	if string(second.ChainHash) != string(want) {
		t.Errorf("second.ChainHash does not chain from the first record's own chain hash")
	}
}

func TestAppend_MissingRequiredFieldIsError(t *testing.T) {
	db := openTestStore(t)
	cases := []Decision{
		{Subject: "s", Determination: "d", DecidedBy: "alice"},
		{Kind: "k", Determination: "d", DecidedBy: "alice"},
		{Kind: "k", Subject: "s", DecidedBy: "alice"},
		{Kind: "k", Subject: "s", Determination: "d"},
	}
	for _, d := range cases {
		if _, err := Append(db, d); !errors.Is(err, ErrIncompleteDecision) {
			t.Errorf("Append(%+v) error = %v, want ErrIncompleteDecision", d, err)
		}
	}
}

func TestAppend_EmptyRationaleIsAllowed(t *testing.T) {
	db := openTestStore(t)
	if _, err := Append(db, Decision{Kind: "candidate-review", Subject: "s", Determination: "rejected", DecidedBy: "alice"}); err != nil {
		t.Errorf("Append with an empty Rationale returned unexpected error: %v", err)
	}
}

func TestList_ReturnsEveryDecisionOldestFirst(t *testing.T) {
	db := openTestStore(t)
	if _, err := Append(db, Decision{Kind: "promotion", Subject: "s1", Determination: "Theory", DecidedBy: "alice"}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if _, err := Append(db, Decision{Kind: "promotion", Subject: "s2", Determination: "Law", DecidedBy: "bob"}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	got, err := List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Subject != "s1" || got[1].Subject != "s2" {
		t.Errorf("List = %+v, want [s1, s2] in that order", got)
	}
}

func TestVerify_TrueForAnUntamperedChain(t *testing.T) {
	db := openTestStore(t)
	for i := 0; i < 3; i++ {
		if _, err := Append(db, Decision{Kind: "promotion", Subject: "s", Determination: "Theory", DecidedBy: "alice"}); err != nil {
			t.Fatalf("Append returned unexpected error: %v", err)
		}
	}
	ok, err := Verify(db)
	if err != nil {
		t.Fatalf("Verify returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("Verify = false for an untampered chain, want true")
	}
}

func TestVerify_FalseWhenARowWasTamperedWith(t *testing.T) {
	db := openTestStore(t)
	for i := 0; i < 3; i++ {
		if _, err := Append(db, Decision{Kind: "promotion", Subject: "s", Determination: "Theory", DecidedBy: "alice"}); err != nil {
			t.Fatalf("Append returned unexpected error: %v", err)
		}
	}
	if _, err := db.Exec(`UPDATE decisions SET determination = 'Law' WHERE id = 2`); err != nil {
		t.Fatalf("tampering with row 2 returned unexpected error: %v", err)
	}

	ok, err := Verify(db)
	if err != nil {
		t.Fatalf("Verify returned unexpected error: %v", err)
	}
	if ok {
		t.Error("Verify = true after tampering with a row, want false")
	}
}

func TestVerify_TrueForAnEmptyLedger(t *testing.T) {
	db := openTestStore(t)
	ok, err := Verify(db)
	if err != nil {
		t.Fatalf("Verify returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("Verify = false for an empty ledger, want true")
	}
}
