package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/teamconfig"
	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

const settingsUsage = "usage: conjecture settings check-floor <name> --local <value> --floor <value> --direction <higher|lower>\n" +
	"       conjecture settings verify <settings.cjtr>\n" +
	"       conjecture settings check <settings.cjtr> <name> --local <value>"

// runSettings implements "conjecture settings <subcommand>", wiring
// team-config's own floor-not-ceiling rule
// (internal/teamconfig/floor.go) into the CLI - disclosed as unwired
// since the package's own introduction (PR #46).
//
// "verify" and "check" read a real settings.cjtr file via
// [loadSettingsFile] - [teamconfig.LoadSettings]'s own schema: a
// top-level "settings" mapping of name to {floor, direction}, going
// through the same format-version and signature pipeline every other
// .cjtr file uses ("Signed like any other .cjtr file"). "check-floor"
// stays flag-only - a lighter path for a value and floor that do not
// (yet) live in any settings.cjtr file.
func runSettings(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, settingsUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]

	switch subcommand {
	case "check-floor":
		return runSettingsCheckFloor(rest, stdout, stderr)
	case "verify":
		return runSettingsVerify(rest, stdout, stderr)
	case "check":
		return runSettingsCheck(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, settingsUsage)
		return 1
	}
}

// loadSettingsFile reads and strictly parses path as a settings.cjtr
// file: format version, the "signed" block (if present), and the
// "settings" mapping itself (see [teamconfig.LoadSettings]) - the
// same loading sequence loadFicha itself uses for a phenomenon file,
// minus everything settings.cjtr has no use for (area/phenomenon,
// concepts, rules, tables): "Signed like any other .cjtr file" means
// this reuses cjtr's own generic signing/versioning primitives rather
// than inventing settings-specific ones.
func loadSettingsFile(path string) (raw []byte, root *yamlstrict.Node, signed *cjtr.Signed, settings map[string]teamconfig.Setting, err error) {
	raw, err = os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	root, err = yamlstrict.Load(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	version, err := cjtr.FormatVersion(root)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := cjtr.CheckSupported(version); err != nil {
		return nil, nil, nil, nil, err
	}
	signed, err = cjtr.ReadSigned(root)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	settings, err = teamconfig.LoadSettings(root)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return raw, root, signed, settings, nil
}

// runSettingsVerify implements "conjecture settings verify
// <settings.cjtr>" - spec-authoring's own unconfirmed-NOTICE and
// signature-invalidation rules applied to a settings.cjtr file,
// mirroring runVerify's own handling of a phenomenon file, then lists
// every declared setting's floor and direction.
func runSettingsVerify(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, settingsUsage)
		return 1
	}

	raw, root, signed, settings, err := loadSettingsFile(args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	exitCode := 0
	if signed == nil || !signed.Confirmed {
		_, _ = fmt.Fprintln(stdout, "NOTICE: this settings file has not had human review (unconfirmed) and may have been shaped by the drafting model.")
	}
	if signed != nil {
		valid, err := cjtr.VerifySignature(raw, root)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			return 1
		}
		if !valid {
			_, _ = fmt.Fprintln(stdout, "SIGNATURE INVALID: the file's content no longer matches its recorded signature hash; it has been modified since it was signed.")
			exitCode = 1
		}
	}

	if len(settings) == 0 {
		_, _ = fmt.Fprintln(stdout, "no settings declared")
		return exitCode
	}
	names := make([]string, 0, len(settings))
	for name := range settings {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		s := settings[name]
		_, _ = fmt.Fprintf(stdout, "%s: floor=%v direction=%s\n", name, s.Floor, s.Direction)
	}
	return exitCode
}

// runSettingsCheck implements "conjecture settings check
// <settings.cjtr> <name> --local <value>" - the real end-to-end
// "floor, not ceiling" scenario: a local value checked against a
// team's own floor and direction, both read from a real file instead
// of flags.
func runSettingsCheck(args []string, stdout, stderr io.Writer) int {
	var path, name, localFlag string
	for i := 0; i < len(args); i++ {
		if args[i] == "--local" {
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, settingsUsage)
				return 1
			}
			localFlag = args[i]
			continue
		}
		switch {
		case path == "":
			path = args[i]
		case name == "":
			name = args[i]
		default:
			_, _ = fmt.Fprintln(stderr, settingsUsage)
			return 1
		}
	}
	if path == "" || name == "" || localFlag == "" {
		_, _ = fmt.Fprintln(stderr, settingsUsage)
		return 1
	}

	local, err := strconv.ParseFloat(localFlag, 64)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: --local %q is not a number\n", localFlag)
		return 1
	}

	_, _, _, settings, err := loadSettingsFile(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	setting, ok := settings[name]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "conjecture: %q is not declared in %s\n", name, path)
		return 1
	}

	if err := teamconfig.CheckFloor(name, local, setting.Floor, setting.Direction); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: rejected: %v\n", name, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s: accepted: local value %v meets the team's floor of %v\n", name, local, setting.Floor)
	return 0
}

// runSettingsCheckFloor parses its own arguments and reports whether
// local meets the team's floor in direction, via
// [teamconfig.CheckFloor]. local, floor, and direction are all
// operator-supplied facts about one already-configured setting
// somewhere in this codebase (a required case count, a tolerated
// violation rate, a risk weight's own ordinal) or a team's own
// candidate value for one - unlike "check", this does not read any
// settings.cjtr file, for a value and floor that do not (yet) live in
// one.
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
