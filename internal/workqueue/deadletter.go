package workqueue

import (
	"database/sql"
	"fmt"
	"time"
)

// DeadLetterItem is a work-queue item that "conjecture work" removed
// from the normal queue because acting on it automatically is
// deterministically impossible (its own disposable implementing
// oracle refused it outright - see cmd/conjecture's own
// ErrDomainTooComplexForOracle) - never the same kind of failure a
// retry could resolve, which is what the normal queue's own
// [workqueue.Add]-it-back-and-try-again path already exists for.
//
// This is a secondary, append-only record, not a field on [Item]
// itself and not a status an item in the normal queue can carry: an
// item here is already gone from work_queue_items (removed by the
// same [Pop] that read it), and [Next]/[Pop] never read this table at
// all, so nothing here is ever picked up automatically again. A human
// reviews it directly (via "conjecture queue review") and decides
// what to do - fix the hypothesis's own declared domain, implement it
// by hand, or anything else - never conjecture itself.
type DeadLetterItem struct {
	ID       string
	Tier     Tier
	Weight   int
	Severity int
	// Reason is the error that caused the removal, exactly as
	// reported (e.g. ErrDomainTooComplexForOracle's own message) -
	// disclosed, not summarized, so a human reviewing this later has
	// the real reason, not this package's own paraphrase of it.
	Reason    string
	CreatedAt time.Time
}

// EnsureDeadLetterSchema creates the work_queue_dead_letter table if
// it does not already exist. It is safe to call every time a store
// is opened.
func EnsureDeadLetterSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS work_queue_dead_letter (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id    TEXT NOT NULL,
			tier       INTEGER NOT NULL,
			weight     INTEGER NOT NULL,
			severity   INTEGER NOT NULL,
			reason     TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("workqueue: creating dead-letter schema: %w", err)
	}
	return nil
}

// AddDeadLetter appends item - unlike [Add]'s own main-queue table,
// this is append-only and never rejects a repeated ItemID: the same
// hypothesis/law can be dead-lettered more than once across separate
// "conjecture work" runs (check.go's own recordEvidence re-queues a
// confirmed violation every time it is re-checked and still fails),
// and each occurrence is its own honest historical record, not a
// single row to overwrite.
func AddDeadLetter(db *sql.DB, item DeadLetterItem) error {
	_, err := db.Exec(
		`INSERT INTO work_queue_dead_letter (item_id, tier, weight, severity, reason, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, int(item.Tier), item.Weight, item.Severity, item.Reason, item.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("workqueue: adding %q to the dead letter: %w", item.ID, err)
	}
	return nil
}

// ListDeadLetter returns every dead-lettered item, oldest first.
func ListDeadLetter(db *sql.DB) ([]DeadLetterItem, error) {
	rows, err := db.Query(`SELECT item_id, tier, weight, severity, reason, created_at FROM work_queue_dead_letter ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("workqueue: listing the dead letter: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []DeadLetterItem
	for rows.Next() {
		var (
			it        DeadLetterItem
			tier      int
			createdAt string
		)
		if err := rows.Scan(&it.ID, &tier, &it.Weight, &it.Severity, &it.Reason, &createdAt); err != nil {
			return nil, fmt.Errorf("workqueue: listing the dead letter: %w", err)
		}
		it.Tier = Tier(tier)
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("workqueue: parsing created_at for dead-lettered %q: %w", it.ID, err)
		}
		it.CreatedAt = parsedCreatedAt
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("workqueue: listing the dead letter: %w", err)
	}
	return items, nil
}
