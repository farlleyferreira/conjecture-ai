package contextcomposition

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

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

func TestParseVersionRef_BareNameHasNoSeq(t *testing.T) {
	ref, err := ParseVersionRef("checkout")
	if err != nil {
		t.Fatalf("ParseVersionRef returned unexpected error: %v", err)
	}
	if ref.Name != "checkout" || ref.Seq != 0 {
		t.Errorf("ParseVersionRef(\"checkout\") = %+v, want {Name: \"checkout\", Seq: 0}", ref)
	}
}

func TestParseVersionRef_NameWithVersion(t *testing.T) {
	ref, err := ParseVersionRef("checkout@v4")
	if err != nil {
		t.Fatalf("ParseVersionRef returned unexpected error: %v", err)
	}
	if ref.Name != "checkout" || ref.Seq != 4 {
		t.Errorf("ParseVersionRef(\"checkout@v4\") = %+v, want {Name: \"checkout\", Seq: 4}", ref)
	}
}

func TestParseVersionRef_MalformedTagIsError(t *testing.T) {
	for _, s := range []string{"checkout@4", "checkout@latest", "checkout@v0", "checkout@v-1", "@v4"} {
		if _, err := ParseVersionRef(s); err == nil {
			t.Errorf("ParseVersionRef(%q) = nil error, want one", s)
		}
	}
}

func TestCreateContext_ThenGetContext(t *testing.T) {
	db := openTestStore(t)

	created, err := CreateContext(db, "checkout", "")
	if err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if created.State != Active {
		t.Errorf("CreateContext's result state = %v, want Active", created.State)
	}

	got, err := GetContext(db, "checkout")
	if err != nil {
		t.Fatalf("GetContext returned unexpected error: %v", err)
	}
	if got.ID != "checkout" || got.State != Active {
		t.Errorf("GetContext = %+v, want an Active \"checkout\"", got)
	}
}

func TestCreateContext_DuplicateNameIsError(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("first CreateContext returned unexpected error: %v", err)
	}
	if _, err := CreateContext(db, "checkout", ""); err == nil {
		t.Error("second CreateContext with the same name = nil error, want one")
	}
}

func TestGetContext_UnknownNameIsErrContextNotFound(t *testing.T) {
	db := openTestStore(t)
	_, err := GetContext(db, "nonexistent")
	if !errors.Is(err, ErrContextNotFound) {
		t.Errorf("GetContext(\"nonexistent\") error = %v, want ErrContextNotFound", err)
	}
}

func TestListContexts_ReturnsEveryRegisteredContext(t *testing.T) {
	db := openTestStore(t)
	for _, name := range []string{"checkout", "cart"} {
		if _, err := CreateContext(db, name, ""); err != nil {
			t.Fatalf("CreateContext(%q) returned unexpected error: %v", name, err)
		}
	}

	got, err := ListContexts(db)
	if err != nil {
		t.Fatalf("ListContexts returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListContexts returned %d contexts, want 2", len(got))
	}
	if got[0].ID != "checkout" || got[1].ID != "cart" {
		t.Errorf("ListContexts = %+v, want [checkout, cart] in creation order", got)
	}
}

func TestListContextsFiltered_ByArea(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", "web"); err != nil {
		t.Fatalf("CreateContext(checkout) returned unexpected error: %v", err)
	}
	if _, err := CreateContext(db, "cart", "mobile"); err != nil {
		t.Fatalf("CreateContext(cart) returned unexpected error: %v", err)
	}

	got, err := ListContextsFiltered(db, ContextFilter{Area: "web"})
	if err != nil {
		t.Fatalf("ListContextsFiltered returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "checkout" {
		t.Errorf("ListContextsFiltered(Area: \"web\") = %+v, want only [checkout]", got)
	}
}

func TestListContextsFiltered_ByDateRange(t *testing.T) {
	db := openTestStore(t)
	insertContextAt(t, db, "old", "2020-01-01T00:00:00Z")
	insertContextAt(t, db, "middle", "2020-06-01T00:00:00Z")
	insertContextAt(t, db, "new", "2020-12-31T00:00:00Z")

	since, err := time.Parse(time.RFC3339, "2020-03-01T00:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse returned unexpected error: %v", err)
	}
	until, err := time.Parse(time.RFC3339, "2020-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse returned unexpected error: %v", err)
	}

	got, err := ListContextsFiltered(db, ContextFilter{Since: since, Until: until})
	if err != nil {
		t.Fatalf("ListContextsFiltered returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "middle" {
		t.Errorf("ListContextsFiltered(2020-03-01..2020-09-01) = %+v, want only [middle]", got)
	}
}

func TestListContextsFiltered_CombinedAreaAndDate(t *testing.T) {
	db := openTestStore(t)
	insertContextAt(t, db, "old-web", "2020-01-01T00:00:00Z")
	insertContextAt(t, db, "new-web", "2020-12-01T00:00:00Z")
	if _, err := db.Exec(`UPDATE contexts SET area = ? WHERE name = ?`, "web", "old-web"); err != nil {
		t.Fatalf("setting area on old-web returned unexpected error: %v", err)
	}
	if _, err := db.Exec(`UPDATE contexts SET area = ? WHERE name = ?`, "web", "new-web"); err != nil {
		t.Fatalf("setting area on new-web returned unexpected error: %v", err)
	}

	since, err := time.Parse(time.RFC3339, "2020-06-01T00:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse returned unexpected error: %v", err)
	}

	got, err := ListContextsFiltered(db, ContextFilter{Area: "web", Since: since})
	if err != nil {
		t.Fatalf("ListContextsFiltered returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "new-web" {
		t.Errorf("ListContextsFiltered(Area: \"web\", Since: 2020-06-01) = %+v, want only [new-web]", got)
	}
}

