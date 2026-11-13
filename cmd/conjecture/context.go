package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/decisionledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
	"github.com/pmezard/go-difflib/difflib"
)

const contextUsage = "usage: conjecture context <list|resume|open|origin|diff|restore|resolve-conflict> [args]"

// runContext implements "conjecture context <subcommand>" for all six
// of context-composition's own lifecycle commands (list/resume/open/
// restore - see internal/contextcomposition/store.go - plus origin
// and diff, below).
//
// Honesty note on origin/diff: the specification itself
// (context-composition's own spec.md) names only that these two
// commands must exist, plus one concrete scenario ("a user can still
// diff the superseded context against the new one") - it gives no
// output shape for either. The lost product-proposal document's own
// paraphrase ("decision, base, excerpt and who signed" for origin;
// "specs, decisions and bases that changed" for diff) is not treated
// as a specification here - it was never verified, and this codebase
// has already spent one session confirming it is permanently
// unrecoverable. Instead, each is grounded in what this codebase can
// actually compose for real, confirmed with the user before writing
// either:
//   - diff compares two versions' own real stored Content via a real
//     unified-diff algorithm (github.com/pmezard/go-difflib, already a
//     transitive dependency via testify, not a new one) - directly
//     answering the one scenario the specification actually gives.
//   - origin reports a context's real source .cjtr file's own
//     "signed" block ([cjtr.ReadSigned] - user/tool/date/confirmed)
//     together with every [decisionledger] decision whose own Subject
//     exactly matches that file's path (joinPhenomenonPath's own
//     convention) - two pieces of real, already-tracked provenance
//     this codebase did not have composed together until now. Decision
//     matching is an exact string match on Subject, not a fuzzy one: a
//     decision recorded against a differently-spelled path for the
//     same file (a different relative prefix, an absolute path) will
//     not be found - a disclosed limitation, not a guess at resolving
//     it.
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
	case "list", "resume", "open", "origin", "diff", "restore":
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
	case "origin":
		return contextOrigin(db, repoRoot, rest, stdout, stderr)
	case "diff":
		return contextDiff(db, rest, stdout, stderr)
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

// contextOrigin implements "conjecture context origin <name>" - see
// this file's own doc comment for what "origin" means here and why.
func contextOrigin(db *sql.DB, repoRoot string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: conjecture context origin <name>")
		return 1
	}
	name := args[0]

	ctx, err := contextcomposition.GetContext(db, name)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s (area %q)\n", ctx.ID, ctx.Area)

	area, phenomenon, ok := splitPhenomenonID(name)
	if !ok {
		_, _ = fmt.Fprintf(stdout, "source file: unknown - %q is not an area::phenomenon composite ID\n", name)
		return 0
	}
	path := joinPhenomenonPath(repoRoot, area, phenomenon)

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "source file: %s (could not be loaded: %v)\n", path, err)
	} else if signed, err := cjtr.ReadSigned(f.root); err != nil {
		_, _ = fmt.Fprintf(stdout, "source file: %s (signed block could not be read: %v)\n", path, err)
	} else if signed == nil {
		_, _ = fmt.Fprintf(stdout, "source file: %s (not signed)\n", path)
	} else {
		_, _ = fmt.Fprintf(stdout, "source file: %s\nsigned by %s (tool %s, date %s, confirmed %v)\n",
			path, signed.User, signed.Tool, signed.Date, signed.Confirmed)
	}

	if err := decisionledger.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	all, err := decisionledger.List(db)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	found := false
	for _, d := range all {
		if d.Subject != path {
			continue
		}
		found = true
		_, _ = fmt.Fprintf(stdout, "decision: %s %q by %s on %s\n", d.Kind, d.Determination, d.DecidedBy, d.CreatedAt.Format(time.RFC3339))
	}
	if !found {
		_, _ = fmt.Fprintln(stdout, "no decisions recorded against this source file")
	}
	return 0
}

// splitPhenomenonID splits a context's own ID back into its area and
// phenomenon - the exact inverse of [cjtr.ComposeID]'s own area/
// phenomenon convention every real context-creating caller (check.go's
// syncCheckContext) already uses, ok is false for anything else.
func splitPhenomenonID(id string) (area, phenomenon string, ok bool) {
	area, phenomenon, found := strings.Cut(id, "::")
	if !found || area == "" || phenomenon == "" || strings.Contains(phenomenon, "::") {
		return "", "", false
	}
	return area, phenomenon, true
}

// contextDiff implements "conjecture context diff <name>[@v<N>]
// <name>[@v<N>]" - a real unified diff between two context versions'
// own stored Content (see this file's own doc comment for why a
// unified diff, not the lost document's own unverified output shape).
// Each argument names either a context's latest version (a bare
// name) or one exact past version ("name@vN") - [contextcomposition.
// ParseVersionRef]'s own two forms, the same ones "conjecture context
// open" already accepts.
func contextDiff(db *sql.DB, args []string, stdout, stderr io.Writer) int {
	usage := "usage: conjecture context diff <name>[@v<N>] <name>[@v<N>]"
	if len(args) != 2 {
		_, _ = fmt.Fprintln(stderr, usage)
		return 1
	}

	a, err := resolveContextVersion(db, args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	b, err := resolveContextVersion(db, args[1])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	if a.Content == b.Content {
		_, _ = fmt.Fprintf(stdout, "%s@%s and %s@%s are identical\n", a.ContextName, a.Tag(), b.ContextName, b.Tag())
		return 0
	}

	text, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(a.Content),
		FromFile: a.ContextName + "@" + a.Tag(),
		B:        difflib.SplitLines(b.Content),
		ToFile:   b.ContextName + "@" + b.Tag(),
		Context:  3,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprint(stdout, text)
	return 0
}

// resolveContextVersion resolves one "conjecture context diff" (or
// future caller's) argument - a bare context name (its latest
// version, via [contextcomposition.Resume]) or "name@vN" (that exact
// version, via [contextcomposition.Open]) - the same two forms
// [contextcomposition.ParseVersionRef] already distinguishes by
// whether it parsed a nonzero Seq.
func resolveContextVersion(db *sql.DB, arg string) (contextcomposition.Version, error) {
	ref, err := contextcomposition.ParseVersionRef(arg)
	if err != nil {
		return contextcomposition.Version{}, err
	}
	if ref.Seq == 0 {
		return contextcomposition.Resume(db, ref.Name)
	}
	return contextcomposition.Open(db, ref)
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
