package conformancecycle

import "testing"

func TestVerdict_StringAndParseRoundTrip(t *testing.T) {
	for _, v := range []Verdict{Conformant, NonConformant, Inconclusive} {
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
	_, err := ParseVerdict("passed")
	if err == nil {
		t.Fatal("ParseVerdict(\"passed\") = nil error, want an error")
	}
}

func TestVerdictFromOutcome_TimeoutAlwaysInconclusive(t *testing.T) {
	// A timeout must win regardless of what "passed" claims - a
	// timed-out check has no real pass/fail result to report.
	if got := VerdictFromOutcome(true, true); got != Inconclusive {
		t.Errorf("VerdictFromOutcome(timedOut=true, passed=true) = %v, want Inconclusive", got)
	}
	if got := VerdictFromOutcome(true, false); got != Inconclusive {
		t.Errorf("VerdictFromOutcome(timedOut=true, passed=false) = %v, want Inconclusive", got)
	}
}

func TestVerdictFromOutcome_NoTimeout(t *testing.T) {
	if got := VerdictFromOutcome(false, true); got != Conformant {
		t.Errorf("VerdictFromOutcome(timedOut=false, passed=true) = %v, want Conformant", got)
	}
	if got := VerdictFromOutcome(false, false); got != NonConformant {
		t.Errorf("VerdictFromOutcome(timedOut=false, passed=false) = %v, want NonConformant", got)
	}
}
