# Proposal

## Why

persistence's own "Append-only, hash-chained tables for evidence" names
three tables: evidence, decisions, and configuration history. Evidence
and configuration history now exist; decisions do not, so the human
decisions this codebase already makes (promotions, harness promotion,
candidate review) leave no attributable record outside stdout. Its
"Git-tracked specification files" requirement also names a
`conjecture/phenomena/<area>/` layout and a materializations path that no
command creates, and "Everything else stays local-only" asks for the
local working directory to be in `.gitignore` - there is no such
directory or entry.

## What Changes

- Add `internal/decisionledger`: an append-only, hash-chained `decisions`
  table (kind, subject, determination, rationale, decided by, created
  at, chain hash) with `Append`, `List`, and `Verify` (recomputes every
  chain link).
- Record a decision wherever this codebase already asks a human for an
  explicit determination: `conjecture promote` (Theory/Law),
  `conjecture harness promote` (stable), and `conjecture base review`
  (accept/reject). The attribution travels with the decision.
- Add `persistence.EnsureWorkingLayout(repoRoot)`: creates
  `conjecture/phenomena/`, `conjecture/materializations/`, and the
  local `.conjecture/` working directory, and adds `.conjecture/` to
  the repository's `.gitignore` (creating the file when absent,
  preserving existing content, idempotent).
- Add `conjecture init [--dir <path>]`, the command that applies that
  layout.
- Materialization *content* generation is not attempted: the layout
  requirement names the path, and the projection feature itself has its
  own design. The read-only-for-the-agent half of the data-directory
  requirement is disclosed as not enforceable without a separate
  process identity (the directory is already created owner-only).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `persistence`: no requirement text changes - "Git-tracked
  specification files", "Everything else stays local-only", and
  "Append-only, hash-chained tables for evidence" already state this
  behavior; this change fills the missing decisions table, the working
  layout, and the `.gitignore` entry. `skip_specs: true`.

## Impact

- Affected paths: `internal/decisionledger` (new),
  `internal/persistence/layout.go` (new), `cmd/conjecture/init.go`
  (new), `promote.go`, `harness.go`, `base.go` (decision records),
  `main.go` (dispatch), tests.
- The local SQLite store gains `decisions` (additive); a repository
  gains three directories and one `.gitignore` line only when
  `conjecture init` runs.
- No new dependency.
