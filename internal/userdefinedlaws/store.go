package userdefinedlaws

import (
	"database/sql"
	"fmt"
)

// EnsureSchema creates the user_defined_law_status table if it does
// not already exist. It is safe to call every time a store is opened.
//
// Queued here, in the same local, per-repository SQLite database
// [persistence.OpenStore] opens for every other piece of non-git-
// tracked state, for the same reason evidence-ledger and work-queue's
// own stores are: a law's status is conformance-run output, not
// specification text, so it belongs outside the Git-tracked .cjtr
// file persistence's own requirement names.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_defined_law_status (
			id                    TEXT PRIMARY KEY,
			status                INTEGER NOT NULL DEFAULT 0,
			counterevidence_count INTEGER NOT NULL DEFAULT 0,
			failure_free_cases    INTEGER NOT NULL DEFAULT 0
		);
	`)
	if err != nil {
		return fmt.Errorf("userdefinedlaws: creating schema: %w", err)
	}
	return nil
}

// SetStatus persists status as id's current status, replacing
// whatever status (if any) id previously had - a law's status
// lifecycle moves forward as new conformance evidence arrives, it
// does not accumulate a history of every past status the way
// evidence-ledger's own append-only records do.
func SetStatus(db *sql.DB, id string, status Status) error {
	_, err := db.Exec(
		`INSERT INTO user_defined_law_status (id, status) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET status = excluded.status`,
		id, int(status),
	)
	if err != nil {
		return fmt.Errorf("userdefinedlaws: setting status for %q: %w", id, err)
	}
	return nil
}

// GetStatus reports id's current status, or [Unverified] if id has
// never had a status recorded - [Status]'s own doc comment: "the zero
// value, Unverified, is a law's starting status before any
// conformance evidence exists."
func GetStatus(db *sql.DB, id string) (Status, error) {
	var status int
	err := db.QueryRow(`SELECT status FROM user_defined_law_status WHERE id = ?`, id).Scan(&status)
	if err == sql.ErrNoRows {
		return Unverified, nil
	}
	if err != nil {
		return 0, fmt.Errorf("userdefinedlaws: reading status for %q: %w", id, err)
	}
	return Status(status), nil
}

// RecordCounterevidence increments id's own accumulated empirical-
// counterevidence count by one and returns the new total - the
// Invalidation pathway's own running tally toward "a configurable
// threshold before the law is marked contested" ([MeetsContestedThreshold]).
// It never touches id's status column: a fresh row it creates defaults
// to [Unverified], same as [GetStatus] on an id never seen before;
// [SetStatus] is still the only way a caller changes status itself.
// The count only ever grows - nothing in the specification's wording
// ("accumulate toward a... threshold") says a later Conformant result
// should reset it, so this does not invent that reset.
func RecordCounterevidence(db *sql.DB, id string) (int, error) {
	if _, err := db.Exec(
		`INSERT INTO user_defined_law_status (id, counterevidence_count) VALUES (?, 1)
		 ON CONFLICT(id) DO UPDATE SET counterevidence_count = counterevidence_count + 1`,
		id,
	); err != nil {
		return 0, fmt.Errorf("userdefinedlaws: recording counterevidence for %q: %w", id, err)
	}
	return GetCounterevidenceCount(db, id)
}

// GetCounterevidenceCount reports id's current accumulated
// counterevidence count, or 0 if id has never had any recorded.
func GetCounterevidenceCount(db *sql.DB, id string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT counterevidence_count FROM user_defined_law_status WHERE id = ?`, id).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("userdefinedlaws: reading counterevidence count for %q: %w", id, err)
	}
	return count, nil
}

// AddFailureFreeCases adds delta to id's own running failure-free case
// count and returns the new total - evidence-ledger's own "number of
// failure-free cases required before promotion" criterion
// ([evidenceledger.Weight.MeetsPromotionThreshold]), tracked here
// because that package has no per-hypothesis weight to look the
// threshold up by for anything except a law (always critical-weighted,
// see contextengine.GatedWeight's own "isUserDefinedLaw's rule wins
// outright"), so this running count only ever matters for a law.
func AddFailureFreeCases(db *sql.DB, id string, delta int) (int, error) {
	if _, err := db.Exec(
		`INSERT INTO user_defined_law_status (id, failure_free_cases) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET failure_free_cases = failure_free_cases + excluded.failure_free_cases`,
		id, delta,
	); err != nil {
		return 0, fmt.Errorf("userdefinedlaws: adding failure-free cases for %q: %w", id, err)
	}
	return GetFailureFreeCases(db, id)
}

// ResetFailureFreeCases clears id's own running failure-free case
// count back to zero - a single confirmed counterexample demotes a
// hypothesis's evidence-ledger stage, and resets whatever progress
// toward the NEXT promotion a prior run of failure-free cases
// represented the same way: the specification's "repeatable results"
// wording for promotion is read here as results that keep surviving
// without a demoting counterexample interrupting the run, not as an
// all-time total that a later violation leaves untouched.
func ResetFailureFreeCases(db *sql.DB, id string) error {
	_, err := db.Exec(
		`INSERT INTO user_defined_law_status (id, failure_free_cases) VALUES (?, 0)
		 ON CONFLICT(id) DO UPDATE SET failure_free_cases = 0`,
		id,
	)
	if err != nil {
		return fmt.Errorf("userdefinedlaws: resetting failure-free cases for %q: %w", id, err)
	}
	return nil
}

// GetFailureFreeCases reports id's current running failure-free case
// count, or 0 if id has never had any recorded.
func GetFailureFreeCases(db *sql.DB, id string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT failure_free_cases FROM user_defined_law_status WHERE id = ?`, id).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("userdefinedlaws: reading failure-free cases for %q: %w", id, err)
	}
	return count, nil
}
