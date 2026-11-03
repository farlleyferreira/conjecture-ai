# conformance-cycle Specification

## Purpose
Defines the loop that checks whether implementation code conforms to a frozen
`.cjtr` specification, from first run through counterexample feedback to a
final verdict.

## Requirements

### Requirement: Frozen specification as the conformance input
Once a `.cjtr` file is signed, the conformance cycle SHALL check implementation
code only against that frozen file. The implementing session SHALL receive the
frozen specification and SHALL NOT be able to alter it mid-cycle.

#### Scenario: Mid-cycle spec change rejected
- **WHEN** an attempt is made to change the frozen `.cjtr` file while a
  conformance cycle against it is in progress
- **THEN** the system rejects the change for the purposes of that cycle

### Requirement: Three levels of universal ("for all") rigor
The conformance cycle SHALL support three rigor levels for a universally
quantified rule: directed sampling, bounded proof, and complete proof. The
system SHALL select the level according to the rule's assigned weight and
available evidence.

#### Scenario: Directed sampling used for low-weight rule
- **WHEN** a low-weight rule is checked with no stronger evidence available
- **THEN** the conformance cycle uses directed sampling rather than bounded or
  complete proof

### Requirement: Minimal counterexample returned on violation
When implementation code violates a rule, the conformance cycle SHALL return
the minimal counterexample found, not an arbitrary failing case.

#### Scenario: Minimal counterexample surfaced to the implementing session
- **WHEN** the conformance cycle finds multiple failing inputs for the same
  violation
- **THEN** it returns the smallest/simplest one it found as the
  counterexample

### Requirement: Sandboxed execution with no network access
All implementation code exercised during the conformance cycle SHALL run in a
sandbox with no network access.

#### Scenario: Network call blocked during verification
- **WHEN** implementation code under test attempts a network call during a
  conformance run
- **THEN** the sandbox blocks the call

### Requirement: Round and token limits
The conformance cycle SHALL enforce a configured maximum number of
judge-to-implementer rounds and a configured maximum token budget per cycle.

#### Scenario: Cycle stops at round limit
- **WHEN** the configured maximum number of rounds is reached without the
  code reaching conformance
- **THEN** the cycle stops and reports the outcome as not conformant rather
  than continuing indefinitely

### Requirement: Three-way conformance verdict
The conformance cycle SHALL produce one of three verdicts for a rule:
conformant, non-conformant, or inconclusive. A timed-out check SHALL be
reported as inconclusive, never as a violation.

#### Scenario: Timeout reported as inconclusive
- **WHEN** a conformance check for a rule times out before reaching a
  definitive result
- **THEN** the system reports that rule as inconclusive, not as violated
