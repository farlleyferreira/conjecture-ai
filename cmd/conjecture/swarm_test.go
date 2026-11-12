package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/adversarialslm"
	"github.com/farlleyferreira/conjecture-ai/internal/attackerswarm"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

// fakeResolver returns a resolveRunner closure backed by outcomes
// (family -> the AttackOutput that family's own fakeAttackRunner
// always returns), so swarmAttack can be tested without any real
// local model server.
func fakeResolver(outcomes map[string]adversarialslm.AttackOutput) func(strategySpec) (attackRunner, error) {
	return func(spec strategySpec) (attackRunner, error) {
		out, ok := outcomes[spec.family]
		if !ok {
			return nil, fmt.Errorf("no fake outcome configured for family %q", spec.family)
		}
		return &fakeAttackRunner{output: out}, nil
	}
}

func fiveAttackerSpecs() []strategySpec {
	return []strategySpec{
		{family: "alpha", runtime: "ollama", model: "m"},
		{family: "beta", runtime: "ollama", model: "m"},
		{family: "gamma", runtime: "ollama", model: "m"},
		{family: "delta", runtime: "ollama", model: "m"},
		{family: "epsilon", runtime: "ollama", model: "m"},
	}
}

func TestRunSwarm_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSwarm(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSwarm with no subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunSwarm_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSwarm([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSwarm with an unknown subcommand: exit code = %d, want 1", code)
	}
}

func TestRunSwarmAttack_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSwarm([]string{"attack", "file.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("swarm attack with one positional arg: exit code = %d, want 1", code)
	}
}

func TestRunSwarmAttack_ModeOffDoesNothing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSwarm([]string{"attack", "file.cjtr", "id", "--mode", "off", "--budget", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("swarm attack with --mode off: exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "off") {
		t.Errorf("stdout should report the swarm is off, got %q", stdout.String())
	}
}

func TestRunSwarmAttack_DefaultModeIsOff(t *testing.T) {
	// No --mode given at all - "disabled by default" - never touches
	// the real file/store.
	var stdout, stderr bytes.Buffer
	code := runSwarm([]string{"attack", "file.cjtr", "id", "--budget", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("swarm attack with no --mode: exit code = %d, want 0", code)
	}
}

func TestRunSwarmAttack_FewerThanMinStrategiesIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"attack", "file.cjtr", "id", "--mode", "attackers-only", "--budget", "5",
		"--attacker", "a|ollama|m", "--attacker", "b|ollama|m"}
	code := runSwarm(args, &stdout, &stderr)
	if code != 1 {
		t.Errorf("swarm attack with only 2 attackers: exit code = %d, want 1", code)
	}
}

func TestRunSwarmAttack_AttackersPlusAuditorsNeedsAnAuditor(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"attack", "file.cjtr", "id", "--mode", "attackers-plus-auditors", "--budget", "5"}
	for _, s := range fiveAttackerSpecs() {
		args = append(args, "--attacker", s.family+"|"+s.runtime+"|"+s.model)
	}
	code := runSwarm(args, &stdout, &stderr)
	if code != 1 {
		t.Errorf("attackers-plus-auditors with no --auditor: exit code = %d, want 1", code)
	}
}

func TestRunSwarmAttack_AuditorSameFamilyAsAttackerIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"attack", "file.cjtr", "id", "--mode", "attackers-plus-auditors", "--budget", "5"}
	for _, s := range fiveAttackerSpecs() {
		args = append(args, "--attacker", s.family+"|"+s.runtime+"|"+s.model)
	}
	args = append(args, "--auditor", "alpha|ollama|m") // same family as an attacker
	code := runSwarm(args, &stdout, &stderr)
	if code != 1 {
		t.Errorf("auditor sharing an attacker's family: exit code = %d, want 1", code)
	}
}

