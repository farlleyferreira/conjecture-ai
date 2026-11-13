# Proposal

## Why

A re-audit for remaining gaps after merging the dead-letter resolve
change (#165) found that `queue_review` - the MCP tool for reading
dead-lettered items, shipped in #164 - was never added to the scaffolded
orchestration skill's own "Everything else this server exposes" table.
An agent following only that skill has no way to learn `queue_review`
exists, so it never surfaces dead-lettered items or tells the user about
`--resolve`/`--all`.

## What Changes

- `cmd/conjecture/templates/conjecture-workflow-skill.md` gains a
  `queue_review` row in its tool table, describing the default-
  unresolved-only/`all` behavior and that resolving stays human-only,
  CLI-only.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - documentation only, no behavior change. `skip_specs: true`.

## Impact

- Affected paths: `cmd/conjecture/templates/conjecture-workflow-skill.md`.
- A repository whose scaffolded skill file still matches its last
  recorded checksum picks this row up automatically on the next
  `conjecture init` (skill-update-tracking's own existing mechanism);
  one a human has since customized is left alone, same as any other
  template change.
- No new dependency.
