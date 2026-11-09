package optimizationsuite

import (
	"database/sql"
	"fmt"
)

// EnsureSchema creates the calibrated_setting table if it does not
// already exist. It is safe to call every time a store is opened.
//
// Queued here, in the same local, per-repository SQLite database
// [persistence.OpenStore] opens for every other piece of non-git-
// tracked state, for the same reason evidence-ledger's and work-
// queue's own stores are: a setting's current calibrated value is
// calibration-run output, not specification text, so it belongs
// outside the git-tracked settings.cjtr file persistence's own
// requirement names.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS calibrated_setting (
			field   TEXT PRIMARY KEY,
			current REAL NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("optimizationsuite: creating schema: %w", err)
	}
	return nil
}

// SetCurrent persists value as field's current calibrated value,
// replacing whatever value (if any) it previously held - the running
// calibrated value moves forward as [ApplyProposal] decides it
// should, it does not accumulate a history of every past value.
func SetCurrent(db *sql.DB, field string, value float64) error {
	_, err := db.Exec(
		`INSERT INTO calibrated_setting (field, current) VALUES (?, ?)
		 ON CONFLICT(field) DO UPDATE SET current = excluded.current`,
		field, value,
	)
	if err != nil {
		return fmt.Errorf("optimizationsuite: setting current value for %q: %w", field, err)
	}
	return nil
}

// GetCurrent reports field's current calibrated value, if one has
// ever been recorded for it. found is false the first time a field is
// calibrated, before any value exists to compare a proposal against -
// a genuinely different state from "a value exists and is zero," so
// this does not invent a default.
func GetCurrent(db *sql.DB, field string) (value float64, found bool, err error) {
	err = db.QueryRow(`SELECT current FROM calibrated_setting WHERE field = ?`, field).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("optimizationsuite: reading current value for %q: %w", field, err)
	}
	return value, true, nil
}
