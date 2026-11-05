// Package persistence implements the persistence capability (openspec/
// specs/persistence/spec.md): where conjecture's non-git-tracked state
// lives - a single local, per-repository SQLite database, outside the
// git repository, at a fixed per-OS path - and the hash-chaining
// mechanism every append-only table in that database uses.
//
// This package only resolves the path and opens the database; it does
// not define any table. The specific tables (evidence, decisions,
// configuration history, …) belong to whichever capability owns that
// data (e.g. evidence-ledger), not to persistence itself.
package persistence

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

// RepoID derives the stable identifier a repository's local data
// directory is keyed by: the first 16 hex characters (64 bits - more
// than enough to avoid an accidental collision for this purpose) of the
// SHA-256 hash of repoRoot's cleaned, absolute path (lowercased on
// Windows, whose filesystem is case-insensitive).
//
// The specification names the directory "<repo-id>" without saying how
// to derive it. Hashing the working tree's own path, rather than
// something like its git remote URL, is the reading that invents the
// least: it needs no git remote to exist, gives every distinct checkout
// of a repository its own local state (which is what that state is
// actually scoped to - two clones of the same remote can hold different
// uncommitted specifications), and never collides by construction, as a
// human-chosen project name could.
func RepoID(repoRoot string) (string, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("persistence: %w", err)
	}
	clean := filepath.Clean(abs)
	if runtime.GOOS == "windows" {
		clean = strings.ToLower(clean)
	}
	sum := sha256.Sum256([]byte(clean))
	return hex.EncodeToString(sum[:])[:16], nil
}

// LocalDataDir returns the per-OS local data directory for the
// repository rooted at repoRoot, exactly as the persistence spec names
// it: "/var/lib/conjecture/<repo-id>/" on Linux, "/Library/Application
// Support/conjecture/<repo-id>/" on macOS, and
// "%ProgramData%\conjecture\<repo-id>\" on Windows.
func LocalDataDir(repoRoot string) (string, error) {
	return localDataDir(repoRoot, runtime.GOOS)
}

func localDataDir(repoRoot, goos string) (string, error) {
	id, err := RepoID(repoRoot)
	if err != nil {
		return "", err
	}

	// Each branch's separator is the literal one the specification
	// itself uses for that OS - not filepath.Join's, which follows the
	// build platform's own convention regardless of which OS's path this
	// function is asked to produce, and so cannot be used here (this
	// function must produce a genuine Windows path when asked for one
	// even when built for Linux, and vice versa).
	switch goos {
	case "linux":
		return "/var/lib/conjecture/" + id, nil
	case "darwin":
		return "/Library/Application Support/conjecture/" + id, nil
	case "windows":
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		return strings.TrimRight(base, `\`) + `\conjecture\` + id, nil
	default:
		return "", fmt.Errorf("persistence: unsupported OS %q (the specification only names Linux, macOS, and Windows paths)", goos)
	}
}

// OpenStore opens (creating it first if necessary) the single per-
// repository SQLite database the persistence spec requires, at its
// fixed per-OS path for repoRoot. The caller is responsible for closing
// the returned *sql.DB.
func OpenStore(repoRoot string) (*sql.DB, error) {
	dir, err := LocalDataDir(repoRoot)
	if err != nil {
		return nil, err
	}
	return OpenStoreAt(dir)
}

// OpenStoreAt opens (creating it first if necessary) the SQLite database
// at dir/conjecture.db, creating dir itself if it does not exist. This
// is [OpenStore]'s path-independent half, split out so it can be tested
// against a temporary directory instead of the real per-OS system path
// OpenStore always resolves to.
func OpenStoreAt(dir string) (*sql.DB, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("persistence: creating local data directory: %w", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, "conjecture.db"))
	if err != nil {
		return nil, fmt.Errorf("persistence: opening store: %w", err)
	}

	// WAL lets concurrent readers proceed while a write is in progress
	// (this store has one writer - the engine - and potentially several
	// readers, e.g. an MCP server and a CLI invocation at once); foreign
	// keys are off by SQLite's own default and this store wants them
	// enforced.
	if _, err := db.Exec(`PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("persistence: configuring store: %w", err)
	}

	return db, nil
}

// ChainHash computes the hash-chain value for a new append-only record
// whose content is data, linked to the previous record's own chain hash
// (previous - or nil/empty for a table's first record) - the generic
// "chained by hash to the previous one" mechanism the persistence spec
// requires of every append-only table (evidence, decisions, and
// configuration history), independent of any one table's own columns,
// which this package does not define.
func ChainHash(previous, data []byte) []byte {
	h := sha256.New()
	h.Write(previous)
	h.Write(data)
	return h.Sum(nil)
}
