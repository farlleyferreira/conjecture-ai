package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/farlleyferreira/conjecture-ai/internal/optimizationsuite"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/teamconfig"
)

const optimizeUsage = "usage: conjecture optimize set-current <field> <value>\n" +
	"       conjecture optimize calibrate <settings.cjtr> <field> --proposal <value> [--approved]"

// runOptimize implements "conjecture optimize <subcommand>", wiring
// optimization-suite's own calibration gate
// ([optimizationsuite.ApplyProposal]) into the CLI for the first
// time - disclosed as unwired since the package's own introduction
// (PR #65): "The system SHALL NOT apply a calibration proposal
// automatically; it SHALL require explicit human approval."
//
// Honesty note: this still does not run any multi-objective
// calibration algorithm - nothing in this codebase proposes a
// CalibrationProposal's own Value on its own (see optimizationsuite's
// own package doc comment). --proposal is the operator's own
// candidate value, same "caller supplies the fact" posture every
// other resolve-conflict-shaped command in this CLI already takes.
// What this wires is the gate around whatever value a human (or,
// eventually, a real calibrator) proposes: never applied without
// --approved, never allowed to cross the floor a real settings.cjtr
// file declares (see loadSettingsFile), and persisted across runs via
// [optimizationsuite.SetCurrent] so the next calibration has a real
// current value to compare against instead of starting from nothing.
func runOptimize(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, optimizeUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]

	switch subcommand {
	case "set-current":
		return runOptimizeSetCurrent(rest, stdout, stderr)
	case "calibrate":
		return runOptimizeCalibrate(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, optimizeUsage)
		return 1
	}
}

// openOptimizeStore opens the local, per-repository store, assuming
// the current working directory is the repository root - the same
// assumption every other command reading this store already makes.
func openOptimizeStore() (*sql.DB, error) {
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := optimizationsuite.EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// runOptimizeSetCurrent parses "set-current"'s own arguments and
// opens the real store - the thin, real-path wrapper around
// [setCurrent], which a test calls directly against an injected
// temporary store instead of the real per-OS one [persistence.
// OpenStore] would otherwise open.
func runOptimizeSetCurrent(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		_, _ = fmt.Fprintln(stderr, optimizeUsage)
		return 1
	}
	field := args[0]
	value, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %q is not a number\n", args[1])
		return 1
	}

	db, err := openOptimizeStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	return setCurrent(db, field, value, stdout, stderr)
}

// setCurrent implements "conjecture optimize set-current <field>
// <value>" against db - a direct, human-initiated write establishing
// field's own starting current value (e.g. the value a team's
// settings.cjtr floor already implies, or whatever is actually
// configured today), before any calibration proposal has run against
// it. This is not itself a calibration decision - no approval gate, no
// floor clamp - it is the one-time seed [calibrate]'s own first real
// proposal needs to have a current value to compare against at all.
func setCurrent(db *sql.DB, field string, value float64, stdout, stderr io.Writer) int {
	if err := optimizationsuite.SetCurrent(db, field, value); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s: current set to %v\n", field, value)
	return 0
}

// runOptimizeCalibrate parses "calibrate"'s own arguments and opens
// the real store - the thin, real-path wrapper around [calibrate],
// which a test calls directly against an injected temporary store.
func runOptimizeCalibrate(args []string, stdout, stderr io.Writer) int {
	var path, field, proposalFlag string
	approved := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--approved":
			approved = true
		case "--proposal":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, optimizeUsage)
				return 1
			}
			proposalFlag = args[i]
		default:
			switch {
			case path == "":
				path = args[i]
			case field == "":
				field = args[i]
			default:
				_, _ = fmt.Fprintln(stderr, optimizeUsage)
				return 1
			}
		}
	}
	if path == "" || field == "" || proposalFlag == "" {
		_, _ = fmt.Fprintln(stderr, optimizeUsage)
		return 1
	}

	proposalValue, err := strconv.ParseFloat(proposalFlag, 64)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: --proposal %q is not a number\n", proposalFlag)
		return 1
	}

	_, _, _, settings, err := loadSettingsFile(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	setting, ok := settings[field]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "conjecture: %q is not declared in %s\n", field, path)
		return 1
	}

	db, err := openOptimizeStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	return calibrate(db, field, proposalValue, approved, setting.Floor, setting.Direction, stdout, stderr)
}

// calibrate implements "conjecture optimize calibrate"'s own real
// logic against db: [optimizationsuite.ApplyProposal] applied for
// real, with field's current value read from and written back to db.
// field must already have a current value (via [setCurrent] or an
// earlier [calibrate] call) - this does not invent a starting point
// for a field nobody has calibrated before.
func calibrate(db *sql.DB, field string, proposalValue float64, approved bool, floor float64, direction teamconfig.Direction, stdout, stderr io.Writer) int {
	current, found, err := optimizationsuite.GetCurrent(db, field)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if !found {
		_, _ = fmt.Fprintf(stderr, "conjecture: %q has no current value yet - seed one with \"conjecture optimize set-current\"\n", field)
		return 1
	}

	proposal := optimizationsuite.CalibrationProposal{Field: field, Value: proposalValue}
	next := optimizationsuite.ApplyProposal(current, proposal, approved, floor, direction)

	if err := optimizationsuite.SetCurrent(db, field, next); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	switch {
	case !approved:
		_, _ = fmt.Fprintf(stdout, "%s: proposal %v not approved - current stays %v\n", field, proposalValue, next)
	case next == proposalValue:
		_, _ = fmt.Fprintf(stdout, "%s: approved - current is now %v\n", field, next)
	default:
		_, _ = fmt.Fprintf(stdout, "%s: approved but clamped at the floor - current is now %v (proposal %v crossed the floor)\n", field, next, proposalValue)
	}
	return 0
}
