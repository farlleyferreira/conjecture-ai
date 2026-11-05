package specjudge

import "testing"

func TestVerdict_StringAndParseRoundTrip(t *testing.T) {
	for _, v := range []Verdict{Accepted, Pending, Rejected} {
		s := v.String()
		parsed, err := ParseVerdict(s)
		if err != nil {
			t.Fatalf("ParseVerdict(%q) returned unexpected error: %v", s, err)
		}
		if parsed != v {
			t.Errorf("Verdict(%d).String() = %q, which parses back to %v", int(v), s, parsed)
		}
	}
}

func TestParseVerdict_Unrecognized(t *testing.T) {
	_, err := ParseVerdict("approved")
	if err == nil {
		t.Fatal("ParseVerdict(\"approved\") = nil error, want an error")
	}
}

func TestJudge_NoFindingsAccepted(t *testing.T) {
	if got := Judge(nil); got != Accepted {
		t.Errorf("Judge(nil) = %v, want Accepted", got)
	}
	if got := Judge([]Finding{}); got != Accepted {
		t.Errorf("Judge([]Finding{}) = %v, want Accepted", got)
	}
}

func TestJudge_OnlyBlockingFindingsArePending(t *testing.T) {
	findings := []Finding{
		{Severity: Blocking, Message: "an uncovered combination"},
		{Severity: Blocking, Message: "an unaddressed boundary"},
	}
	if got := Judge(findings); got != Pending {
		t.Errorf("Judge(only Blocking findings) = %v, want Pending", got)
	}
}

func TestJudge_AnyRejectingFindingRejects(t *testing.T) {
	findings := []Finding{
		{Severity: Blocking, Message: "an uncovered combination"},
		{Severity: Rejecting, Message: "an unfalsifiable hypothesis"},
	}
	if got := Judge(findings); got != Rejected {
		t.Errorf("Judge(with a Rejecting finding present) = %v, want Rejected", got)
	}
}
