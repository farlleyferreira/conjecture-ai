package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_CreatesLayoutAtGivenDir(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := runInit([]string{"--dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runInit returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "initialized conjecture layout") {
		t.Errorf("stdout = %q, want it to report the layout was initialized", stdout.String())
	}

	for _, rel := range []string{filepath.Join("conjecture", "phenomena"), filepath.Join("conjecture", "materializations"), ".conjecture"} {
		if info, err := os.Stat(filepath.Join(dir, rel)); err != nil || !info.IsDir() {
			t.Errorf("%s was not created as a directory", rel)
		}
	}
}

func TestRunInit_DefaultsToCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned unexpected error: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	var stdout, stderr bytes.Buffer
	code := runInit(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runInit returned exit code %d, stderr: %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "conjecture", "phenomena")); err != nil {
		t.Errorf("runInit with no --dir did not initialize the current directory: %v", err)
	}
}

func TestRunInit_ScaffoldsOrchestrationSkill(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := runInit([]string{"--dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runInit returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "orchestration skill") {
		t.Errorf("stdout = %q, want it to report the skill was written", stdout.String())
	}

	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(skillRelPath)))
	if err != nil {
		t.Fatalf("reading scaffolded skill returned unexpected error: %v", err)
	}
	if !strings.HasPrefix(string(content), "---\nname: conjecture-workflow\n") {
		t.Errorf("scaffolded skill does not start with the expected frontmatter, got:\n%s", content)
	}
}

func TestRunInit_DoesNotOverwriteExistingSkillFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(skillRelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	const customized = "a user's own customized skill content"
	if err := os.WriteFile(path, []byte(customized), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := runInit([]string{"--dir", dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("runInit returned exit code %d, stderr: %s", code, stderr.String())
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading skill file returned unexpected error: %v", err)
	}
	if string(content) != customized {
		t.Errorf("skill file content = %q, want the user's own customized content left untouched", content)
	}
}

func TestRunInit_UnknownFlagIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runInit([]string{"--bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runInit with an unknown flag: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunInit_MissingDirValueIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runInit([]string{"--dir"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runInit with --dir and no value: exit code = %d, want 1", code)
	}
}
