package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/contextengine"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

const graphUsage = "usage: conjecture graph set <node-id> --weight|--relevance|--confidence <low|medium|high|critical>\n" +
	"       conjecture graph inspect <node-id>"

// runGraph implements "conjecture graph <subcommand>", the first CLI
// surface over context-engine's own per-node storage
// (internal/contextengine/store.go): a place to actually set and read
// back a node's three axes, rather than computing one (as check.go's
// own [contextengine.GatedWeight] call already does for a law) and
// leaving it unrecorded.
//
// "inspect" is this command's own grounding for the specification's
// "Axes reported separately" scenario: "WHEN a node's context is
// inspected THEN its weight, relevance, and confidence are reported as
// separate values, not as one combined score." It prints the three
// axes on three lines, each marked "not set" independently when that
// axis has never been recorded - never a combined score, and never a
// guessed default for an axis nothing has reported.
//
// This command deliberately does not implement relevance propagation
// ([contextengine.Propagate]) or anything about edges: the
// specification gives no concrete representation for what connects two
// nodes, and nothing elsewhere in this codebase derives one from real
// rule relationships yet - inventing an edge model here would be
// guessing, not grounding. A node's identity is likewise left as an
// opaque, caller-supplied string, the same posture [contextengine.
// GatedWeight] and [contextengine.Propagate] already take.
func runGraph(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, graphUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "set":
		return runGraphSet(rest, stdout, stderr)
	case "inspect":
		return runGraphInspect(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, graphUsage)
		return 1
	}
}

// openGraphStore opens the local, per-repository store, assuming the
// current working directory is the repository root - the same
// assumption every other command reading this store already makes.
func openGraphStore() (*sql.DB, error) {
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := contextengine.EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// runGraphSet parses "set"'s own arguments and opens the real store -
// the thin, real-path wrapper around [graphSet], which a test calls
// directly against an injected temporary store instead of the real
// per-OS one [persistence.OpenStore] would otherwise open.
func runGraphSet(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		_, _ = fmt.Fprintln(stderr, graphUsage)
		return 1
	}
	nodeID, flag, valueFlag := args[0], args[1], args[2]

	value, err := evidenceledger.ParseWeight(valueFlag)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	var setFn func(*sql.DB, string, evidenceledger.Weight) error
	var axisName string
	switch flag {
	case "--weight":
		setFn, axisName = contextengine.SetWeight, "weight"
	case "--relevance":
		setFn, axisName = contextengine.SetRelevance, "relevance"
	case "--confidence":
		setFn, axisName = contextengine.SetConfidence, "confidence"
	default:
		_, _ = fmt.Fprintln(stderr, graphUsage)
		return 1
	}

	db, err := openGraphStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	return graphSet(db, setFn, axisName, nodeID, value, stdout, stderr)
}

// graphSet implements "conjecture graph set"'s own real logic against
// db: persists value for nodeID via setFn (one of [contextengine.
// SetWeight], [contextengine.SetRelevance], or [contextengine.
// SetConfidence] - already resolved by the caller from its own
// --weight/--relevance/--confidence flag) and reports what was set.
func graphSet(db *sql.DB, setFn func(*sql.DB, string, evidenceledger.Weight) error, axisName, nodeID string, value evidenceledger.Weight, stdout, stderr io.Writer) int {
	if err := setFn(db, nodeID, value); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s: %s set to %s\n", nodeID, axisName, value)
	return 0
}

// runGraphInspect parses "inspect"'s own arguments and opens the real
// store - the thin, real-path wrapper around [graphInspect], which a
// test calls directly against an injected temporary store.
func runGraphInspect(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, graphUsage)
		return 1
	}
	nodeID := args[0]

	db, err := openGraphStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	return graphInspect(db, nodeID, stdout, stderr)
}

// graphInspect implements "conjecture graph inspect"'s own real logic
// against db - see this file's own doc comment for how this grounds
// "Axes reported separately."
func graphInspect(db *sql.DB, nodeID string, stdout, stderr io.Writer) int {
	axes, found, err := contextengine.GetAxes(db, nodeID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if !found {
		_, _ = fmt.Fprintf(stdout, "%s: no axes recorded\n", nodeID)
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "%s:\n", nodeID)
	_, _ = fmt.Fprintf(stdout, "  weight:     %s\n", axisString(axes.Weight))
	_, _ = fmt.Fprintf(stdout, "  relevance:  %s\n", axisString(axes.Relevance))
	_, _ = fmt.Fprintf(stdout, "  confidence: %s\n", axisString(axes.Confidence))
	return 0
}

// axisString reports one axis's own value, or "not set" when nothing
// has ever recorded it - the per-axis independence [contextengine.
// Axes] itself documents, carried through to this command's own
// output.
func axisString(w *evidenceledger.Weight) string {
	if w == nil {
		return "not set"
	}
	return w.String()
}
