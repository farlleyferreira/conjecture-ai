package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/expressioncore"
	"github.com/farlleyferreira/conjecture-ai/internal/omissiondetection"
)

const verifyUsage = "usage: conjecture verify <file.cjtr> [--scope <composite-id-prefix>] [--catalog <cat1,cat2,...> [--addressed <cat1,cat2,...>]]"

// runVerify loads a .cjtr file and checks every hypothesis's declared
// examples for internal consistency (see cjtr.CheckExample). It does not
// yet check generated code against anything - that is the conformance
// cycle capability, not started. This only checks that a ficha's own
// examples agree with its own given/then/otherwise.
//
// --scope wires cjtr-language's own "Prefix-based verification scope"
// scenario: "WHEN a user runs verification scoped to a composite ID
// prefix... THEN the system verifies exactly the nodes whose composite
// ID starts with that prefix." A "node" here is a rule (its own
// three-segment compositeID) or the file-level user_defined_laws group
// (the phenomenon's own two-segment compositeID, per this file's own
// long-standing comment on that placement) - matching is a literal
// string prefix test, nothing segment-aware, since the specification's
// own wording and worked example never ask for more than that. Every
// check this command runs against a hypothesis or law (falsifiability,
// example consistency, the solver's consistency check) is scoped this
// way when --scope is given; Type D omission reporting is not, since
// it checks a whole area's own category coverage, not individual
// rule/hypothesis nodes --scope names.
//
// --catalog/--addressed wire in omission-detection's Type D check
// ("requirement nobody thought of"): the catalog of requirement
// categories to check for, and which of them this file's own rules
// address, are both caller-supplied - nothing in the cjtr-language
// grammar lets a rule declare its own category, so this command does
// not guess one (see [omissiondetection.CheckTypeD]'s own doc
// comment). Omitting --catalog skips Type D entirely, the same as an
// empty catalog would: there is nothing configured to check for.
//
// Honesty note: this only sees the one phenomenon file given on the
// command line, not every file in its area - "the relevant area"
// [omissiondetection.CheckTypeD] reports against is only as complete
// as what this single run loaded. A category this file does not
// address but a sibling phenomenon file in the same area already does
// would still be reported here as missing; this command does not
// claim area-wide completeness beyond what it actually read.
func runVerify(args []string, stdout, stderr io.Writer) int {
	path, scope, catalog, addressed, ok := parseVerifyArgs(args)
	if !ok {
		_, _ = fmt.Fprintln(stderr, verifyUsage)
		return 1
	}

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
	var allHyps []cjtr.Hypothesis

	if strings.HasPrefix(f.phenomenonID, scope) {
		allHyps = append(allHyps, f.fileLaws...)

		t, c, failed := checkHypotheses(f.fileLaws, f.phenomenonID, concepts, tables, stdout)
		total += t
		consistent += c
		if failed {
			exitCode = 1
		}
	}

	for _, rule := range rules {
		if !strings.HasPrefix(rule.CompositeID, scope) {
			continue
		}

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
		allHyps = append(allHyps, hyps...)

		t, c, failed := checkHypotheses(hyps, rule.CompositeID, concepts, tables, stdout)
		total += t
		consistent += c
		if failed {
			exitCode = 1
		}
	}

	// spec-judge's "Consistency check via solver" - run once, across
	// every unconditional statement in the whole file, not per rule:
	// a genuine contradiction is just as real (and just as worth
	// catching) between two different rules as within one. See
	// checkConsistency's own doc comment for why only an unconditional
	// (IsUniversal) statement is ever included.
	if findings := checkConsistency(allHyps, concepts, expressioncore.Solver{}, stdout); len(findings) > 0 {
		exitCode = 1
	}

	// omission-detection's Type D ("requirement nobody thought of"):
	// reported, not a failure - the same posture Type A omissions
	// already take in check.go/implement.go (something to flag for a
	// human, not something that fails the run on its own).
	if len(catalog) > 0 {
		for _, omission := range omissiondetection.CheckTypeD(catalog, f.area, addressed) {
			_, _ = fmt.Fprintf(stdout, "%s: TYPE D OMISSION: category %q has no rule addressing it in this file's own area\n", f.phenomenonID, omission.Category)
		}
	}

	_, _ = fmt.Fprintf(stdout, "\n%d/%d examples consistent\n", consistent, total)
	return exitCode
}

// parseVerifyArgs parses runVerify's own arguments: the required
// file path, plus the optional --scope/--catalog/--addressed flags
// (see runVerify's own doc comment). ok is false on any malformed
// invocation - an unrecognized flag, a flag missing its value, more
// than one positional argument, or --addressed without --catalog
// (addressed categories mean nothing without a catalog to check them
// against). scope defaults to "", which matches every node ([strings.
// HasPrefix] against an empty prefix is always true) - unscoped,
// whole-file verification, exactly today's existing behavior.
func parseVerifyArgs(args []string) (path, scope string, catalog omissiondetection.Catalog, addressed map[string]bool, ok bool) {
	var catalogFlag, addressedFlag string
	sawCatalog := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scope", "--catalog", "--addressed":
			flag := args[i]
			i++
			if i >= len(args) {
				return "", "", nil, nil, false
			}
			switch flag {
			case "--scope":
				scope = args[i]
			case "--catalog":
				catalogFlag = args[i]
				sawCatalog = true
			case "--addressed":
				addressedFlag = args[i]
			}
		default:
			if path != "" {
				return "", "", nil, nil, false
			}
			path = args[i]
		}
	}
	if path == "" || (!sawCatalog && addressedFlag != "") {
		return "", "", nil, nil, false
	}

	if sawCatalog {
		catalog = omissiondetection.Catalog(splitNonEmpty(catalogFlag))
	}
	addressed = make(map[string]bool)
	for _, c := range splitNonEmpty(addressedFlag) {
		addressed[c] = true
	}
	return path, scope, catalog, addressed, true
}

// splitNonEmpty splits s on commas, dropping empty entries - so both
// "" and a trailing/leading/doubled comma produce no spurious blank
// category rather than an error this command would otherwise have to
// invent wording for.
func splitNonEmpty(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
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
