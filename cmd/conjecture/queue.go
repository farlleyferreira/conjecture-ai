package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

const queueUsage = "usage: conjecture queue add <id> <blocker|current-task|background> [--weight <N>] [--severity <N>]\n" +
	"       conjecture queue list [--area <area>] [--phenomenon <phenomenon>]\n" +
	"       conjecture queue next"

// queueAgingRate and queueAgingUnit are this CLI's own default aging
// parameters for [workqueue.Next]/[workqueue.Pop]: +1.0 effective
// weight per 24 hours an item waits. workqueue's own EffectiveWeight
// doc comment is explicit that the specification gives no concrete
// rate or unit - "calibrating agingRate and unit for a real deployment
// is a policy decision" - so this is this command's own chosen
// default, not a value the specification names, picked only so the
// CLI has something concrete to run with rather than requiring every
// invocation to supply it.
const (
	queueAgingRate = 1.0
	queueAgingUnit = 24 * time.Hour
)

// runQueue implements "conjecture queue <subcommand>": "add" (queues
// one item), "list" (a per-phenomenon summary by default, or - given
// both --area and --phenomenon - every item under exactly one
// phenomenon; see [computeQueueList]'s own doc comment for why), and
// "next" (pops and reports the single highest-priority item, by
// [workqueue.Less]'s own tier/weight/severity/age order) - work-
// queue's own "single, three-tier work queue that drives what
// conjecture works on next."
func runQueue(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, queueUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]

	switch subcommand {
	case "add":
		return runQueueAdd(rest, stdout, stderr)
	case "list":
		return runQueueList(rest, stdout, stderr)
	case "next":
		return runQueueNext(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, queueUsage)
		return 1
	}
}

func runQueueAdd(args []string, stdout, stderr io.Writer) int {
	var id, tierFlag string
	weight, severity := 1, 0
	var positional []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--weight", "--severity":
			flag := args[i]
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, queueUsage)
				return 1
			}
			n, err := strconv.Atoi(args[i])
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "conjecture: %s must be an integer, got %q\n", flag, args[i])
				return 1
			}
			if flag == "--weight" {
				weight = n
			} else {
				severity = n
			}
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 2 {
		_, _ = fmt.Fprintln(stderr, queueUsage)
		return 1
	}
	id, tierFlag = positional[0], positional[1]

	tier, err := parseTier(tierFlag)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	db, err := openRepoStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := workqueue.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	item := workqueue.Item{ID: id, Tier: tier, Weight: weight, Severity: severity, CreatedAt: time.Now().UTC()}
	if err := workqueue.Add(db, item); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "queued %s (%s, weight %d, severity %d)\n", id, tierFlag, weight, severity)
	return 0
}

// runQueueList implements "conjecture queue list [--area <area>]
// [--phenomenon <phenomenon>]" - see [computeQueueList]'s own doc
// comment for why its default view is a per-phenomenon summary
// rather than [workqueue.List]'s own raw, uncapped item list.
func runQueueList(args []string, stdout, stderr io.Writer) int {
	var area, phenomenon string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--area", "--phenomenon":
			flag := args[i]
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, queueUsage)
				return 1
			}
			if flag == "--area" {
				area = args[i]
			} else {
				phenomenon = args[i]
			}
		default:
			_, _ = fmt.Fprintln(stderr, queueUsage)
			return 1
		}
	}

	db, err := openRepoStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := workqueue.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	items, err := workqueue.List(db)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	summaries, queueItems, warning, err := computeQueueList(items, area, phenomenon)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	if area != "" && phenomenon != "" {
		if warning != "" {
			_, _ = fmt.Fprintf(stdout, "WARNING: %s\n", warning)
		}
		if len(queueItems) == 0 {
			_, _ = fmt.Fprintf(stdout, "%s::%s: queue is empty\n", area, phenomenon)
			return 0
		}
		for _, it := range queueItems {
			_, _ = fmt.Fprintf(stdout, "%s: %s, weight %d, severity %d, queued %s\n", it.ID, it.Tier, it.Weight, it.Severity, it.Queued)
		}
		return 0
	}

	if len(summaries) == 0 {
		_, _ = fmt.Fprintln(stdout, "queue is empty")
		return 0
	}
	for _, s := range summaries {
		_, _ = fmt.Fprintf(stdout, "%s: %d items (blocker %d, current-task %d, background %d)\n", s.PhenomenonID, s.Total, s.Blocker, s.CurrentTask, s.Background)
	}
	return 0
}

