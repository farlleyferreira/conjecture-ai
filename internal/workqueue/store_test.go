package workqueue

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

func openTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestAdd_ThenList(t *testing.T) {
	db := openTestStore(t)
	now := time.Now().UTC()

	item := Item{ID: "a", Tier: Blocker, Weight: 5, Severity: 1, CreatedAt: now}
	if err := Add(db, item); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	items, err := List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "a" || items[0].Tier != Blocker || items[0].Weight != 5 || items[0].Severity != 1 {
		t.Errorf("items[0] = %+v, want the item just added", items[0])
	}
	if !items[0].CreatedAt.Equal(now) {
		t.Errorf("items[0].CreatedAt = %v, want %v", items[0].CreatedAt, now)
	}
}

func TestAdd_DuplicateIDIsError(t *testing.T) {
	db := openTestStore(t)
	item := Item{ID: "dup", Tier: Background, Weight: 1, CreatedAt: time.Now().UTC()}
	if err := Add(db, item); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := Add(db, item); err == nil {
		t.Fatal("Add with a duplicate ID = nil error, want one")
	}
}

func TestNext_RanksByTierThenWeight(t *testing.T) {
	db := openTestStore(t)
	now := time.Now().UTC()
	if err := Add(db, Item{ID: "background-heavy", Tier: Background, Weight: 100, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := Add(db, Item{ID: "blocker-light", Tier: Blocker, Weight: 1, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	item, found, err := Next(db, now, 0, time.Hour)
	if err != nil {
		t.Fatalf("Next returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("Next found = false, want true")
	}
	if item.ID != "blocker-light" {
		t.Errorf("Next returned %q, want the blocker-tier item regardless of weight", item.ID)
	}
}

func TestNext_EmptyQueue(t *testing.T) {
	db := openTestStore(t)
	_, found, err := Next(db, time.Now(), 0, time.Hour)
	if err != nil {
		t.Fatalf("Next returned unexpected error: %v", err)
	}
	if found {
		t.Error("Next on an empty queue found = true, want false")
	}
}

func TestPop_RemovesTheReturnedItem(t *testing.T) {
	db := openTestStore(t)
	now := time.Now().UTC()
	if err := Add(db, Item{ID: "only", Tier: CurrentTask, Weight: 1, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	item, found, err := Pop(db, now, 0, time.Hour)
	if err != nil {
		t.Fatalf("Pop returned unexpected error: %v", err)
	}
	if !found || item.ID != "only" {
		t.Fatalf("Pop = %+v, %v, want {ID: \"only\"}, true", item, found)
	}

	items, err := List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) after Pop = %d, want 0", len(items))
	}
}

func TestPop_EmptyQueue(t *testing.T) {
	db := openTestStore(t)
	_, found, err := Pop(db, time.Now(), 0, time.Hour)
	if err != nil {
		t.Fatalf("Pop returned unexpected error: %v", err)
	}
	if found {
		t.Error("Pop on an empty queue found = true, want false")
	}
}
