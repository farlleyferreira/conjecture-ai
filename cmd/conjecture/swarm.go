package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/adversarialslm"
	"github.com/farlleyferreira/conjecture-ai/internal/attackerswarm"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
)

const swarmUsage = "usage: conjecture swarm attack <file.cjtr> <hypothesis-id> --mode <off|attackers-only|attackers-plus-auditors> --budget <N>\n" +
	"       --attacker <family>|<runtime>|<model> (repeated, at least " +
	"5)\n" +
	"       [--auditor <family>|<runtime>|<model> (repeated, required for attackers-plus-auditors)]"

// swarmProducedBy is this command's own [evidenceledger.Evidence.
// ProducedBy] label - distinct from [adversarialSLMProducedBy], since
// a swarm result is a deterministic aggregate over several
// independent agents, not one agent's own direct finding.
const swarmProducedBy = "attacker swarm"

// runSwarm implements "conjecture swarm <subcommand>" - attacker-
// swarm's own real orchestration wired into the CLI for the first
// time: the package's own pure mode-validation, bandit allocator, and
// deterministic aggregator (all already built and tested) now
// actually dispatch real attacks, via the same local-model runners
// [adversarialslm] and "conjecture attack" already use.
//
// Honesty note: nothing in this codebase's own specification corpus
// defines what the five-plus "distinct attack strategies" the bandit
// allocates across concretely are (grepped every openspec/specs/
// file - "strategy"/"strategies" appears only in attacker-swarm's own
// requirement text and optimization-suite's own reference to it,
// never with a worked example). Inventing five canonical strategy
// names here would be guessing a taxonomy the specification never
// gives. Instead, a "strategy" is whatever the caller configures via
// --attacker: a named local model/runtime pairing - the same "caller
// supplies the fact" posture this codebase already takes for every
// other undefined concept this session has wired (evidence-ledger's
// own Theory/Law promotion conditions, context-engine's own
// isUserDefinedLaw). The bandit itself ([attackerswarm.AllocateNext])
// is strategy-agnostic by construction - it only ever sees Name/
// Pulls/Successes - so this reading changes nothing about how it
// works, only what a caller happens to name its own strategies.
func runSwarm(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, swarmUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "attack":
		return runSwarmAttack(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, swarmUsage)
		return 1
	}
}

// strategySpec is one --attacker/--auditor flag's own parsed value.
type strategySpec struct {
	family, runtime, model string
}

// parseStrategySpec parses "<family>|<runtime>|<model>" - family and
// runtime are required, model may be empty (llamacpp needs none).
func parseStrategySpec(s string) (strategySpec, error) {
	parts := strings.Split(s, "|")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return strategySpec{}, fmt.Errorf("%q is not \"<family>|<runtime>|<model>\"", s)
	}
	return strategySpec{family: parts[0], runtime: parts[1], model: parts[2]}, nil
}

// runSwarmAttack parses "attack"'s own arguments and loads the real
// file/store/runners - the thin, real-path wrapper around
// [swarmAttack].
func runSwarmAttack(args []string, stdout, stderr io.Writer) int {
	var path, hypothesisID, modeFlag string
	budget := 0
	var attackerFlags, auditorFlags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode", "--budget", "--attacker", "--auditor":
			flag := args[i]
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, swarmUsage)
				return 1
			}
			switch flag {
			case "--mode":
				modeFlag = args[i]
			case "--budget":
				if _, err := fmt.Sscanf(args[i], "%d", &budget); err != nil {
					_, _ = fmt.Fprintf(stderr, "conjecture: --budget %q is not a number\n", args[i])
					return 1
				}
			case "--attacker":
				attackerFlags = append(attackerFlags, args[i])
			case "--auditor":
				auditorFlags = append(auditorFlags, args[i])
			}
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 2 || budget <= 0 {
		_, _ = fmt.Fprintln(stderr, swarmUsage)
		return 1
	}
	path, hypothesisID = positional[0], positional[1]

	var mode attackerswarm.Mode
	switch modeFlag {
	case "", "off":
		// mode's own zero value is already attackerswarm.Off.
	case "attackers-only":
		mode = attackerswarm.AttackersOnly
	case "attackers-plus-auditors":
		mode = attackerswarm.AttackersPlusAuditors
	default:
		_, _ = fmt.Fprintf(stderr, "conjecture: %q is not a recognized --mode\n", modeFlag)
		return 1
	}
	if mode == attackerswarm.Off {
		_, _ = fmt.Fprintln(stdout, "swarm: off (the default) - nothing run")
		return 0
	}

	attackers, err := parseStrategySpecs(attackerFlags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: --attacker %v\n", err)
		return 1
	}
	if len(attackers) < attackerswarm.MinStrategies {
		_, _ = fmt.Fprintf(stderr, "conjecture: %d --attacker strategies given, want at least %d\n", len(attackers), attackerswarm.MinStrategies)
		return 1
	}
	auditors, err := parseStrategySpecs(auditorFlags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: --auditor %v\n", err)
		return 1
	}
	if mode == attackerswarm.AttackersPlusAuditors {
		if len(auditors) == 0 {
			_, _ = fmt.Fprintln(stderr, "conjecture: attackers-plus-auditors mode needs at least one --auditor")
			return 1
		}
		for _, a := range attackers {
			for _, j := range auditors {
				if err := attackerswarm.ValidateMode(mode, a.family, j.family); err != nil {
					_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
					return 1
				}
			}
		}
	}

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := openPromoteStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	resolveRunner := func(spec strategySpec) (attackRunner, error) {
		return buildAttackRunner(spec.runtime, spec.model, "")
	}
	return swarmAttack(context.Background(), db, f, hypothesisID, mode, budget, attackers, auditors, resolveRunner, stdout, stderr)
}

