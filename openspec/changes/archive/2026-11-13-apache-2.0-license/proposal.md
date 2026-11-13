# Proposal

## Why

While scoping closed-beta readiness, a permissive non-commercial
restriction was considered (PolyForm Noncommercial / BUSL) for
preventing unauthorized commercial use. Explicit direction was to keep
the project permissively licensed instead, switching from MIT to the
Apache License, Version 2.0 - still a fully permissive OSI-approved
open source license (commercial use allowed, same as MIT), but adding
an explicit patent grant and patent-litigation retaliation clause MIT
does not have.

## What Changes

- `LICENSE` replaced with the canonical Apache License, Version 2.0
  text, with the Appendix's copyright line attributed to
  "farlleyferreira, 2026" (the same holder/year the prior MIT license
  named).
- `README.md`'s License section now names Apache 2.0 instead of MIT.
- The scaffolded orchestration skill's own frontmatter (`cmd/conjecture/
  templates/conjecture-workflow-skill.md`) updated from `license: MIT`
  to `license: Apache-2.0`, so a project scaffolded by `conjecture init`
  carries the same license this project itself uses, not a stale one.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - a licensing change, not a behavior change. `skip_specs: true`.

## Impact

- Affected paths: `LICENSE`, `README.md`,
  `cmd/conjecture/templates/conjecture-workflow-skill.md`.
- No code behavior changes; `go build`/`go vet`/`go test ./...` all
  still pass unchanged.
- A repository whose scaffolded skill file still matches its last
  recorded checksum picks up the `license: Apache-2.0` line
  automatically on its next `conjecture init`
  (skill-update-tracking's own existing mechanism); one a human has
  since customized is left alone, same as any other template change.
- This does not retroactively relicense any prior release already
  distributed under MIT - only the repository's current state going
  forward.
