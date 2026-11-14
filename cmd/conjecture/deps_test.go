package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/languageagnosticcontracts"
)

// requireDependencyAnalysis skips a test that needs the real
// Tree-sitter-backed check, which only a CGO build carries (see
// languageagnosticcontracts.ErrDependencyAnalysisUnavailable).
func requireDependencyAnalysis(t *testing.T) {
	t.Helper()
	if !languageagnosticcontracts.DependencyAnalysisAvailable {
		t.Skip("dependency analysis needs a CGO build")
	}
}

func TestCheckGoForbiddenImports_WithoutCGOIsAPlainError(t *testing.T) {
	if languageagnosticcontracts.DependencyAnalysisAvailable {
		t.Skip("this binary was built with CGO")
	}
	var stdout, stderr bytes.Buffer
	code := checkGoForbiddenImports("demo.go", []byte("package demo\n"), []string{"os"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("checkGoForbiddenImports without CGO: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "CGO") {
		t.Errorf("stderr should say a CGO build is needed, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should stay empty (never a guessed \"no forbidden imports\"), got %q", stdout.String())
	}
}

func TestRunDeps_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDeps(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runDeps with no subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunDeps_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDeps([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runDeps with an unknown subcommand: exit code = %d, want 1", code)
	}
}

func TestRunDepsCheck_MissingForbidIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDeps([]string{"check", "file.go"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("deps check with no --forbid: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunDepsCheck_MissingPathIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDeps([]string{"check", "--forbid", "os"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("deps check with no path: exit code = %d, want 1", code)
	}
}

func TestRunDepsCheck_ForbidMissingValueIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDeps([]string{"check", "file.go", "--forbid"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("deps check with a valueless --forbid: exit code = %d, want 1", code)
	}
}

func TestRunDepsCheck_TwoPositionalArgsIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDeps([]string{"check", "a.go", "b.go", "--forbid", "os"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("deps check with two positional args: exit code = %d, want 1", code)
	}
}

func TestRunDepsCheck_MissingFileIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDeps([]string{"check", filepath.Join(t.TempDir(), "nope.go"), "--forbid", "os"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("deps check with a missing file: exit code = %d, want 1", code)
	}
}

func TestCheckGoForbiddenImports_ReportsViolation(t *testing.T) {
	requireDependencyAnalysis(t)
	var stdout, stderr bytes.Buffer
	source := []byte("package demo\n\nimport \"github.com/example/forbidden\"\n")
	code := checkGoForbiddenImports("demo.go", source, []string{"github.com/example/forbidden"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("checkGoForbiddenImports with a violation: exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "forbidden import") {
		t.Errorf("stdout should report the forbidden import, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "demo.go:3") {
		t.Errorf("stdout should report the file and line, got %q", stdout.String())
	}
}

func TestCheckGoForbiddenImports_NoViolationsReportsPlainly(t *testing.T) {
	requireDependencyAnalysis(t)
	var stdout, stderr bytes.Buffer
	source := []byte("package demo\n\nimport \"fmt\"\n")
	code := checkGoForbiddenImports("demo.go", source, []string{"os"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("checkGoForbiddenImports with no violations: exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "no forbidden imports") {
		t.Errorf("stdout should report no forbidden imports, got %q", stdout.String())
	}
}

func TestRunDepsCheck_EndToEnd(t *testing.T) {
	requireDependencyAnalysis(t)
	path := filepath.Join(t.TempDir(), "demo.go")
	if err := os.WriteFile(path, []byte("package demo\n\nimport \"github.com/example/forbidden\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := runDeps([]string{"check", path, "--forbid", "github.com/example/forbidden"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("runDeps check: exit code = %d, want 1; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "github.com/example/forbidden") {
		t.Errorf("stdout should mention the forbidden import, got %q", stdout.String())
	}
}
