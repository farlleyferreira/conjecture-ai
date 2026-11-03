# Spec Delta

## Purpose

Defines how conjecture observes a target system purely through declared
contracts, so it works across any programming language without understanding
that language's semantics.

## ADDED Requirements

### Requirement: Observation only through declared contracts
The system SHALL observe a target system's behavior only through explicitly
declared contracts: commands, external interfaces (HTTP, CLI, files, stdio,
messages), and standard result formats. The system SHALL NOT parse or
interpret target source code to determine behavior.

#### Scenario: Behavior observed via declared command, not source parsing
- **WHEN** the system verifies a rule against a target system
- **THEN** it invokes the declared command/interface and reads its declared
  output format, without parsing the target's source code for semantics

### Requirement: Standard result formats
The system SHALL accept target output in standard formats, including JSON
validated against a schema, and test-result formats such as JUnit XML or
TAP.

#### Scenario: JUnit XML result consumed
- **WHEN** a target system reports its test results in JUnit XML
- **THEN** the system parses that standard format to determine pass/fail
  outcomes

### Requirement: Read-only dependency analysis
The system SHALL use a read-only, language-agnostic parsing approach (such as
Tree-sitter) only for dependency-rule checks, and SHALL NOT use it to infer
business semantics.

#### Scenario: Dependency rule checked without semantic parsing
- **WHEN** a dependency rule (for example, a forbidden import) is checked
- **THEN** the check uses read-only structural parsing, not semantic
  interpretation of the code's behavior

### Requirement: Optional runtime adapters for introspection
The system MAY use an optional, per-runtime adapter for deeper introspection
where available, but a rule's conformance verdict SHALL NOT depend on any
such adapter being present.

#### Scenario: Verification still works without a runtime adapter
- **WHEN** no runtime-specific adapter is installed for a target's runtime
- **THEN** the system still produces a conformance verdict using the
  declared contracts alone

### Requirement: Generic accessibility driver for GUI targets
For a GUI target, the system SHALL use a generic accessibility-based driver
to observe behavior, rather than a framework-specific UI automation
integration.

#### Scenario: GUI observed via accessibility driver
- **WHEN** the target under test is a GUI application
- **THEN** the system observes its behavior through the generic
  accessibility driver
