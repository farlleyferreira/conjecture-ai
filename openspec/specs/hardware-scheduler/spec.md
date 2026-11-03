# hardware-scheduler Specification

## Purpose
Defines minimum hardware requirements and the local scheduling rules that
keep the adversarial SLM, the system under test, and heavy background work
from competing for the same machine at the wrong time.

## Requirements

### Requirement: Minimum hardware requirement
The system SHALL require at least 6 GB of RAM on the machine running
conjecture and the adversarial SLM.

#### Scenario: Startup checks available RAM
- **WHEN** conjecture starts on a machine with less than 6 GB of RAM
- **THEN** the system reports that the minimum hardware requirement is not
  met

### Requirement: Always-local execution for the SLM
The adversarial SLM SHALL always run locally, on the developer's own
machine; the system SHALL NOT offload it to a remote inference service.

#### Scenario: No remote inference fallback
- **WHEN** local SLM inference would be slow on a given machine
- **THEN** the system does not fall back to a remote inference service for
  the SLM

### Requirement: Mutual exclusion between SLM and system-under-test load
The system SHALL avoid scheduling heavy adversarial SLM work at the same time
as heavy load from the system under test on the same machine.

#### Scenario: Heavy SLM work deferred during active system load
- **WHEN** the system under test is under heavy load
- **THEN** the system defers heavy adversarial SLM work rather than running
  both simultaneously

### Requirement: Heavy work only on an idle machine
The system SHALL only run heavy background verification work when the
machine is idle, and SHALL NOT make the user's interactive work feel slower
as a result of background verification.

#### Scenario: Heavy work paused on active interactive use
- **WHEN** the user is actively using the machine
- **THEN** the system does not run heavy background verification work that
  would compete for resources

### Requirement: Timeout never counts as a violation
When a check exceeds its allotted time on constrained hardware, the system
SHALL report the result as inconclusive, never as a rule violation.

#### Scenario: Slow machine produces inconclusive, not failing, result
- **WHEN** a check times out due to hardware constraints
- **THEN** the system reports that result as inconclusive
