// Package llmorchestration implements pieces of the llm-orchestration
// capability (openspec/specs/llm-orchestration/spec.md): the
// drafting/implementing session-separation check, hidden-case
// withholding, and the trust-boundary gate that decides what an LLM
// actor may do directly versus only propose. It does not open or run
// any LLM session itself - no provider, API key, or network call is
// involved in anything here, only the policy a caller enforces around
// one.
package llmorchestration

import "fmt"

// Role is which side of the drafting/implementing split a Session was
// opened for.
type Role int

const (
	Drafter Role = iota
	Implementer
)

// Session identifies one LLM session and the role it was opened for.
type Session struct {
	ID   string
	Role Role
}

// RequireSeparateSessions enforces "drafting and implementing are
// always separate sessions": drafting must actually hold the Drafter
// role, implementing must hold the Implementer role, and the two must
// be distinct sessions. It does not and cannot check the specification's
// further requirement that the drafting session "has no visibility
// into the implementation approach" - that is a property of what
// information a real session was given, which this package, holding
// only session identities, has no way to inspect.
func RequireSeparateSessions(drafting, implementing Session) error {
	if drafting.Role != Drafter {
		return fmt.Errorf("llmorchestration: drafting session %q does not hold the Drafter role", drafting.ID)
	}
	if implementing.Role != Implementer {
		return fmt.Errorf("llmorchestration: implementing session %q does not hold the Implementer role", implementing.ID)
	}
	if drafting.ID == implementing.ID {
		return fmt.Errorf("llmorchestration: drafting and implementing sessions must be distinct, both are %q", drafting.ID)
	}
	return nil
}

// JudgeCase is one case a judge checks implementation code against.
// Hidden marks a case the implementing session is never shown.
type JudgeCase struct {
	Input  string
	Hidden bool
}

// VisibleCases returns the subset of cases an implementing session may
// see, withholding every case flagged Hidden ("the system does not
// provide the hidden cases it is being judged against").
func VisibleCases(cases []JudgeCase) []JudgeCase {
	var visible []JudgeCase
	for _, c := range cases {
		if !c.Hidden {
			visible = append(visible, c)
		}
	}
	return visible
}

// Actor is who is requesting an Action through the integration channel.
type Actor int

const (
	Human Actor = iota
	LLM
)

// Action is one thing that can be requested through the integration
// channel. SubmitProposal is the one action the specification says an
// LLM SHALL be able to do ("only proposals cross the boundary"); every
// other value here is one the specification names as off-limits for an
// LLM, regardless of which side of the drafting/implementing split it
// is on.
type Action int

const (
	SubmitProposal Action = iota
	WriteEvidenceRecord
	CallAdversarialSLMDirectly
	SignSpecOnBehalfOfAnotherUser
	ChangeConfiguration
	ConfirmSpecification
	PromoteHypothesis
	MarkVerdictConformant
)

// Allowed reports whether actor may perform action directly through
// the integration channel. A Human actor may perform any of these
// directly - none of the specification's restrictions here are about
// humans. An LLM actor may only SubmitProposal: everything else is
// something "no LLM involved in drafting or implementing SHALL be able
// to" do directly ("LLM stays outside the trust boundary"; "only
// proposals cross the boundary").
func Allowed(actor Actor, action Action) bool {
	if actor == Human {
		return true
	}
	return action == SubmitProposal
}
