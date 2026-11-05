package specjudge

import "testing"

func TestFalsifiabilityFinding_Falsifiable(t *testing.T) {
	if got := FalsifiabilityFinding("never-negative", "the shipping fee is at least 50.00", true); got != nil {
		t.Errorf("FalsifiabilityFinding(falsifiable=true) = %+v, want nil", got)
	}
}

func TestFalsifiabilityFinding_Unfalsifiable(t *testing.T) {
	got := FalsifiabilityFinding("never-negative", "the shipping fee is at least -1.00", false)
	if got == nil {
		t.Fatal("FalsifiabilityFinding(falsifiable=false) = nil, want a Finding")
	}
	if got.Severity != Rejecting {
		t.Errorf("got.Severity = %v, want Rejecting", got.Severity)
	}
}

func TestFalsifiabilityFinding_ComposesWithJudge(t *testing.T) {
	// A worked end-to-end example: one unfalsifiable hypothesis among
	// several falsifiable ones must still reject the whole
	// specification.
	results := []struct {
		name string
		stmt string
		ok   bool
	}{
		{"never-negative", "the shipping fee is at least -1.00", false},
		{"free-above", "the shipping fee is 0.00", true},
	}

	var findings []Finding
	for _, r := range results {
		if f := FalsifiabilityFinding(r.name, r.stmt, r.ok); f != nil {
			findings = append(findings, *f)
		}
	}

	if got := Judge(findings); got != Rejected {
		t.Errorf("Judge(findings) = %v, want Rejected (one hypothesis was unfalsifiable)", got)
	}
}