func TestParseStrategySpec_Valid(t *testing.T) {
	got, err := parseStrategySpec("alpha|ollama|qwen2.5:0.5b")
	if err != nil {
		t.Fatalf("parseStrategySpec returned unexpected error: %v", err)
	}
	want := strategySpec{family: "alpha", runtime: "ollama", model: "qwen2.5:0.5b"}
	if got != want {
		t.Errorf("parseStrategySpec = %+v, want %+v", got, want)
	}
}

func TestParseStrategySpec_LlamaCppNeedsNoModel(t *testing.T) {
	got, err := parseStrategySpec("alpha|llamacpp|")
	if err != nil {
		t.Fatalf("parseStrategySpec returned unexpected error: %v", err)
	}
	if got.model != "" {
		t.Errorf("got.model = %q, want empty", got.model)
	}
}

func TestParseStrategySpec_WrongPartCountIsError(t *testing.T) {
	if _, err := parseStrategySpec("alpha|ollama"); err == nil {
		t.Error("parseStrategySpec with 2 parts = nil error, want one")
	}
}

func TestParseStrategySpec_EmptyFamilyIsError(t *testing.T) {
	if _, err := parseStrategySpec("|ollama|m"); err == nil {
		t.Error("parseStrategySpec with an empty family = nil error, want one")
	}
}

