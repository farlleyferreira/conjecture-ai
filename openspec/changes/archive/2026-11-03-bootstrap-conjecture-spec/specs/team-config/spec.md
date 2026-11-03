# Spec Delta

## Purpose

Defines the shared, versioned `settings.cjtr` file that sets a team-wide
configuration floor, never a ceiling, for individual contributors.

## ADDED Requirements

### Requirement: Versioned, git-tracked team settings
The system SHALL store team-wide configuration in a `settings.cjtr` file,
tracked in git alongside `.cjtr` phenomenon files.

#### Scenario: Settings file reviewed like a spec change
- **WHEN** `settings.cjtr` changes
- **THEN** the change is visible in the normal git history and review flow,
  like any other tracked file

### Requirement: Floor, not ceiling
A setting in `settings.cjtr` SHALL define a minimum requirement that every
contributor's local configuration must meet. A local configuration SHALL be
allowed to exceed it, but SHALL NOT be allowed to fall below it.

#### Scenario: Local override below the floor rejected
- **WHEN** a local configuration attempts to set a value stricter-than or
  equal-to the team floor is required but instead sets a weaker value
- **THEN** the system rejects the local override

#### Scenario: Local override above the floor accepted
- **WHEN** a local configuration sets a stricter value than the team floor
  requires
- **THEN** the system accepts the local override

### Requirement: Signed like any other `.cjtr` file
`settings.cjtr` SHALL go through the same signature mechanism as other
`.cjtr` files, with a content hash excluding its own `signed` block.

#### Scenario: Settings change requires re-signing
- **WHEN** `settings.cjtr` content changes
- **THEN** its previous signature no longer matches and it must be re-signed

### Requirement: Compiled binary stays local
The team configuration's compiled form SHALL only exist locally on each
machine; it SHALL NOT be distributed as a shared pre-compiled artifact.

#### Scenario: Compilation happens on each machine
- **WHEN** `settings.cjtr` is loaded on a given machine
- **THEN** its compiled form is produced locally on that machine rather than
  fetched from a shared location
