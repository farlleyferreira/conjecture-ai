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
// "list" now also filters by area and by a created-at date range (see
// contextList's own doc comment) - area is auto-derived, not asked of
// the operator, now that "conjecture check" (see check.go's own
// syncCheckContext) is this codebase's first real context-creating
// caller. "tags" ("the same table: 'lists by name, area, date and
// tags'") is deliberately not attempted: with exactly one context-
// creating path in this codebase, a tag naming "which command created
// this" could never take more than one value, which is not a filter
// worth building ahead of a second value actually existing.
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
		return contextList(db, rest, stdout, stderr)
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

// contextList implements "conjecture context list [--area <area>]
// [--since <YYYY-MM-DD>] [--until <YYYY-MM-DD>]" - "Ver o que existe" /
// "Lista por nome, área, data e etiquetas" (see this file's own doc
// comment for what is and is not implemented of that: area and date,
// not tags). --since/--until are inclusive, parsed as whole UTC days
// (--since at 00:00:00, --until at 23:59:59.999999999), since
// CreatedAt carries no timezone of its own for an operator to reason
// about more precisely.
func contextList(db *sql.DB, args []string, stdout, stderr io.Writer) int {
	usage := "usage: conjecture context list [--area <area>] [--since <YYYY-MM-DD>] [--until <YYYY-MM-DD>]"
	var filter contextcomposition.ContextFilter
	for i := 0; i < len(args); i++ {
		if i+1 >= len(args) {
			_, _ = fmt.Fprintln(stderr, usage)
			return 1
		}
		flag, value := args[i], args[i+1]
		i++
		switch flag {
		case "--area":
			filter.Area = value
		case "--since":
			t, err := time.Parse("2006-01-02", value)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "conjecture: --since: %v\n", err)
				return 1
			}
			filter.Since = t
		case "--until":
			t, err := time.Parse("2006-01-02", value)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "conjecture: --until: %v\n", err)
				return 1
			}
			filter.Until = t.Add(24*time.Hour - time.Nanosecond)
		default:
			_, _ = fmt.Fprintln(stderr, usage)
			return 1
		}
	}

	contexts, err := contextcomposition.ListContextsFiltered(db, filter)
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
		if c.Area != "" {
			_, _ = fmt.Fprintf(stdout, "\t%s", c.Area)
		}
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
