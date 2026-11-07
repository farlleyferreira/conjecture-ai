package workqueue

import (
	"database/sql"
	"fmt"
	"time"
)

// EnsureSchema creates the work_queue_items table if it does not
// already exist. It is safe to call every time a store is opened.
//
// Queued here, in the same local, per-repository SQLite database
// [persistence.OpenStore] opens for every other piece of non-git-
// tracked state, for the same reason evidence-ledger and context-
// composition's own stores are: persistence's own "Git-tracked
// specification files" requirement names a closed list that does not
// mention a work-queue item, and its "Everything else stays local-
// only" requirement is written to cover exactly what that list
// excludes.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS work_queue_items (
			id         TEXT PRIMARY KEY,
			tier       INTEGER NOT NULL,
			weight     INTEGER NOT NULL,
			severity   INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("workqueue: creating schema: %w", err)
	}
	return nil
}

// Add persists item. It fails if item.ID is already queued - an item
// is identified by its own caller-chosen ID, and silently re-adding
// one under the same ID would reset CreatedAt, defeating "Aging
// prevents starvation" for whatever work the earlier entry already
// represented. A caller that means to re-surface the same work
// chooses a new ID, or removes the old entry first via [Pop].
func Add(db *sql.DB, item Item) error {
	_, err := db.Exec(
		`INSERT INTO work_queue_items (id, tier, weight, severity, created_at) VALUES (?, ?, ?, ?, ?)`,
		item.ID, int(item.Tier), item.Weight, item.Severity, item.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("workqueue: adding %q: %w", item.ID, err)
	}
	return nil
}

// List returns every queued item, in no particular order - a caller
// wanting [Less]'s own tier/weight/severity/age order calls [Next] or
// [Pop] instead.
func List(db *sql.DB) ([]Item, error) {
	rows, err := db.Query(`SELECT id, tier, weight, severity, created_at FROM work_queue_items`)
	if err != nil {
		return nil, fmt.Errorf("workqueue: listing: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []Item
	for rows.Next() {
		var (
			it        Item
			tier      int
			createdAt string
		)
		if err := rows.Scan(&it.ID, &tier, &it.Weight, &it.Severity, &createdAt); err != nil {
			return nil, fmt.Errorf("workqueue: listing: %w", err)
		}
		it.Tier = Tier(tier)
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("workqueue: parsing created_at for %q: %w", it.ID, err)
		}
		it.CreatedAt = parsedCreatedAt
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("workqueue: listing: %w", err)
	}
	return items, nil
}

// Next returns the item [Less] ranks highest at now, without removing
// it, or (Item{}, false, nil) if the queue is empty - the persisted
// equivalent of [Queue.Next].
func Next(db *sql.DB, now time.Time, agingRate float64, unit time.Duration) (Item, bool, error) {
	items, err := List(db)
	if err != nil {
		return Item{}, false, err
	}
	if len(items) == 0 {
		return Item{}, false, nil
	}
	best := items[0]
	for _, it := range items[1:] {
		if Less(it, best, now, agingRate, unit) {
			best = it
		}
	}
	return best, true, nil
}

// Pop removes and returns the item [Next] would return - the
// persisted equivalent of [Queue.Pop]: "The system SHALL keep
// checking the queue in a loop until no actionable item remains,
// rather than stopping after a fixed number of steps," a caller
// drives by calling Pop until it reports found=false.
func Pop(db *sql.DB, now time.Time, agingRate float64, unit time.Duration) (Item, bool, error) {
	item, found, err := Next(db, now, agingRate, unit)
	if err != nil || !found {
		return item, found, err
	}
	if _, err := db.Exec(`DELETE FROM work_queue_items WHERE id = ?`, item.ID); err != nil {
		return Item{}, false, fmt.Errorf("workqueue: removing %q: %w", item.ID, err)
	}
	return item, true, nil
}
