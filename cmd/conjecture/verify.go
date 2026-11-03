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

	rules, err := cjtr.Rules(root, area, phenomenon, concepts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	exitCode := 0
	total, consistent := 0, 0

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

		for _, h := range hyps {
			if len(h.Examples) == 0 {
				_, _ = fmt.Fprintf(stdout, "%s: %s: no examples to check\n", rule.CompositeID, h.Name)
				continue
			}
			for i, example := range h.Examples {
				total++
				ok, err := cjtr.CheckExample(h, concepts, tables, example)
				switch {
				case err != nil:
					_, _ = fmt.Fprintf(stdout, "%s: %s: example %d: ERROR: %v\n", rule.CompositeID, h.Name, i+1, err)
					exitCode = 1
				case ok:
					consistent++
					_, _ = fmt.Fprintf(stdout, "%s: %s: example %d: PASS\n", rule.CompositeID, h.Name, i+1)
				default:
					_, _ = fmt.Fprintf(stdout, "%s: %s: example %d: FAIL (inconsistent with %v)\n", rule.CompositeID, h.Name, i+1, example)
					exitCode = 1
				}
			}
		}
	}

	_, _ = fmt.Fprintf(stdout, "\n%d/%d examples consistent\n", consistent, total)
	return exitCode
}
