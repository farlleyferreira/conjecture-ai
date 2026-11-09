package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

const contextUsage = "usage: conjecture context <list|resume|open|restore|resolve-conflict> [args]"

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
// [contextcomposition.Context]. This is a real, named gap, not an
// oversight: it would need its own grounded design, the same way
// origin/diff's did when they were deferred out of the storage PR this
// builds on.
//
// "resume" now also answers the table's other stated half of what it
// does - the next work to pick up - by reporting [workqueue.Next]'s
// own highest-priority item alongside the resumed content, now that
// work-queue has real persistence (internal/workqueue/store.go) to
// query. It peeks rather than pops: resuming a context is read-only by
// the requirement this file's own "Prior context resumed" scenario
// names, and nothing says resuming should also consume queue state -
// a caller still removes the item via "conjecture queue next" once
// they act on it.
//
// There is also no "conjecture context" command to create a context or
// add a version in the first place - the product proposal's own
// command table names no such verb. A context's rows are created by
// whatever eventually drives this (a conformance cycle, say) calling
// [contextcomposition.CreateContext]/[contextcomposition.AddVersion]
// directly, not by this CLI surface.
//
// "resolve-conflict" wires [contextcomposition.ResolveConflict] -
// PrecedenceOrder's own conflict rule, unused by any command until
// now - into the CLI. It needs no store: the two composition modes in
// conflict are the operator's own facts about what is actually
// conflicting, the same "caller supplies the fact" posture "conjecture
// verify"'s --catalog/--addressed flags and "conjecture law resolve-
// conflict" already take.
func runContext(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, contextUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "list", "resume", "open", "restore":
	case "resolve-conflict":
		return contextResolveConflict(rest, stdout, stderr)
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
// restores name's latest version's content for continued use, then
// reports the highest-priority work-queue item (if any) as the next
// work to pick up (see this file's own doc comment).
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

	if err := workqueue.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	item, found, err := workqueue.Next(db, time.Now().UTC(), queueAgingRate, queueAgingUnit)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if !found {
		_, _ = fmt.Fprintln(stdout, "next work: queue is empty")
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "next work: %s (%s)\n", item.ID, tierString(item.Tier))
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

// contextResolveConflict implements "conjecture context resolve-
// conflict --mode-a <mode> --mode-b <mode>" - [contextcomposition.
// ResolveConflict] applied to two operator-named composition modes.
// When decided is false (the two modes are the same one), this
// reports that honestly rather than printing a winner
// [contextcomposition.ResolveConflict]'s own doc comment says is not
// meaningful - the same "flag it for a human, decide nothing
// unilaterally" posture "conjecture law resolve-conflict"'s own
// RaiseQuestion outcome already takes.
func contextResolveConflict(args []string, stdout, stderr io.Writer) int {
	usage := "usage: conjecture context resolve-conflict --mode-a <ByReference|ByEmbedding|ByInheritance|ByView> --mode-b <...>"
	var modeAFlag, modeBFlag string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode-a", "--mode-b":
			flag := args[i]
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, usage)
				return 1
			}
			switch flag {
			case "--mode-a":
				modeAFlag = args[i]
			case "--mode-b":
				modeBFlag = args[i]
			}
		default:
			_, _ = fmt.Fprintln(stderr, usage)
			return 1
		}
	}
	if modeAFlag == "" || modeBFlag == "" {
		_, _ = fmt.Fprintln(stderr, usage)
		return 1
	}

	modeA, err := contextcomposition.ParseCompositionMode(modeAFlag)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	modeB, err := contextcomposition.ParseCompositionMode(modeBFlag)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	winner, decided := contextcomposition.ResolveConflict(modeA, modeB)
	if !decided {
		_, _ = fmt.Fprintf(stdout, "undecided: %s and %s are the same mode - PrecedenceOrder ranks the four modes against each other, not two sources composed the same way\n", modeA, modeB)
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "%s outranks %s\n", winner, otherMode(winner, modeA, modeB))
	return 0
}

// otherMode returns whichever of a and b is not winner - the loser,
// for contextResolveConflict's own report line.
func otherMode(winner, a, b contextcomposition.CompositionMode) contextcomposition.CompositionMode {
	if winner == a {
		return b
	}
	return a
}
