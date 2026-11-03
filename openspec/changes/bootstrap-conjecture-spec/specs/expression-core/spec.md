# Spec Delta

## Purpose

Defines the deterministic expression engine that every `.cjtr` statement
compiles to, which the judge and conformance checks evaluate directly instead
of interpreting natural language at verification time.

## ADDED Requirements

### Requirement: Deterministic, side-effect-free evaluation
The expression core SHALL evaluate every compiled statement deterministically,
with no side effects and with a guaranteed termination bound. The same
statement and inputs SHALL always produce the same result.

#### Scenario: Repeated evaluation is identical
- **WHEN** the same compiled statement is evaluated twice with the same inputs
- **THEN** both evaluations return the identical result

### Requirement: Exact decimal arithmetic
The expression core SHALL evaluate decimal values exactly, without the
precision loss of binary floating-point representation, for any statement
involving monetary or other exact-decimal quantities.

#### Scenario: Monetary comparison stays exact
- **WHEN** a statement compares a monetary value against a threshold expressed
  with two decimal places
- **THEN** the comparison result is not affected by binary floating-point
  rounding

### Requirement: CEL as the primary target, FEEL as fallback
The system SHALL compile statements to CEL by default. The system SHALL fall
back to FEEL for a statement only when CEL cannot represent it with exact
decimal semantics or within the provable subset.

#### Scenario: Fallback triggered by unsupported semantics
- **WHEN** a statement needs exact-decimal or provable-subset semantics that
  the CEL compilation cannot provide
- **THEN** the system compiles that statement to FEEL instead

### Requirement: Provable subset translatable to a solver
A defined subset of compiled statements SHALL be translatable into a form
usable by an SMT solver, to support consistency checking without executing
code.

#### Scenario: Consistency check without execution
- **WHEN** two statements in the provable subset are checked for mutual
  consistency
- **THEN** the system performs that check via solver translation rather than
  by running generated code

### Requirement: Local performance within hardware bounds
The expression core SHALL be able to perform at least thousands of
evaluations per second on a machine with 6 GB of RAM.

#### Scenario: High-volume evaluation stays responsive
- **WHEN** a rule requires thousands of sampled evaluations for its promotion
  criteria
- **THEN** the expression core completes them within the product's local
  performance budget on a 6 GB RAM machine
