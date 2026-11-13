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
	// RowID is this occurrence's own row id (work_queue_dead_letter's
	// AUTOINCREMENT primary key) - never part of the composite ID a
	// hypothesis/law already carries, and needed precisely because
	// [AddDeadLetter] allows that same ID to appear more than once:
	// [ResolveDeadLetter] targets one specific occurrence by this
	// value, not by ID.
	RowID    int64
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
	// ResolvedAt is the zero [time.Time] for an item no human has
	// resolved yet, or the time [ResolveDeadLetter] marked it resolved.
	ResolvedAt time.Time
}

// EnsureDeadLetterSchema creates the work_queue_dead_letter table if
// it does not already exist. It is safe to call every time a store
// is opened.
func EnsureDeadLetterSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS work_queue_dead_letter (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id     TEXT NOT NULL,
			tier        INTEGER NOT NULL,
			weight      INTEGER NOT NULL,
			severity    INTEGER NOT NULL,
			reason      TEXT NOT NULL,
			created_at  TEXT NOT NULL,
			resolved_at TEXT NOT NULL DEFAULT ''
		);
	`)
	if err != nil {
		return fmt.Errorf("workqueue: creating dead-letter schema: %w", err)
	}
	return ensureDeadLetterResolvedAtColumn(db)
}

// ensureDeadLetterResolvedAtColumn adds resolved_at to a
// work_queue_dead_letter table created before this column existed (a
// repository that already ran "conjecture work" under this feature's
// first release) - CREATE TABLE IF NOT EXISTS above only takes effect
// for a table that does not exist yet, never one that already does.
func ensureDeadLetterResolvedAtColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(work_queue_dead_letter)`)
	if err != nil {
		return fmt.Errorf("workqueue: inspecting the dead-letter schema: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hasColumn := false
	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("workqueue: inspecting the dead-letter schema: %w", err)
		}
		if name == "resolved_at" {
			hasColumn = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("workqueue: inspecting the dead-letter schema: %w", err)
	}
	if hasColumn {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE work_queue_dead_letter ADD COLUMN resolved_at TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("workqueue: adding resolved_at to the dead-letter schema: %w", err)
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

// ListDeadLetter returns dead-lettered items, oldest first. With
// includeResolved false (the default view "conjecture queue review"
// and the queue_review MCP tool both use), an item [ResolveDeadLetter]
// already marked resolved is left out - the point of resolving one is
// to drop it from the list a human works through next. includeResolved
// true also returns resolved items, as an audit trail.
func ListDeadLetter(db *sql.DB, includeResolved bool) ([]DeadLetterItem, error) {
	query := `SELECT id, item_id, tier, weight, severity, reason, created_at, resolved_at FROM work_queue_dead_letter`
	if !includeResolved {
		query += ` WHERE resolved_at = ''`
	}
	query += ` ORDER BY id ASC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("workqueue: listing the dead letter: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []DeadLetterItem
	for rows.Next() {
		var (
			it         DeadLetterItem
			tier       int
			createdAt  string
			resolvedAt string
		)
		if err := rows.Scan(&it.RowID, &it.ID, &tier, &it.Weight, &it.Severity, &it.Reason, &createdAt, &resolvedAt); err != nil {
			return nil, fmt.Errorf("workqueue: listing the dead letter: %w", err)
		}
		it.Tier = Tier(tier)
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("workqueue: parsing created_at for dead-lettered %q: %w", it.ID, err)
		}
		it.CreatedAt = parsedCreatedAt
		if resolvedAt != "" {
			parsedResolvedAt, err := time.Parse(time.RFC3339Nano, resolvedAt)
			if err != nil {
				return nil, fmt.Errorf("workqueue: parsing resolved_at for dead-lettered %q: %w", it.ID, err)
			}
			it.ResolvedAt = parsedResolvedAt
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("workqueue: listing the dead letter: %w", err)
	}
	return items, nil
}

// ResolveDeadLetter marks one specific dead-letter occurrence (by its
// own RowID - see [DeadLetterItem.RowID]'s own doc comment for why ID
// alone cannot target one) resolved, so [ListDeadLetter]'s default
// view stops returning it. It errors if rowID does not exist or was
// already resolved - never silently a no-op, so a human re-running the
// same "--resolve" by mistake finds out rather than assuming it worked
// twice.
func ResolveDeadLetter(db *sql.DB, rowID int64) error {
	result, err := db.Exec(
		`UPDATE work_queue_dead_letter SET resolved_at = ? WHERE id = ? AND resolved_at = ''`,
		time.Now().UTC().Format(time.RFC3339Nano), rowID,
	)
	if err != nil {
		return fmt.Errorf("workqueue: resolving dead-letter row %d: %w", rowID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("workqueue: resolving dead-letter row %d: %w", rowID, err)
	}
	if affected == 0 {
		return fmt.Errorf("workqueue: dead-letter row %d does not exist or is already resolved", rowID)
	}
	return nil
}
