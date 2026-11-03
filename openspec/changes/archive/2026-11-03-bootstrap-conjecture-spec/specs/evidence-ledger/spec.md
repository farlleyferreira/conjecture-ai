# Spec Delta

## Purpose

Defines the per-hypothesis evidence profile and the maturity scale a
hypothesis climbs as it survives increasingly strong verification, replacing a
binary pass/fail verdict with honest, itemized evidence.

## ADDED Requirements

### Requirement: Evidence profile per hypothesis
The system SHALL maintain, for every hypothesis, an evidence profile recording
what was attacked, what survived, what remains open, what nobody has checked
yet, who/what produced each result, and the validity hash of that result.

#### Scenario: Unchecked area reported honestly
- **WHEN** no rule in an area addresses a given concern (for example,
  concurrency)
- **THEN** the evidence profile states that nobody has checked that concern,
  rather than implying it is covered

### Requirement: Four-stage maturity scale
The system SHALL track each hypothesis on a maturity scale of Hypothesis,
Thesis, Theory, and Law (with Theorem available at any stage for a formally
proved statement), and SHALL only advance a hypothesis along this scale
according to its defined promotion criteria.

#### Scenario: Hypothesis promoted to Thesis
- **WHEN** a hypothesis is accepted by the judge and found conformant by the
  conformance cycle with repeatable results
- **THEN** the system promotes it to Thesis

#### Scenario: Thesis promoted to Theory
- **WHEN** every thesis in a phenomenon is confirmed, the mechanism is
  confirmed, a second round with a different-family model reproduces the
  result, and swarm/attack robustness checks find no violation
- **THEN** the system promotes the phenomenon's theses to Theory

#### Scenario: Theory promoted to Law
- **WHEN** a universally quantified (always) hypothesis remains conformant
  against hidden samples and across new code versions
- **THEN** the system promotes it to Law

### Requirement: Weight-dependent numeric promotion criteria
The number of failure-free cases required before promotion SHALL depend on
the hypothesis's assigned risk weight: 100 cases for low weight, 300 for
medium, 1000 for high, and 3000 for critical, corresponding to maximum
tolerated violation rates of 3%, 1%, 0.3%, and 0.1% respectively at 95%
confidence.

#### Scenario: Critical rule requires 3000 cases
- **WHEN** a hypothesis is assigned critical weight
- **THEN** the system requires 3000 failure-free cases before considering it
  for promotion

### Requirement: Single-counterexample demotion
A single confirmed counterexample SHALL demote a hypothesis from its current
stage, with the demotion recorded alongside the counterexample that caused it.

#### Scenario: Law demoted by a new counterexample
- **WHEN** a counterexample is confirmed against a hypothesis currently at
  Law stage
- **THEN** the system demotes it and records the counterexample as the cause

### Requirement: Validity hash chain
Each evidence record SHALL be chained by hash to the specification, code,
harness, SLM, and operator versions it was produced against, so that a change
to any of them is visible as evidence debt rather than silently stale
evidence.

#### Scenario: Evidence flagged stale after a dependency changes
- **WHEN** the harness or code version an evidence record depended on changes
- **THEN** the system flags that evidence record as stale via the hash chain
