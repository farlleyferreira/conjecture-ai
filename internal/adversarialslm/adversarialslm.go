// Package adversarialslm implements pieces of the adversarial-slm
// capability (openspec/specs/adversarial-slm/spec.md): default/reserve
// selection by family and license, the grammar-constrained output
// shape, and the blind-input boundary. It does not run any model -
// "runs locally, on the same machine... with no network dependency" is
// a property of an actual model invocation this codebase does not
// perform yet.
package adversarialslm

import (
	"encoding/json"
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/jsonschema"
)

// License is an SLM candidate's distribution license.
type License int

const (
	Permissive License = iota
	ResearchOnly
	NonCommercial
)

func (l License) String() string {
	switch l {
	case Permissive:
		return "permissive"
	case ResearchOnly:
		return "research-only"
	case NonCommercial:
		return "non-commercial"
	default:
		return "unknown license"
	}
}

// Candidate is one adversarial-SLM candidate: the facts SelectSLM needs
// to decide whether it may be used at all, and whether it collides with
// the code-generating model.
type Candidate struct {
	Name    string
	Family  string
	License License
}

// SelectSLM picks the adversarial SLM to use, given the configured
// default and reserve candidates and the code-generating model's own
// family.
//
// "Permissive licensing requirement": a candidate under a research-only
// or non-commercial license SHALL NOT be used as a default or reserve
// SLM - checked for both regardless of which one ends up selected,
// since the specification's own wording forbids configuring either role
// with one, not just whichever role is actually picked for a given run.
//
// "Vendor and family independence from the generator": def is used
// unless its family matches generatorFamily, in which case reserve is
// used instead ("family collision triggers reserve SLM").
func SelectSLM(def, reserve Candidate, generatorFamily string) (Candidate, error) {
	if def.License != Permissive {
		return Candidate{}, fmt.Errorf("adversarialslm: default SLM %q has a %v license, want permissive", def.Name, def.License)
	}
	if reserve.License != Permissive {
		return Candidate{}, fmt.Errorf("adversarialslm: reserve SLM %q has a %v license, want permissive", reserve.Name, reserve.License)
	}

	if def.Family == generatorFamily {
		return reserve, nil
	}
	return def, nil
}

// AttackInput is everything the adversarial SLM receives for one attack
// or audit. "Blind input": it SHALL receive only the assertion under
// test, the interface, and prior results - no field here carries the
// code-generating model's reasoning trace or chat history, by
// construction, rather than by a runtime check that could be bypassed.
type AttackInput struct {
	Assertion    string
	Interface    string
	PriorResults []AttackOutput
}

// AttackOutcome is the adversarial SLM's typed verdict on one attack.
// The specification requires the output be grammar-constrained, not a
// specific set of outcome values; this package reads the two the
// specification's own scenario names ("flags a potential violation" /
// implicitly, does not) as the closed set, rather than inventing a
// third state the specification never mentions.
type AttackOutcome int

const (
	NoViolationFound AttackOutcome = iota
	ViolationFound
)

func (o AttackOutcome) String() string {
	switch o {
	case NoViolationFound:
		return "no_violation_found"
	case ViolationFound:
		return "violation_found"
	default:
		return "unknown outcome"
	}
}

// AttackOutput is the adversarial SLM's parsed, grammar-conformant
// output for one attack.
type AttackOutput struct {
	Outcome        AttackOutcome
	Counterexample string
}

// attackOutputSchema is AttackOutput's defined output grammar
// ("grammar-constrained typed output... so results are machine-
// parseable without free-form text interpretation"): a closed JSON
// shape, so a malformed or free-form response is rejected outright by
// ParseAttackOutput rather than loosely interpreted.
const attackOutputSchema = `{
	"type": "object",
	"properties": {
		"outcome": {"type": "string", "enum": ["violation_found", "no_violation_found"]},
		"counterexample": {"type": "string"}
	},
	"required": ["outcome"],
	"additionalProperties": false
}`

// ParseAttackOutput parses raw against [attackOutputSchema], rejecting
// it outright ("the system SHALL reject that output rather than
// attempting to interpret it loosely") when it does not conform.
func ParseAttackOutput(raw []byte) (AttackOutput, error) {
	if err := jsonschema.Validate([]byte(attackOutputSchema), raw); err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: output does not conform to the grammar: %w", err)
	}

	var decoded struct {
		Outcome        string `json:"outcome"`
		Counterexample string `json:"counterexample"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: decoding output: %w", err)
	}

	var outcome AttackOutcome
	switch decoded.Outcome {
	case "violation_found":
		outcome = ViolationFound
	case "no_violation_found":
		outcome = NoViolationFound
	default:
		// Unreachable given the schema's own enum constraint above,
		// kept as a hard error rather than a silent default so a
		// change to the schema's enum without a matching case here
		// fails loudly instead of miscategorizing the outcome.
		return AttackOutput{}, fmt.Errorf("adversarialslm: unrecognized outcome %q", decoded.Outcome)
	}

	return AttackOutput{Outcome: outcome, Counterexample: decoded.Counterexample}, nil
}

// EvidenceFacts reports the two evidenceledger.Evidence-shaped facts an
// attack output contributes - attacked is always true (an attack was
// run), survived reports whether the hypothesis held up against it -
// without this package importing evidenceledger directly. This mirrors
// specjudge.FalsifiabilityFinding's own decoupling: the caller already
// holds an evidenceledger.Evidence value and only needs these two bools
// filled in.
//
// "SLM opinion never counts as proof on its own": this only shapes the
// fact into evidence-ledger's own vocabulary. It does not itself record
// anything, invalidate a law, or promote/demote a stage - a caller
// still has to append it as one evidence item among the ones a
// promotion decision weighs, same as every other evidence source.
func (o AttackOutput) EvidenceFacts() (attacked, survived bool) {
	return true, o.Outcome == NoViolationFound
}
