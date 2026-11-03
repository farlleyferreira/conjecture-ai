// Package tap parses the Test Anything Protocol: the other "standard
// result format" the language-agnostic-contracts capability (openspec/
// specs/language-agnostic-contracts/spec.md) names, alongside JUnit XML
// (see internal/junit). TAP is a real, pre-existing, line-oriented
// protocol (https://testanything.org) most languages' test runners can
// already emit (Perl's own test suite, TAP::Harness, prove, and TAP
// producers for Go, JavaScript, Python, and others); this package
// follows its published grammar, including its own documented
// tolerance for unrecognized lines (TAP output may be interleaved with
// a program's other output on the same stream, so a conforming consumer
// does not fail the whole parse over one line it doesn't recognize - it
// only reports an error for a line that looks like a test result or
// plan but is malformed).
package tap

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Result is a parsed TAP stream.
type Result struct {
	// Version is the declared "TAP version N" line's number, or 0 if the
	// stream never declared one (version 12 is the implied default when
	// omitted).
	Version int
	Plan    Plan
	Tests   []TestLine
	// BailOut is the reason given on a "Bail out!" line, or "" if none
	// occurred. A bail-out ends the run; this package still returns
	// whatever it parsed before that line.
	BailOut string
}

// Plan is a stream's "start..end" plan line.
type Plan struct {
	Declared   bool
	Start, End int
	Skip       bool
	SkipReason string
}

// TestLine is one "ok"/"not ok" result line.
type TestLine struct {
	// Number is the line's own test number, or - if the line omitted one
	// (a valid TAP line need not number its tests) - its position among
	// every test line seen so far, starting at 1, matching how TAP
	// consumers conventionally number an unnumbered line.
	Number      int
	OK          bool
	Description string
	// Directive is "SKIP", "TODO", or "" (TAP's only two directives).
	Directive       string
	DirectiveReason string
}

var (
	versionRe = regexp.MustCompile(`^TAP version (\d+)$`)
	planRe    = regexp.MustCompile(`(?i)^(\d+)\.\.(\d+)(?:\s*#\s*SKIP\b\s*(.*))?$`)
	testRe    = regexp.MustCompile(`^(not )?ok\b(?:\s+(\d+))?(?:\s+-?\s*(.*))?$`)
	bailRe    = regexp.MustCompile(`^Bail out!\s*(.*)$`)
	// directiveRe finds a trailing "# SKIP ..."/"# TODO ..." within a
	// test line's remainder - the only two directives TAP defines - so
	// it is never confused with a literal "#" that happens to appear
	// inside a test's own description.
	directiveRe = regexp.MustCompile(`(?i)#\s*(SKIP|TODO)\b\s*(.*)$`)
)

// Parse reads a TAP stream from r.
func Parse(r io.Reader) (Result, error) {
	var result Result
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "" || strings.HasPrefix(trimmed, "#"):
			continue
		case versionRe.MatchString(trimmed):
			m := versionRe.FindStringSubmatch(trimmed)
			n, err := strconv.Atoi(m[1])
			if err != nil {
				return Result{}, fmt.Errorf("tap: invalid version line %q: %w", line, err)
			}
			result.Version = n
		case planRe.MatchString(trimmed):
			m := planRe.FindStringSubmatch(trimmed)
			start, err := strconv.Atoi(m[1])
			if err != nil {
				return Result{}, fmt.Errorf("tap: invalid plan line %q: %w", line, err)
			}
			end, err := strconv.Atoi(m[2])
			if err != nil {
				return Result{}, fmt.Errorf("tap: invalid plan line %q: %w", line, err)
			}
			// TAP's documented "skip the whole plan" form is "1..0"
			// (an empty range), optionally with a "# SKIP ..." reason;
			// this generalizes to any empty range (end < start) rather
			// than only literally "1..0".
			result.Plan = Plan{Declared: true, Start: start, End: end, Skip: m[3] != "" || end < start, SkipReason: m[3]}
		case testRe.MatchString(trimmed):
			tc, err := parseTestLine(trimmed, len(result.Tests)+1)
			if err != nil {
				return Result{}, err
			}
			result.Tests = append(result.Tests, tc)
		case bailRe.MatchString(trimmed):
			m := bailRe.FindStringSubmatch(trimmed)
			result.BailOut = m[1]
			return result, nil // a bail-out ends the run; nothing after it is meaningful
		default:
			// TAP's own tolerance for unrecognized lines (see the
			// package doc comment): a line that is not a version, plan,
			// test, or bail-out line is a diagnostic, not an error.
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("tap: %w", err)
	}

	return result, nil
}

func parseTestLine(line string, defaultNumber int) (TestLine, error) {
	m := testRe.FindStringSubmatch(line)
	tc := TestLine{OK: m[1] == "", Number: defaultNumber}

	if m[2] != "" {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			return TestLine{}, fmt.Errorf("tap: invalid test number in %q: %w", line, err)
		}
		tc.Number = n
	}

	rest := m[3]
	if dm := directiveRe.FindStringSubmatchIndex(rest); dm != nil {
		tc.Description = strings.TrimSpace(rest[:dm[0]])
		tc.Directive = strings.ToUpper(rest[dm[2]:dm[3]])
		tc.DirectiveReason = strings.TrimSpace(rest[dm[4]:dm[5]])
	} else {
		tc.Description = strings.TrimSpace(rest)
	}

	return tc, nil
}
