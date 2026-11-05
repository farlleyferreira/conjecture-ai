package knowledgebases

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

func TestAdd_NormAuthorityComplete(t *testing.T) {
	db := openTestStore(t)

	stored, err := Add(db, Base{
		Path: "./normas/lgpd.pdf", Authority: Norm, Scope: "project",
		ContentHash: []byte{0xde, 0xad}, LastReviewDate: "2026-01-01",
	})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if stored.ID == 0 {
		t.Error("stored.ID = 0, want a non-zero assigned id")
	}
	if stored.Incomplete {
		t.Error("stored.Incomplete = true, want false (both required fields were given)")
	}
}

func TestAdd_NormAuthorityMissingReviewDate(t *testing.T) {
	db := openTestStore(t)

	stored, err := Add(db, Base{
		Path: "./normas/lgpd.pdf", Authority: Norm, Scope: "project",
		ContentHash: []byte{0xde, 0xad},
	})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if !stored.Incomplete {
		t.Error("stored.Incomplete = false, want true (no last review date given)")
	}
}

func TestAdd_NormAuthorityMissingContentHash(t *testing.T) {
	db := openTestStore(t)

	stored, err := Add(db, Base{
		Path: "./normas/lgpd.pdf", Authority: Norm, Scope: "project",
		LastReviewDate: "2026-01-01",
	})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if !stored.Incomplete {
		t.Error("stored.Incomplete = false, want true (no content hash given)")
	}
}

func TestAdd_NonNormAuthorityNeverIncomplete(t *testing.T) {
	db := openTestStore(t)

	stored, err := Add(db, Base{Path: "./wiki/glossario.md", Authority: Reference, Scope: "project"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if stored.Incomplete {
		t.Error("stored.Incomplete = true, want false (a reference-authority base needs neither field)")
	}
}

func TestList_OrderAndRoundTrip(t *testing.T) {
	db := openTestStore(t)

	if _, err := Add(db, Base{Path: "a", Authority: Norm, Scope: "project", ContentHash: []byte{1}, LastReviewDate: "2026-01-01"}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if _, err := Add(db, Base{Path: "b", Authority: Reference, Scope: "area:shipping"}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	bases, err := List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(bases) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(bases))
	}
	if bases[0].Path != "a" || bases[0].Authority != Norm || bases[0].Scope != "project" {
		t.Errorf("bases[0] = %+v", bases[0])
	}
	if bases[1].Path != "b" || bases[1].Authority != Reference || bases[1].Scope != "area:shipping" {
		t.Errorf("bases[1] = %+v", bases[1])
	}
}

func TestInScope_Project(t *testing.T) {
	b := Base{Scope: "project"}
	if !InScope(b, "shipping") {
		t.Error("InScope with scope \"project\" = false, want true (applies everywhere)")
	}
	if !InScope(b, "billing") {
		t.Error("InScope with scope \"project\" = false, want true (applies everywhere)")
	}
}

func TestInScope_Area(t *testing.T) {
	b := Base{Scope: "area:shipping"}
	if !InScope(b, "shipping") {
		t.Error("InScope(area:shipping, \"shipping\") = false, want true")
	}
	if InScope(b, "billing") {
		t.Error("InScope(area:shipping, \"billing\") = true, want false")
	}
}
