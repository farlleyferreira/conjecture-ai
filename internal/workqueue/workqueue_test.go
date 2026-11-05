package workqueue

import (
	"testing"
	"time"
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestLess_TierAlwaysWinsFirst(t *testing.T) {
	// A blocker with the lowest possible weight/severity must still
	// outrank a background item with the highest, regardless of age.
	blocker := Item{ID: "b", Tier: Blocker, Weight: 0, Severity: 0, CreatedAt: epoch}
	background := Item{ID: "bg", Tier: Background, Weight: 100, Severity: 100, CreatedAt: epoch.Add(-1000 * time.Hour)}

	if !Less(blocker, background, epoch, 0, time.Hour) {
		t.Error("Less(blocker, background) = false, want true (a blocker preempts background regardless of weight/severity/age)")
	}
	if Less(background, blocker, epoch, 0, time.Hour) {
		t.Error("Less(background, blocker) = true, want false")
	}
}

func TestLess_CurrentTaskBetweenBlockerAndBackground(t *testing.T) {
	blocker := Item{Tier: Blocker, CreatedAt: epoch}
	current := Item{Tier: CurrentTask, CreatedAt: epoch}
	background := Item{Tier: Background, CreatedAt: epoch}

	if !Less(blocker, current, epoch, 0, time.Hour) {
		t.Error("Less(blocker, current) = false, want true")
	}
	if !Less(current, background, epoch, 0, time.Hour) {
		t.Error("Less(current, background) = false, want true")
	}
}

func TestLess_WithinTierOrdersByWeightThenSeverityThenAge(t *testing.T) {
	higherWeight := Item{Tier: Background, Weight: 10, Severity: 0, CreatedAt: epoch}
	lowerWeight := Item{Tier: Background, Weight: 5, Severity: 100, CreatedAt: epoch.Add(-1000 * time.Hour)}
	if !Less(higherWeight, lowerWeight, epoch, 0, time.Hour) {
		t.Error("Less(higherWeight, lowerWeight) = false, want true (weight is compared before severity or age)")
	}

	higherSeverity := Item{Tier: Background, Weight: 5, Severity: 10, CreatedAt: epoch}
	lowerSeverity := Item{Tier: Background, Weight: 5, Severity: 1, CreatedAt: epoch.Add(-1000 * time.Hour)}
	if !Less(higherSeverity, lowerSeverity, epoch, 0, time.Hour) {
		t.Error("Less(higherSeverity, lowerSeverity) = false, want true (severity is compared before age, given equal weight)")
	}

	older := Item{Tier: Background, Weight: 5, Severity: 5, CreatedAt: epoch.Add(-10 * time.Hour)}
	newer := Item{Tier: Background, Weight: 5, Severity: 5, CreatedAt: epoch}
	if !Less(older, newer, epoch, 0, time.Hour) {
		t.Error("Less(older, newer) = false, want true (oldest first, given equal weight and severity)")
	}
}

func TestEffectiveWeight_AgingEventuallySurfacesALowWeightItem(t *testing.T) {
	// "Long-waiting item eventually surfaces": a low-weight item that
	// has waited long enough must outrank a higher-weight, newer one.
	lowWeightOld := Item{Tier: Background, Weight: 1, CreatedAt: epoch.Add(-1000 * time.Hour)}
	highWeightNew := Item{Tier: Background, Weight: 50, CreatedAt: epoch}

	const agingRate = 1.0
	const unit = time.Hour

	if Less(highWeightNew, lowWeightOld, epoch, agingRate, unit) {
		t.Error("the higher-weight, newer item still outranked the long-waiting one with aging applied")
	}
	if !Less(lowWeightOld, highWeightNew, epoch, agingRate, unit) {
		t.Error("the long-waiting low-weight item did not outrank the higher-weight, newer one")
	}
}

func TestEffectiveWeight_NoAgingLeavesWeightUnchanged(t *testing.T) {
	item := Item{Weight: 5, CreatedAt: epoch.Add(-1000 * time.Hour)}
	got := EffectiveWeight(item, epoch, 0, time.Hour)
	if got != 5 {
		t.Errorf("EffectiveWeight with agingRate=0 = %v, want 5 (unchanged regardless of age)", got)
	}
}

func TestQueue_PopReturnsHighestPriorityFirst(t *testing.T) {
	var q Queue
	q.Add(Item{ID: "background", Tier: Background, Weight: 100, CreatedAt: epoch})
	q.Add(Item{ID: "blocker", Tier: Blocker, Weight: 0, CreatedAt: epoch})
	q.Add(Item{ID: "current", Tier: CurrentTask, Weight: 0, CreatedAt: epoch})

	item, ok := q.Pop(epoch, 0, time.Hour)
	if !ok || item.ID != "blocker" {
		t.Fatalf("Pop = (%+v, %v), want the blocker item", item, ok)
	}
	item, ok = q.Pop(epoch, 0, time.Hour)
	if !ok || item.ID != "current" {
		t.Fatalf("Pop = (%+v, %v), want the current-task item", item, ok)
	}
	item, ok = q.Pop(epoch, 0, time.Hour)
	if !ok || item.ID != "background" {
		t.Fatalf("Pop = (%+v, %v), want the background item", item, ok)
	}
}

func TestQueue_RunsToCompleteness(t *testing.T) {
	var q Queue
	for i := 0; i < 5; i++ {
		q.Add(Item{Tier: Background, CreatedAt: epoch})
	}

	processed := 0
	for {
		_, ok := q.Pop(epoch, 0, time.Hour)
		if !ok {
			break
		}
		processed++
	}

	if processed != 5 {
		t.Errorf("processed %d items, want 5 (the loop should run until no actionable item remains)", processed)
	}
	if q.Len() != 0 {
		t.Errorf("q.Len() = %d, want 0", q.Len())
	}
}

func TestQueue_EmptyQueue(t *testing.T) {
	var q Queue
	if _, ok := q.Next(epoch, 0, time.Hour); ok {
		t.Error("Next on an empty queue = ok true, want false")
	}
	if _, ok := q.Pop(epoch, 0, time.Hour); ok {
		t.Error("Pop on an empty queue = ok true, want false")
	}
}
