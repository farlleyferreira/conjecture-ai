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
