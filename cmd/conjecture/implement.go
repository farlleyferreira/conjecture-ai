package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/llmclient"
	"github.com/farlleyferreira/conjecture-ai/internal/omissiondetection"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

const implementUsage = "usage: conjecture implement [--provider <name>] [--max-rounds <N>] <file.cjtr> <rule-id> <hypothesis-id>"

// implementDefaultMaxRounds is this command's own default for
// conformance-cycle's "configured maximum number of judge-to-
// implementer rounds" - the specification gives no number, so this is
// a caller-overridable default (--max-rounds), not a value the
// specification names.
const implementDefaultMaxRounds = 5

// implementImage is the Docker image the generated implementation
// code runs in. This command always asks the implementer LLM for a
// single self-contained Python 3 script (implementCodeFileName), so
// unlike "conjecture check" - which runs arbitrary caller-supplied
// code and therefore takes the image as an argument - this command
// fixes the image itself, since it is the one that decides what
// language the code it generates is written in. Python was picked for
// having no separate compile step: one sandboxed "python3 <file>" run
// per round, nothing to build first.
const implementImage = "python:3-slim"

// implementCodeFileName is the generated script's name inside the
// sandboxed working directory - conformance-cycle's own
// harness.go reads/writes conjecture_input.json/conjecture_output.json
// in that same directory, which the generated script must honor.
const implementCodeFileName = "impl.py"

// implementerSystemPrompt teaches the model the one real contract its
// generated code must satisfy: internal/conformancecycle's own
// invented JSON input/output file convention (harness.go) - a flat
// JSON object of a concept's own "variable" path to string value, read
// from conjecture_input.json and written to conjecture_output.json
// before the process exits. This is this project's own invention
// (harness.go's own doc comment - nothing in any specification names
// an input delivery mechanism), not something a real integration's
// adapter would necessarily use - picked here only so this command's
// own generated code has something concrete and already-testable to
// target.
const implementerSystemPrompt = `You write a single self-contained Python 3 script implementing a described business rule.

Contract (this project's own convention - follow it exactly):
- Read a JSON object from "conjecture_input.json" in the current directory: a flat mapping of dotted variable names (e.g. "input.subtotal") to their value, always as a JSON string even for numbers - parse numeric values yourself (use the decimal module for any monetary or otherwise exact-decimal value, never float, to avoid rounding error).
- Compute whatever output variables the rule describes.
- Write a JSON object to "conjecture_output.json" in the current directory: a flat mapping of dotted output variable names (e.g. "output.shipping_fee") to their value, rendered back as a JSON string with the same number of decimal places the rule's own examples use.
- Exit 0 on success. Exit non-zero only if the input is genuinely malformed.

Output ONLY the Python source code - no markdown code fences, no commentary before or after it.`

func runImplement(args []string, stdout, stderr io.Writer) int {
	provider := "anthropic"
	maxRounds := implementDefaultMaxRounds
	var positional []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, implementUsage)
				return 1
			}
			provider = args[i]
		case "--max-rounds":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, implementUsage)
				return 1
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				_, _ = fmt.Fprintf(stderr, "conjecture: --max-rounds must be a positive integer, got %q\n", args[i])
				return 1
			}
			maxRounds = n
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 3 {
		_, _ = fmt.Fprintln(stderr, implementUsage)
		return 1
	}
	path, ruleID, hypothesisID := positional[0], positional[1], positional[2]

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	rule, h, isLaw, err := findHypothesis(f, ruleID, hypothesisID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	client, err := llmclient.New(provider)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := evidenceledger.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	// workqueue/userdefinedlaws schemas are required here for the same
	// reason "conjecture check" ensures them: recordEvidence (shared
	// with check.go) enqueues a blocker item on NonConformant and, when
	// isLaw, updates a law's status - both need their own table to
	// already exist, which this path never ensured before this change
	// (a pre-existing gap: the enqueue in #89 only wired "check", not
	// "implement", so it silently no-op'd here until now).
	if err := workqueue.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	sb := &sandbox.Sandbox{}
	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	verdict, err := runImplementLoop(context.Background(), db, sb, client, rule, h, maxRounds, specVersion, isLaw, f.concepts, f.tables, stdout)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if verdict != conformancecycle.Conformant {
		return 1
	}
	return 0
}

