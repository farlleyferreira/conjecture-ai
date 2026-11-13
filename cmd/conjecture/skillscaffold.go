package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// skillChecksumRelPath records, in the repository's own local working
// directory (never git-tracked - persistence's own "Everything else
// stays local-only"), the SHA-256 of the exact bytes
// [ensureWorkflowSkill] itself last wrote to skillRelPath. Its only
// purpose is answering one question on a later "conjecture init" run:
// "is the file on disk still exactly what we wrote, or has the user
// edited it since?" - see [ensureWorkflowSkill]'s own doc comment for
// what each answer does.
const skillChecksumRelPath = ".conjecture/skill-checksum"

// SkillScaffoldAction reports what [ensureWorkflowSkill] actually did,
// so "conjecture init" can tell the user the true story instead of
// one fixed message regardless of outcome.
type SkillScaffoldAction int

const (
	SkillCreated SkillScaffoldAction = iota
	SkillUpToDate
	SkillUpdated
	SkillLeftCustomized
)

func (a SkillScaffoldAction) String() string {
	switch a {
	case SkillCreated:
		return "created"
	case SkillUpToDate:
		return "already up to date"
	case SkillUpdated:
		return "updated to the newer bundled version"
	case SkillLeftCustomized:
		return "left as-is (edited since conjecture last wrote it)"
	default:
		return "unknown"
	}
}

// ensureWorkflowSkill writes or updates the bundled orchestration
// skill (templates/conjecture-workflow-skill.md, embedded at build
// time, rendered via [renderWorkflowSkill]) at repoRoot.
//
// A file that does not exist yet is created (SkillCreated). A file
// that exists is only ever overwritten when it is still exactly the
// bytes this function itself last wrote there, verified via the
// SHA-256 recorded at skillChecksumRelPath the previous time this ran
// (SkillUpdated when the newly rendered content actually differs,
// SkillUpToDate when it is already identical) - never when the
// recorded checksum is missing or does not match what is on disk
// (SkillLeftCustomized), since either means a human may have edited
// the file since, and clobbering that edit would defeat the entire
// point of idempotence [persistence.EnsureWorkingLayout] already
// established for the rest of this command's own layout. A repository
// scaffolded before this checksum tracking existed has no recorded
// checksum at all, so its first "conjecture init" under this version
// also reports SkillLeftCustomized - conservatively correct (this
// function genuinely cannot tell whether that file was ever edited),
// not a false claim either way.
func ensureWorkflowSkill(repoRoot string) (SkillScaffoldAction, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(skillRelPath))
	checksumPath := filepath.Join(repoRoot, filepath.FromSlash(skillChecksumRelPath))
	rendered := renderWorkflowSkill()
	renderedSum := sha256Hex(rendered)

	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return 0, fmt.Errorf("conjecture: creating %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, rendered, 0o644); err != nil {
			return 0, fmt.Errorf("conjecture: writing %s: %w", path, err)
		}
		if err := writeSkillChecksum(checksumPath, renderedSum); err != nil {
			return 0, err
		}
		return SkillCreated, nil
	}
	if err != nil {
		return 0, fmt.Errorf("conjecture: reading %s: %w", path, err)
	}

	storedSum, readErr := os.ReadFile(checksumPath)
	existingSum := sha256Hex(existing)
	if readErr != nil || strings.TrimSpace(string(storedSum)) != existingSum {
		// Missing, unreadable, or stale checksum - this file may have
		// been edited by a human since it was last written; never
		// overwrite on anything less than certainty that it was not.
		return SkillLeftCustomized, nil
	}

	if existingSum == renderedSum {
		return SkillUpToDate, nil
	}
	if err := os.WriteFile(path, rendered, 0o644); err != nil {
		return 0, fmt.Errorf("conjecture: writing %s: %w", path, err)
	}
	if err := writeSkillChecksum(checksumPath, renderedSum); err != nil {
		return 0, err
	}
	return SkillUpdated, nil
}

// writeSkillChecksum records sum at path, creating its own parent
// directory (".conjecture/", already created 0o700 by
// [persistence.EnsureWorkingLayout] in the real "conjecture init"
// path; created here too in case a caller - a test, say - calls
// [ensureWorkflowSkill] on its own without that having already run).
func writeSkillChecksum(path, sum string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("conjecture: creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(sum), 0o600); err != nil {
		return fmt.Errorf("conjecture: writing %s: %w", path, err)
	}
	return nil
}

func sha256Hex(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// renderWorkflowSkill substitutes the shared grammar reference
// (cjtrGrammarYAML/cjtrGrammarNotes, draft.go) into the embedded
// skill template's own "{{CJTR_GRAMMAR_YAML}}"/"{{CJTR_GRAMMAR_NOTES}}"
// placeholders, each re-indented by three spaces to match the
// surrounding numbered list item's own continuation indentation - the
// one piece of real re-formatting this sharing needs, since a skill
// file and an LLM system prompt otherwise render the identical text
// in a different structural context.
func renderWorkflowSkill() []byte {
	content := conjectureWorkflowSkill
	content = bytes.ReplaceAll(content, []byte("{{CJTR_GRAMMAR_YAML}}"), []byte(indentNonEmptyLines(cjtrGrammarYAML, "   ")))
	content = bytes.ReplaceAll(content, []byte("{{CJTR_GRAMMAR_NOTES}}"), []byte(indentNonEmptyLines(cjtrGrammarNotes, "   ")))
	return content
}

// indentNonEmptyLines prefixes every non-empty line of s with prefix,
// leaving a blank line (a paragraph break) exactly as it is - a
// trailing prefix on an empty line would be invisible but still
// technically trailing whitespace, worth avoiding on a file meant to
// be read and possibly edited by a human.
func indentNonEmptyLines(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
