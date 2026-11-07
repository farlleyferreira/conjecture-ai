package main

import (
	"context"
	"fmt"
	"io"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/expressioncore"
	"github.com/farlleyferreira/conjecture-ai/internal/specjudge"
)

// checkConsistency implements spec-judge's "Consistency check via
// solver": "The judge SHALL check statements within the provable
// subset for logical contradictions using an SMT solver, and SHALL
// report any contradiction found between two or more statements."
//
// The provable subset here is deliberately narrow: only an
// unconditional statement - an IsUniversal hypothesis's (or law's)
// own Then, never a Given/Otherwise or a conditional hypothesis's
// Then - is treated as an always-true fact about its term, safe to
// check for mutual contradiction with another such fact about the
// same term. A conditional hypothesis's Then only holds when its own
// Given holds; two conditional hypotheses asserting different things
// about the same term under different, possibly mutually exclusive
// Givens (the project's own canonical shipping-fee example: a
// subscription's fee and a regular order's fee) are not a
// contradiction at all, and this function must not report them as
// one. Each bound must also resolve to a constant (a literal, or a
// ConstantConcept) - a VariableConcept's value does not exist yet at
// judge time, before any implementation code or binding does.
//
// A solver error (most likely: no z3 install on this machine/CI -
// never exercised against a real one here, same disclosed gap as
// Docker throughout this project) is reported to stdout and skipped,
// never treated as a judging failure: whether this check can run at
// all depends on optional, external tooling, which must not make
// "conjecture verify" itself unusable without it.
func checkConsistency(hyps []cjtr.Hypothesis, concepts map[string]cjtr.Concept, solver expressioncore.Solver, stdout io.Writer) []specjudge.Finding {
	type universalStatement struct {
		hypothesisName string
		stmt           cjtr.Statement
	}
	byTerm := make(map[string][]universalStatement)
	for _, h := range hyps {
		if !h.IsUniversal {
			continue
		}
		byTerm[h.Then.Term] = append(byTerm[h.Then.Term], universalStatement{hypothesisName: h.Name, stmt: h.Then})
	}

	var findings []specjudge.Finding
	for term, stmts := range byTerm {
		if len(stmts) < 2 {
			continue
		}
		concept, ok := concepts[term]
		if !ok || concept.Type != "decimal" {
			continue
		}

		var names []string
		var constraints []string
		for _, s := range stmts {
			bound1, ok1 := resolveConstantDecimal(s.stmt.Value, s.stmt.ValueIsConcept, concepts)
			if !ok1 {
				continue
			}
			var bound2 expressioncore.Decimal
			if s.stmt.Value2 != "" {
				b2, ok2 := resolveConstantDecimal(s.stmt.Value2, s.stmt.Value2IsConcept, concepts)
				if !ok2 {
					continue
				}
				bound2 = b2
			}
			constraint, err := expressioncore.EncodeComparatorConstraint("x", s.stmt.Comparator, bound1, bound2)
			if err != nil {
				continue
			}
			names = append(names, s.hypothesisName)
			constraints = append(constraints, constraint)
		}
		if len(constraints) < 2 {
			continue
		}

		query := expressioncore.BuildSatisfiabilityQuery("x", constraints)
		result, err := solver.CheckSatisfiability(context.Background(), query)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "consistency check for %q: skipped: %v\n", term, err)
			continue
		}
		if finding := specjudge.ConsistencyFinding(names, result == expressioncore.Unsat); finding != nil {
			_, _ = fmt.Fprintf(stdout, "%q: CONTRADICTION: %s\n", term, finding.Message)
			findings = append(findings, *finding)
		}
	}
	return findings
}

// resolveConstantDecimal reports value as an [expressioncore.Decimal]
// when it resolves to a constant without needing any binding: a
// literal decimal text, or (when isConcept) a [cjtr.ConstantConcept]'s
// own declared value. Anything else (a VariableConcept or TableConcept
// reference, or text that is not a valid exact decimal) reports ok=false,
// since this check cannot resolve it before any implementation code
// or runtime binding exists.
func resolveConstantDecimal(value string, isConcept bool, concepts map[string]cjtr.Concept) (d expressioncore.Decimal, ok bool) {
	text := value
	if isConcept {
		concept, found := concepts[value]
		if !found || concept.Kind != cjtr.ConstantConcept {
			return expressioncore.Decimal{}, false
		}
		text = concept.Value
	}
	d, err := expressioncore.NewDecimal(text)
	if err != nil {
		return expressioncore.Decimal{}, false
	}
	return d, true
}
