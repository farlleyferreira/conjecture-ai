# bootstrap-material Specification

## Purpose
Defines how conjecture ingests a reference architecture document and bridges
to an initial observation harness before a stable, first-party harness
exists for a given codebase.

## Requirements

### Requirement: Reference document ingestion as authority material
The system SHALL accept a reference architecture document as a knowledge base
with norm-level authority, and SHALL extract candidate statements from it for
human review rather than accepting them automatically.

#### Scenario: Extracted statement requires review
- **WHEN** the system extracts a candidate statement from an ingested
  reference document
- **THEN** that statement is presented for human review before it is treated
  as an accepted rule

### Requirement: Initial harness as temporary source of truth
Before a stable harness exists, the system SHALL treat a minimal initial
harness as the source of truth for observation, following a documented
minimal protocol.

#### Scenario: Initial harness used before a stable one exists
- **WHEN** no stable harness is yet registered for a codebase
- **THEN** the system uses the initial harness's minimal protocol to observe
  behavior

### Requirement: Known-answer probes detect a wrong harness
The system SHALL run known-answer probes against any harness (initial or
stable) and SHALL flag the harness as wrong when a probe's result does not
match its known answer, when output does not conform to the expected schema,
or when the same input and seed produce different results across runs.

#### Scenario: Harness flagged wrong on probe mismatch
- **WHEN** a known-answer probe's result does not match its expected answer
- **THEN** the system flags the harness as wrong rather than trusting its
  other results

#### Scenario: Non-reproducible output flagged
- **WHEN** the same input and seed produce different results on repeated
  runs against the same harness
- **THEN** the system flags the harness as wrong

### Requirement: Harness promotion requires human confirmation
A harness SHALL only be promoted from initial to stable, and accepted as the
source of truth going forward, after explicit human confirmation.

#### Scenario: Harness not promoted without confirmation
- **WHEN** a harness has passed its probes but has not been explicitly
  confirmed by a human
- **THEN** the system does not treat it as the stable source of truth
