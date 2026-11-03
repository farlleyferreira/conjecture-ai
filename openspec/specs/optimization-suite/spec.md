# optimization-suite Specification

## Purpose
Defines the three optimization components that improve input generation,
swarm budget allocation, and configuration calibration, all gated behind
human approval so nothing tunes itself unsupervised.

## Requirements

### Requirement: Evolutionary input generator
The system SHALL generate test inputs using an evolutionary (genetic)
algorithm that targets meaningful behavior change deterministically,
without requiring the generated inputs to be hand-written.

#### Scenario: Generated input targets a behavior change
- **WHEN** the evolutionary input generator produces a new candidate input
- **THEN** that input is selected to target an observable change in
  behavior, not chosen arbitrarily

### Requirement: Bandit-based swarm budget allocation
The system SHALL use a multi-armed bandit algorithm to allocate attack
budget across the swarm's attack strategies, as described in
`attacker-swarm`.

#### Scenario: Budget allocation reused from swarm strategy data
- **WHEN** the optimization suite allocates attack budget
- **THEN** it uses the same bandit mechanism the attacker swarm capability
  defines

### Requirement: Multi-objective calibration proposes, never applies
The system SHALL use a multi-objective calibration algorithm (optimizing
across multiple objectives at once) to propose configuration changes. The
system SHALL NOT apply a calibration proposal automatically; it SHALL require
explicit human approval before any proposed change takes effect.

#### Scenario: Calibration proposal requires explicit approval
- **WHEN** the calibrator produces a proposed configuration change
- **THEN** that change has no effect until a human explicitly approves it

### Requirement: Fixed floors never crossed by calibration
The system SHALL enforce fixed configuration floors (such as critical-weight
thresholds and negative-control requirements) that no calibration proposal,
even an approved one, is allowed to cross.

#### Scenario: Approved proposal still respects fixed floors
- **WHEN** an approved calibration proposal would otherwise cross a fixed
  floor
- **THEN** the system clamps the applied change at the floor rather than
  crossing it

### Requirement: Calibration runs offline
The multi-objective calibrator SHALL run offline, separate from the
synchronous verification path, so it does not add latency to interactive
verification.

#### Scenario: Interactive verification unaffected by calibration
- **WHEN** the calibrator is running
- **THEN** interactive, synchronous verification requests are not delayed by
  it
