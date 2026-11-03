# Spec Delta

## Purpose

Defines how a `.cjtr` specification is judged for soundness before any
implementation code exists, so defects in intent are caught before they are
inherited by generated code.

## ADDED Requirements

### Requirement: Falsifiability check
The judge SHALL reject any hypothesis that is not written in a way that could
be disproven by a concrete case.

#### Scenario: Unfalsifiable hypothesis rejected
- **WHEN** a hypothesis is worded so that no possible input could contradict
  it
- **THEN** the judge rejects the hypothesis and reports it as unfalsifiable

### Requirement: Consistency check via solver
The judge SHALL check statements within the provable subset for logical
contradictions using an SMT solver, and SHALL report any contradiction found
between two or more statements.

#### Scenario: Contradictory thresholds detected
- **WHEN** two statements in the provable subset impose contradictory
  conditions on the same input range
- **THEN** the judge reports the contradiction and the statements involved

### Requirement: Completeness check via decision structure
The judge SHALL check a phenomenon's rules against a decision table or
state/event matrix to surface combinations of conditions that no rule covers.

#### Scenario: Uncovered combination surfaced
- **WHEN** a phenomenon's decision table has an input combination with no
  matching rule
- **THEN** the judge reports that combination as uncovered, not as "no issue
  found"

### Requirement: Boundary and side-effect checks
The judge SHALL check each rule's stated boundaries and SHALL check for
unstated side effects implied by the rule's statements.

#### Scenario: Boundary condition flagged
- **WHEN** a rule's boundary value (for example, exactly at a threshold) is
  not explicitly addressed by any hypothesis
- **THEN** the judge flags the boundary as unaddressed

### Requirement: Concept mutation instead of code mutation
The judge SHALL generate mutations of the specified concept itself (by
consequence and by perturbation) and SHALL discard any mutation provably
equivalent to the original via the consistency solver, rather than mutating
generated code.

#### Scenario: Equivalent mutant discarded by proof
- **WHEN** a generated concept mutation is provably equivalent to the
  original statement
- **THEN** the judge discards it without running it against any code

#### Scenario: Distinguishing mutation kept
- **WHEN** a generated concept mutation is not provably equivalent to the
  original statement
- **THEN** the judge keeps it as a case a correct implementation must
  distinguish

### Requirement: Judge verdict on the specification
The judge SHALL produce one of three verdicts on a `.cjtr` specification:
accepted, pending, or rejected, and SHALL report the specific findings that
led to a pending or rejected verdict.

#### Scenario: Pending verdict reports open findings
- **WHEN** the judge finds at least one unresolved issue but no outright
  rejection
- **THEN** it returns a pending verdict together with the list of unresolved
  issues
