# context-composition Specification

## Purpose
Defines versioned, composable contexts that can be listed, resumed, compared,
and restored, so different verification contexts do not have to be rebuilt
from scratch or silently lost.

## Requirements

### Requirement: Context lifecycle commands
The system SHALL support listing, resuming, opening, inspecting origin,
diffing, and restoring a context, corresponding to `conjecture context
list/resume/open/origin/diff/restore`.

#### Scenario: Prior context resumed
- **WHEN** a user runs `conjecture context resume` for an existing context
- **THEN** the system restores that context's state for continued use

### Requirement: Contexts are archived, never deleted
The system SHALL archive a superseded context rather than deleting it, so it
remains available via `origin`, `diff`, or `restore`.

#### Scenario: Superseded context still diffable
- **WHEN** a context is superseded by a newer one
- **THEN** a user can still diff the superseded context against the new one

### Requirement: Four composition modes
The system SHALL support four ways to compose contexts: by reference, by
embedding, by inheritance, and by view.

#### Scenario: Composition by reference does not copy content
- **WHEN** a context is composed by reference
- **THEN** the composed context points to the source context's content
  rather than duplicating it

### Requirement: Explicit conflict precedence
When composed contexts conflict, the system SHALL resolve the conflict
according to an explicit, documented precedence order, not an unspecified or
arbitrary rule.

#### Scenario: Conflict resolved by documented precedence
- **WHEN** two composed contexts assert conflicting values for the same
  node
- **THEN** the system resolves the conflict using its documented precedence
  order and reports which source won
