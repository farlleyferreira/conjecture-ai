package workqueue

import (
	"testing"
	"time"
)

func TestAddDeadLetter_ThenListDeadLetter(t *testing.T) {
	db := openTestStore(t)
	if err := EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}

	now := time.Now().UTC()
	if err := AddDeadLetter(db, DeadLetterItem{
		ID: "area::demo::rule::hyp", Tier: Blocker, Weight: 2, Severity: 1,
		Reason: "domain not boundable", CreatedAt: now,
	}); err != nil {
		t.Fatalf("AddDeadLetter returned unexpected error: %v", err)
	}

	items, err := ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.ID != "area::demo::rule::hyp" || got.Tier != Blocker || got.Weight != 2 || got.Severity != 1 || got.Reason != "domain not boundable" {
		t.Errorf("items[0] = %+v, want the item added", got)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("items[0].CreatedAt = %v, want %v", got.CreatedAt, now)
	}
	if got.RowID == 0 {
		t.Errorf("items[0].RowID = 0, want the table's own assigned row id")
	}
	if !got.ResolvedAt.IsZero() {
		t.Errorf("items[0].ResolvedAt = %v, want zero (unresolved)", got.ResolvedAt)
	}
}

func TestListDeadLetter_EmptyReturnsNilNotError(t *testing.T) {
	db := openTestStore(t)
	if err := EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}

	items, err := ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("items = %+v, want empty", items)
	}
}

func TestAddDeadLetter_AllowsRepeatedID(t *testing.T) {
	db := openTestStore(t)
	if err := EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := AddDeadLetter(db, DeadLetterItem{
			ID: "area::demo::rule::hyp", Tier: Background, Weight: 1,
			Reason: "domain not boundable", CreatedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatalf("AddDeadLetter #%d returned unexpected error: %v", i, err)
		}
	}

	items, err := ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2 (append-only, duplicates allowed)", len(items))
	}
}

func TestResolveDeadLetter_RemovesItFromTheDefaultListing(t *testing.T) {
	db := openTestStore(t)
	if err := EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}
	if err := AddDeadLetter(db, DeadLetterItem{
		ID: "area::demo::rule::hyp", Tier: Blocker, Weight: 1,
		Reason: "domain not boundable", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AddDeadLetter returned unexpected error: %v", err)
	}
	items, err := ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	rowID := items[0].RowID

	if err := ResolveDeadLetter(db, rowID); err != nil {
		t.Fatalf("ResolveDeadLetter returned unexpected error: %v", err)
	}

	unresolved, err := ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(unresolved) != 0 {
		t.Errorf("unresolved = %+v, want empty after resolving the only row", unresolved)
	}

	all, err := ListDeadLetter(db, true)
	if err != nil {
		t.Fatalf("ListDeadLetter(includeResolved=true) returned unexpected error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("len(all) = %d, want 1 (resolved items stay visible with includeResolved)", len(all))
	}
	if all[0].ResolvedAt.IsZero() {
		t.Errorf("all[0].ResolvedAt is zero, want the resolve time")
	}
}

func TestResolveDeadLetter_UnknownOrAlreadyResolvedRowErrors(t *testing.T) {
	db := openTestStore(t)
	if err := EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}

	if err := ResolveDeadLetter(db, 999); err == nil {
		t.Error("ResolveDeadLetter on a nonexistent row returned nil error, want an error")
	}

	if err := AddDeadLetter(db, DeadLetterItem{
		ID: "area::demo::rule::hyp", Tier: Background, Weight: 1,
		Reason: "domain not boundable", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AddDeadLetter returned unexpected error: %v", err)
	}
	items, err := ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	rowID := items[0].RowID

	if err := ResolveDeadLetter(db, rowID); err != nil {
		t.Fatalf("ResolveDeadLetter returned unexpected error: %v", err)
	}
	if err := ResolveDeadLetter(db, rowID); err == nil {
		t.Error("ResolveDeadLetter on an already-resolved row returned nil error, want an error")
	}
}

func TestEnsureDeadLetterResolvedAtColumn_MigratesAnExistingPreResolveTable(t *testing.T) {
	db := openTestStore(t)
	if _, err := db.Exec(`
		CREATE TABLE work_queue_dead_letter (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id    TEXT NOT NULL,
			tier       INTEGER NOT NULL,
			weight     INTEGER NOT NULL,
			severity   INTEGER NOT NULL,
			reason     TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("creating the pre-resolve table returned unexpected error: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO work_queue_dead_letter (item_id, tier, weight, severity, reason, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"area::demo::rule::hyp", int(Blocker), 1, 0, "domain not boundable", time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("seeding the pre-resolve table returned unexpected error: %v", err)
	}

	if err := EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}

	items, err := ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1 (the row from before the migration, preserved)", len(items))
	}
	if !items[0].ResolvedAt.IsZero() {
		t.Errorf("items[0].ResolvedAt = %v, want zero for a row migrated from before resolved_at existed", items[0].ResolvedAt)
	}

	if err := ResolveDeadLetter(db, items[0].RowID); err != nil {
		t.Fatalf("ResolveDeadLetter after migration returned unexpected error: %v", err)
	}
}
