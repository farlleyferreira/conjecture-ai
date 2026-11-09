package main

import (
	"fmt"
	"io"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
)

const translateUsage = "usage: conjecture translate <file.cjtr>"

// runTranslate implements "conjecture translate" - spec-authoring's
// own "Bilingual comments carry deterministic translation"
// requirement: "the system can generate its # pt: translation
// deterministically from the vocabulary source, without invoking a
// model." [cjtr.GeneratePTComment] already does the generation; this
// command was the one piece missing, a way to actually ask for it.
//
// This only prints the generated "# pt:" comment for every statement
// in the file (every "given" clause, "then", and "otherwise" when
// present) next to the English statement it translates - it does not
// write the comment into the file itself. cjtr-language's own "system
// proposes a diff, never a direct edit" rule (Two-layer separation of
// statements and code) means an automatic file edit would need its own
// diff-review flow this codebase does not have yet; printing the
// translation for a human to copy in is the one mechanism this change
// does not have to invent.
func runTranslate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, translateUsage)
		return 1
	}
	path := args[0]

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	exitCode := 0
	exitCode = translateHypotheses(f.fileLaws, f.phenomenonID, stdout, stderr, exitCode)

	for _, rule := range f.rules {
		hyps, err := cjtr.Hypotheses(rule.Body, f.concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %v\n", rule.CompositeID, err)
			exitCode = 1
			continue
		}
		laws, err := cjtr.UserDefinedLaws(rule.Body, f.concepts)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %v\n", rule.CompositeID, err)
			exitCode = 1
			continue
		}
		exitCode = translateHypotheses(append(hyps, laws...), rule.CompositeID, stdout, stderr, exitCode)
	}

	return exitCode
}

// translateHypotheses prints a generated "# pt:" comment for every
// statement (each "given" clause, "then", and "otherwise" when
// present) of every hypothesis or law in hyps, reporting each against
// compositeID.
func translateHypotheses(hyps []cjtr.Hypothesis, compositeID string, stdout, stderr io.Writer, exitCode int) int {
	for _, h := range hyps {
		for _, stmt := range h.Given {
			exitCode = printTranslation(compositeID, h.Name, "given", stmt, stdout, stderr, exitCode)
		}
		thenLabel := "then"
		if h.IsUniversal {
			thenLabel = "always"
		}
		exitCode = printTranslation(compositeID, h.Name, thenLabel, h.Then, stdout, stderr, exitCode)
		if h.Otherwise != nil {
			exitCode = printTranslation(compositeID, h.Name, "otherwise", *h.Otherwise, stdout, stderr, exitCode)
		}
	}
	return exitCode
}

// printTranslation prints stmt's English form (labeled by clause)
// followed by its generated "# pt:" comment on an indented line
// beneath it.
func printTranslation(compositeID, hypothesisName, clause string, stmt cjtr.Statement, stdout, stderr io.Writer, exitCode int) int {
	_, _ = fmt.Fprintf(stdout, "%s: %s: %s: %s\n", compositeID, hypothesisName, clause, formatStatement(stmt))
	comment, err := cjtr.GeneratePTComment(stmt)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "  %s\n", comment)
	return exitCode
}