// PhenomenonQueueSummary is one area::phenomenon's own queue summary -
// "conjecture queue list"'s own default view.
type PhenomenonQueueSummary struct {
	PhenomenonID string `json:"phenomenon_id"`
	Blocker      int    `json:"blocker"`
	CurrentTask  int    `json:"current_task"`
	Background   int    `json:"background"`
	Total        int    `json:"total"`
}

// QueueItemOut is one work-queue item's own MCP/JSON-facing shape -
// the same fields [workqueue.Item] already has, rendered the same way
// [tierString] and RFC3339 already render them for the CLI's own
// per-item line.
type QueueItemOut struct {
	ID       string `json:"id"`
	Tier     string `json:"tier"`
	Weight   int    `json:"weight"`
	Severity int    `json:"severity"`
	Queued   string `json:"queued"`
}

// queueScaleWarningMultiplier is how far above this project's own
// typical phenomenon size a drill-down's own item count has to be
// before [computeQueueList] warns about it - explicitly chosen (by
// the person asking for this, not derived from anything this codebase
// measures): "avisa quando o número for 3x o que se entende como
// ideal." No specification or prior design in this codebase names a
// concrete "ideal" hypotheses-per-phenomenon count to compare
// against, and inventing one here would be exactly the guess this
// project's own discipline avoids - see [computeQueueList]'s own doc
// comment for what this is compared against instead.
const queueScaleWarningMultiplier = 3

// computeQueueList is "conjecture queue list"'s and the "queue_list"
// MCP tool's own shared read.
//
// [workqueue.List] itself returns every queued item with no cap -
// fine for a handful of items, but for a pathologically large queue
// even that first, supposedly cheap call stops being cheap, and
// nothing bounds how much an agent reading it would have to read. The
// composite id every queued item already carries
// ("area::phenomenon::rule::hypothesis", or "area::phenomenon::
// hypothesis" for a file-level law - resolveQueueItem's own two
// shapes, work.go) gives a real, already-grounded structural boundary
// to group by instead of an arbitrary item-count limit that would
// carry no meaning of its own: the number of distinct phenomena with
// queued items does not grow the way the number of items can.
//
// With no filter, this returns the per-phenomenon summary (grouped
// via [queueSummaries]), bounded by phenomenon count. With area alone,
// the same summary narrowed to that area. With both area and
// phenomenon, it drills all the way to that one phenomenon's own
// items instead - never a mix of both in one call, so a caller always
// knows which of the two return values is populated. phenomenon alone
// (no area) is rejected: two distinct areas may each have a
// phenomenon of the same name, so a bare phenomenon name cannot be
// resolved to one.
//
// The drill-down's own item list has no cap of its own - realistically
// bounded by how many hypotheses/laws one .cjtr file declares, a much
// smaller scale than the whole-queue problem this function's own
// summary already solves. warning is non-empty only when that
// assumption looks wrong for real: the phenomenon being drilled into
// has more than [queueScaleWarningMultiplier] times the average item
// count of every *other* phenomenon currently in the queue - this
// project's own real, observed typical size, not an invented universal
// constant. With no other phenomenon to compare against (this is the
// only one in the queue), there is no baseline to be unusual relative
// to, so warning stays empty rather than guessing one.
func computeQueueList(items []workqueue.Item, area, phenomenon string) (summaries []PhenomenonQueueSummary, queueItems []QueueItemOut, warning string, err error) {
	if phenomenon != "" && area == "" {
		return nil, nil, "", errors.New("--phenomenon requires --area (a phenomenon name alone cannot be resolved to one area)")
	}

	if area != "" && phenomenon != "" {
		phenomenonID := area + "::" + phenomenon
		filtered := queueItemsForPhenomenon(items, phenomenonID)
		queueItems = make([]QueueItemOut, len(filtered))
		for i, it := range filtered {
			queueItems[i] = QueueItemOut{
				ID: it.ID, Tier: tierString(it.Tier), Weight: it.Weight,
				Severity: it.Severity, Queued: it.CreatedAt.Format(time.RFC3339),
			}
		}

		var otherTotal, otherCount int
		for _, s := range queueSummaries(items) {
			if s.PhenomenonID == phenomenonID {
				continue
			}
			otherTotal += s.Total
			otherCount++
		}
		if otherCount > 0 {
			average := float64(otherTotal) / float64(otherCount)
			if average > 0 && float64(len(filtered)) > queueScaleWarningMultiplier*average {
				warning = fmt.Sprintf(
					"%s has %d queued items, more than %dx this project's own average of %.1f per phenomenon elsewhere in the queue",
					phenomenonID, len(filtered), queueScaleWarningMultiplier, average,
				)
			}
		}
		return nil, queueItems, warning, nil
	}

	summaries = queueSummaries(items)
	if area != "" {
		summaries = filterSummariesByArea(summaries, area)
	}
	return summaries, nil, "", nil
}

