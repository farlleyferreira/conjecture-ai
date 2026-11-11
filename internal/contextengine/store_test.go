package contextengine

import (
	"database/sql"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

func openTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("OpenStoreAt failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}
	return db
}

func TestGetAxes_UnknownNodeIsNotFound(t *testing.T) {
	db := openTestStore(t)
	_, found, err := GetAxes(db, "nope")
	if err != nil {
		t.Fatalf("GetAxes returned unexpected error: %v", err)
	}
	if found {
		t.Error("GetAxes found = true for a node nothing has ever set, want false")
	}
}

func TestSetWeight_ThenGetAxesReportsItAlone(t *testing.T) {
	db := openTestStore(t)
	if err := SetWeight(db, "n1", evidenceledger.High); err != nil {
		t.Fatalf("SetWeight failed: %v", err)
	}

	axes, found, err := GetAxes(db, "n1")
	if err != nil {
		t.Fatalf("GetAxes returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("GetAxes found = false, want true")
	}
	if axes.Weight == nil || *axes.Weight != evidenceledger.High {
		t.Errorf("Weight = %v, want High", axes.Weight)
	}
	if axes.Relevance != nil {
		t.Errorf("Relevance = %v, want nil (never set)", axes.Relevance)
	}
	if axes.Confidence != nil {
		t.Errorf("Confidence = %v, want nil (never set)", axes.Confidence)
	}
}

func TestSetAllThreeAxes_EachReportedIndependently(t *testing.T) {
	db := openTestStore(t)
	if err := SetWeight(db, "n1", evidenceledger.Critical); err != nil {
		t.Fatalf("SetWeight failed: %v", err)
	}
	if err := SetRelevance(db, "n1", evidenceledger.Low); err != nil {
		t.Fatalf("SetRelevance failed: %v", err)
	}
	if err := SetConfidence(db, "n1", evidenceledger.Medium); err != nil {
		t.Fatalf("SetConfidence failed: %v", err)
	}

	axes, found, err := GetAxes(db, "n1")
	if err != nil {
		t.Fatalf("GetAxes returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("GetAxes found = false, want true")
	}
	if *axes.Weight != evidenceledger.Critical {
		t.Errorf("Weight = %v, want Critical", *axes.Weight)
	}
	if *axes.Relevance != evidenceledger.Low {
		t.Errorf("Relevance = %v, want Low", *axes.Relevance)
	}
	if *axes.Confidence != evidenceledger.Medium {
		t.Errorf("Confidence = %v, want Medium", *axes.Confidence)
	}
}

func TestSetWeight_UpdatingLeavesOtherAxesUntouched(t *testing.T) {
	db := openTestStore(t)
	if err := SetRelevance(db, "n1", evidenceledger.Low); err != nil {
		t.Fatalf("SetRelevance failed: %v", err)
	}
	if err := SetWeight(db, "n1", evidenceledger.High); err != nil {
		t.Fatalf("SetWeight failed: %v", err)
	}
	if err := SetWeight(db, "n1", evidenceledger.Critical); err != nil {
		t.Fatalf("second SetWeight failed: %v", err)
	}

	axes, _, err := GetAxes(db, "n1")
	if err != nil {
		t.Fatalf("GetAxes returned unexpected error: %v", err)
	}
	if *axes.Weight != evidenceledger.Critical {
		t.Errorf("Weight = %v, want Critical (the second Set should win)", *axes.Weight)
	}
	if axes.Relevance == nil || *axes.Relevance != evidenceledger.Low {
		t.Errorf("Relevance = %v, want Low (untouched by SetWeight)", axes.Relevance)
	}
}