// runImplementLoop is conformance-cycle's own "loop that checks
// whether implementation code conforms... from first run through
// counterexample feedback to a final verdict": each round asks the
// implementer for code (the first round from scratch, every later
// round revising the previous attempt against the counterexample the
// last round found), runs it for real via
// [conformancecycle.CheckHypothesisDirectedSampling], and stops at the
// first Conformant result, an Inconclusive one (nothing to revise
// against), or when maxRounds is reached - "Round and token limits":
// "the cycle stops and reports the outcome as not conformant rather
// than continuing indefinitely."
//
// Honesty note: this enforces only the round half of conformance-
// cycle's budget, not the token half - none of this codebase's
// llmclient.Client implementations report how many tokens a call
// used, so there is nothing real to count. Tracking it would need
// extending that interface first, not inventing a number here.
func runImplementLoop(
	ctx context.Context,
	db *sql.DB,
	sb *sandbox.Sandbox,
	client llmclient.Client,
	rule cjtr.Rule,
	h cjtr.Hypothesis,
	maxRounds int,
	specVersion string,
	isLaw bool,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	stdout io.Writer,
) (conformancecycle.Verdict, error) {
	// omission-detection's "Type A - asserted but not implemented":
	// checked once, up front - cases do not depend on any round's own
	// generated code, only on h/concepts - since asking an implementer
	// to revise code against a hypothesis no case will ever exercise
	// would spend rounds (and, eventually, tokens) on something that
	// can never be demonstrated conformant either way.
	var cases []cjtr.Bindings
	if len(h.Examples) > 0 {
		var err error
		cases, err = conformancecycle.DirectedSamplingInputs(h, concepts)
		if err != nil {
			return 0, err
		}
	}
	if _, omitted := omissiondetection.CheckTypeA(rule.CompositeID+"::"+h.ID, len(cases)); omitted {
		_, _ = fmt.Fprintf(stdout, "TYPE A OMISSION: %s has no worked example or no input concept with a declared domain to sample - nothing to implement against\n", h.Name)
		return conformancecycle.Inconclusive, nil
	}

	budget := conformancecycle.Budget{MaxRounds: maxRounds, MaxTokens: math.MaxInt}

	dir, err := os.MkdirTemp("", "conjecture-implement")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	codePath := dir + string(os.PathSeparator) + implementCodeFileName

	opts := sandbox.Options{Image: implementImage, Dir: dir, Command: []string{"python3", implementCodeFileName}}

	var previousCode string
	var previousCounterexample *conformancecycle.Counterexample
	var verdict conformancecycle.Verdict
	exhausted := true

	for round := 0; !budget.Exhausted(round, 0); round++ {
		userPrompt := implementUserPrompt(rule, h, concepts, previousCode, previousCounterexample)
		code, err := client.Complete(ctx, implementerSystemPrompt, userPrompt)
		if err != nil {
			return 0, fmt.Errorf("round %d: asking the implementer for code: %w", round, err)
		}
		code = stripCodeFence(code)

		if err := os.WriteFile(codePath, []byte(code), 0o644); err != nil {
			return 0, fmt.Errorf("round %d: writing generated code: %w", round, err)
		}
		previousCode = code

		var counterexample *conformancecycle.Counterexample
		verdict, counterexample, err = conformancecycle.CheckHypothesisDirectedSampling(ctx, sb, opts, h, concepts, tables)
		if err != nil {
			return 0, fmt.Errorf("round %d: running conformance check: %w", round, err)
		}
		_, _ = fmt.Fprintf(stdout, "round %d: %s\n", round, verdict)

		if verdict == conformancecycle.Conformant || verdict == conformancecycle.Inconclusive {
			exhausted = false
			break
		}
		previousCounterexample = counterexample
	}

	// "the cycle stops and reports the outcome as not conformant rather
	// than continuing indefinitely" - reaching the round limit without
	// ever seeing Conformant or Inconclusive reports
	// VerdictOnBudgetExhausted, which is NonConformant by the
	// specification's own wording, not whatever the last round's own
	// verdict happened to be computed as (the two agree today, since
	// the loop only ever exits this way after a NonConformant round,
	// but this states the rule explicitly rather than relying on that
	// coincidence).
	if exhausted {
		verdict = conformancecycle.VerdictOnBudgetExhausted
	}

	recordEvidence(db, rule.CompositeID, h, verdict, previousCounterexample, implementImage, specVersion, isLaw, stdout)
	return verdict, nil
}

