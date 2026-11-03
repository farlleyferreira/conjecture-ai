# Spec Delta

## Purpose

Defines how external knowledge bases (norms, definitions, references,
history) are attached to a project and scoped, so conjecture can cite
authority material instead of relying only on the `.cjtr` files themselves.

## ADDED Requirements

### Requirement: CLI-based base attachment
The system SHALL provide a command to attach a knowledge base by path,
authority type, and scope, in the form
`conjecture base add <path> --authority <norm|definition|reference|history>
--scope <...>`.

#### Scenario: Base attached with authority and scope
- **WHEN** a user runs `conjecture base add ./normas/ --authority norm
  --scope project`
- **THEN** the system registers that path as a norm-authority base scoped to
  the project

### Requirement: Authority levels govern precedence
The system SHALL support at least the authority levels norm, definition,
reference, and history, and SHALL use a base's declared authority level when
resolving a conflict between bases.

#### Scenario: Norm authority takes precedence over reference
- **WHEN** a norm-authority base and a reference-authority base disagree on
  the same point
- **THEN** the system resolves the conflict in favor of the norm-authority
  base

### Requirement: Required metadata for norm-authority bases
A base registered with norm authority SHALL carry a content hash and a last
review date. The system SHALL reject or flag a norm-authority base missing
either.

#### Scenario: Norm base missing review date flagged
- **WHEN** a base is registered with `--authority norm` and no review date
- **THEN** the system flags the registration as incomplete

### Requirement: Local indexing for retrieval
The system SHALL index attached bases locally using both textual search and
quantized embeddings, without sending base content to a remote service.

#### Scenario: Base content stays local
- **WHEN** a knowledge base is indexed
- **THEN** no base content is transmitted to a remote service as part of
  indexing

### Requirement: Scoped application
Each registered base SHALL have an explicit scope (for example, project-wide
or limited to an area), and the system SHALL only apply a base's content
within its declared scope.

#### Scenario: Area-scoped base not applied outside its area
- **WHEN** a base is registered with `--scope area:shipping`
- **THEN** the system does not apply that base's content when judging or
  verifying a different area
