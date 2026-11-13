package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/conjecture-workflow-skill.md
var conjectureWorkflowSkill []byte

// skillRelPath is where "conjecture init" writes the bundled
// orchestration skill, relative to the repository root - the Claude
// Code skill-file convention (".claude/skills/<name>/SKILL.md") this
// very repository's own OpenSpec dependency already uses to ship its
// own skills into a consuming project (see this repo's own
// ".claude/skills/openspec-*/SKILL.md"). Following that already-
// established market convention, rather than inventing a distribution
// mechanism of this project's own, is deliberate: see cli-mcp-
// integration's own "Onboarding skill scaffold" requirement.
const skillRelPath = ".claude/skills/conjecture-workflow/SKILL.md"

// ensureWorkflowSkill writes the bundled orchestration skill
// (templates/conjecture-workflow-skill.md, embedded at build time)
// into repoRoot, idempotently: an already-present file is left alone,
// never overwritten, so a user's own edits to it are never clobbered
// by a later "conjecture init" re-run - the same "existing directories
// are left alone" posture [persistence.EnsureWorkingLayout] already
// takes for its own idempotent writes.
//
// What the skill itself teaches (draft -> verify -> human-signs ->
// implement -> check, never confirming a specification on the user's
// own behalf) lives entirely in that one file, not duplicated here -
// this function only ever places it.
func ensureWorkflowSkill(repoRoot string) error {
	path := filepath.Join(repoRoot, filepath.FromSlash(skillRelPath))
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("conjecture: checking %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("conjecture: creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, conjectureWorkflowSkill, 0o644); err != nil {
		return fmt.Errorf("conjecture: writing %s: %w", path, err)
	}
	return nil
}
