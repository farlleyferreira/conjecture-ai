# Proposal

## Why

A prior change (the `implement`/`work` complexity gate) disclosed and
accepted a trade-off: a work-queue item `runImplementLoop` refuses
outright (`ErrDomainTooComplexForOracle`) was simply dropped from the
queue on first attempt - "the agent driving conjecture still finds it
via `status` or a direct `check`," not a lost signal, just not one
`work` surfaced proactively. A later usability review re-opened this
as still genuinely worth fixing: explicit direction was to move the
item to a review queue, immediately (no wasted retries, since the
refusal is deterministic), and - if the item was blocker-tier - stop
`work`'s own run outright until a human resolves it, extending
work-queue's own "blocker preempts background work" precedence to "an
unresolvable blocker halts progress, it does not just get skipped."

Verifying this for real as a non-root user (the same discipline this
whole round of work has followed) surfaced a second, unrelated but
real bug: `joinPhenomenonPath` (work.go) resolved a queue item's
composite id to `root/phenomena/<area>/<phenomenon>.cjtr` - missing
the `conjecture/` segment every real `conjecture init`-created
repository actually has. `conjecture work` could never find a real
file in a real repository; every existing test passed only because
its own fixtures were built to match that same missing-segment
convention, never against persistence's own real layout. Explicit
direction was to fix this in the same change, since it blocked
verifying the dead-letter feature at all.

## What Changes

- `internal/workqueue` gains a secondary, append-only dead-letter
  table (`DeadLetterItem`/`EnsureDeadLetterSchema`/`AddDeadLetter`/
  `ListDeadLetter`) - distinct from the main queue's own schema, never
  read by `Next`/`Pop`, so nothing here is ever picked up
  automatically again.
- `workLoop` (work.go), on `ErrDomainTooComplexForOracle`, moves the
  item to the dead letter instead of dropping it, then: stops the run
  outright if the item was blocker-tier (reporting that a human must
  resolve it via `conjecture queue review` before continuing); logs it
  and continues to the rest of the queue otherwise.
- Adds `conjecture queue review` (and the `queue_review` MCP tool) -
  pure read, lists every dead-lettered item; nothing resolves or
  removes an entry but a human, directly.
- Fixes `joinPhenomenonPath` to resolve through `conjecture/phenomena/`
  (persistence's own real layout), not `phenomena/` alone. Updates
  every fixture/test that built its own directory to match the old,
  wrong convention.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `work-queue`: no requirement text changes - a dead letter is this
  project's own invention for a case ("deterministically unresolvable
  by automation") the specification does not name; the blocker-halts
  behavior is this project's own extension of "blocker preempts
  background work," not a change to that requirement's own text.
  `skip_specs: true`.

## Impact

- Affected paths: `internal/workqueue/deadletter.go` (new),
  `internal/workqueue/deadletter_test.go` (new), `cmd/conjecture/
  work.go`, `cmd/conjecture/work_test.go`, `cmd/conjecture/queue.go`,
  `cmd/conjecture/queue_test.go`, `cmd/conjecture/check_test.go`,
  `cmd/conjecture/context_test.go`, `cmd/conjecture/implement_test.go`,
  `cmd/conjecture/mcp.go`, `cmd/conjecture/mcp_read_tools.go`,
  `cmd/conjecture/mcp_read_tools_test.go`, `cmd/conjecture/mcp_test.go`.
- `conjecture work` on a real repository behaves differently going
  forward even for cases unrelated to the complexity gate: it can now
  actually find the `.cjtr` files it is supposed to act on, which it
  could not before this change in any real (non-test-fixture)
  repository.
- A repository gains one more local table (no new file - the same
  per-repository SQLite store every other piece of local state already
  lives in).
- Verified for real as a non-root user in WSL: queued a blocker-tier
  item for an unbounded-domain hypothesis, ran `conjecture work` -
  found the real file (the path fix), moved the item to the dead
  letter, stopped the run, and `conjecture queue review` reported it
  with its own real reason.
- No new dependency.