// parseStrategySpecs parses every flag in flags via
// [parseStrategySpec], stopping at the first malformed one.
func parseStrategySpecs(flags []string) ([]strategySpec, error) {
	specs := make([]strategySpec, 0, len(flags))
	for _, flag := range flags {
		spec, err := parseStrategySpec(flag)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// swarmAttack implements "conjecture swarm attack"'s own real logic
// against db: runs budget attack rounds, each one picking a strategy
// via the real multi-armed bandit ([attackerswarm.AllocateNext]) and
// dispatching a real, independent attack through it (same runner
// construction "conjecture attack" itself uses) - "no communication
// between attacking agents" holds by construction, since every
// round's own [adversarialslm.AttackInput] is built once, up front,
// from already-recorded history, never from another round's own
// finding within this same run. In attackers-plus-auditors mode,
// every configured auditor also runs once, each independently too,
// feeding the same final aggregate. The swarm's own deterministic
// aggregate ([attackerswarm.Aggregate]) is recorded as one evidence
// item - cross-family agreement reported as a signal in its own text,
// never folded into Survived/Attacked as if it settled anything by
// itself, per "cross-family agreement SHALL be treated as a signal,
// never as evidence by itself."
func swarmAttack(ctx context.Context, db *sql.DB, f loadedFicha, hypothesisID string, mode attackerswarm.Mode, budget int, attackerSpecs, auditorSpecs []strategySpec, resolveRunner func(strategySpec) (attackRunner, error), stdout, stderr io.Writer) int {
	h, compositeID, interfaceText, found := findHypothesisByID(f, hypothesisID)
	if !found {
		_, _ = fmt.Fprintf(stderr, "conjecture: no hypothesis with id %q in %s\n", hypothesisID, f.phenomenon)
		return 1
	}

	history, err := evidenceledger.History(db, compositeID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	input := buildAttackInput(h, interfaceText, history)

	strategies := make([]attackerswarm.Strategy, len(attackerSpecs))
	for i, s := range attackerSpecs {
		strategies[i] = attackerswarm.Strategy{Name: s.family}
	}

	var findings []attackerswarm.Finding
	for round := 0; round < budget; round++ {
		idx, err := attackerswarm.AllocateNext(strategies)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			return 1
		}
		spec := attackerSpecs[idx]
		runner, err := resolveRunner(spec)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: strategy %q: %v\n", spec.family, err)
			return 1
		}
		output, err := runner.Attack(ctx, input)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: round %d (strategy %q): %v\n", round, spec.family, err)
			return 1
		}

		violated := output.Outcome == adversarialslm.ViolationFound
		strategies[idx].Pulls++
		if violated {
			strategies[idx].Successes++
		}
		findings = append(findings, attackerswarm.Finding{AgentFamily: spec.family, Violated: violated})
		_, _ = fmt.Fprintf(stdout, "%s: round %d: strategy %q: %s\n", compositeID, round, spec.family, output.Outcome)
	}

	if mode == attackerswarm.AttackersPlusAuditors {
		for _, spec := range auditorSpecs {
			runner, err := resolveRunner(spec)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "conjecture: auditor %q: %v\n", spec.family, err)
				return 1
			}
			output, err := runner.Attack(ctx, input)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "conjecture: auditor %q: %v\n", spec.family, err)
				return 1
			}
			findings = append(findings, attackerswarm.Finding{AgentFamily: spec.family, Violated: output.Outcome == adversarialslm.ViolationFound})
			_, _ = fmt.Fprintf(stdout, "%s: auditor %q: %s\n", compositeID, spec.family, output.Outcome)
		}
	}

	result := attackerswarm.Aggregate(findings)

	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	evidence := evidenceledger.Evidence{
		HypothesisID: compositeID,
		ProducedBy:   swarmProducedBy,
		ValidityHash: evidenceledger.ValidityHash(specVersion, "", version.String(), mode.String(), ""),
	}
	if result.ViolationReported {
		evidence.Attacked = fmt.Sprintf("%s (%s, budget %d)", swarmProducedBy, mode, budget)
	} else {
		evidence.Attacked = fmt.Sprintf("%s (%s, budget %d)", swarmProducedBy, mode, budget)
		evidence.Survived = "no violation found by swarm"
	}
	if _, err := evidenceledger.Append(db, evidence); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%s: swarm result: violation reported: %v, cross-family agreement (signal only, not evidence by itself): %v\n",
		compositeID, result.ViolationReported, result.CrossFamilyAgreement)
	for i, s := range strategies {
		_, _ = fmt.Fprintf(stdout, "%s: strategy %q: %d/%d pulls found violations\n", compositeID, attackerSpecs[i].family, s.Successes, s.Pulls)
	}
	return 0
}
