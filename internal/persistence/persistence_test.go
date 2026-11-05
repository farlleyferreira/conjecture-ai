package persistence

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoID_Deterministic(t *testing.T) {
	id1, err := RepoID("/some/repo")
	if err != nil {
		t.Fatalf("RepoID returned unexpected error: %v", err)
	}
	id2, err := RepoID("/some/repo")
	if err != nil {
		t.Fatalf("RepoID returned unexpected error: %v", err)
	}
	if id1 != id2 {
		t.Errorf("RepoID(%q) = %q then %q, want the same value both times", "/some/repo", id1, id2)
	}
	if len(id1) != 16 {
		t.Errorf("len(RepoID(...)) = %d, want 16", len(id1))
	}
}

func TestRepoID_DistinctPathsDiffer(t *testing.T) {
	id1, err := RepoID("/some/repo")
	if err != nil {
		t.Fatalf("RepoID returned unexpected error: %v", err)
	}
	id2, err := RepoID("/some/other-repo")
	if err != nil {
		t.Fatalf("RepoID returned unexpected error: %v", err)
	}
	if id1 == id2 {
		t.Errorf("RepoID of two distinct paths produced the same id %q", id1)
	}
}

func TestLocalDataDir_Linux(t *testing.T) {
	dir, err := localDataDir("/some/repo", "linux")
	if err != nil {
		t.Fatalf("localDataDir returned unexpected error: %v", err)
	}
	if !strings.HasPrefix(dir, "/var/lib/conjecture/") {
		t.Errorf("localDataDir(..., \"linux\") = %q, want a prefix of /var/lib/conjecture/", dir)
	}
}

func TestLocalDataDir_Darwin(t *testing.T) {
	dir, err := localDataDir("/some/repo", "darwin")
	if err != nil {
		t.Fatalf("localDataDir returned unexpected error: %v", err)
	}
	if !strings.HasPrefix(dir, "/Library/Application Support/conjecture/") {
		t.Errorf("localDataDir(..., \"darwin\") = %q, want a prefix of /Library/Application Support/conjecture/", dir)
	}
}

func TestLocalDataDir_Windows(t *testing.T) {
	t.Setenv("ProgramData", `C:\ProgramData`)
	dir, err := localDataDir(`C:\some\repo`, "windows")
	if err != nil {
		t.Fatalf("localDataDir returned unexpected error: %v", err)
	}
	want := `C:\ProgramData\conjecture\`
	if !strings.HasPrefix(dir, want) {
		t.Errorf("localDataDir(..., \"windows\") = %q, want a prefix of %q", dir, want)
	}
}

func TestLocalDataDir_UnsupportedOS(t *testing.T) {
	_, err := localDataDir("/some/repo", "plan9")
	if err == nil {
		t.Fatal("localDataDir with an unsupported OS = nil error, want an error")
	}
}

func TestOpenStoreAt(t *testing.T) {
	dir := t.TempDir()

	db, err := OpenStoreAt(filepath.Join(dir, "nested", "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping returned unexpected error: %v", err)
	}

	// Foreign keys must actually be enforced: a fresh connection to the
	// same database should still reject a dangling reference.
	if _, err := db.Exec(`CREATE TABLE parent (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("CREATE TABLE parent: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id))`); err != nil {
		t.Fatalf("CREATE TABLE child: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO child (id, parent_id) VALUES (1, 999)`); err == nil {
		t.Error("inserting a child row with a dangling parent_id succeeded, want foreign-key enforcement to reject it")
	}
}

func TestOpenStoreAt_ReopensExistingDatabase(t *testing.T) {
	dir := t.TempDir()

	db1, err := OpenStoreAt(dir)
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	if _, err := db1.Exec(`CREATE TABLE t (v TEXT)`); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := db1.Exec(`INSERT INTO t (v) VALUES ('hello')`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	db1.Close()

	db2, err := OpenStoreAt(dir)
	if err != nil {
		t.Fatalf("OpenStoreAt (reopen) returned unexpected error: %v", err)
	}
	defer db2.Close()

	var got string
	if err := db2.QueryRow(`SELECT v FROM t`).Scan(&got); err != nil {
		t.Fatalf("querying the reopened store: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestChainHash_Deterministic(t *testing.T) {
	h1 := ChainHash([]byte("previous"), []byte("data"))
	h2 := ChainHash([]byte("previous"), []byte("data"))
	if !bytes.Equal(h1, h2) {
		t.Error("ChainHash gave different results for the same inputs")
	}
}

func TestChainHash_DependsOnPrevious(t *testing.T) {
	h1 := ChainHash([]byte("previous-a"), []byte("data"))
	h2 := ChainHash([]byte("previous-b"), []byte("data"))
	if bytes.Equal(h1, h2) {
		t.Error("ChainHash ignored the previous hash: two different chains collided")
	}
}

func TestChainHash_GenesisHasNoPrevious(t *testing.T) {
	h := ChainHash(nil, []byte("first record"))
	if len(h) == 0 {
		t.Error("ChainHash with no previous record returned an empty hash")
	}
}
