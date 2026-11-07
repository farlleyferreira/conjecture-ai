package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/knowledgebases"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

const baseUsage = "usage: conjecture base add <path> --authority <norm|definition|reference|history> --scope <project|area:NAME> [--review-date <date>]\n       conjecture base search <query>"

// runBase implements "conjecture base <subcommand>": "add"
// (knowledge-bases spec, "CLI-based base attachment") and "search"
// ("Local indexing for retrieval"'s textual-search half - see
// [knowledgebases.Search]). Both act on the local, per-repository
// store (see [knowledgebases.EnsureSchema]'s doc comment for why
// there, not a git-tracked file), assuming the current working
// directory is the repository root - this codebase has no repository-
// root-discovery mechanism of its own yet, so this command does not
// invent one, and simply expects to be run from that root, the same
// way many CLI tools do.
func runBase(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, baseUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]

	repoRoot, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	switch subcommand {
	case "add":
		return runBaseAdd(db, rest, stdout, stderr)
	case "search":
		return runBaseSearch(db, rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, baseUsage)
		return 1
	}
}

func runBaseAdd(db *sql.DB, args []string, stdout, stderr io.Writer) int {
	var path, authorityFlag, scope, reviewDate string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--authority", "--scope", "--review-date":
			flag := args[i]
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, baseUsage)
				return 1
			}
			switch flag {
			case "--authority":
				authorityFlag = args[i]
			case "--scope":
				scope = args[i]
			case "--review-date":
				reviewDate = args[i]
			}
		default:
			if path != "" {
				_, _ = fmt.Fprintln(stderr, baseUsage)
				return 1
			}
			path = args[i]
		}
	}
	if path == "" || authorityFlag == "" || scope == "" {
		_, _ = fmt.Fprintln(stderr, baseUsage)
		return 1
	}

	stored, err := registerBase(db, path, authorityFlag, scope, reviewDate)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	// Indexed for local full-text search right away, as part of
	// registration itself - there is no separate "conjecture base
	// index" verb the product proposal names, and a newly-attached
	// base being unsearchable until some other command is run would be
	// a surprising gap, not a deliberate scope boundary.
	if err := knowledgebases.EnsureSearchSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := knowledgebases.IndexBase(db, stored); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "registered %s as %s authority, scope %s", stored.Path, stored.Authority, stored.Scope)
	if stored.Incomplete {
		_, _ = fmt.Fprint(stdout, " (INCOMPLETE: a norm-authority base needs a last review date)")
	}
	_, _ = fmt.Fprintln(stdout)
	return 0
}

// runBaseSearch implements "conjecture base search <query>" -
// knowledge-bases' "Local indexing for retrieval" textual-search half,
// over whatever every "conjecture base add" has indexed so far.
func runBaseSearch(db *sql.DB, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, baseUsage)
		return 1
	}

	if err := knowledgebases.EnsureSearchSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	results, err := knowledgebases.Search(db, args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no matches")
		return 0
	}
	for _, r := range results {
		_, _ = fmt.Fprintf(stdout, "%s: %s\n", r.FilePath, r.Snippet)
	}
	return 0
}

// registerBase validates authorityFlag, computes path's content hash,
// and registers the base in db - the part of "conjecture base add"
// independent of where its store lives, so a test can exercise it
// against a temporary store instead of the real per-OS one
// [persistence.OpenStore] always resolves to.
func registerBase(db *sql.DB, path, authorityFlag, scope, reviewDate string) (knowledgebases.Base, error) {
	authority, err := knowledgebases.ParseAuthority(authorityFlag)
	if err != nil {
		return knowledgebases.Base{}, err
	}

	hash, err := knowledgebases.ComputeContentHash(path)
	if err != nil {
		return knowledgebases.Base{}, err
	}

	if err := knowledgebases.EnsureSchema(db); err != nil {
		return knowledgebases.Base{}, err
	}

	return knowledgebases.Add(db, knowledgebases.Base{
		Path: path, Authority: authority, Scope: scope,
		ContentHash: hash, LastReviewDate: reviewDate,
	})
}
