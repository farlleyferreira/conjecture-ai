package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/adversarialslm"
	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
)

const attackUsage = "usage: conjecture attack <file.cjtr> <hypothesis-id> [--runtime ollama --model <name> | --runtime llamacpp] [--base-url <url>]"

// adversarialSLMProducedBy is the fixed [evidenceledger.Evidence.
// ProducedBy] prefix every record this command appends carries -
// attackHypothesis's own way of telling a past adversarial-SLM
// attack's evidence apart from any other producer's, when building
// [adversarialslm.AttackInput.PriorResults] back out of
// [evidenceledger.History].
const adversarialSLMProducedBy = "adversarial SLM"

// violationCounterexamplePrefix is the fixed text
// attackHypothesis writes before a found counterexample in
// [evidenceledger.Evidence.Attacked] - the same string
// attackInputFromHistory strips back off to recover the raw
// counterexample when rebuilding prior results. Keeping both uses of
// this literal in one place is what keeps them from drifting apart.
const violationCounterexamplePrefix = adversarialSLMProducedBy + " attack; found counterexample: "

// attackRunner is the subset of [adversarialslm.OllamaRunner]'s and
// [adversarialslm.LlamaCppRunner]'s shared behavior this command
// needs - a local interface (neither type itself declares one) so a
// test can inject a fake runner instead of needing a real local
// Ollama or llama-server instance running.
type attackRunner interface {
	Attack(ctx context.Context, input adversarialslm.AttackInput) (adversarialslm.AttackOutput, error)
}

// runAttack implements "conjecture attack" - adversarial-slm's own
// real runtime wired into the CLI for the first time: [adversarialslm.
// OllamaRunner] (since PR #73) and [adversarialslm.LlamaCppRunner]
// already called a real, locally running server; nothing in this
// codebase ever invoked either or recorded a result as evidence.
//
// "Blind input": the assertion sent to the model is built only from
// the named hypothesis's own Given/Then statements and its rule's
// declared "when" interface (see buildAttackInput) - never anything
// else from the file, and never the code-generating model's reasoning
// or chat history, since this command has no access to either in the
// first place.
//
// Honesty note: this does not implement SelectSLM's own default/
// reserve, family-collision selection - that needs a real, configured
// notion of "the code-generating model's family," which nothing in
// this codebase tracks yet; --runtime/--model/--base-url name exactly
// one runner directly instead. It also does not attempt attacker-
// swarm's own multi-agent modes, strategy budget, or bandit
// allocation - that capability's own separate, still-disclosed gap,
// layered on top of a single working attacker, which is what this
// command adds.
func runAttack(args []string, stdout, stderr io.Writer) int {
	var path, hypothesisID, runtime, model, baseURL string
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--runtime", "--model", "--base-url":
			flag := args[i]
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, attackUsage)
				return 1
			}
			switch flag {
			case "--runtime":
				runtime = args[i]
			case "--model":
				model = args[i]
			case "--base-url":
				baseURL = args[i]
			}
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 2 {
		_, _ = fmt.Fprintln(stderr, attackUsage)
		return 1
	}
	path, hypothesisID = positional[0], positional[1]

	runner, err := buildAttackRunner(runtime, model, baseURL)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	f, err := loadFicha(path)
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

	return attackHypothesis(context.Background(), db, f, hypothesisID, runner, attackSLMLabel(runtime, model), stdout, stderr)
}

// attackSLMLabel names which runner/model actually ran an attack -
// [evidenceledger.ValidityHash]'s own slmVersion dependency, so
// switching which local model answers for this hypothesis makes its
// past adversarial-SLM evidence visibly stale ([evidenceledger.Stale]),
// the same "a change to any of them is visible as evidence debt"
// property every other validity-hash dependency already gets.
func attackSLMLabel(runtime, model string) string {
	if runtime == "" {
		runtime = "ollama"
	}
	if model == "" {
		return runtime
	}
	return runtime + ":" + model
}

// buildAttackRunner resolves runtime/model/baseURL into a real
// [attackRunner] - "ollama" (the default, needing --model) or
// "llamacpp" (needing no model: llama-server already has exactly one
// loaded). An empty baseURL leaves each runner's own standard local
// default in place.
func buildAttackRunner(runtime, model, baseURL string) (attackRunner, error) {
	switch runtime {
	case "", "ollama":
		if model == "" {
			return nil, fmt.Errorf("--runtime ollama needs --model <name>")
		}
		r := adversarialslm.NewOllamaRunner(model)
		if baseURL != "" {
			r.BaseURL = baseURL
		}
		return r, nil
	case "llamacpp":
		r := adversarialslm.NewLlamaCppRunner()
		if baseURL != "" {
			r.BaseURL = baseURL
		}
		return r, nil
	default:
		return nil, fmt.Errorf("%q is not a recognized --runtime (want \"ollama\" or \"llamacpp\")", runtime)
	}
}

