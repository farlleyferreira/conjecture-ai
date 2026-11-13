# Design

## Context

See proposal.md - Why. This follows from two prior discussions in the
same round: the `queue_list` scaling hole identified while auditing the
"many queued items" question, and the real hierarchy grounding
(area/phenomenon/rule/axiom/hypothesis, distinct from the maturity
ladder) that supplies the grouping key.

## Goals / Non-Goals

**Goals:**
- Bound the default view by a quantity that does not grow unboundedly in
  normal use (distinct phenomena with queued items), not an arbitrary
  item count.
- Reuse the one structural boundary this codebase's own composite ids
  already carry, rather than inventing a new one.

**Non-Goals:**
- Pagination of the drill-down (`--area` + `--phenomenon`) item list
  itself. A single phenomenon's own queued items are bounded by how many
  hypotheses/laws it declares, which this project has no reason to
  expect at a scale needing its own cap; if that assumption turns out
  wrong in real use, that is a separate, later change.
- Deriving the grouping from anything other than the item's own id
  (e.g. a cached join against the `.cjtr` file itself) - the id already
  carries the exact prefix needed, and reading the file would add a
  real filesystem dependency this read does not otherwise have.

## Decisions

1. **Group by the id's own first two segments, not by re-parsing the
   source file.** `queueItemPhenomenonID` only ever looks at the id
   string already stored on the item - no filesystem access, no new
   failure mode this read did not already have.
2. **A malformed id is grouped under itself, not dropped.** Consistent
   with every other "disclose, don't swallow" choice in this codebase:
   an id `workqueue.Add` never validates the shape of (it only requires
   it be non-empty) should still be visible somewhere if one ever
   existed, not silently absorbed into a count that does not account
   for it.
3. **Exactly one of `summaries`/`items` is ever populated, never both.**
   A caller checking which one is non-nil always gets an unambiguous
   answer; mixing both into one call's response for the three-way
   no-filter/area-only/area+phenomenon branching would need its own
   disambiguation anyway.
4. **`queue_list`'s MCP output breaks from the shared `textReadOutput`
   shape the other seven read-only tools in `mcp_read_tools.go` use.**
   Those seven wrap an existing CLI function with no richer per-field
   structure worth a bespoke type; this one now does, the same
   reasoning `status` already applies elsewhere.

## Risks / Trade-offs

- **A caller that depended on `queue_list`'s old flat-item-list default
  breaks.** Disclosed in proposal.md - Impact: no caller inside this
  codebase exists, but an external agent that already memorized the old
  shape (from the skill text, or its own prior session) would need to
  learn the new one. Not mitigated with a compatibility flag - the old
  shape was the exact scaling hole this change exists to close, so
  preserving it as an option would just move the hole behind a flag
  nobody is told to avoid.
