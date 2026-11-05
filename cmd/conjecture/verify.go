package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// runVerify loads a .cjtr file and checks every hypothesis's declared
// examples for internal consistency (see cjtr.CheckExample). It does not
// yet check generated code against anything - that is the conformance
// cycle capability, not started. This only checks that a ficha's own
// examples agree with its own given/then/otherwise.
func runVerify(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: conjecture verify <file.cjtr>")
		return 1
	}
	path := args[0]

	f, err := os.Open(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = f.Close() }()

	root, err := yamlstrict.Load(f)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	version, err := cjtr.FormatVersion(root)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := cjtr.CheckSupported(version); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	// spec-authoring's "Confirmed flag gates downstream use": an
	// unconfirmed file's verdicts must carry a visible caveat that the
	// specification lacked human review. This is advisory, not a
	// failure - it is printed to stdout alongside the verdict, and does
	// not affect runVerify's exit code.
	signed, err := cjtr.ReadSigned(root)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if signed == nil || !signed.Confirmed {
		_, _ = fmt.Fprintln(stdout, "NOTICE: this specification has not had human review (unconfirmed) and may have been shaped by the drafting model.")
	}

	// Per the cjtr-language spec, a file's area is its directory
	// (conjecture/phenomena/<area>/<phenomenon>.cjtr) and its phenomenon
	// slug is its filename - neither comes from the file's content.
	area := filepath.Base(filepath.Dir(path))
	phenomenon := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	// A table concept's "table" path (e.g. "bases/shipping-table.csv") is
	// relative to the conjecture tree's root, the parent of "phenomena/"
	// - three directories above a phenomenon file at
	// phenomena/<area>/<phenomenon>.cjtr. The specification never states
	// this explicitly; it is the only reading that doesn't invent a path
	// convention of its own, since the worked example's own path comment
	// and its "bases/" directory both sit beside "phenomena/" that way.
	conjectureRoot := filepath.Dir(filepath.Dir(filepath.Dir(path)))

	concepts, err := cjtr.Concepts(root)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	tables, err := cjtr.LoadTables(concepts, conjectureRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	observation, err := cjtr.ReadObservation(root)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	// "scope" and "simplifications" are declarative documentation (see
	// [cjtr.Scope]'s doc comment) with no effect on verification yet, but
	// are still read and validated here so a malformed one is caught
	// rather than silently ignored, same as every other top-level field
	// this command reads.
	if _, err := cjtr.ReadScope(root); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if _, err := cjtr.ReadSimplifications(root); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	// A file-level "axioms"/"assumes" clause (see [cjtr.Axioms]'s doc
	// comment) is likewise only parsed and validated here, not yet
	// checked against anything, same as a rule-level one on [cjtr.Rule].
	if _, err := cjtr.Axioms(root, concepts); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	rules, err := cjtr.Rules(root, area, phenomenon, concepts, observation)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	exitCode := 0
	total, consistent := 0, 0

	// "user_defined_laws" at a file's top level (sibling to "rules", for
	// the whole phenomenon rather than one rule - see [cjtr.UserDefinedLaws]'s
	// doc comment) has no rule to scope it under, so it is checked against
	// the phenomenon's own composite ID (area::phenomenon) - the same
	// two-segment prefix the cjtr-language spec's "Prefix-based
	// verification scope" scenario already treats as a valid scope on its
	// own, not a new convention invented here.
	phenomenonID, err := cjtr.ComposeID(area, phenomenon)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	fileLaws, err := cjtr.UserDefinedLaws(root, concepts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	t, c, failed := checkHypotheses(fileLaws, phenomenonID, concepts, tables, stdout)
	total += t
	consistent += c
	if failed {
		exitCode = 1
	}

	for _, rule := range rules {
		hyps, err := cjtr.Hypotheses(rule.Body, concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %v\n", rule.CompositeID, err)
			exitCode = 1
			continue
		}

		laws, err := cjtr.UserDefinedLaws(rule.Body, concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %v\n", rule.CompositeID, err)
			exitCode = 1
			continue
		}
		hyps = append(hyps, laws...)

		t, c, failed := checkHypotheses(hyps, rule.CompositeID, concepts, tables, stdout)
		total += t
		consistent += c
		if failed {
			exitCode = 1
		}
	}

	_, _ = fmt.Fprintf(stdout, "\n%d/%d examples consistent\n", consistent, total)
	return exitCode
}

// checkHypotheses checks every hypothesis's (or user-defined law's)
// declared examples for internal consistency, reporting each one against
// compositeID, and returns how many were checked, how many were
// consistent, and whether any error or failure occurred.
func checkHypotheses(hyps []cjtr.Hypothesis, compositeID string, concepts map[string]cjtr.Concept, tables cjtr.Tables, stdout io.Writer) (total, consistent int, failed bool) {
	for _, h := range hyps {
		if len(h.Examples) == 0 {
			_, _ = fmt.Fprintf(stdout, "%s: %s: no examples to check\n", compositeID, h.Name)
			continue
		}
		for i, example := range h.Examples {
			total++
			ok, err := cjtr.CheckExample(h, concepts, tables, example)
			switch {
			case err != nil:
				_, _ = fmt.Fprintf(stdout, "%s: %s: example %d: ERROR: %v\n", compositeID, h.Name, i+1, err)
				failed = true
			case ok:
				consistent++
				_, _ = fmt.Fprintf(stdout, "%s: %s: example %d: PASS\n", compositeID, h.Name, i+1)
			default:
				_, _ = fmt.Fprintf(stdout, "%s: %s: example %d: FAIL (inconsistent with %v)\n", compositeID, h.Name, i+1, example)
				failed = true
			}
		}
	}
	return total, consistent, failed
}
