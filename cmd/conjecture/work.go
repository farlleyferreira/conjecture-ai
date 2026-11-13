package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/llmclient"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

const workUsage = "usage: conjecture work [--provider <name>] [--max-rounds <N>]"

// workMaxAttemptsPerItem is this command's own disclosed default for
// how many times a single item may be popped and retried within one
// "conjecture work" run before this run leaves it alone. "Loop runs
// to completeness, not to a fixed step count" ("the system SHALL keep
// checking the queue in a loop until no actionable item remains")
// does not say what to do with an item that keeps failing and being
// re-enqueued (recordEvidence's own existing behavior on a
// NonConformant verdict - see check.go) - retrying it forever within
// one invocation would spend unbounded LLM budget chasing one
// possibly-unfixable hypothesis while other, genuinely actionable
// items wait. Once exhausted, the item is restored to the real queue
// (never lost) for a later run, or "conjecture queue next", to pick
// up - this run simply stops contending it for the top spot.
const workMaxAttemptsPerItem = 3

// runWork implements "conjecture work" - work-queue's own "Loop runs
// to completeness" requirement, the one piece of that capability
// nothing before this wired: the queue itself, its ordering, and one
// producer (check.go, on a confirmed violation) and one consumer
// (context.go's own "resume", read-only) already existed; this is
// the first command that actually pops an item and acts on it.
//
// "Acting on" a work-queue item means running it through
// runImplementLoop - the exact same round-loop "conjecture implement"
// already drives by hand, one hypothesis at a time - since every item
// this codebase's own producer ever enqueues (check.go's
// recordEvidence) is a confirmed violation: a hypothesis or law
// needing its implementation fixed, which is exactly what implement
// does. This command does not invent a second notion of "work."
//
// An item runImplementLoop refuses outright (ErrDomainTooComplexForOracle
// - implement.go) is never retried: a deterministic refusal, not a
// failure another attempt could resolve. It moves to the dead letter
// instead (workqueue.AddDeadLetter) - still queryable via "conjecture
// queue review," never automatically re-selected again. A dead-
// lettered blocker-tier item stops this run outright, the same
// "blocker preempts background work" precedence work-queue's own
// spec already gives extended to "an unresolvable blocker halts
// progress, it does not just get skipped"; a current-task or
// background-tier one is logged and this run moves on to whatever
// else remains queued.
func runWork(args []string, stdout, stderr io.Writer) int {
	provider := "anthropic"
	maxRounds := implementDefaultMaxRounds
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, workUsage)
				return 1
			}
			provider = args[i]
		case "--max-rounds":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, workUsage)
				return 1
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				_, _ = fmt.Fprintf(stderr, "conjecture: --max-rounds must be a positive integer, got %q\n", args[i])
				return 1
			}
			maxRounds = n
		default:
			_, _ = fmt.Fprintln(stderr, workUsage)
			return 1
		}
	}

	client, err := llmclient.New(provider)
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
	if err := workqueue.EnsureDeadLetterSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := evidenceledger.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	sb := &sandbox.Sandbox{}
	return workLoop(context.Background(), db, sb, client, ".", maxRounds, stdout)
}

