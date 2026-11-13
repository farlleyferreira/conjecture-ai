# Proposal

## Why

A usability review of `queue_list` (added in an earlier block of this
round) found a real scaling hole: `workqueue.List` returns every queued
item with no cap, so even that "cheap first call" an agent would use to
decide whether to drill further stops being cheap for a pathologically
large queue. An arbitrary item-count limit (`--limit 50`) would fix the
raw size problem but carry no meaning of its own - which 50, chosen how.

A separate discussion, grounding this project's own real hierarchy
(area → phenomenon → rule → hypothesis/axiom/law, distinct from the
Hypothesis→Thesis→Theory→Law maturity ladder), supplied the missing
piece: every queued item's own composite id already carries its
area::phenomenon prefix (`resolveQueueItem`'s own two shapes, work.go).
That is a real, already-grounded structural boundary to group by -
bounded by how many distinct phenomena have queued items, which does not
grow the way the number of items can, unlike any count-based limit.

## What Changes

- `conjecture queue list` and the `queue_list` MCP tool now return a
  per-area::phenomenon summary (item count, broken down by tier) by
  default, instead of every raw item.
- `--area <area>` narrows the summary to that area's own phenomena.
- `--area <area> --phenomenon <phenomenon>` drills all the way down to
  that one phenomenon's own raw items instead of the summary - the only
  way to see individual items now. `--phenomenon` alone (no `--area`) is
  rejected: a phenomenon name alone cannot be resolved to one area, since
  two distinct areas may each have a phenomenon of the same name.
- `queue_list`'s own MCP output gains real structure (`summaries`/
  `items`, mirroring `status`'s own precedent) instead of the shared
  `{ok, report}` text-only shape every other read-only tool in that file
  still uses - this is the one of them with real per-field structure
  worth a bespoke type.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `work-queue`: no requirement text changes - no requirement names
  `list`'s own output shape (it is this project's own CLI invention, the
  same honesty note `runQueue`'s own doc comment already carries).
  `skip_specs: true`.

## Impact

- Affected paths: `cmd/conjecture/queue.go`, `cmd/conjecture/queue_test.go`,
  `cmd/conjecture/mcp.go`, `cmd/conjecture/mcp_read_tools.go`,
  `cmd/conjecture/mcp_read_tools_test.go`.
- Anything that previously parsed `conjecture queue list`'s flat,
  per-item text output breaks: the default view's shape changed. No
  internal caller of this command's own text output exists in this
  codebase today (`runQueueList` has no other caller besides `runQueue`'s
  own dispatch), so nothing inside this repository is affected.
- No new dependency.