// attackHypothesis implements "conjecture attack"'s own real logic
// against db and runner - the thin, real-path wrapper around
// findHypothesisByID/buildAttackInput, so a test can inject a fake
// runner and a temporary store instead of a real local model server
// and the real per-OS one [persistence.OpenStore] would otherwise
// open.
func attackHypothesis(ctx context.Context, db *sql.DB, f loadedFicha, hypothesisID string, runner attackRunner, slmLabel string, stdout, stderr io.Writer) int {
	h, compositeID, interfaceText, isLaw, found := findHypothesisByID(f, hypothesisID)
	if !found {
		_, _ = fmt.Fprintf(stderr, "conjecture: no hypothesis with id %q in %s\n", hypothesisID, f.phenomenon)
		return 1
	}
	if isLaw {
		_, _ = fmt.Fprintf(stdout, "%s: WARNING: this is a law (always Critical weight) - a single local-model opinion is weaker evidence than that stakes warrant; consider \"conjecture swarm attack\" for diversified coverage instead\n", compositeID)
	}

	history, err := evidenceledger.History(db, compositeID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	input := buildAttackInput(h, interfaceText, history)

	output, err := runner.Attack(ctx, input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	// Carries forward whatever stage this hypothesis already stands
	// at - the same "stage = previous.Stage" rule check.go's own
	// recordEvidence already applies - rather than leaving Stage at
	// its zero value (Hypothesis), which would silently demote a
	// hypothesis already at Thesis or beyond every time an attack is
	// recorded against it.
	stage := evidenceledger.Hypothesis
	if len(history) > 0 {
		stage = history[len(history)-1].Stage
	}

	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	evidence := evidenceledger.Evidence{
		HypothesisID: compositeID,
		Stage:        stage,
		ProducedBy:   adversarialSLMProducedBy,
		ValidityHash: evidenceledger.ValidityHash(specVersion, "", version.String(), slmLabel, ""),
	}
	if isLaw {
		// Disclosed the same way check.go's own unrunRigorLevelsNote
		// discloses what a verdict's own rigor level did not cover -
		// this evidence is a single, correlated-risk local-model
		// opinion, weaker than a law's always-Critical weight warrants.
		evidence.Unchecked = "law attacked by a single local model (one strategy) - consider \"conjecture swarm attack\" for diversified coverage"
	}
	switch output.Outcome {
	case adversarialslm.ViolationFound:
		evidence.Attacked = violationCounterexamplePrefix + output.Counterexample
	case adversarialslm.NoViolationFound:
		evidence.Attacked = adversarialSLMProducedBy + " attack"
		evidence.Survived = "no violation found by adversarial SLM"
	}
	// "SLM opinion never counts as proof on its own": this appends one
	// evidence item, exactly as every other evidence producer in this
	// codebase does - it does not promote/demote a stage or otherwise
	// treat the finding as settling anything by itself.
	if _, err := evidenceledger.Append(db, evidence); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%s: adversarial SLM: %s\n", compositeID, output.Outcome)
	if output.Outcome == adversarialslm.ViolationFound {
		_, _ = fmt.Fprintf(stdout, "%s: counterexample: %s\n", compositeID, output.Counterexample)
	}
	return 0
}

// findHypothesisByID searches f's own file-level laws and every
// rule's hypotheses and user-defined laws for one whose own declared
// ID matches hypothesisID, returning it alongside the composite ID
// [evidenceledger.Evidence.HypothesisID] already uses for it (the
// same compositeID+"::"+h.ID scheme check.go's own recordEvidence
// already builds), its rule's own declared "when" interface ("" for a
// file-level law, which has no associated rule), and whether it was
// found among the file's laws rather than its plain hypotheses - see
// [attackHypothesis]'s own law-weight warning for why that distinction
// matters to this command specifically.
func findHypothesisByID(f loadedFicha, hypothesisID string) (h cjtr.Hypothesis, compositeID, interfaceText string, isLaw, found bool) {
	for _, candidate := range f.fileLaws {
		if candidate.ID == hypothesisID {
			return candidate, f.phenomenonID + "::" + candidate.ID, "", true, true
		}
	}
	for _, rule := range f.rules {
		hyps, err := cjtr.Hypotheses(rule.Body, f.concepts)
		if err == nil {
			for _, candidate := range hyps {
				if candidate.ID == hypothesisID {
					return candidate, rule.CompositeID + "::" + candidate.ID, rule.When, false, true
				}
			}
		}
		laws, err := cjtr.UserDefinedLaws(rule.Body, f.concepts)
		if err == nil {
			for _, candidate := range laws {
				if candidate.ID == hypothesisID {
					return candidate, rule.CompositeID + "::" + candidate.ID, rule.When, true, true
				}
			}
		}
	}
	return cjtr.Hypothesis{}, "", "", false, false
}

// buildAttackInput builds h's own [adversarialslm.AttackInput] -
// "Blind input": Assertion is built only from h's own Given/Then
// statements (via formatStatement, the same rendering
// judgeFalsifiability already uses for its own messages), Interface
// is interfaceText (empty for a file-level law), and PriorResults is
// every past evidence record for this hypothesis this command itself
// produced (identified by [adversarialSLMProducedBy], never a rigor-
// level conformance record, which uses the same free-text Attacked
// field for an unrelated purpose) - nothing about the code-generating
// model's reasoning or chat history is available to build from in the
// first place.
func buildAttackInput(h cjtr.Hypothesis, interfaceText string, history []evidenceledger.Evidence) adversarialslm.AttackInput {
	var assertion string
	if h.IsUniversal {
		assertion = "always, " + formatStatement(h.Then)
	} else {
		given := make([]string, len(h.Given))
		for i, g := range h.Given {
			given[i] = formatStatement(g)
		}
		assertion = fmt.Sprintf("given %s, then %s", strings.Join(given, " and "), formatStatement(h.Then))
	}

	var prior []adversarialslm.AttackOutput
	for _, e := range history {
		if e.ProducedBy != adversarialSLMProducedBy {
			continue
		}
		if e.Survived != "" {
			prior = append(prior, adversarialslm.AttackOutput{Outcome: adversarialslm.NoViolationFound})
			continue
		}
		prior = append(prior, adversarialslm.AttackOutput{
			Outcome:        adversarialslm.ViolationFound,
			Counterexample: strings.TrimPrefix(e.Attacked, violationCounterexamplePrefix),
		})
	}

	return adversarialslm.AttackInput{Assertion: assertion, Interface: interfaceText, PriorResults: prior}
}