// insertContextAt inserts an Active context named name with the given
// RFC3339 createdAt, bypassing CreateContext's own time.Now() so date-
// range filtering tests can control CreatedAt directly.
func insertContextAt(t *testing.T, db *sql.DB, name, createdAt string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO contexts (name, state, superseded_by, area, created_at) VALUES (?, ?, ?, ?, ?)`,
		name, Active.String(), "", "", createdAt,
	); err != nil {
		t.Fatalf("inserting context %q returned unexpected error: %v", name, err)
	}
}

func TestAddVersion_SeqStartsAtOneAndIncrements(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	v1, err := AddVersion(db, "checkout", "first content")
	if err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if v1.Seq != 1 || v1.Tag() != "v1" {
		t.Errorf("first AddVersion = %+v, want Seq 1 (\"v1\")", v1)
	}

	v2, err := AddVersion(db, "checkout", "second content")
	if err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if v2.Seq != 2 || v2.Tag() != "v2" {
		t.Errorf("second AddVersion = %+v, want Seq 2 (\"v2\")", v2)
	}
}

func TestAddVersion_SeqNumberingIsIndependentPerContext(t *testing.T) {
	db := openTestStore(t)
	for _, name := range []string{"checkout", "cart"} {
		if _, err := CreateContext(db, name, ""); err != nil {
			t.Fatalf("CreateContext(%q) returned unexpected error: %v", name, err)
		}
	}

	if _, err := AddVersion(db, "checkout", "a"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	v, err := AddVersion(db, "cart", "b")
	if err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if v.Seq != 1 {
		t.Errorf("cart's first version Seq = %d, want 1 (independent of checkout's own numbering)", v.Seq)
	}
}

func TestResume_ReturnsLatestVersion(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "checkout", "old"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "checkout", "new"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	got, err := Resume(db, "checkout")
	if err != nil {
		t.Fatalf("Resume returned unexpected error: %v", err)
	}
	if got.Content != "new" || got.Seq != 2 {
		t.Errorf("Resume = %+v, want the latest version (seq 2, content \"new\")", got)
	}
}

func TestResume_NoVersionsIsErrVersionNotFound(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	_, err := Resume(db, "checkout")
	if !errors.Is(err, ErrVersionNotFound) {
		t.Errorf("Resume on a context with no versions, error = %v, want ErrVersionNotFound", err)
	}
}

func TestOpen_ReturnsTheExactVersionAsked(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "checkout", "v1 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "checkout", "v2 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	got, err := Open(db, VersionRef{Name: "checkout", Seq: 1})
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	if got.Content != "v1 content" {
		t.Errorf("Open(checkout@v1) content = %q, want %q", got.Content, "v1 content")
	}
}

func TestOpen_WithNoVersionIsError(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "checkout", "content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	_, err := Open(db, VersionRef{Name: "checkout"})
	if err == nil {
		t.Error("Open with no explicit version = nil error, want one (open always needs an explicit version)")
	}
}

func TestOpen_UnknownVersionIsErrVersionNotFound(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	_, err := Open(db, VersionRef{Name: "checkout", Seq: 99})
	if !errors.Is(err, ErrVersionNotFound) {
		t.Errorf("Open(checkout@v99) error = %v, want ErrVersionNotFound", err)
	}
}

func TestArchiveContext_ThenRestore(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	archived, err := ArchiveContext(db, "checkout", "new-checkout")
	if err != nil {
		t.Fatalf("ArchiveContext returned unexpected error: %v", err)
	}
	if archived.State != Archived || archived.SupersededBy != "new-checkout" {
		t.Errorf("ArchiveContext result = %+v, want Archived, SupersededBy \"new-checkout\"", archived)
	}

	// Archiving must persist, not just return a transformed value.
	reloaded, err := GetContext(db, "checkout")
	if err != nil {
		t.Fatalf("GetContext returned unexpected error: %v", err)
	}
	if reloaded.State != Archived {
		t.Errorf("GetContext after ArchiveContext = %+v, want State Archived", reloaded)
	}

	restored, err := Restore(db, "checkout")
	if err != nil {
		t.Fatalf("Restore returned unexpected error: %v", err)
	}
	if restored.State != Active || restored.SupersededBy != "" {
		t.Errorf("Restore result = %+v, want Active with no SupersededBy", restored)
	}

	reloadedAgain, err := GetContext(db, "checkout")
	if err != nil {
		t.Fatalf("GetContext returned unexpected error: %v", err)
	}
	if reloadedAgain.State != Active {
		t.Errorf("GetContext after Restore = %+v, want State Active", reloadedAgain)
	}
}

func TestRestore_NotArchivedIsError(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	_, err := Restore(db, "checkout")
	if err == nil {
		t.Error("Restore on an Active context = nil error, want one (nothing to restore)")
	}
}
