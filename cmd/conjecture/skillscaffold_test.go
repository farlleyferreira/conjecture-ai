package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndentNonEmptyLines_PrefixesOnlyNonEmptyLines(t *testing.T) {
	got := indentNonEmptyLines("a\n\nb\nc", "  ")
	want := "  a\n\n  b\n  c"
	if got != want {
		t.Errorf("indentNonEmptyLines = %q, want %q", got, want)
	}
}

func TestIndentNonEmptyLines_TrimsTrailingNewlines(t *testing.T) {
	got := indentNonEmptyLines("a\nb\n\n", "> ")
	want := "> a\n> b"
	if got != want {
		t.Errorf("indentNonEmptyLines = %q, want %q", got, want)
	}
}

func TestRenderWorkflowSkill_SubstitutesGrammarPlaceholders(t *testing.T) {
	content := string(renderWorkflowSkill())

	if strings.Contains(content, "{{CJTR_GRAMMAR_YAML}}") || strings.Contains(content, "{{CJTR_GRAMMAR_NOTES}}") {
		t.Error("renderWorkflowSkill left an unsubstituted placeholder")
	}
	// The shared grammar text, re-indented by renderWorkflowSkill,
	// must still appear verbatim (modulo indentation) - the whole
	// point of sharing it with draftSystemPrompt (draft.go) instead of
	// each copy drifting independently.
	for _, line := range strings.Split(strings.TrimRight(cjtrGrammarYAML, "\n"), "\n") {
		if line == "" {
			continue
		}
		if !strings.Contains(content, "   "+line) {
			t.Errorf("rendered skill is missing indented grammar line %q", line)
		}
	}
	if !strings.Contains(content, `The only recognized comparators are: is / equals`) {
		t.Error("rendered skill is missing the shared comparator notes")
	}
}

func TestEnsureWorkflowSkill_CreatesWhenMissing(t *testing.T) {
	repoRoot := t.TempDir()
	action, err := ensureWorkflowSkill(repoRoot)
	if err != nil {
		t.Fatalf("ensureWorkflowSkill returned unexpected error: %v", err)
	}
	if action != SkillCreated {
		t.Errorf("action = %v, want SkillCreated", action)
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(skillRelPath)))
	if err != nil {
		t.Fatalf("reading scaffolded skill returned unexpected error: %v", err)
	}
	if string(got) != string(renderWorkflowSkill()) {
		t.Error("scaffolded skill content does not match renderWorkflowSkill's own output")
	}
}

func TestEnsureWorkflowSkill_UpToDateOnSecondRun(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := ensureWorkflowSkill(repoRoot); err != nil {
		t.Fatalf("first ensureWorkflowSkill returned unexpected error: %v", err)
	}

	action, err := ensureWorkflowSkill(repoRoot)
	if err != nil {
		t.Fatalf("second ensureWorkflowSkill returned unexpected error: %v", err)
	}
	if action != SkillUpToDate {
		t.Errorf("action = %v, want SkillUpToDate", action)
	}
}

func TestEnsureWorkflowSkill_UpdatesWhenUnchangedSincePriorWrite(t *testing.T) {
	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, filepath.FromSlash(skillRelPath))
	checksumPath := filepath.Join(repoRoot, filepath.FromSlash(skillChecksumRelPath))

	// Simulates what an older "conjecture init" would have left behind:
	// a file this function itself wrote (so its own recorded checksum
	// matches it), whose content happens to differ from what the
	// current binary would now render - exactly the situation a real
	// template update produces.
	oldContent := []byte("an older bundled version of the skill\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	if err := os.WriteFile(path, oldContent, 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}
	if err := writeSkillChecksum(checksumPath, sha256Hex(oldContent)); err != nil {
		t.Fatalf("writeSkillChecksum returned unexpected error: %v", err)
	}

	action, err := ensureWorkflowSkill(repoRoot)
	if err != nil {
		t.Fatalf("ensureWorkflowSkill returned unexpected error: %v", err)
	}
	if action != SkillUpdated {
		t.Errorf("action = %v, want SkillUpdated", action)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading updated skill returned unexpected error: %v", err)
	}
	if string(got) != string(renderWorkflowSkill()) {
		t.Error("updated skill content does not match renderWorkflowSkill's own output")
	}
}

func TestEnsureWorkflowSkill_LeavesCustomizedFileAlone(t *testing.T) {
	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, filepath.FromSlash(skillRelPath))

	// No checksum recorded at all - this function cannot tell whether
	// a human wrote this file, edited a prior scaffold, or neither;
	// it must never guess by overwriting it.
	customized := []byte("the user's own customized skill content\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	if err := os.WriteFile(path, customized, 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	action, err := ensureWorkflowSkill(repoRoot)
	if err != nil {
		t.Fatalf("ensureWorkflowSkill returned unexpected error: %v", err)
	}
	if action != SkillLeftCustomized {
		t.Errorf("action = %v, want SkillLeftCustomized", action)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading skill returned unexpected error: %v", err)
	}
	if string(got) != string(customized) {
		t.Error("customized skill content was overwritten, want it left untouched")
	}
}

func TestEnsureWorkflowSkill_StaleChecksumLeavesFileAlone(t *testing.T) {
	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, filepath.FromSlash(skillRelPath))
	checksumPath := filepath.Join(repoRoot, filepath.FromSlash(skillChecksumRelPath))

	original := []byte("what conjecture originally wrote\n")
	customized := []byte("the user edited this after conjecture wrote it\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	// The recorded checksum matches the *original* content, not the
	// customized content now on disk - exactly what a human edit after
	// a real "conjecture init" leaves behind.
	if err := writeSkillChecksum(checksumPath, sha256Hex(original)); err != nil {
		t.Fatalf("writeSkillChecksum returned unexpected error: %v", err)
	}
	if err := os.WriteFile(path, customized, 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	action, err := ensureWorkflowSkill(repoRoot)
	if err != nil {
		t.Fatalf("ensureWorkflowSkill returned unexpected error: %v", err)
	}
	if action != SkillLeftCustomized {
		t.Errorf("action = %v, want SkillLeftCustomized", action)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading skill returned unexpected error: %v", err)
	}
	if string(got) != string(customized) {
		t.Error("customized skill content was overwritten, want it left untouched")
	}
}

func TestDraftSystemPrompt_ContainsSharedGrammar(t *testing.T) {
	for _, want := range []string{
		"conjecture: 1.0.0",
		"phenomenon: <short name>",
		`The only recognized comparators are: is / equals`,
		`Every hypothesis should carry at least one example`,
	} {
		if !strings.Contains(draftSystemPrompt, want) {
			t.Errorf("draftSystemPrompt is missing %q", want)
		}
	}
}
