package userdefinedlaws

import (
	"database/sql"
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

func TestGetStatus_NeverSetIsUnverified(t *testing.T) {
	db := openTestStore(t)
	status, err := GetStatus(db, "demo::some-law")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != Unverified {
		t.Errorf("GetStatus on an unset law = %v, want Unverified", status)
	}
}

func TestSetStatus_ThenGetStatus_RoundTrips(t *testing.T) {
	db := openTestStore(t)
	if err := SetStatus(db, "demo::some-law", Violated); err != nil {
		t.Fatalf("SetStatus returned unexpected error: %v", err)
	}
	status, err := GetStatus(db, "demo::some-law")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != Violated {
		t.Errorf("GetStatus = %v, want Violated", status)
	}
}

func TestGetCounterevidenceCount_NeverRecordedIsZero(t *testing.T) {
	db := openTestStore(t)
	count, err := GetCounterevidenceCount(db, "demo::some-law")
	if err != nil {
		t.Fatalf("GetCounterevidenceCount returned unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("GetCounterevidenceCount on an unrecorded law = %d, want 0", count)
	}
}

func TestRecordCounterevidence_AccumulatesAcrossCalls(t *testing.T) {
	db := openTestStore(t)
	for i, want := range []int{1, 2, 3} {
		got, err := RecordCounterevidence(db, "demo::some-law")
		if err != nil {
			t.Fatalf("RecordCounterevidence call %d returned unexpected error: %v", i, err)
		}
		if got != want {
			t.Errorf("RecordCounterevidence call %d = %d, want %d", i, got, want)
		}
	}

	count, err := GetCounterevidenceCount(db, "demo::some-law")
	if err != nil {
		t.Fatalf("GetCounterevidenceCount returned unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("GetCounterevidenceCount after 3 RecordCounterevidence calls = %d, want 3", count)
	}
}

func TestRecordCounterevidence_DoesNotResetExistingStatus(t *testing.T) {
	db := openTestStore(t)
	if err := SetStatus(db, "demo::some-law", Violated); err != nil {
		t.Fatalf("SetStatus returned unexpected error: %v", err)
	}
	if _, err := RecordCounterevidence(db, "demo::some-law"); err != nil {
		t.Fatalf("RecordCounterevidence returned unexpected error: %v", err)
	}

	status, err := GetStatus(db, "demo::some-law")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != Violated {
		t.Errorf("GetStatus after RecordCounterevidence = %v, want Violated (unchanged)", status)
	}
}

func TestSetStatus_DoesNotResetCounterevidenceCount(t *testing.T) {
	db := openTestStore(t)
	if _, err := RecordCounterevidence(db, "demo::some-law"); err != nil {
		t.Fatalf("RecordCounterevidence returned unexpected error: %v", err)
	}
	if err := SetStatus(db, "demo::some-law", Contested); err != nil {
		t.Fatalf("SetStatus returned unexpected error: %v", err)
	}

	count, err := GetCounterevidenceCount(db, "demo::some-law")
	if err != nil {
		t.Fatalf("GetCounterevidenceCount returned unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("GetCounterevidenceCount after SetStatus = %d, want 1 (unchanged)", count)
	}
}

func TestSetStatus_OverwritesPriorStatus(t *testing.T) {
	db := openTestStore(t)
	if err := SetStatus(db, "demo::some-law", Violated); err != nil {
		t.Fatalf("SetStatus returned unexpected error: %v", err)
	}
	if err := SetStatus(db, "demo::some-law", Conformant); err != nil {
		t.Fatalf("SetStatus returned unexpected error: %v", err)
	}
	status, err := GetStatus(db, "demo::some-law")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != Conformant {
		t.Errorf("GetStatus after overwrite = %v, want Conformant", status)
	}
}
