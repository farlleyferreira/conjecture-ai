# Design

## Context

See proposal.md - Why. `persistence` already owns the local store path
and `ChainHash`; `evidenceledger` and `teamconfig` each own one
append-only table. Human decisions currently happen in three commands
and are only printed. `conjecture init` does not exist; `draft` writes a
user-supplied path and nothing creates the phenomenon tree.

## Goals / Non-Goals

**Goals:**
- Give every human determination this codebase asks for a durable,
  attributable, tamper-evident record in the same local store.
- Make the repository layout the specification names a one-command,
  idempotent operation with the local directory gitignored.

**Non-Goals:**
- Materialization content. The requirement names the path; generating
  projections is a separate capability with its own design.
- Enforcing "read-only to the implementing agent" at the OS level.
  There is no separate process identity to deny: the agent runs as the
  same user as the engine, and the directory is already created 0700.
  Claiming a denial the code cannot perform would be theater; this is
  recorded as the remaining boundary.
- A generic audit UI; `List` is the read surface.

## Decisions

1. **Decisions are their own append-only, hash-chained table.** A
   decision is not evidence (it is a determination, not an
   observation), and mixing them would blur the evidence profile
   promotion weighs. Each row's `chain_hash` covers the previous row's
   hash plus a canonical JSON payload, reusing
   `persistence.ChainHash`; `Verify` recomputes the whole chain so a
   corrupted or reordered history is detectable.

2. **A decision names its kind, subject, determination, rationale, and
   who decided.** The three wired producers have stable kinds
   (`promotion`, `harness-promotion`, `candidate-review`); `Append`
   refuses an empty kind/subject/determination/decider, so an
   unattributed decision cannot be recorded. Rationale may be empty
   (a rejected candidate needs none), unlike the other fields.

3. **Recording a decision is part of the action, so a failure fails
   the command.** If `promote` appended its evidence but could not
   append the decision, the command reports an error instead of
   claiming success: the point of the record is that the determination
   is attributable, and silently missing it would reopen the gap this
   change closes. This can leave the primary action applied with a
   reported error; that ordering is disclosed here rather than hidden.

4. **The layout is created only by `conjecture init`, never as a side
   effect of another command.** Creating directories and editing
   `.gitignore` from `draft` or `check` would surprise a repository the
   user has not opted into. `init` is idempotent: existing directories
   are left alone, and `.conjecture/` is appended to `.gitignore` only
   when no line already names it, preserving whatever the file
   contained.

5. **The local working directory is `.conjecture/` at the repository
   root.** The specification says the local conjecture working
   directory must be gitignored but does not name it; `.conjecture/`
   is the obvious repo-local scratch location and does not collide with
   the git-tracked `conjecture/` tree. Durable state stays where it
   already is (the per-OS SQLite store outside the repository); this
   directory is for local scratch, and making it exist plus gitignore
   it is the enforceable half of "everything else stays local-only".

## Risks / Trade-offs

- **Decision recording happens after the primary write** → on a
  store failure the action is applied without its decision and the
  command reports an error. Retrying the command may then hit an
  already-promoted state; the error text is the signal to inspect.
- **Hash-chain verification is only called on demand** (`Verify`), not
  on every append: it is O(n) and not needed for the write path. A
  caller that wants tamper detection (a future `conjecture decisions`
  audit command) invokes it.
- **`.gitignore` editing is minimal**: append one line, no rewrite.
  A repository with an unusual ignore structure still gets a correct
  entry; `init` does not reformat the file.
- **Read-only data directory remains unenforced** for the reason above;
  the proposal and this design say so instead of claiming otherwise.
