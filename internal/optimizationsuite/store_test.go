package optimizationsuite

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

func openStoreTestStore(t *testing.T) *sql.DB {
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

func TestGetCurrent_NeverSetIsNotFound(t *testing.T) {
	db := openStoreTestStore(t)

	_, found, err := GetCurrent(db, "queue_aging_rate")
	if err != nil {
		t.Fatalf("GetCurrent returned unexpected error: %v", err)
	}
	if found {
		t.Error("found = true for a field never calibrated, want false")
	}
}

func TestSetCurrent_ThenGetCurrentRoundTrips(t *testing.T) {
	db := openStoreTestStore(t)

	if err := SetCurrent(db, "queue_aging_rate", 2.5); err != nil {
		t.Fatalf("SetCurrent returned unexpected error: %v", err)
	}

	value, found, err := GetCurrent(db, "queue_aging_rate")
	if err != nil {
		t.Fatalf("GetCurrent returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if value != 2.5 {
		t.Errorf("value = %v, want 2.5", value)
	}
}

func TestSetCurrent_OverwritesPreviousValue(t *testing.T) {
	db := openStoreTestStore(t)

	if err := SetCurrent(db, "queue_aging_rate", 2.5); err != nil {
		t.Fatalf("SetCurrent returned unexpected error: %v", err)
	}
	if err := SetCurrent(db, "queue_aging_rate", 4.0); err != nil {
		t.Fatalf("SetCurrent returned unexpected error: %v", err)
	}

	value, found, err := GetCurrent(db, "queue_aging_rate")
	if err != nil {
		t.Fatalf("GetCurrent returned unexpected error: %v", err)
	}
	if !found || value != 4.0 {
		t.Errorf("value = %v, found = %v, want 4.0, true", value, found)
	}
}

func TestGetCurrent_DistinctFieldsIndependent(t *testing.T) {
	db := openStoreTestStore(t)

	if err := SetCurrent(db, "queue_aging_rate", 2.5); err != nil {
		t.Fatalf("SetCurrent returned unexpected error: %v", err)
	}

	_, found, err := GetCurrent(db, "law_contested_threshold")
	if err != nil {
		t.Fatalf("GetCurrent returned unexpected error: %v", err)
	}
	if found {
		t.Error("found = true for an unrelated field, want false")
	}
}
