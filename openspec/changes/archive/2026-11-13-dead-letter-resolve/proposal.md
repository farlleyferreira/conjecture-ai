# Proposal

## Why

The dead-letter queue (`work_queue_dead_letter`, this same round of work's
immediately prior change) gave "conjecture queue review" a way to list
every item a disposable implementing oracle refused outright, but no way
to clear one: `AddDeadLetter`/`ListDeadLetter` were the only two
operations. A human reviewing the same unresolved item run after run,
with no way to say "handled," is a genuine gap, found by re-checking the
dead-letter source file's own function list directly rather than assuming
completeness from memory. Explicit direction was to resolve it.

## What Changes

- `internal/workqueue` gains a `resolved_at` column on
  `work_queue_dead_letter` (added to the `CREATE TABLE IF NOT EXISTS`
  for a fresh store, and migrated onto an existing one via
  `ALTER TABLE ... ADD COLUMN`, guarded by a `PRAGMA table_info` check,
  since the table already shipped in the prior change and may already
  exist on a real repository without this column).
- `DeadLetterItem` gains `RowID` (the table's own auto-increment id,
  needed to target one specific occurrence since `AddDeadLetter` already
  allows the same composite ID to recur) and `ResolvedAt` (zero =
  unresolved).
- `ListDeadLetter` takes an `includeResolved bool`: false (the default)
  returns only unresolved items; true also returns resolved ones, as an
  audit trail.
- New `ResolveDeadLetter(db, rowID)` marks one row resolved; it errors if
  the row does not exist or is already resolved, rather than silently
  no-op-ing.
- `conjecture queue review` gains `--resolve <row-id>` and `--all`; each
  listed row now leads with its own row id so a human has what
  `--resolve` needs. The `queue_review` MCP tool stays pure read (gains
  an `all` argument only) - resolving stays CLI-only, the same
  "decisions stay human, never exposed via MCP" precedent `sign
  --confirmed`/`promote` already set.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `work-queue`: no requirement text changes - the dead letter itself is
  this project's own invention, not spec text; this change only adds a
  way to clear an entry from it. `skip_specs: true`.

## Impact

- Affected paths: `internal/workqueue/deadletter.go`,
  `internal/workqueue/deadletter_test.go`, `cmd/conjecture/queue.go`,
  `cmd/conjecture/queue_test.go`, `cmd/conjecture/mcp.go`,
  `cmd/conjecture/mcp_read_tools.go`,
  `cmd/conjecture/mcp_read_tools_test.go`.
- A repository that already has a `work_queue_dead_letter` table from the
  prior change gets `resolved_at` added to it in place on the next
  `EnsureDeadLetterSchema` call (every dead-letter-touching command runs
  this), losing no existing rows.
- Verified for real in WSL: seeded a dead-letter row via the real
  `workqueue` package against the real per-OS store the compiled
  `conjecture` binary itself opens, then ran `queue review`,
  `queue review --resolve <row-id>`, confirmed it dropped from the
  default listing, confirmed `--all` still shows it with its own
  resolved time, and confirmed resolving the same row twice errors
  instead of silently succeeding.
- No new dependency.
