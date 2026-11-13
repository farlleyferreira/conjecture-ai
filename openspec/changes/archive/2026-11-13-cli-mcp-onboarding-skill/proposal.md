# Proposal

## Why

Blocks 0-3 of this round of work gave an agent a real MCP surface
(verify, check, status, read_evidence, and eight more read-only
context/queue/graph tools) and a real, confirmed-working local store.
What is still missing, confirmed directly by trying to use this
project: nothing teaches an agent *how* to drive that surface, or in
what order. Without it, every agent that picks up conjecture has to
rediscover the whole draft -> verify -> human-signs -> implement ->
check loop by reading this codebase's own Go source, the same way
this session did before any of blocks 0-3 existed, and the same risk
exists of an agent running `conjecture sign --confirmed` or `conjecture
promote` itself via a shell tool, defeating the human-only boundary
cli-mcp-integration's own permission table already establishes.

This project's own OpenSpec dependency already solves the equivalent
problem for itself: it ships its own orchestration know-how as Claude
Code skill files (`.claude/skills/<name>/SKILL.md`) placed directly
into the consuming repository - this project's own `.claude/skills/
openspec-*/` directories are exactly that, already present in this
checkout. Matching that already-established distribution convention,
rather than inventing a new one, is the explicit direction given for
this change.

## What Changes

- `conjecture init` now also writes a bundled orchestration skill to
  `.claude/skills/conjecture-workflow/SKILL.md` in the repository it
  initializes - idempotently, same as the rest of its own layout: an
  already-present file (a user's own edit) is never overwritten.
- The skill's content (`cmd/conjecture/templates/conjecture-workflow-
  skill.md`, embedded at build time) teaches the real loop this
  codebase actually supports today: draft the `.cjtr` file directly,
  verify it via the "verify" tool, stop and have the human run
  `conjecture sign --confirmed` themselves (explicitly never run on
  the agent's own behalf, by any channel), write the real
  implementation with the agent's own normal coding ability (not
  `conjecture implement`/`work`/`attack`/`swarm`, which is a separate,
  already-complete feature of this tool), verify it for real via the
  "check" tool, and report the result via "status". It also tables
  every other read-only tool blocks 0-3 added, and an explicit "never
  do this" list naming the two human-only actions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cli-mcp-integration`: adds one requirement, "Onboarding skill
  scaffold" - `conjecture init` ships the orchestration know-how into
  the repository using this project's own already-established
  distribution convention, rather than leaving an agent to rediscover
  the workflow from source. Edited directly in this change
  (`skip_specs: true` - no delta-spec sync needed, the main spec.md is
  edited in place, same as the two earlier changes this round).

## Impact

- Affected paths: `cmd/conjecture/skillscaffold.go` (new),
  `cmd/conjecture/templates/conjecture-workflow-skill.md` (new,
  embedded), `cmd/conjecture/init.go`, `openspec/specs/cli-mcp-
  integration/spec.md`.
- A repository gains one more file (and, where needed, the
  `.claude/skills/conjecture-workflow/` directory) only when
  `conjecture init` runs, same trigger as the rest of the layout.
- No new dependency (`embed` is standard library).
