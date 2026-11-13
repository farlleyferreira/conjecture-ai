# Design

## Context

See proposal.md - Why. Confirmed by running the real built binary as a
non-root user: every command that touches the local store
(`persistence.OpenStore`) fails at startup, before reaching any tool
or command logic. This affects `conjecture mcp` (fails before any MCP
tool can even be registered), `conjecture check`, and `conjecture
sign --confirmed`.

## Goals / Non-Goals

**Goals:**
- Let the ordinary, non-root/non-admin user account conjecture
  actually runs as open and use its local store, on all three named
  platforms, with no manual setup.
- Keep the directory owner-only (`0700`), the same real boundary the
  prior persistence change already settled on.

**Non-Goals:**
- Enforcing "read-only to the implementing agent" at the OS level.
  Already disclosed as not enforceable in the prior
  `persistence-decisions-and-layout` design.md, for the same reason
  restated in proposal.md - Why. This change does not attempt it
  either; moving the path does not change who the directory's owning
  process is.
- A migration path for any pre-existing root-owned store. There is
  essentially no real data to migrate: a non-root user could never
  write to the old path, so any data there belongs to whoever ran
  conjecture as root, not to the ordinary user switching to the new
  path.

## Decisions

1. **Per-user data directories, the standard convention for each OS,
   not a conjecture-invented location.** `$XDG_DATA_HOME` (or `$HOME/
   .local/share`) on Linux is the XDG Base Directory spec's own
   answer to "where does a user-level application store its data."
   `$HOME/Library/Application Support` on macOS and `%LOCALAPPDATA%`
   on Windows are each platform's own standard per-user app-data
   location - chosen because they are already guaranteed writable by
   whichever account is running the process, without this codebase
   inventing its own convention or asking the user to create anything.

2. **`localDataDir`'s per-OS `goos` parameter is preserved**, so the
   existing cross-platform test pattern (exercising all three branches
   from a single dev machine) keeps working unchanged - only the
   literal path construction inside each branch changes.

3. **A missing required env var is a reported error, not a silent bad
   path.** If `HOME` is empty on Linux/macOS, or `LOCALAPPDATA` is
   empty on Windows, `localDataDir` returns an error rather than
   silently building a path like `/.local/share/conjecture/<id>`
   (filesystem root) - which would reintroduce exactly the permission
   failure this change exists to remove, just one directory shallower.
   `XDG_DATA_HOME` being empty is not an error: falling back to
   `$HOME/.local/share` is the XDG spec's own documented default, not
   a missing-input failure.

## Risks / Trade-offs

- **No automatic migration from the old root-owned path.** Disclosed
  above, not attempted - there is no real non-root data there to move.
- **A user who ran conjecture as root before this change** finds a
  fresh, empty store under their own account afterward. This is the
  expected, disclosed consequence of fixing a path that was never
  genuinely usable without root to begin with.
