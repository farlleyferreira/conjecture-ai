# Proposal

## Why

Explicit direction: README.md should be understandable by someone with
zero technical background, not just a developer already familiar with
spec-driven tooling. Took structural inspiration (not text) from
Fission-AI/OpenSpec's own README - lead with a plain value-proposition
sentence, a philosophy block, a "see it in action" transcript, then
Quick Start - while keeping every fact grounded in this project's own
real commands and behavior, unchanged.

Follow-up explicit direction: the default `README.md` should be in
pt-BR, with an option to read it in English - not the other way around.

## What Changes

- `README.md` restructured: adds a one-sentence tagline, a plain-words
  analogy ("a rule for a game") ahead of the existing business-rule
  example, a "what `conjecture` also does" bullet list, real CI/license
  badges, and `> [!NOTE]` callouts for the build/LLM-provider asides that
  were previously dense prose paragraphs.
- No fact changed: the worked example (shipping rule, the real bug, the
  real fix), the commands table, the "what this is not for" section, and
  the "Status" section are carried over with their content intact (the
  commands table additionally gained `queue`'s own `review` subcommand,
  already real but not previously mentioned there).
- Added a real badge for the existing `ci.yml` workflow and a static MIT
  license badge - no new CI job, nothing invented.
- `README.md` is now the pt-BR translation (prose only - every code
  block, command name, flag, env var, and file path left byte-identical
  to the English version, never translated); the prior English content
  moved to a new `README.en.md`. Each file carries a one-line language
  switcher at the top linking to the other.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - documentation only, no behavior change. `skip_specs: true`.

## Impact

- Affected paths: `README.md`, `README.en.md` (new).
- No code or test changes; `go build`/`go vet`/`go test ./...` all pass
  unchanged (docs-only).
- Verified every link added or kept resolves to something real in this
  repository: `.github/workflows/ci.yml` exists, `LICENSE` exists,
  `internal/llmclient/openaicompatible.go` exists, issue #144 was already
  linked before this change.
