# user-defined-laws Specification

## Purpose
Defines laws that are declared directly by the user rather than promoted from
a hypothesis, including their precedence, status lifecycle, and invalidation
rules.

## Requirements

### Requirement: Direct declaration of a user-defined law
The system SHALL allow a law to be declared directly, nested under
`user_defined_laws` at the level where it applies, instead of only being
reached by promotion from a hypothesis.

#### Scenario: Law declared without prior hypothesis
- **WHEN** a user declares a statement directly under `user_defined_laws`
- **THEN** the system accepts it as a law candidate without requiring it to
  have first existed as a hypothesis

### Requirement: User-defined laws still go through judgment and conformance
A user-defined law SHALL NOT skip the judge or the conformance cycle. It
SHALL be judged as part of the specification and checked on every
conformance run like any other rule.

#### Scenario: User-defined law judged like any rule
- **WHEN** a specification containing a user-defined law is judged
- **THEN** the judge evaluates it with the same checks applied to promoted
  hypotheses

### Requirement: Precedence over promoted hypotheses
When a user-defined law and a promoted hypothesis would otherwise conflict at
the same scope, the system SHALL give precedence to the user-defined law.

#### Scenario: Conflict resolved in favor of the user-defined law
- **WHEN** a user-defined law and a hypothesis-derived rule disagree on the
  same case
- **THEN** the system's verdict follows the user-defined law

### Requirement: Explicit status lifecycle
A user-defined law SHALL carry one of the following statuses: unverified,
conformant, conformant-under-generalization, violated, contested, or
invalidated.

#### Scenario: Violation changes status to violated
- **WHEN** a confirmed counterexample is found against a user-defined law
- **THEN** its status changes to violated

### Requirement: Invalidation pathway
The system SHALL invalidate a user-defined law automatically when a formal
proof contradicts it. When two statements of equal authority conflict without
a formal proof, the system SHALL raise a question rather than deciding
unilaterally. Empirical counterevidence SHALL accumulate toward a configurable
threshold before the law is marked contested.

#### Scenario: Equal-authority conflict raises a question
- **WHEN** two statements with equal declared authority contradict each other
  with no formal proof available
- **THEN** the system raises a decisive question instead of invalidating
  either one automatically

#### Scenario: Empirical evidence accumulates toward contested status
- **WHEN** empirical counterevidence against a user-defined law crosses its
  configured threshold
- **THEN** the system marks the law as contested
