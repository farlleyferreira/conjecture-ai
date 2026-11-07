package knowledgebases

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

func openSearchTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if err := EnsureSearchSchema(db); err != nil {
		t.Fatalf("EnsureSearchSchema returned unexpected error: %v", err)
	}
	return db
}

func TestIndexBase_SingleFileIsSearchable(t *testing.T) {
	db := openSearchTestStore(t)
	path := filepath.Join(t.TempDir(), "lgpd.txt")
	if err := os.WriteFile(path, []byte("o titular dos dados pode solicitar a exclusao"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	b, err := Add(db, Base{Path: path, Authority: Reference, Scope: "project"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	if err := IndexBase(db, b); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	results, err := Search(db, "exclusao")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].BaseID != b.ID || results[0].FilePath != path {
		t.Errorf("results[0] = %+v, want BaseID %d and FilePath %q", results[0], b.ID, path)
	}
}

func TestIndexBase_DirectoryIndexesEveryFile(t *testing.T) {
	db := openSearchTestStore(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("primeiro arquivo sobre transporte"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("segundo arquivo sobre armazenamento"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	b, err := Add(db, Base{Path: dir, Authority: Reference, Scope: "project"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	if err := IndexBase(db, b); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	results, err := Search(db, "transporte")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].FilePath != filepath.Join(dir, "a.txt") {
		t.Errorf("results = %+v, want exactly a.txt", results)
	}
}

func TestIndexBase_ReindexingClearsStaleEntries(t *testing.T) {
	db := openSearchTestStore(t)
	path := filepath.Join(t.TempDir(), "doc.txt")
	if err := os.WriteFile(path, []byte("conteudo original"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	b, err := Add(db, Base{Path: path, Authority: Reference, Scope: "project"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := IndexBase(db, b); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	if err := os.WriteFile(path, []byte("conteudo atualizado"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := IndexBase(db, b); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	results, err := Search(db, "original")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search(\"original\") after reindexing = %+v, want no matches (stale content should be cleared)", results)
	}

	results, err = Search(db, "atualizado")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search(\"atualizado\") = %+v, want 1 match", results)
	}
}

func TestSearch_NoMatchesReturnsEmpty(t *testing.T) {
	db := openSearchTestStore(t)
	results, err := Search(db, "nothing-indexed-matches-this")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}