// workLoop is "conjecture work"'s own store/root-independent half, so
// a test can exercise it against a temporary store and a temporary
// root directory instead of the real per-OS ones runWork always
// resolves to.
func workLoop(ctx context.Context, db *sql.DB, sb *sandbox.Sandbox, client llmclient.Client, root string, maxRounds int, stdout io.Writer) int {
	attempts := make(map[string]int)
	var held []workqueue.Item
	exitCode := 0

	for {
		item, found, err := workqueue.Pop(db, time.Now().UTC(), queueAgingRate, queueAgingUnit)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "conjecture: ERROR popping the queue: %v\n", err)
			exitCode = 1
			break
		}
		if !found {
			_, _ = fmt.Fprintln(stdout, "queue is empty")
			break
		}

		if attempts[item.ID] >= workMaxAttemptsPerItem {
			_, _ = fmt.Fprintf(stdout, "%s: exhausted %d attempts this run, leaving for a later run\n", item.ID, workMaxAttemptsPerItem)
			held = append(held, item)
			exitCode = 1
			continue
		}
		attempts[item.ID]++

		path, ruleID, hypothesisID, err := resolveQueueItem(item.ID, root)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: ERROR: %v\n", item.ID, err)
			exitCode = 1
			continue
		}

		f, err := loadFicha(path)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: ERROR loading %s: %v\n", item.ID, path, err)
			exitCode = 1
			continue
		}

		rule, h, isLaw, err := findHypothesis(f, ruleID, hypothesisID)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: ERROR: %v\n", item.ID, err)
			exitCode = 1
			continue
		}

		_, _ = fmt.Fprintf(stdout, "%s: working (attempt %d/%d)\n", item.ID, attempts[item.ID], workMaxAttemptsPerItem)
		specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
		verdict, err := runImplementLoop(ctx, db, sb, client, rule, h, maxRounds, specVersion, isLaw, f.concepts, f.tables, stdout)
		if errors.Is(err, ErrDomainTooComplexForOracle) {
			if dlErr := workqueue.AddDeadLetter(db, workqueue.DeadLetterItem{
				ID: item.ID, Tier: item.Tier, Weight: item.Weight, Severity: item.Severity,
				Reason: err.Error(), CreatedAt: time.Now().UTC(),
			}); dlErr != nil {
				_, _ = fmt.Fprintf(stdout, "%s: ERROR moving to the dead letter: %v\n", item.ID, dlErr)
			}
			exitCode = 1
			if item.Tier == workqueue.Blocker {
				_, _ = fmt.Fprintf(stdout, "%s: blocker item moved to the dead letter (%v) - stopping; run \"conjecture queue review\" and resolve it before continuing\n", item.ID, err)
				break
			}
			_, _ = fmt.Fprintf(stdout, "%s: moved to the dead letter (%v)\n", item.ID, err)
			continue
		}
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: ERROR: %v\n", item.ID, err)
			exitCode = 1
			continue
		}
		_, _ = fmt.Fprintf(stdout, "%s: %s\n", item.ID, verdict)
		if verdict != conformancecycle.Conformant {
			exitCode = 1
		}
	}

	for _, item := range held {
		if err := workqueue.Add(db, item); err != nil {
			_, _ = fmt.Fprintf(stdout, "conjecture: ERROR restoring %q to the queue: %v\n", item.ID, err)
		}
	}

	return exitCode
}

// resolveQueueItem splits a work-queue item's own id - exactly the
// "area::phenomenon::rule::hypothesis" or "area::phenomenon::
// hypothesis" (a file-level law) composite id check.go's own
// recordEvidence builds it from - back into the .cjtr file path
// cjtr-language's own path convention already names (area is its
// directory, phenomenon is its filename, both already used by
// loadFicha), relative to root, and the ruleID/hypothesisID
// findHypothesis needs.
func resolveQueueItem(id, root string) (path, ruleID, hypothesisID string, err error) {
	segments := strings.Split(id, "::")
	switch len(segments) {
	case 3:
		return joinPhenomenonPath(root, segments[0], segments[1]), "", segments[2], nil
	case 4:
		return joinPhenomenonPath(root, segments[0], segments[1]), segments[2], segments[3], nil
	default:
		return "", "", "", fmt.Errorf("work: %q is not a recognized composite id (want area::phenomenon::hypothesis or area::phenomenon::rule::hypothesis)", id)
	}
}

// joinPhenomenonPath resolves a .cjtr file's real on-disk path from
// root (the repository root) and its own area/phenomenon -
// persistence's own "Git-tracked specification files" layout
// ([persistence.EnsureWorkingLayout]'s own "conjecture/phenomena/"
// tree, the one "conjecture init" actually creates), not a layout
// this command invents of its own. Confirmed by running "conjecture
// work" for real against a real "conjecture init"-created repository
// as a non-root user: the prior version of this function (missing
// the "conjecture/" segment) could never find a real file at all.
func joinPhenomenonPath(root, area, phenomenon string) string {
	return filepath.Join(root, "conjecture", "phenomena", area, phenomenon+".cjtr")
}
