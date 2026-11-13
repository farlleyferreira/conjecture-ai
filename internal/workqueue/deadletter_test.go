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

	items, err := ListDeadLetter(db)
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
}

func TestListDeadLetter_EmptyReturnsNilNotError(t *testing.T) {
	db := openTestStore(t)
	if err := EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}

	items, err := ListDeadLetter(db)
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

	items, err := ListDeadLetter(db)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2 (append-only, duplicates allowed)", len(items))
	}
}