// queueItemPhenomenonID extracts an item's own area::phenomenon
// prefix - the first two "::"-separated segments, present whether the
// item's own id has 3 segments (a file-level law) or 4 (a rule-scoped
// hypothesis/law), the same two shapes [resolveQueueItem] (work.go)
// already distinguishes for a different purpose (resolving a file
// path, not grouping).
func queueItemPhenomenonID(id string) (string, bool) {
	segments := strings.Split(id, "::")
	if len(segments) != 3 && len(segments) != 4 {
		return "", false
	}
	return segments[0] + "::" + segments[1], true
}

// queueSummaries groups items by their own area::phenomenon prefix,
// in deterministic (sorted) order. An id [queueItemPhenomenonID]
// cannot parse - never produced by this codebase's own producer
// (check.go's recordEvidence), but not rejected by [workqueue.Add]
// either - is grouped under its own literal id instead of being
// silently dropped, the same "disclose, don't swallow" posture every
// other command in this codebase already takes.
func queueSummaries(items []workqueue.Item) []PhenomenonQueueSummary {
	byKey := make(map[string]*PhenomenonQueueSummary)
	var order []string
	for _, it := range items {
		key, ok := queueItemPhenomenonID(it.ID)
		if !ok {
			key = it.ID
		}
		s, found := byKey[key]
		if !found {
			s = &PhenomenonQueueSummary{PhenomenonID: key}
			byKey[key] = s
			order = append(order, key)
		}
		switch it.Tier {
		case workqueue.Blocker:
			s.Blocker++
		case workqueue.CurrentTask:
			s.CurrentTask++
		case workqueue.Background:
			s.Background++
		}
		s.Total++
	}
	sort.Strings(order)
	summaries := make([]PhenomenonQueueSummary, len(order))
	for i, key := range order {
		summaries[i] = *byKey[key]
	}
	return summaries
}

// queueItemsForPhenomenon returns every item in items whose own
// [queueItemPhenomenonID] equals phenomenonID exactly.
func queueItemsForPhenomenon(items []workqueue.Item, phenomenonID string) []workqueue.Item {
	var out []workqueue.Item
	for _, it := range items {
		if key, ok := queueItemPhenomenonID(it.ID); ok && key == phenomenonID {
			out = append(out, it)
		}
	}
	return out
}

// filterSummariesByArea keeps only the summaries whose own
// PhenomenonID starts with "<area>::" - safe as a plain string
// prefix match (not a false-positive risk against a differently-named
// area that merely starts with the same letters) because "::" never
// appears inside an area name itself, only as the segment separator.
func filterSummariesByArea(summaries []PhenomenonQueueSummary, area string) []PhenomenonQueueSummary {
	prefix := area + "::"
	var out []PhenomenonQueueSummary
	for _, s := range summaries {
		if strings.HasPrefix(s.PhenomenonID, prefix) {
			out = append(out, s)
		}
	}
	return out
}

func runQueueNext(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		_, _ = fmt.Fprintln(stderr, queueUsage)
		return 1
	}

	db, err := openRepoStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := workqueue.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	return popNext(db, stdout)
}

// popNext is "conjecture queue next"'s store-independent half, so a
// test can exercise it against a temporary store instead of the real
// per-OS one [persistence.OpenStore] always resolves to.
func popNext(db *sql.DB, stdout io.Writer) int {
	item, found, err := workqueue.Pop(db, time.Now().UTC(), queueAgingRate, queueAgingUnit)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "conjecture: %v\n", err)
		return 1
	}
	if !found {
		_, _ = fmt.Fprintln(stdout, "queue is empty")
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "%s: %s, weight %d, severity %d\n", item.ID, tierString(item.Tier), item.Weight, item.Severity)
	return 0
}

func parseTier(s string) (workqueue.Tier, error) {
	switch s {
	case "blocker":
		return workqueue.Blocker, nil
	case "current-task":
		return workqueue.CurrentTask, nil
	case "background":
		return workqueue.Background, nil
	default:
		return 0, fmt.Errorf("%q is not a recognized tier (want blocker, current-task, or background)", s)
	}
}

func tierString(t workqueue.Tier) string {
	switch t {
	case workqueue.Blocker:
		return "blocker"
	case workqueue.CurrentTask:
		return "current-task"
	case workqueue.Background:
		return "background"
	default:
		return "unknown tier"
	}
}
