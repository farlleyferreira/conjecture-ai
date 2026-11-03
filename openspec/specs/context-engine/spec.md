# context-engine Specification

## Purpose
Defines the context graph that tracks weight, relevance, and confidence for
each node as three independent axes, and the gating rules used to compute
weight.

## Requirements

### Requirement: Three independent axes, never multiplied
The context engine SHALL track weight (importance), relevance, and
confidence as three independent values per node. The system SHALL NOT
combine them into a single multiplied score.

#### Scenario: Axes reported separately
- **WHEN** a node's context is inspected
- **THEN** its weight, relevance, and confidence are reported as separate
  values, not as one combined score

### Requirement: Weight-gating rules
The system SHALL assign a node's weight using fixed gating rules: a
user-defined law SHALL always be weighted critical; a rule involving money,
identity, permissions, or personal data SHALL always be weighted high or
above.

#### Scenario: User-defined law always critical
- **WHEN** a node is a user-defined law
- **THEN** its weight is set to critical regardless of other signals

#### Scenario: Money-related rule weighted at least high
- **WHEN** a rule's statement involves a monetary amount
- **THEN** its weight is set to high or above

### Requirement: Single-level relevance propagation
The system SHALL propagate a relevance signal exactly one level through the
graph, and SHALL NOT propagate it transitively beyond that.

#### Scenario: Relevance does not propagate two levels
- **WHEN** a relevance signal is applied to a node
- **THEN** only its immediate neighbors are affected, not their further
  neighbors

### Requirement: Weight is never auto-optimized
The system SHALL NOT automatically adjust a node's weight as an optimization
target. A weight change SHALL only occur through an explicit gating rule or
an approved calibration proposal.

#### Scenario: Optimizer cannot silently raise or lower weight
- **WHEN** the optimization suite runs
- **THEN** it does not directly change any node's weight outside an approved
  calibration proposal