// implementUserPrompt renders the request this round sends the
// implementer: the hypothesis and its declaring rule's own given/then/
// otherwise/examples on the first round, or the previous round's code
// plus the counterexample that violated it on every later one.
func implementUserPrompt(rule cjtr.Rule, h cjtr.Hypothesis, concepts map[string]cjtr.Concept, previousCode string, counterexample *conformancecycle.Counterexample) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Rule: %s\n", rule.Name)
	fmt.Fprintf(&b, "Hypothesis: %s\n", h.Name)
	for _, stmt := range h.Given {
		fmt.Fprintf(&b, "Given: %s\n", formatStatement(stmt))
	}
	if h.IsUniversal {
		fmt.Fprintf(&b, "Always: %s\n", formatStatement(h.Then))
	} else {
		fmt.Fprintf(&b, "Then: %s\n", formatStatement(h.Then))
		if h.Otherwise != nil {
			fmt.Fprintf(&b, "Otherwise: %s\n", formatStatement(*h.Otherwise))
		}
	}
	for _, example := range h.Examples {
		fmt.Fprintf(&b, "Example: %v\n", example)
	}
	b.WriteString("\nConcepts (name: variable or value, type):\n")
	for _, name := range sortedConceptNames(concepts) {
		c := concepts[name]
		switch c.Kind {
		case cjtr.VariableConcept:
			fmt.Fprintf(&b, "- %s: variable %s, type %s\n", name, c.Variable, c.Type)
		case cjtr.ConstantConcept:
			fmt.Fprintf(&b, "- %s: fixed value %s\n", name, c.Value)
		}
	}

	if previousCode != "" {
		b.WriteString("\nYour previous attempt:\n")
		b.WriteString(previousCode)
		b.WriteString("\n\nIt was INCORRECT. ")
		if counterexample != nil {
			fmt.Fprintf(&b, "A conformance check found this failing case: %v\n", counterexample.Value)
		}
		b.WriteString("Fix it. Output the complete corrected script, not a diff.\n")
	}

	return b.String()
}

func sortedConceptNames(concepts map[string]cjtr.Concept) []string {
	names := make([]string, 0, len(concepts))
	for name := range concepts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// findHypothesis locates the hypothesis (or user-defined law) declared
// ruleID/hypothesisID names, returning the rule it belongs to (for
// recordEvidence's own compositeID::id scoping) and whether it was
// declared as a law rather than a plain hypothesis (so the caller can
// thread that through to recordEvidence's own isLaw parameter).
func findHypothesis(f loadedFicha, ruleID, hypothesisID string) (cjtr.Rule, cjtr.Hypothesis, bool, error) {
	for _, rule := range f.rules {
		if rule.ID != ruleID {
			continue
		}
		hyps, err := cjtr.Hypotheses(rule.Body, f.concepts)
		if err != nil {
			return cjtr.Rule{}, cjtr.Hypothesis{}, false, err
		}
		laws, err := cjtr.UserDefinedLaws(rule.Body, f.concepts)
		if err != nil {
			return cjtr.Rule{}, cjtr.Hypothesis{}, false, err
		}
		for _, h := range hyps {
			if h.ID == hypothesisID {
				return rule, h, false, nil
			}
		}
		for _, h := range laws {
			if h.ID == hypothesisID {
				return rule, h, true, nil
			}
		}
		return cjtr.Rule{}, cjtr.Hypothesis{}, false, fmt.Errorf("rule %q has no hypothesis with id %q", ruleID, hypothesisID)
	}
	return cjtr.Rule{}, cjtr.Hypothesis{}, false, fmt.Errorf("no rule with id %q", ruleID)
}
