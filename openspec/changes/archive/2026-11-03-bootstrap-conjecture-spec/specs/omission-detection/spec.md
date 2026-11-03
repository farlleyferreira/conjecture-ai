# Spec Delta

## Purpose

Defines the four classes of specification omission conjecture looks for and
the techniques used to surface each one, so gaps are found before they become
missing behavior in shipped code.

## ADDED Requirements

### Requirement: Type A - asserted but not implemented
The system SHALL detect when a `.cjtr` statement asserts behavior that has no
corresponding implementation evidence, using observation and generated cases
against the running system.

#### Scenario: Asserted behavior with no evidence flagged
- **WHEN** a statement asserts a behavior but no case or observation exercises
  it
- **THEN** the system flags it as a Type A omission

### Requirement: Type B - case forgotten within the domain
The system SHALL detect a domain case that the specification does not
address, using property-based testing combined with targeted attacks.

#### Scenario: Unaddressed domain case surfaced by property testing
- **WHEN** property-based test generation finds an input within the domain
  that no rule addresses
- **THEN** the system flags it as a Type B omission

### Requirement: Type C - case not specified
The system SHALL detect a case missing from a decision table or a
state/event matrix for a phenomenon.

#### Scenario: Missing matrix cell surfaced
- **WHEN** a state/event matrix for a phenomenon has an empty cell with no
  corresponding rule
- **THEN** the system flags it as a Type C omission

### Requirement: Type D - requirement nobody thought of
The system SHALL maintain a configurable catalog of requirement categories
(for example, concurrency, localization, accessibility) and SHALL check
whether each category is addressed by at least one rule in the relevant
area, while explicitly acknowledging that a category outside this catalog
cannot be detected.

#### Scenario: Uncovered catalog category flagged
- **WHEN** a configured requirement category has no corresponding rule in an
  area it applies to
- **THEN** the system flags that category as a Type D omission for that area

#### Scenario: Category outside the catalog stays invisible
- **WHEN** a requirement category is not present in the configured catalog
- **THEN** the system does not claim to have checked for it, and does not
  report its absence as "no omissions found"
