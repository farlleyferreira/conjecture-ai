# Proposal

## Why

`queue_list`'s own drill-down (`--area` + `--phenomenon`) returns one
phenomenon's own raw items with no cap - a disclosed, accepted Non-Goal
of the prior `work-queue-hierarchical-list` change, reasoned to be low-
risk since it is bounded by how many hypotheses/laws one `.cjtr` file
declares, not by project scale. Explicit instruction: warn when that
assumption looks wrong for real, specifically when a drill-down's own
count is more than 3x "what's understood as ideal."

No specification or prior design in this codebase names a concrete
"ideal" hypotheses-per-phenomenon count - inventing one here would be
exactly the guess this project's own discipline has avoided throughout.
Grounded instead in the project's own real, observed data: the average
item count of every *other* phenomenon currently in the queue.

## What Changes

- `computeQueueList`'s drill-down branch (`--area` + `--phenomenon`)
  now also returns a `warning` string, non-empty only when the
  phenomenon's own item count exceeds `queueScaleWarningMultiplier`
  (3) times the average item count of every other phenomenon currently
  in the queue.
- With no other phenomenon to compare against, `warning` stays empty -
  there is no baseline to be unusual relative to, so this never
  guesses one.
- `conjecture queue list` prints it as a `WARNING:` line; the
  `queue_list` MCP tool's own output gains a `warning` field.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `work-queue`: no requirement text changes - this is a CLI/MCP-level
  disclosure refinement of a Non-Goal the prior change already named,
  not a new boundary. `skip_specs: true`.

## Impact

- Affected paths: `cmd/conjecture/queue.go`, `cmd/conjecture/
  queue_test.go`, `cmd/conjecture/mcp_read_tools.go`,
  `cmd/conjecture/mcp_read_tools_test.go`.
- `computeQueueList`'s own signature gains a fourth return value
  (`warning`); both of its existing callers (`runQueueList`,
  `queueListTool`) are updated.
- Verified for real as a non-root user in WSL: 6 queued items across 3
  phenomena (1 each in two, 4 in a third) - the drill-down into the
  4-item phenomenon reported the warning; the other two did not.
- No new dependency.
