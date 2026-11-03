package tap

import (
	"strings"
	"testing"
)

// This is the Test Anything Protocol's own canonical example stream
// (https://testanything.org), used verbatim so this parser is checked
// against the format's real reference text rather than a fixture this
// package invented on its own.
const canonicalExample = `TAP version 13
1..4
ok 1 - Input file opened
not ok 2 - First line of the input valid
ok 3 - Read the rest of the file
not ok 4 - Summarized correctly # TODO Not written yet
`

func TestParse_CanonicalExample(t *testing.T) {
	got, err := Parse(strings.NewReader(canonicalExample))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if got.Version != 13 {
		t.Errorf("Version = %d, want 13", got.Version)
	}
	if got.Plan != (Plan{Declared: true, Start: 1, End: 4}) {
		t.Errorf("Plan = %+v, want {Declared:true Start:1 End:4}", got.Plan)
	}
	if len(got.Tests) != 4 {
		t.Fatalf("len(Tests) = %d, want 4", len(got.Tests))
	}

	want := []TestLine{
		{Number: 1, OK: true, Description: "Input file opened"},
		{Number: 2, OK: false, Description: "First line of the input valid"},
		{Number: 3, OK: true, Description: "Read the rest of the file"},
		{Number: 4, OK: false, Description: "Summarized correctly", Directive: "TODO", DirectiveReason: "Not written yet"},
	}
	for i, w := range want {
		if got.Tests[i] != w {
			t.Errorf("Tests[%d] = %+v, want %+v", i, got.Tests[i], w)
		}
	}
}

func TestParse_UnnumberedLinesAreNumberedByPosition(t *testing.T) {
	got, err := Parse(strings.NewReader("ok\nok\nnot ok\n"))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(got.Tests) != 3 {
		t.Fatalf("len(Tests) = %d, want 3", len(got.Tests))
	}
	for i, tc := range got.Tests {
		if tc.Number != i+1 {
			t.Errorf("Tests[%d].Number = %d, want %d", i, tc.Number, i+1)
		}
	}
}

func TestParse_SkipDirective(t *testing.T) {
	got, err := Parse(strings.NewReader("ok 1 - a test # SKIP no network\n"))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	tc := got.Tests[0]
	if tc.Directive != "SKIP" || tc.DirectiveReason != "no network" || tc.Description != "a test" {
		t.Errorf("Tests[0] = %+v", tc)
	}
}

func TestParse_SkipWholePlan(t *testing.T) {
	got, err := Parse(strings.NewReader("1..0 # SKIP no network available\n"))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !got.Plan.Skip || got.Plan.SkipReason != "no network available" {
		t.Errorf("Plan = %+v, want Skip=true with the given reason", got.Plan)
	}
}

func TestParse_BailOut(t *testing.T) {
	got, err := Parse(strings.NewReader("1..4\nok 1\nBail out! Database on fire\nok 2\n"))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if got.BailOut != "Database on fire" {
		t.Errorf("BailOut = %q", got.BailOut)
	}
	// Whatever came before the bail-out line is still returned; nothing
	// at or after it needs to be, since a bail-out ends the run.
	if len(got.Tests) != 1 {
		t.Errorf("len(Tests) = %d, want 1 (only what was seen before the bail-out)", len(got.Tests))
	}
}

func TestParse_UnrecognizedLinesAreTolerated(t *testing.T) {
	// TAP's own documented tolerance: output that is not itself a TAP
	// line (here, a line a build tool might print to the same stream)
	// does not fail the parse.
	got, err := Parse(strings.NewReader("1..1\nSome unrelated program output\nok 1 - a test\n"))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(got.Tests) != 1 || !got.Tests[0].OK {
		t.Errorf("Tests = %+v", got.Tests)
	}
}

func TestParse_CommentLinesIgnored(t *testing.T) {
	got, err := Parse(strings.NewReader("1..1\n# a diagnostic comment\nok 1 - a test\n"))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(got.Tests) != 1 {
		t.Errorf("len(Tests) = %d, want 1", len(got.Tests))
	}
}
