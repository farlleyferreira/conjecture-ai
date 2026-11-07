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
			id     TEXT PRIMARY KEY,
			status INTEGER NOT NULL
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
