package knowledgebases

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeContentHash_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lgpd.txt")
	if err := os.WriteFile(path, []byte("art. 1 ..."), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	h1, err := ComputeContentHash(path)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}
	h2, err := ComputeContentHash(path)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}
	if !bytes.Equal(h1, h2) {
		t.Error("ComputeContentHash gave different results for the same unchanged file")
	}
}

func TestComputeContentHash_FileContentChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lgpd.txt")

	if err := os.WriteFile(path, []byte("version 1"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	h1, err := ComputeContentHash(path)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}

	if err := os.WriteFile(path, []byte("version 2"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	h2, err := ComputeContentHash(path)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}

	if bytes.Equal(h1, h2) {
		t.Error("ComputeContentHash did not change after the file's content changed")
	}
}

func TestComputeContentHash_Directory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("A"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("B"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	h1, err := ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}

	// Changing a nested file must change the directory's overall hash.
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("B-changed"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	h2, err := ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}
	if bytes.Equal(h1, h2) {
		t.Error("ComputeContentHash did not change after a nested file's content changed")
	}
}

func TestComputeContentHash_PathMatters(t *testing.T) {
	// Two trees with the same single file content, but at different
	// relative paths, must not hash the same: a rename is a real
	// change to what the base registered, not a no-op.
	dirA := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirA, "one.txt"), []byte("same content"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	dirB := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirB, "two.txt"), []byte("same content"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	hA, err := ComputeContentHash(dirA)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}
	hB, err := ComputeContentHash(dirB)
	if err != nil {
		t.Fatalf("ComputeContentHash returned unexpected error: %v", err)
	}
	if bytes.Equal(hA, hB) {
		t.Error("ComputeContentHash was the same for two trees whose file had the same content but a different name")
	}
}

func TestComputeContentHash_MissingPath(t *testing.T) {
	_, err := ComputeContentHash(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("ComputeContentHash with a missing path = nil error, want an error")
	}
}
