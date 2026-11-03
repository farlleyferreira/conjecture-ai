# persistence Specification

## Purpose
Defines where conjecture state lives: what is tracked in the project's git
repository versus what stays local-only, and where local-only data is stored
per operating system.

## Requirements

### Requirement: Git-tracked specification files
The system SHALL store `.cjtr` files under `conjecture/phenomena/<area>/`,
one file per phenomenon, as regular git-tracked files. Materialization output
SHALL be stored under a `materializations` path, and team configuration SHALL
be stored as `settings.cjtr`, both also git-tracked.

#### Scenario: Phenomenon file is one per phenomenon
- **WHEN** a new phenomenon is specified
- **THEN** the system creates one `.cjtr` file for it under
  `conjecture/phenomena/<area>/`

### Requirement: Everything else stays local-only
The system SHALL store all evidence, verdicts, context graph data, and
decisions outside the git repository, in a local, per-repository SQLite
database, and SHALL add the local conjecture working directory to
`.gitignore`.

#### Scenario: Verdicts never committed to git
- **WHEN** a verdict is produced
- **THEN** it is written to the local SQLite store, never into a `.cjtr`
  file or any other git-tracked file

### Requirement: Single SQLite store, no dedicated graph database
The system SHALL use exactly one local SQLite database per repository for
all non-git-tracked state. The system SHALL NOT introduce a second database
or a dedicated graph database.

#### Scenario: Context graph queried from the same store
- **WHEN** the context engine needs to traverse the graph
- **THEN** it queries the same per-repository SQLite database used for
  evidence and verdicts

### Requirement: Append-only, hash-chained tables for evidence
Evidence, decisions, and configuration history SHALL be stored in
append-only tables, each record chained by hash to the previous one.

#### Scenario: Historical record not overwritten
- **WHEN** a new evidence record is written
- **THEN** the previous record for that hypothesis remains in the table,
  linked by hash, rather than being overwritten

### Requirement: Per-OS local data directory
The local data directory SHALL be read-only to the implementing agent and
located per operating system: `/var/lib/conjecture/<repo-id>/` on Linux,
`/Library/Application Support/conjecture/<repo-id>/` on macOS, and
`%ProgramData%\conjecture\<repo-id>\` on Windows.

#### Scenario: Agent cannot write to the local data directory
- **WHEN** an implementing or drafting LLM session attempts to write
  directly to the local data directory
- **THEN** the system denies the write