func TestSwarmAttack_AllCleanReportsNoViolation(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	attackers := fiveAttackerSpecs()
	outcomes := map[string]adversarialslm.AttackOutput{
		"alpha": {Outcome: adversarialslm.NoViolationFound},
		"beta":  {Outcome: adversarialslm.NoViolationFound},
		"gamma": {Outcome: adversarialslm.NoViolationFound},
		"delta": {Outcome: adversarialslm.NoViolationFound},
		// epsilon deliberately unreachable this run (budget 5, 5
		// strategies, every strategy gets exactly its one bootstrap
		// pull - epsilon's own outcome is still needed too.
		"epsilon": {Outcome: adversarialslm.NoViolationFound},
	}

	var stdout, stderr bytes.Buffer
	code := swarmAttack(context.Background(), db, f, "fee-matches", attackerswarm.AttackersOnly, 5, attackers, nil, fakeResolver(outcomes), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("swarmAttack returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "violation reported: false") {
		t.Errorf("stdout should report no violation, got %q", stdout.String())
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].ProducedBy != swarmProducedBy {
		t.Errorf("ProducedBy = %q, want %q", history[0].ProducedBy, swarmProducedBy)
	}
	if history[0].Survived == "" {
		t.Error("Survived is empty, want it to report no violation")
	}
}

func TestSwarmAttack_AnyViolationReportsIt(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	attackers := fiveAttackerSpecs()
	outcomes := map[string]adversarialslm.AttackOutput{
		"alpha":   {Outcome: adversarialslm.NoViolationFound},
		"beta":    {Outcome: adversarialslm.ViolationFound, Counterexample: "x = -1"},
		"gamma":   {Outcome: adversarialslm.NoViolationFound},
		"delta":   {Outcome: adversarialslm.NoViolationFound},
		"epsilon": {Outcome: adversarialslm.NoViolationFound},
	}

	var stdout, stderr bytes.Buffer
	code := swarmAttack(context.Background(), db, f, "fee-matches", attackerswarm.AttackersOnly, 5, attackers, nil, fakeResolver(outcomes), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("swarmAttack returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "violation reported: true") {
		t.Errorf("stdout should report a violation, got %q", stdout.String())
	}

	history, err := evidenceledger.History(db, "demo-area::demo::shipping-rule::fee-matches")
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 || history[0].Survived != "" {
		t.Errorf("history = %+v, want one record with Survived empty (a violation was found)", history)
	}
}

func TestSwarmAttack_CrossFamilyAgreementReported(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	attackers := fiveAttackerSpecs()
	// Every family agrees (all clean) - at least two distinct clean
	// families is cross-family agreement.
	outcomes := map[string]adversarialslm.AttackOutput{
		"alpha":   {Outcome: adversarialslm.NoViolationFound},
		"beta":    {Outcome: adversarialslm.NoViolationFound},
		"gamma":   {Outcome: adversarialslm.NoViolationFound},
		"delta":   {Outcome: adversarialslm.NoViolationFound},
		"epsilon": {Outcome: adversarialslm.NoViolationFound},
	}

	var stdout, stderr bytes.Buffer
	code := swarmAttack(context.Background(), db, f, "fee-matches", attackerswarm.AttackersOnly, 5, attackers, nil, fakeResolver(outcomes), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("swarmAttack returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "cross-family agreement (signal only, not evidence by itself): true") {
		t.Errorf("stdout should report cross-family agreement as a signal, got %q", stdout.String())
	}
}

func TestSwarmAttack_AttackersPlusAuditorsRunsAuditorsToo(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	attackers := fiveAttackerSpecs()
	auditors := []strategySpec{{family: "zeta", runtime: "ollama", model: "m"}}
	outcomes := map[string]adversarialslm.AttackOutput{
		"alpha":   {Outcome: adversarialslm.NoViolationFound},
		"beta":    {Outcome: adversarialslm.NoViolationFound},
		"gamma":   {Outcome: adversarialslm.NoViolationFound},
		"delta":   {Outcome: adversarialslm.NoViolationFound},
		"epsilon": {Outcome: adversarialslm.NoViolationFound},
		// Only the auditor finds a violation.
		"zeta": {Outcome: adversarialslm.ViolationFound, Counterexample: "audited failure"},
	}

	var stdout, stderr bytes.Buffer
	code := swarmAttack(context.Background(), db, f, "fee-matches", attackerswarm.AttackersPlusAuditors, 5, attackers, auditors, fakeResolver(outcomes), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("swarmAttack returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "auditor \"zeta\"") {
		t.Errorf("stdout should mention the auditor ran, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "violation reported: true") {
		t.Errorf("stdout should report a violation (found only by the auditor), got %q", stdout.String())
	}
}

func TestSwarmAttack_UnknownHypothesisIDIsError(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)

	var stdout, stderr bytes.Buffer
	code := swarmAttack(context.Background(), db, f, "no-such-id", attackerswarm.AttackersOnly, 5, fiveAttackerSpecs(), nil, fakeResolver(nil), &stdout, &stderr)
	if code != 1 {
		t.Errorf("swarmAttack with an unknown id: exit code = %d, want 1", code)
	}
}

func TestSwarmAttack_EveryStrategyGetsItsBootstrapPullFirst(t *testing.T) {
	f, err := loadFicha(writeCheckFixture(t))
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	db := openCheckTestStore(t)
	attackers := fiveAttackerSpecs()
	outcomes := map[string]adversarialslm.AttackOutput{
		"alpha": {Outcome: adversarialslm.NoViolationFound}, "beta": {Outcome: adversarialslm.NoViolationFound},
		"gamma": {Outcome: adversarialslm.NoViolationFound}, "delta": {Outcome: adversarialslm.NoViolationFound},
		"epsilon": {Outcome: adversarialslm.NoViolationFound},
	}

	// Budget exactly equals the strategy count: AllocateNext's own
	// "never-pulled strategy first" rule means every strategy must
	// get exactly one pull, in order - a deterministic, fully
	// specified outcome worth asserting directly.
	var stdout, stderr bytes.Buffer
	code := swarmAttack(context.Background(), db, f, "fee-matches", attackerswarm.AttackersOnly, len(attackers), attackers, nil, fakeResolver(outcomes), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("swarmAttack returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	for _, s := range attackers {
		if !strings.Contains(stdout.String(), fmt.Sprintf("strategy %q: 0/1 pulls", s.family)) {
			t.Errorf("stdout should show strategy %q got exactly 1 pull, got:\n%s", s.family, stdout.String())
		}
	}
}
