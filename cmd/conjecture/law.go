package main

import (
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
)

const lawUsage = "usage: conjecture law resolve-conflict <law-id> [--formal-proof-contradicts] [--equal-authority]"

// runLaw implements "conjecture law <subcommand>": "resolve-conflict"
// wires [userdefinedlaws.ResolveConflict] - user-defined-laws' own
// "Invalidation pathway" and "Precedence over promoted hypotheses"
// requirements - into the CLI for the first time. Both of
// ResolveConflict's own facts (whether a formal proof contradicts the
// law, whether the conflicting statement carries equal declared
// authority) are supplied by the operator as explicit flags: nothing
// in this codebase produces a formal proof against an arbitrary law or
// compares declared authority on its own, so this command does not
// guess either - the same "caller supplies the fact" posture
// "conjecture verify"'s own --catalog/--addressed flags already take
// for omission-detection's Type D check.
func runLaw(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, lawUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]

	switch subcommand {
	case "resolve-conflict":
		return runLawResolveConflict(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, lawUsage)
		return 1
	}
}

// runLawResolveConflict applies [userdefinedlaws.ResolveConflict] to
// lawID and reports the outcome. An AutoInvalidated outcome also
// records the law's status as [userdefinedlaws.Invalidated] - the one
// status this codebase could not reach from any other path until now
// (see check.go's own honesty note on recordEvidence). A RaiseQuestion
// outcome is reported on stdout as a flagged line for a human to act
// on - the same "print it for a human, decide nothing unilaterally"
// posture recordEvidence's own staleness warning already takes - and
// does not change the law's stored status, since the specification
// only names a status transition for the automatic-invalidation and
// contested-threshold cases, not for raising a question. A LawPrevails
// outcome changes nothing: it is "Precedence over promoted hypotheses"
// reported back, not a status this package's own lifecycle tracks.
func runLawResolveConflict(args []string, stdout, stderr io.Writer) int {
	var lawID string
	var formalProofContradicts, equalAuthority bool
	for _, a := range args {
		switch a {
		case "--formal-proof-contradicts":
			formalProofContradicts = true
		case "--equal-authority":
			equalAuthority = true
		default:
			if lawID != "" {
				_, _ = fmt.Fprintln(stderr, lawUsage)
				return 1
			}
			lawID = a
		}
	}
	if lawID == "" {
		_, _ = fmt.Fprintln(stderr, lawUsage)
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
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	outcome := userdefinedlaws.ResolveConflict(formalProofContradicts, equalAuthority)
	_, _ = fmt.Fprintf(stdout, "%s: conflict outcome: %s\n", lawID, outcome)

	switch outcome {
	case userdefinedlaws.AutoInvalidated:
		if err := userdefinedlaws.SetStatus(db, lawID, userdefinedlaws.Invalidated); err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "%s: status set to invalidated\n", lawID)
	case userdefinedlaws.RaiseQuestion:
		_, _ = fmt.Fprintf(stdout, "%s: QUESTION: two statements of equal declared authority conflict with no formal proof available - a human must decide, the system will not\n", lawID)
	}

	return 0
}
