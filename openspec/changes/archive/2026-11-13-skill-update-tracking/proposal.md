# Proposal

## Why

The `conjecture-workflow` skill's own design.md (block 4 of this round)
disclosed a real gap and left it open: a repository's already-
scaffolded skill file never picks up a later improvement to the
bundled template - only a fresh `conjecture init` on a repository that
never had the file would. A later usability review re-surfaced this
as a genuinely open item, not yet decided permanently closed.

## What Changes

- `ensureWorkflowSkill` now records the SHA-256 of whatever it writes
  to `.claude/skills/conjecture-workflow/SKILL.md` at
  `.conjecture/skill-checksum` (the repository's own local, never
  git-tracked working directory - persistence's "Everything else
  stays local-only").
- On a later `conjecture init`, if the file on disk still matches that
  recorded checksum (never edited since conjecture last wrote it) and
  the bundled template has since changed, the file is updated and the
  checksum refreshed. If the file is already identical to the current
  template, nothing happens. If the recorded checksum is missing or
  does not match what is on disk - including a repository scaffolded
  before this change existed, which has no recorded checksum at all -
  the file is left exactly as it is: this function can only ever prove
  a file is *unedited*, never that it has been, so anything less than
  that proof defaults to leaving it alone.
- `ensureWorkflowSkill` returns which of the four outcomes
  (`SkillCreated`/`SkillUpToDate`/`SkillUpdated`/`SkillLeftCustomized`)
  actually happened; "conjecture init" reports the real one instead of
  one fixed message regardless of outcome.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cli-mcp-integration`: no requirement text changes - "Onboarding
  skill scaffold" already covers writing the skill; this only
  refines what re-running "conjecture init" does to an
  already-scaffolded one. `skip_specs: true`.

## Impact

- Affected paths: `cmd/conjecture/skillscaffold.go`, `cmd/conjecture/
  skillscaffold_test.go`, `cmd/conjecture/init.go`.
- A repository gains one more local file
  (`.conjecture/skill-checksum`), only when `conjecture init` writes
  or updates the skill - never git-tracked.
- `ensureWorkflowSkill`'s own signature changes (`error` ->
  `(SkillScaffoldAction, error)`); its one caller (`runInit`) is
  updated.
- Verified for real as a non-root user in WSL: created, left
  unchanged on a second run, left a human edit untouched, and updated
  a file that matched its own recorded checksum but predated a
  (simulated) newer bundled version - all four outcomes confirmed.
- No new dependency.
