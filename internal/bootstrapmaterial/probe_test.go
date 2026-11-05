package bootstrapmaterial

import "testing"

func TestCheckProbe_MatchingAnswer(t *testing.T) {
	probe := Probe{Input: "2+2", Seed: "1", ExpectedAnswer: "4"}
	wrong, reasons := CheckProbe(probe, []Result{{Output: []byte("4")}})
	if wrong {
		t.Errorf("CheckProbe = wrong=true, reasons=%v, want wrong=false", reasons)
	}
}

func TestCheckProbe_MismatchedAnswer(t *testing.T) {
	probe := Probe{Input: "2+2", Seed: "1", ExpectedAnswer: "4"}
	wrong, reasons := CheckProbe(probe, []Result{{Output: []byte("5")}})
	if !wrong {
		t.Fatal("CheckProbe with a mismatched answer = wrong=false, want true")
	}
	if len(reasons) != 1 {
		t.Errorf("len(reasons) = %d, want 1", len(reasons))
	}
}

func TestCheckProbe_NoExpectedAnswerSkipsThatCheck(t *testing.T) {
	probe := Probe{Input: "anything", Seed: "1"}
	wrong, reasons := CheckProbe(probe, []Result{{Output: []byte("whatever")}})
	if wrong {
		t.Errorf("CheckProbe with no ExpectedAnswer = wrong=true, reasons=%v, want false", reasons)
	}
}

const feeSchema = `{
  "type": "object",
  "properties": {"fee": {"type": "number"}},
  "required": ["fee"]
}`

func TestCheckProbe_SchemaConformant(t *testing.T) {
	probe := Probe{ExpectedSchema: []byte(feeSchema)}
	wrong, reasons := CheckProbe(probe, []Result{{Output: []byte(`{"fee": 7.5}`)}})
	if wrong {
		t.Errorf("CheckProbe = wrong=true, reasons=%v, want false", reasons)
	}
}

func TestCheckProbe_SchemaNonConformant(t *testing.T) {
	probe := Probe{ExpectedSchema: []byte(feeSchema)}
	wrong, reasons := CheckProbe(probe, []Result{{Output: []byte(`{}`)}})
	if !wrong {
		t.Fatal("CheckProbe with schema-non-conformant output = wrong=false, want true")
	}
	if len(reasons) != 1 {
		t.Errorf("len(reasons) = %d, want 1", len(reasons))
	}
}

func TestCheckProbe_Reproducible(t *testing.T) {
	probe := Probe{Input: "x", Seed: "1"}
	results := []Result{{Output: []byte("same")}, {Output: []byte("same")}, {Output: []byte("same")}}
	wrong, reasons := CheckProbe(probe, results)
	if wrong {
		t.Errorf("CheckProbe with identical repeated results = wrong=true, reasons=%v, want false", reasons)
	}
}

func TestCheckProbe_NonReproducible(t *testing.T) {
	probe := Probe{Input: "x", Seed: "1"}
	results := []Result{{Output: []byte("a")}, {Output: []byte("b")}}
	wrong, reasons := CheckProbe(probe, results)
	if !wrong {
		t.Fatal("CheckProbe with differing repeated results = wrong=false, want true")
	}
	if len(reasons) != 1 {
		t.Errorf("len(reasons) = %d, want 1", len(reasons))
	}
}

func TestCheckProbe_MultipleReasonsAtOnce(t *testing.T) {
	probe := Probe{ExpectedAnswer: "4", ExpectedSchema: []byte(feeSchema)}
	wrong, reasons := CheckProbe(probe, []Result{{Output: []byte("not json, not 4")}})
	if !wrong {
		t.Fatal("CheckProbe = wrong=false, want true")
	}
	if len(reasons) != 2 {
		t.Errorf("len(reasons) = %d, want 2 (both the answer mismatch and the schema failure)", len(reasons))
	}
}

func TestCheckProbe_NoResults(t *testing.T) {
	probe := Probe{ExpectedAnswer: "4"}
	wrong, reasons := CheckProbe(probe, nil)
	if wrong {
		t.Errorf("CheckProbe with no results = wrong=true, reasons=%v, want false (nothing observed to judge)", reasons)
	}
}
