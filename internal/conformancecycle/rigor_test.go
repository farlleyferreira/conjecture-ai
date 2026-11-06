package conformancecycle

import (
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

func TestUsesDirectedSampling_LowWeightNoStrongerEvidence(t *testing.T) {
	if !UsesDirectedSampling(evidenceledger.Low, false) {
		t.Error("UsesDirectedSampling(Low, false) = false, want true (the specification's own named scenario)")
	}
}

func TestUsesDirectedSampling_LowWeightWithStrongerEvidenceIsFalse(t *testing.T) {
	if UsesDirectedSampling(evidenceledger.Low, true) {
		t.Error("UsesDirectedSampling(Low, true) = true, want false (stronger evidence is available)")
	}
}

func TestUsesDirectedSampling_HigherWeightsAreFalse(t *testing.T) {
	for _, w := range []evidenceledger.Weight{evidenceledger.Medium, evidenceledger.High, evidenceledger.Critical} {
		if UsesDirectedSampling(w, false) {
			t.Errorf("UsesDirectedSampling(%v, false) = true, want false (the specification only names Low)", w)
		}
	}
}

func TestRigorLevel_String(t *testing.T) {
	cases := map[RigorLevel]string{
		DirectedSampling: "directed sampling",
		BoundedProof:     "bounded proof",
		CompleteProof:    "complete proof",
	}
	for level, want := range cases {
		if got := level.String(); got != want {
			t.Errorf("%v.String() = %q, want %q", level, got, want)
		}
	}
}
