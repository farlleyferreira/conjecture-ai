package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

const contextUsage = "usage: conjecture context <list|resume|open|restore> [args]"

// runContext implements "conjecture context <subcommand>" for the four
// of context-composition's six lifecycle commands this codebase has
// storage for (list/resume/open/restore - see
// internal/contextcomposition/store.go). "origin" and "diff" are not
// wired here: the product proposal document's own worked-example table
// describes origin's output as "decision, base, excerpt and who
// signed" and diff's as "specs, decisions and bases that changed" -
// both need real integration across spec-judge, knowledge-bases, and
// signing this codebase does not yet compose together, not just a
// store lookup.
//
// This surface also does not implement "list"/"search" filtering by
// area, date, or tags (the same table: "lists by name, area, date and
// tags"), since nothing stores an area or tags for a context yet - see
// [contextcomposition.Context] - nor does "resume" "answer the next
// work," the table's other stated half of what it does, which needs
// work-queue integration this codebase has not built either. Every one
// of these is a real, named gap, not an oversight: each would need its
// own grounded design, the same way origin/diff's did when they were
// deferred out of the storage PR this builds on.
//
// There is also no "conjecture context" command to create a context or
// add a version in the first place - the product proposal's own
// command table names no such verb. A context's rows are created by
// whatever eventually drives this (a conformance cycle, say) calling
// [contextcomposition.CreateContext]/[contextcomposition.AddVersion]
// directly, not by this CLI surface.
func runContext(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, contextUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "list", "resume", "open", "restore":
	default:
		_, _ = fmt.Fprintln(stderr, contextUsage)
		return 1
	}

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

	if err := contextcomposition.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	switch subcommand {
	case "list":
		return contextList(db, stdout, stderr)
	case "resume":
		return contextResume(db, rest, stdout, stderr)
	case "open":
		return contextOpen(db, rest, stdout, stderr)
	case "restore":
		return contextRestore(db, rest, stdout, stderr)
	default:
		// Unreachable: the subcommand switch above already rejected
		// anything else.
		_, _ = fmt.Fprintln(stderr, contextUsage)
		return 1
	}
}

// contextList implements "conjecture context list" - "Ver o que
// existe" / "Lista por nome, área, data e etiquetas" (see this file's
// own doc comment for what is and is not implemented of that).
func contextList(db *sql.DB, stdout, stderr io.Writer) int {
	contexts, err := contextcomposition.ListContexts(db)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if len(contexts) == 0 {
		_, _ = fmt.Fprintln(stdout, "no contexts")
		return 0
	}
	for _, c := range contexts {
		_, _ = fmt.Fprintf(stdout, "%s\t%s", c.ID, c.State)
		if c.State == contextcomposition.Archived && c.SupersededBy != "" {
			_, _ = fmt.Fprintf(stdout, " (superseded by %s)", c.SupersededBy)
		}
		_, _ = fmt.Fprintln(stdout)
	}
	return 0
}

// contextResume implements "conjecture context resume <name>" -
// restores name's latest version's content for continued use.
func contextResume(db *sql.DB, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: conjecture context resume <name>")
		return 1
	}
	v, err := contextcomposition.Resume(db, args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s@%s\n%s\n", v.ContextName, v.Tag(), v.Content)
	return 0
}

// contextOpen implements "conjecture context open <name>@v<N>" -
// opens that exact past version, read-only.
func contextOpen(db *sql.DB, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: conjecture context open <name>@v<N>")
		return 1
	}
	ref, err := contextcomposition.ParseVersionRef(args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	v, err := contextcomposition.Open(db, ref)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s@%s (read-only)\n%s\n", v.ContextName, v.Tag(), v.Content)
	return 0
}

// contextRestore implements "conjecture context restore <name>" -
// brings an archived context back to Active.
func contextRestore(db *sql.DB, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: conjecture context restore <name>")
		return 1
	}
	c, err := contextcomposition.Restore(db, args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "restored %s (%s)\n", c.ID, c.State)
	return 0
}
