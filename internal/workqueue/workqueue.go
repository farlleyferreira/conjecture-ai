// Package workqueue implements the work-queue capability (openspec/
// specs/work-queue/spec.md): the single, three-tier queue that drives
// what conjecture works on next, replacing fixed phases or personas
// with a queue derived from current state.
package workqueue

import "time"

// Tier is one of the specification's three priority tiers, in
// descending priority order: a Blocker item always preempts a
// CurrentTask item, which always preempts a Background item ("The
// system SHALL maintain exactly one work queue with three tiers...
// SHALL NOT maintain separate per-phase or per-persona queues").
type Tier int

const (
	Blocker Tier = iota
	CurrentTask
	Background
)

// Item is one unit of work in the queue.
type Item struct {
	ID        string
	Tier      Tier
	Weight    int
	Severity  int
	CreatedAt time.Time
}

// EffectiveWeight returns item's Weight as adjusted for how long it has
// waited, at the given instant - "The system SHALL increase an item's
// effective priority the longer it remains unaddressed, so a low-
// weight item is not starved indefinitely." agingRate (added to Weight
// for every unit of age elapsed) must be positive for that guarantee to
// hold: effective weight must grow without bound as age grows, so that
// any item, however low its raw Weight, eventually outranks any fixed
// higher-weight item - "Long-waiting item eventually surfaces."
//
// The specification gives no concrete rate or unit, only this
// qualitative requirement, so this package does not hardcode one:
// calibrating agingRate and unit for a real deployment is a policy
// decision - exactly the kind of thing optimization-suite names as a
// human-gated calibration activity, not something to invent here.
func EffectiveWeight(item Item, now time.Time, agingRate float64, unit time.Duration) float64 {
	age := now.Sub(item.CreatedAt)
	return float64(item.Weight) + agingRate*(float64(age)/float64(unit))
}

// Less reports whether a should be worked on before b: by Tier first
// (a Blocker always before a CurrentTask item, always before a
// Background one - the one and only tier ordering, since the
// specification rules out any other queue), then within a tier by
// EffectiveWeight (higher first), then Severity (higher first), then
// CreatedAt (oldest first) - "Ordering by weight, then severity, then
// age."
func Less(a, b Item, now time.Time, agingRate float64, unit time.Duration) bool {
	if a.Tier != b.Tier {
		return a.Tier < b.Tier
	}
	aw, bw := EffectiveWeight(a, now, agingRate, unit), EffectiveWeight(b, now, agingRate, unit)
	if aw != bw {
		return aw > bw
	}
	if a.Severity != b.Severity {
		return a.Severity > b.Severity
	}
	return a.CreatedAt.Before(b.CreatedAt)
}

// Queue is the specification's single, three-tier work queue.
type Queue struct {
	items []Item
}

// Add appends item to the queue.
func (q *Queue) Add(item Item) {
	q.items = append(q.items, item)
}

// Len reports how many items remain in the queue.
func (q *Queue) Len() int {
	return len(q.items)
}

// Next returns the item [Less] ranks highest at now, without removing
// it, or (Item{}, false) if the queue is empty.
func (q *Queue) Next(now time.Time, agingRate float64, unit time.Duration) (Item, bool) {
	i, ok := q.bestIndex(now, agingRate, unit)
	if !ok {
		return Item{}, false
	}
	return q.items[i], true
}

// Pop removes and returns the item [Next] would return - "The system
// SHALL keep checking the queue in a loop until no actionable item
// remains, rather than stopping after a fixed number of steps": a
// caller drives that loop by calling Pop until it reports false.
func (q *Queue) Pop(now time.Time, agingRate float64, unit time.Duration) (Item, bool) {
	i, ok := q.bestIndex(now, agingRate, unit)
	if !ok {
		return Item{}, false
	}
	item := q.items[i]
	q.items = append(q.items[:i], q.items[i+1:]...)
	return item, true
}

func (q *Queue) bestIndex(now time.Time, agingRate float64, unit time.Duration) (int, bool) {
	if len(q.items) == 0 {
		return 0, false
	}
	best := 0
	for i := 1; i < len(q.items); i++ {
		if Less(q.items[i], q.items[best], now, agingRate, unit) {
			best = i
		}
	}
	return best, true
}
