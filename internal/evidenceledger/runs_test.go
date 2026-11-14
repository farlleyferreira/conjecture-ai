package evidenceledger

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func openRunTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureRunSchema(db); err != nil {
		t.Fatalf("EnsureRunSchema: %v", err)
	}
	return db
}

func TestAppendRunAndRuns_RoundTripInOrder(t *testing.T) {
	db := openRunTestDB(t)
	want := []Run{
		{HypothesisID: "a::b::c", SpecVersion: "s1", CodeVersion: "c1", Kind: RunDirected, Cases: 4, Conformant: true},
		{HypothesisID: "a::b::c", SpecVersion: "s1", CodeVersion: "c1", Kind: RunHidden, Cases: 20, Conformant: true, ProducerFamily: "anthropic", Seed: 18446744073709551615},
	}
	for _, r := range want {
		if err := AppendRun(db, r); err != nil {
			t.Fatalf("AppendRun: %v", err)
		}
	}
	if err := AppendRun(db, Run{HypothesisID: "other", SpecVersion: "s1", CodeVersion: "c1", Kind: RunHidden, Cases: 9, Conformant: true}); err != nil {
		t.Fatalf("AppendRun: %v", err)
	}

	got, err := Runs(db, "a::b::c")
	if err != nil {
		t.Fatalf("Runs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(Runs) = %d, want 2 (another hypothesis's run must not leak in)", len(got))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Kind != w.Kind || g.Cases != w.Cases || g.Conformant != w.Conformant || g.ProducerFamily != w.ProducerFamily ||
			g.Seed != w.Seed || g.SpecVersion != w.SpecVersion || g.CodeVersion != w.CodeVersion || g.CreatedAt.IsZero() {
			t.Errorf("Runs()[%d] = %+v, want %+v with a creation time", i, g, w)
		}
	}
}

func hidden(code string, cases int, ok bool, family string) Run {
	return Run{SpecVersion: "s1", CodeVersion: code, Kind: RunHidden, Cases: cases, Conformant: ok, ProducerFamily: family}
}

func TestMeasure_CountsOnlyHiddenCasesAndResetsOnAnyFailure(t *testing.T) {
	runs := []Run{
		hidden("c1", 60, true, ""),
		{SpecVersion: "s1", CodeVersion: "c1", Kind: RunDirected, Cases: 500, Conformant: true},
		hidden("c1", 50, true, ""),
	}
	if got := Measure(runs, "s1").FailureFree("c1"); got != 110 {
		t.Errorf("FailureFree = %d, want 110 (directed cases are not counted)", got)
	}

	runs = append(runs, Run{SpecVersion: "s1", CodeVersion: "c1", Kind: RunDirected, Cases: 4, Conformant: false})
	if got := Measure(runs, "s1").FailureFree("c1"); got != 0 {
		t.Errorf("FailureFree after a failing directed run = %d, want 0", got)
	}

	runs = append(runs, hidden("c1", 30, true, ""))
	if got := Measure(runs, "s1").FailureFree("c1"); got != 30 {
		t.Errorf("FailureFree after recovering = %d, want 30", got)
	}
}

func TestMeasure_KeepsCodeVersionsApartAndIgnoresOtherSpecVersions(t *testing.T) {
	runs := []Run{
		hidden("c1", 100, true, "anthropic"),
		hidden("c2", 40, true, "openai"),
		hidden("c2", 0, false, ""),
		{SpecVersion: "old", CodeVersion: "c1", Kind: RunHidden, Cases: 5000, Conformant: true},
	}
	m := Measure(runs, "s1")
	if m.FailureFree("c1") != 100 || m.FailureFree("c2") != 0 {
		t.Errorf("FailureFreeByCode = %v, want c1:100 c2:0", m.FailureFreeByCode)
	}
	if got := strings.Join(m.QualifiedCodeVersions(Low), ","); got != "c1" {
		t.Errorf("QualifiedCodeVersions(Low) = %q, want \"c1\"", got)
	}
	if got := m.QualifiedCodeVersions(Medium); len(got) != 0 {
		t.Errorf("QualifiedCodeVersions(Medium) = %v, want none (100 < 300)", got)
	}
}

func TestMeasure_QualifiedFamiliesNeedAQualifiedCodeVersionAndADeclaredFamily(t *testing.T) {
	runs := []Run{
		hidden("c1", 100, true, "anthropic"),
		hidden("c2", 100, true, "openai"),
		hidden("c3", 100, true, ""),       // qualified, but nobody said who wrote it
		hidden("c4", 10, true, "mistral"), // declared, but not enough cases
		hidden("c5", 100, true, "anthropic"),
	}
	got := strings.Join(Measure(runs, "s1").QualifiedFamilies(Low), ",")
	if got != "anthropic,openai" {
		t.Errorf("QualifiedFamilies = %q, want \"anthropic,openai\"", got)
	}
}
