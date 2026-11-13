package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureWorkingLayout_CreatesEveryDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	if err := EnsureWorkingLayout(repoRoot); err != nil {
		t.Fatalf("EnsureWorkingLayout returned unexpected error: %v", err)
	}

	for _, dir := range []string{
		filepath.Join(repoRoot, "conjecture", "phenomena"),
		filepath.Join(repoRoot, "conjecture", "materializations"),
		filepath.Join(repoRoot, ".conjecture"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Stat(%s) returned unexpected error: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s exists but is not a directory", dir)
		}
	}
}

func TestEnsureWorkingLayout_AddsGitignoreEntry(t *testing.T) {
	repoRoot := t.TempDir()
	if err := EnsureWorkingLayout(repoRoot); err != nil {
		t.Fatalf("EnsureWorkingLayout returned unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore returned unexpected error: %v", err)
	}
	if string(got) != ".conjecture/\n" {
		t.Errorf(".gitignore = %q, want %q", string(got), ".conjecture/\n")
	}
}

func TestEnsureWorkingLayout_PreservesExistingGitignoreContent(t *testing.T) {
	repoRoot := t.TempDir()
	existing := "node_modules/\n*.log\n"
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatalf("seeding .gitignore returned unexpected error: %v", err)
	}

	if err := EnsureWorkingLayout(repoRoot); err != nil {
		t.Fatalf("EnsureWorkingLayout returned unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore returned unexpected error: %v", err)
	}
	want := existing + ".conjecture/\n"
	if string(got) != want {
		t.Errorf(".gitignore = %q, want %q", string(got), want)
	}
}

func TestEnsureWorkingLayout_IdempotentDoesNotDuplicateGitignoreEntry(t *testing.T) {
	repoRoot := t.TempDir()
	if err := EnsureWorkingLayout(repoRoot); err != nil {
		t.Fatalf("first EnsureWorkingLayout returned unexpected error: %v", err)
	}
	if err := EnsureWorkingLayout(repoRoot); err != nil {
		t.Fatalf("second EnsureWorkingLayout returned unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore returned unexpected error: %v", err)
	}
	if string(got) != ".conjecture/\n" {
		t.Errorf(".gitignore = %q after running twice, want it still just %q", string(got), ".conjecture/\n")
	}
}

func TestEnsureWorkingLayout_IdempotentLeavesExistingDirectoriesAlone(t *testing.T) {
	repoRoot := t.TempDir()
	if err := EnsureWorkingLayout(repoRoot); err != nil {
		t.Fatalf("first EnsureWorkingLayout returned unexpected error: %v", err)
	}

	marker := filepath.Join(repoRoot, "conjecture", "phenomena", "marker.txt")
	if err := os.WriteFile(marker, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("writing marker file returned unexpected error: %v", err)
	}

	if err := EnsureWorkingLayout(repoRoot); err != nil {
		t.Fatalf("second EnsureWorkingLayout returned unexpected error: %v", err)
	}

	if _, err := os.Stat(marker); err != nil {
		t.Errorf("marker file was removed by a second EnsureWorkingLayout call: %v", err)
	}
}
