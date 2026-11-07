package specjudge

import (
	"strings"
	"testing"
)

func TestConsistencyFinding_Satisfiable(t *testing.T) {
	if got := ConsistencyFinding([]string{"never-negative", "free-above"}, false); got != nil {
		t.Errorf("ConsistencyFinding(unsat=false) = %+v, want nil", got)
	}
}

func TestConsistencyFinding_Unsatisfiable(t *testing.T) {
	got := ConsistencyFinding([]string{"at-least-fifty", "at-most-ten"}, true)
	if got == nil {
		t.Fatal("ConsistencyFinding(unsat=true) = nil, want a Finding")
	}
	if got.Severity != Rejecting {
		t.Errorf("got.Severity = %v, want Rejecting", got.Severity)
	}
	for _, name := range []string{"at-least-fifty", "at-most-ten"} {
		if !strings.Contains(got.Message, name) {
			t.Errorf("got.Message = %q, want it to mention %q", got.Message, name)
		}
	}
}

func TestConsistencyFinding_ComposesWithJudge(t *testing.T) {
	findings := []Finding{}
	if f := ConsistencyFinding([]string{"ok-one", "ok-two"}, false); f != nil {
		findings = append(findings, *f)
	}
	if f := ConsistencyFinding([]string{"at-least-fifty", "at-most-ten"}, true); f != nil {
		findings = append(findings, *f)
	}

	if got := Judge(findings); got != Rejected {
		t.Errorf("Judge(findings) = %v, want Rejected (one group was contradictory)", got)
	}
}
