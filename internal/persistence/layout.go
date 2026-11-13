package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// workingDirEntry is the local working directory's own .gitignore
// line - see [EnsureWorkingLayout]'s own doc comment for why
// ".conjecture/" specifically.
const workingDirEntry = ".conjecture/"

// EnsureWorkingLayout creates the repository layout the persistence
// spec names - "Git-tracked specification files" (a
// "conjecture/phenomena/<area>/" tree and a materializations path) and
// "Everything else stays local-only" (a local working directory,
// gitignored) - idempotently: an existing directory is left alone,
// and ".conjecture/" is appended to .gitignore only when no line
// already names it, preserving whatever the file already contained
// (creating it when absent).
//
// This is the layout half of the specification only, not the durable
// state half: the per-OS SQLite store [OpenStore] already resolves to
// lives outside the repository entirely and is unaffected by this
// function. ".conjecture/" is this project's own choice for the local
// working directory's name - the specification says it must be
// gitignored but does not name it, and this does not collide with the
// git-tracked "conjecture/" tree this same function creates.
//
// This function is never called as a side effect of another command
// (see cmd/conjecture/init.go's own doc comment for why) - a caller
// invokes it only for "conjecture init" itself.
func EnsureWorkingLayout(repoRoot string) error {
	gitTracked := []string{
		filepath.Join(repoRoot, "conjecture", "phenomena"),
		filepath.Join(repoRoot, "conjecture", "materializations"),
	}
	for _, dir := range gitTracked {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("persistence: creating %s: %w", dir, err)
		}
	}

	// Owner-only: the local working directory holds conjecture's own
	// scratch state, never git-tracked - see this function's own doc
	// comment, and [EnsureWorkingLayout]'s design's own "Non-Goals"
	// for why this is as far as "read-only to the implementing agent"
	// goes (there is no separate process identity to deny further).
	localDir := filepath.Join(repoRoot, ".conjecture")
	if err := os.MkdirAll(localDir, 0o700); err != nil {
		return fmt.Errorf("persistence: creating %s: %w", localDir, err)
	}

	return ensureGitignoreEntry(repoRoot, workingDirEntry)
}

// ensureGitignoreEntry appends entry as its own line to repoRoot's own
// .gitignore - creating the file when it does not exist, and leaving
// it untouched when entry (after trimming surrounding whitespace) is
// already one of its lines, exactly as written.
func ensureGitignoreEntry(repoRoot, entry string) error {
	path := filepath.Join(repoRoot, ".gitignore")
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("persistence: reading %s: %w", path, err)
	}

	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	content := string(raw)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("persistence: writing %s: %w", path, err)
	}
	return nil
}
