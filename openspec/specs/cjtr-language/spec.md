# cjtr-language Specification

## Purpose
Defines the `.cjtr` declarative specification format: a closed grammar and
controlled vocabulary that lets a person write falsifiable rules without
writing a programming language in disguise.

## Requirements

### Requirement: Closed vocabulary and grammar
Every `.cjtr` statement SHALL be composed only from a fixed, English-phrase
controlled vocabulary and a fixed grammar. The system SHALL reject a statement
that uses a word or construct outside that vocabulary.

#### Scenario: Unknown word rejected
- **WHEN** a `.cjtr` file contains a statement using a word not present in the
  controlled vocabulary
- **THEN** the system rejects the file and reports the unknown word and its
  location

### Requirement: Single source of truth for vocabulary
The system SHALL generate its parser, JSON Schema, autocomplete data,
Portuguese translation, and tests from one single vocabulary data source,
never from independently maintained copies.

#### Scenario: Vocabulary entry added
- **WHEN** a new word is added to the vocabulary source
- **THEN** the parser, schema, autocomplete, translation, and tests all reflect
  the new word without any of them being edited by hand

### Requirement: Strict YAML loading
The system SHALL load a `.cjtr` file as YAML under a strict profile: no
anchors, no references, no custom tags, and duplicate keys SHALL be rejected
rather than silently overwritten. Every scalar value SHALL be read as text and
then typed according to the concept it belongs to, never by YAML's own
implicit type inference.

#### Scenario: Implicit YAML typing avoided
- **WHEN** a `.cjtr` file contains the scalar `100.00` for a value that the
  concept declares as an exact decimal
- **THEN** the system preserves it as an exact decimal and does not convert it
  through a lossy floating-point representation

#### Scenario: Anchor or tag rejected
- **WHEN** a `.cjtr` file contains a YAML anchor, reference, or custom tag
- **THEN** the system rejects the file instead of resolving or executing it

### Requirement: Composite, content-derived identifiers
Every node in a `.cjtr` file SHALL have a composite identifier built from its
parent's identifier and its own name (`parent-id::child-id`), derived from the
area, phenomenon, rule, and hypothesis names actually written. The system
SHALL NOT assign freestanding generated identifiers (such as UUIDs).

#### Scenario: Rename detected by content
- **WHEN** a hypothesis is renamed but its parent rule and content stay the
  same
- **THEN** the system can recognize it as the same node moved/renamed rather
  than a new, unrelated node

#### Scenario: Prefix-based verification scope
- **WHEN** a user runs verification scoped to a composite ID prefix (for
  example `shipping::free-shipping::free-above`)
- **THEN** the system verifies exactly the nodes whose composite ID starts with
  that prefix

### Requirement: Format versioning
Each `.cjtr` file SHALL declare a `conjecture: MAJOR.MINOR.PATCH` format
version. The system SHALL treat a MAJOR change as altering which guarantees,
verdicts, or decisions the file depends on; a MINOR change as additive and
non-breaking; and a PATCH change as a wording-only correction. This version
SHALL be included in the file's content hash.

#### Scenario: Unsupported MAJOR version rejected
- **WHEN** a `.cjtr` file declares a `conjecture` MAJOR version the installed
  system does not support
- **THEN** the system refuses to load the file rather than guessing at
  compatibility

### Requirement: Two-layer separation of statements and code
The grammar SHALL keep natural-language statements and any supporting
executable code in two separate layers. The system SHALL only ever write, into
the `.cjtr` file, pending markers (`???`) and bilingual comments; it SHALL
never write executable code or evidence into the file on its own.

#### Scenario: System proposes a diff, never a direct edit
- **WHEN** the system determines a `.cjtr` file should change
- **THEN** it proposes a diff for human review rather than writing the change
  directly into the signed file
