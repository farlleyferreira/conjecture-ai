package evidenceledger

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// Run is one conformance run against one hypothesis, as a structured
// fact: how many cases, against which code, whether every one of them
// conformed.
//
// It exists beside [Evidence], not instead of it. An Evidence record
// is written for a person to read ("directed sampling; found
// counterexample: ..."), and promotion past Thesis used to be decided
// by a person reading those and passing --different-family-reproduced
// or --hidden-samples-conformant by hand - an attestation, not a
// measurement. A Run is what those conditions can be computed from.
type Run struct {
	HypothesisID string
	// SpecVersion and CodeVersion identify what the run was against: a
	// hash of the .cjtr file, and of the implementation (its source
	// tree, or its container image).
	SpecVersion, CodeVersion string
	// Kind is [RunHidden] for independently drawn random cases - the
	// only kind the numeric promotion criteria count - or [RunDirected]
	// for any deterministic check (directed sampling, a bounded or
	// complete proof), which is recorded so that its failure resets
	// the count.
	Kind string
	// Cases is how many cases the run exercised.
	Cases int
	// Conformant is true only when every case conformed.
	Conformant bool
	// ProducerFamily is the model family that wrote the implementation,
	// as declared by whoever ran the check ("anthropic", "openai",
	// "human"), or "" when not declared. Nothing here can verify it.
	ProducerFamily string
	// Seed reproduces a hidden run's cases.
	Seed      uint64
	CreatedAt time.Time
}

const (
	RunHidden   = "hidden"
	RunDirected = "directed"
)

// EnsureRunSchema creates the conformance_run table if it does not
// exist yet.
func EnsureRunSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conformance_run (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			hypothesis_id   TEXT NOT NULL,
			spec_version    TEXT NOT NULL,
			code_version    TEXT NOT NULL,
			kind            TEXT NOT NULL,
			cases           INTEGER NOT NULL,
			conformant      INTEGER NOT NULL,
			producer_family TEXT NOT NULL,
			seed            TEXT NOT NULL,
			created_at      TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS conformance_run_hypothesis_id ON conformance_run (hypothesis_id, id);
	`)
	if err != nil {
		return fmt.Errorf("evidenceledger: creating run schema: %w", err)
	}
	return nil
}

// AppendRun records r.
func AppendRun(db *sql.DB, r Run) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	_, err := db.Exec(
		`INSERT INTO conformance_run
			(hypothesis_id, spec_version, code_version, kind, cases, conformant, producer_family, seed, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.HypothesisID, r.SpecVersion, r.CodeVersion, r.Kind, r.Cases, r.Conformant, r.ProducerFamily,
		fmt.Sprintf("%d", r.Seed), r.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("evidenceledger: appending run: %w", err)
	}
	return nil
}

// Runs returns every run recorded for hypothesisID, oldest first.
func Runs(db *sql.DB, hypothesisID string) ([]Run, error) {
	rows, err := db.Query(
		`SELECT spec_version, code_version, kind, cases, conformant, producer_family, seed, created_at
		 FROM conformance_run WHERE hypothesis_id = ? ORDER BY id ASC`,
		hypothesisID,
	)
	if err != nil {
		return nil, fmt.Errorf("evidenceledger: reading runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var runs []Run
	for rows.Next() {
		r := Run{HypothesisID: hypothesisID}
		var seed, createdAt string
		if err := rows.Scan(&r.SpecVersion, &r.CodeVersion, &r.Kind, &r.Cases, &r.Conformant, &r.ProducerFamily, &seed, &createdAt); err != nil {
			return nil, fmt.Errorf("evidenceledger: reading runs: %w", err)
		}
		_, _ = fmt.Sscanf(seed, "%d", &r.Seed)
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("evidenceledger: reading runs: %w", err)
	}
	return runs, nil
}

// Measured is what a hypothesis's runs add up to against one version
// of its specification - every number the promotion criteria ask for,
// computed rather than asserted.
type Measured struct {
	// FailureFreeByCode is, for each code version, how many hidden
	// cases conformed in a row since that code version's last failing
	// run of any kind. A failing run puts its code version back to
	// zero: "a single counterexample demotes".
	FailureFreeByCode map[string]int
	// FamilyByCode is the producer family declared for each code
	// version (the latest non-empty declaration).
	FamilyByCode map[string]string
}

// Measure folds runs (oldest first, as [Runs] returns them) into
// [Measured], keeping only runs against specVersion: cases that
// conformed to an earlier text of the rule say nothing about this
// one.
func Measure(runs []Run, specVersion string) Measured {
	m := Measured{FailureFreeByCode: map[string]int{}, FamilyByCode: map[string]string{}}
	for _, r := range runs {
		if r.SpecVersion != specVersion {
			continue
		}
		if r.ProducerFamily != "" {
			m.FamilyByCode[r.CodeVersion] = r.ProducerFamily
		}
		switch {
		case !r.Conformant:
			m.FailureFreeByCode[r.CodeVersion] = 0
		case r.Kind == RunHidden:
			m.FailureFreeByCode[r.CodeVersion] += r.Cases
		default:
			// A conformant deterministic run: recorded, not counted.
			if _, seen := m.FailureFreeByCode[r.CodeVersion]; !seen {
				m.FailureFreeByCode[r.CodeVersion] = 0
			}
		}
	}
	return m
}

// FailureFree returns the failure-free hidden-case count for one code
// version.
func (m Measured) FailureFree(codeVersion string) int {
	return m.FailureFreeByCode[codeVersion]
}

// QualifiedCodeVersions returns the code versions whose failure-free
// hidden-case count meets weight's required count, sorted.
func (m Measured) QualifiedCodeVersions(weight Weight) []string {
	var out []string
	for code, n := range m.FailureFreeByCode {
		if weight.MeetsPromotionThreshold(n) {
			out = append(out, code)
		}
	}
	sort.Strings(out)
	return out
}

// QualifiedFamilies returns the distinct declared producer families
// among the code versions that meet weight's required count, sorted.
// A qualified code version with no declared family contributes
// nothing: an unknown author cannot be shown to differ from another.
func (m Measured) QualifiedFamilies(weight Weight) []string {
	seen := map[string]bool{}
	for _, code := range m.QualifiedCodeVersions(weight) {
		if family := m.FamilyByCode[code]; family != "" {
			seen[family] = true
		}
	}
	out := make([]string, 0, len(seen))
	for family := range seen {
		out = append(out, family)
	}
	sort.Strings(out)
	return out
}
