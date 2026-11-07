package main

import (
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

const queueUsage = "usage: conjecture queue add <id> <blocker|current-task|background> [--weight <N>] [--severity <N>]\n       conjecture queue list\n       conjecture queue next"

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
// one item), "list" (shows every queued item, unordered), and "next"
// (pops and reports the single highest-priority item, by
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

func runQueueList(args []string, stdout, stderr io.Writer) int {
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

	items, err := workqueue.List(db)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(stdout, "queue is empty")
		return 0
	}
	for _, it := range items {
		_, _ = fmt.Fprintf(stdout, "%s: %s, weight %d, severity %d, queued %s\n", it.ID, tierString(it.Tier), it.Weight, it.Severity, it.CreatedAt.Format(time.RFC3339))
	}
	return 0
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
