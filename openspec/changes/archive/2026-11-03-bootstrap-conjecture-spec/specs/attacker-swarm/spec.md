# Spec Delta

## Purpose

Defines the optional multi-agent adversarial attack mode that scales attack
diversity beyond a single SLM without letting attacking agents collude or
debate.

## ADDED Requirements

### Requirement: Opt-in, three operating modes
The attacker swarm SHALL be disabled by default. When enabled, it SHALL
support exactly three modes: off, attackers only, and attackers plus
cross-family auditors.

#### Scenario: Swarm disabled by default
- **WHEN** no swarm mode is explicitly configured
- **THEN** the system runs with the swarm off

#### Scenario: Auditor mode requires a different model family
- **WHEN** the attackers-plus-auditors mode is selected
- **THEN** the auditors used are from a different model family than the
  attackers

### Requirement: No communication between attacking agents
Attacking agents in the swarm SHALL NOT communicate with each other during an
attack run.

#### Scenario: Agents attack independently
- **WHEN** multiple attacking agents run against the same hypothesis
- **THEN** none of them receives another attacking agent's output during the
  run

### Requirement: Deterministic result aggregation
The system SHALL aggregate swarm results deterministically from the
individual agents' findings, and cross-family agreement SHALL be treated as a
signal, never as evidence by itself.

#### Scenario: Agreement treated as signal, not evidence
- **WHEN** attackers from different families reach the same finding
- **THEN** the system records the agreement as a signal and still requires
  the underlying evidence to stand on its own

### Requirement: Bandit-based strategy budget allocation
The swarm SHALL support at least five distinct attack strategies and SHALL
use a multi-armed bandit algorithm to allocate a configured attack budget
across strategies.

#### Scenario: Budget shifts toward a more productive strategy
- **WHEN** one attack strategy finds violations more often than others within
  a run
- **THEN** the bandit allocator shifts a larger share of the remaining
  budget toward that strategy
