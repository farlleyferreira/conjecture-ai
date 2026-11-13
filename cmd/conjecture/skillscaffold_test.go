package main

import (
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
