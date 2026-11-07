package main

import (
	"fmt"
	"io"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
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

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	raw, root, signed := f.raw, f.root, f.signed

	// spec-authoring's "Confirmed flag gates downstream use": an
	// unconfirmed file's verdicts must carry a visible caveat that the
	// specification lacked human review. This is advisory, not a
	// failure - it is printed to stdout alongside the verdict, and does
	// not affect runVerify's exit code.
	if signed == nil || !signed.Confirmed {
		_, _ = fmt.Fprintln(stdout, "NOTICE: this specification has not had human review (unconfirmed) and may have been shaped by the drafting model.")
	}

	// spec-authoring's signature invalidation rule: "any part of a
	// .cjtr file outside the signed block changes" -> "the recorded
	// signature no longer matches the file's content hash." Unlike the
	// unconfirmed NOTICE above (advisory only), a signed file whose
	// content no longer matches its recorded hash has had its
	// authorship record falsified by the edit, so this is a hard
	// failure, not a caveat: the signature exists specifically to make
	// that tampering detectable.
	var signatureInvalid bool
	if signed != nil {
		valid, err := cjtr.VerifySignature(raw, root)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			return 1
		}
		if !valid {
			_, _ = fmt.Fprintln(stdout, "SIGNATURE INVALID: the file's content no longer matches its recorded signature hash; it has been modified since it was signed.")
			signatureInvalid = true
		}
	}

	concepts, tables, rules := f.concepts, f.tables, f.rules

	exitCode := 0
	if signatureInvalid {
		exitCode = 1
	}
	total, consistent := 0, 0

	// "user_defined_laws" at a file's top level (sibling to "rules", for
	// the whole phenomenon rather than one rule - see [cjtr.UserDefinedLaws]'s
	// doc comment) has no rule to scope it under, so it is checked against
	// the phenomenon's own composite ID (area::phenomenon) - the same
	// two-segment prefix the cjtr-language spec's "Prefix-based
	// verification scope" scenario already treats as a valid scope on its
	// own, not a new convention invented here.
	t, c, failed := checkHypotheses(f.fileLaws, f.phenomenonID, concepts, tables, stdout)
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
		// spec-judge's "Falsifiability check": "The judge SHALL reject
		// any hypothesis that is not written in a way that could be
		// disproven by a concrete case." Checked against each
		// statement's own term's declared domain (see
		// [cjtr.Falsifiable]'s doc comment for what this does and does
		// not catch) - not an advisory notice like the unconfirmed-
		// specification one above, since the specification's own
		// wording is "reject," not "flag."
		for _, stmt := range unfalsifiabilityCandidates(h) {
			ok, err := cjtr.Falsifiable(stmt, concepts)
			switch {
			case err != nil:
				_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR checking falsifiability: %v\n", compositeID, h.Name, err)
				failed = true
			case !ok:
				_, _ = fmt.Fprintf(stdout, "%s: %s: UNFALSIFIABLE: %q holds for every value %q's own declared domain allows\n", compositeID, h.Name, formatStatement(stmt), stmt.Term)
				failed = true
			}
		}

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

// unfalsifiabilityCandidates returns the statements of h worth checking
// for falsifiability: Then always, and Otherwise when h declares one.
// Given is deliberately excluded - it is h's existence condition, not
// a claim the hypothesis makes, so the specification's falsifiability
// requirement does not apply to it the same way.
func unfalsifiabilityCandidates(h cjtr.Hypothesis) []cjtr.Statement {
	stmts := []cjtr.Statement{h.Then}
	if h.Otherwise != nil {
		stmts = append(stmts, *h.Otherwise)
	}
	return stmts
}

// formatStatement renders stmt approximately as it was written, for
// use in a diagnostic message - not a parser round-trip guarantee.
func formatStatement(stmt cjtr.Statement) string {
	value := stmt.Value
	if stmt.ValueIsConcept {
		value = "the " + value
	}
	if stmt.Value2 == "" {
		return fmt.Sprintf("the %s %s %s", stmt.Term, stmt.Comparator, value)
	}
	value2 := stmt.Value2
	if stmt.Value2IsConcept {
		value2 = "the " + value2
	}
	return fmt.Sprintf("the %s %s %s and %s", stmt.Term, stmt.Comparator, value, value2)
}
