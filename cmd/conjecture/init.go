package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

const initUsage = "usage: conjecture init [--dir <path>]"

// runInit implements "conjecture init" - the one command that applies
// persistence.EnsureWorkingLayout, and the only one that ever does:
// creating the git-tracked "conjecture/phenomena/"/"conjecture/
// materializations/" tree and the local, gitignored ".conjecture/"
// working directory from any other command (draft, check, …) would
// surprise a repository the user has not opted into. --dir defaults
// to the current working directory, the same assumption every other
// command that reads this repository's own local store already makes.
//
// It also scaffolds the bundled orchestration skill
// (ensureWorkflowSkill, skillscaffold.go) into the repository, the
// same market-standard distribution mechanism this project's own
// OpenSpec dependency already uses to ship its own skills into a
// consuming project - creating it, updating it to a newer bundled
// version, or leaving a human-edited copy alone, never guessing which
// applies (see ensureWorkflowSkill's own doc comment).
func runInit(args []string, stdout, stderr io.Writer) int {
	dir := "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, initUsage)
				return 1
			}
			dir = args[i]
		default:
			_, _ = fmt.Fprintln(stderr, initUsage)
			return 1
		}
	}

	repoRoot, err := filepath.Abs(dir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	if err := persistence.EnsureWorkingLayout(repoRoot); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	action, err := ensureWorkflowSkill(repoRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "initialized conjecture layout at %s\n", repoRoot)
	_, _ = fmt.Fprintf(stdout, "orchestration skill at %s: %s\n", filepath.Join(repoRoot, filepath.FromSlash(skillRelPath)), action)
	return 0
}
