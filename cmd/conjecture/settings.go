package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/farlleyferreira/conjecture-ai/internal/teamconfig"
)

const settingsUsage = "usage: conjecture settings check-floor <name> --local <value> --floor <value> --direction <higher|lower>"

// runSettings implements "conjecture settings <subcommand>":
// "check-floor" wires team-config's own floor-not-ceiling rule
// (internal/teamconfig/floor.go) into the CLI for the first time -
// disclosed as unwired since the package's own introduction (PR #46):
// "this package only implements that comparison... it does not parse
// settings.cjtr itself."
//
// This still does not parse a real settings.cjtr file: the
// specification gives no worked example of that file's own content
// (team-config's own doc comment), so this command does not invent a
// concrete settings schema to go with the comparison rule either. What
// it does wire is the comparison itself, with both the team's floor
// and a local override's value supplied as explicit flags - the same
// "caller supplies the fact" posture every other resolve-conflict-
// shaped command in this CLI already takes. A real settings.cjtr
// loader, when the specification or the user gives this project a
// concrete schema to parse, would call the exact same
// [teamconfig.CheckFloor] this command already exercises end to end.
func runSettings(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, settingsUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]

	switch subcommand {
	case "check-floor":
		return runSettingsCheckFloor(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, settingsUsage)
		return 1
	}
}

// runSettingsCheckFloor parses its own arguments and reports whether
// local meets the team's floor in direction, via
// [teamconfig.CheckFloor]. local, floor, and direction are all
// operator-supplied facts about one already-configured setting
// somewhere in this codebase (a required case count, a tolerated
// violation rate, a risk weight's own ordinal) or a team's own
// candidate value for one - this command does not look any setting's
// current value up itself, since there is no settings store yet for
// it to read from.
func runSettingsCheckFloor(args []string, stdout, stderr io.Writer) int {
	var name, localFlag, floorFlag, directionFlag string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local", "--floor", "--direction":
			flag := args[i]
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, settingsUsage)
				return 1
			}
			switch flag {
			case "--local":
				localFlag = args[i]
			case "--floor":
				floorFlag = args[i]
			case "--direction":
				directionFlag = args[i]
			}
		default:
			if name != "" {
				_, _ = fmt.Fprintln(stderr, settingsUsage)
				return 1
			}
			name = args[i]
		}
	}
	if name == "" || localFlag == "" || floorFlag == "" || directionFlag == "" {
		_, _ = fmt.Fprintln(stderr, settingsUsage)
		return 1
	}

	local, err := strconv.ParseFloat(localFlag, 64)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: --local %q is not a number\n", localFlag)
		return 1
	}
	floor, err := strconv.ParseFloat(floorFlag, 64)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: --floor %q is not a number\n", floorFlag)
		return 1
	}
	var direction teamconfig.Direction
	switch directionFlag {
	case "higher":
		direction = teamconfig.Higher
	case "lower":
		direction = teamconfig.Lower
	default:
		_, _ = fmt.Fprintf(stderr, "conjecture: --direction %q must be \"higher\" or \"lower\"\n", directionFlag)
		return 1
	}

	if err := teamconfig.CheckFloor(name, local, floor, direction); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: rejected: %v\n", name, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s: accepted: local value %v meets the team's floor of %v\n", name, local, floor)
	return 0
}
