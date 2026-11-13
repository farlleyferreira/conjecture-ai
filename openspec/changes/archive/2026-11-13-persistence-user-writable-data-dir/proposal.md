# Proposal

## Why

`persistence`'s "Per-OS local data directory" requirement names a
system-owned path for every OS: `/var/lib/conjecture/<repo-id>/` on
Linux, `/Library/Application Support/conjecture/<repo-id>/` on macOS,
`%ProgramData%\conjecture\<repo-id>\` on Windows. Every one of these is
only writable by root/an administrator, not by the ordinary user
account conjecture actually runs as.

Confirmed for real: running the built `conjecture` binary as a non-root
user in a fresh WSL Ubuntu session (`su -s /bin/sh nobody -c '... conjecture
mcp'`) fails immediately -

```
conjecture: persistence: creating local data directory: mkdir /var/lib/conjecture/50ca9e21acdb681a: permission denied
```

`persistence.OpenStore` is called at the top of `runMCP`, `runCheck`,
and `runSign`'s confirmed path, before any tool or command logic runs.
A non-root user cannot use any of them - the CLI, the MCP server, or
the main verification loop - at all, on any of the three named
platforms, without first manually creating and chowning the directory
(something nothing in this codebase does for them, and something a
real end user would never think to do).

The prior `persistence-decisions-and-layout` change's own design.md
already disclosed that the "read-only to the implementing agent" half
of this requirement is not enforced at the OS level and cannot be
without a separate process identity the agent and the engine do not
have: "the agent runs as the same user as the engine, and the
directory is already created 0700. Claiming a denial the code cannot
perform would be theater." Putting the store under a root/admin-owned
path adds nothing toward that already-conceded boundary - whoever runs
`conjecture` already has full read/write access to it regardless of
which directory it is, root-owned or not. The system path only adds a
real, confirmed failure, in exchange for no real enforcement.

## What Changes

- `internal/persistence.localDataDir` resolves to a per-user-writable
  path instead: `$XDG_DATA_HOME/conjecture/<repo-id>` (falling back to
  `$HOME/.local/share/conjecture/<repo-id>`) on Linux, `$HOME/Library/
  Application Support/conjecture/<repo-id>` on macOS, `%LOCALAPPDATA%\
  conjecture\<repo-id>` on Windows. The directory is still created
  `0700` (owner-only), exactly as today.
- `openspec/specs/persistence/spec.md`'s "Per-OS local data directory"
  requirement is updated to name these paths instead of the
  root/admin-owned ones. The "read-only to the implementing agent"
  scenario text is left exactly as it is - already a disclosed,
  separate gap from a prior change, not something this change claims
  to close or reopens.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `persistence`: "Per-OS local data directory" requirement text
  changes (the three named paths). Edited directly in this change
  (`skip_specs: true` - no delta-spec sync needed, the main spec.md is
  edited in place, same as the prior persistence change).

## Impact

- Affected paths: `internal/persistence/persistence.go`,
  `internal/persistence/persistence_test.go`,
  `openspec/specs/persistence/spec.md`.
- Every repository's local store moves location on upgrade - this is
  a fresh, previously-unusable-for-most-users path, not a migration:
  there is no real prior data at the old root-owned paths to carry
  forward for a non-root user, since they could never write there to
  begin with. A user who happened to run conjecture as root before this
  change will see a fresh, empty store the first time they run it as
  themselves after upgrading; this is disclosed, not silently patched
  over.
- No new dependency.
