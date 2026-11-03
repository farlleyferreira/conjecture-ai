# adversarial-slm Specification

## Purpose
Defines the local small language model that attacks and audits hypotheses
with no visibility into the code-generating model's reasoning, to provide an
adversarial check independent of the implementer.

## Requirements

### Requirement: Local execution only
The adversarial SLM SHALL run locally, on the same machine as the rest of
conjecture, with no network dependency for its attack or audit function.

#### Scenario: SLM runs without network access
- **WHEN** the adversarial SLM performs an attack or audit
- **THEN** it completes the operation without making a network call

### Requirement: Blind input
The adversarial SLM SHALL receive only the assertion under test, the
interface, and prior results. It SHALL NOT receive the code-generating
model's reasoning or conversation history.

#### Scenario: Generator reasoning withheld from the SLM
- **WHEN** the adversarial SLM attacks a hypothesis
- **THEN** its input does not include the code-generating model's reasoning
  trace or chat history

### Requirement: Vendor and family independence from the generator
The default adversarial SLM SHALL come from a different model family than
the code-generating model. When the default SLM's family coincides with the
generator's family, the system SHALL switch to the configured reserve SLM
from a different family.

#### Scenario: Family collision triggers reserve SLM
- **WHEN** the default adversarial SLM's family matches the code-generating
  model's family
- **THEN** the system uses the reserve SLM instead

### Requirement: Permissive licensing requirement
The system SHALL only use an adversarial SLM distributed under a permissive
license. A model variant available only under a research-only or
non-commercial license SHALL NOT be used as a default or reserve SLM.

#### Scenario: Research-only license excluded
- **WHEN** a candidate SLM variant is available only under a research-only
  license
- **THEN** the system does not select it as a default or reserve SLM

### Requirement: Grammar-constrained typed output
The adversarial SLM's output SHALL be constrained to a defined output
grammar, so results are machine-parseable without free-form text
interpretation.

#### Scenario: Malformed output rejected
- **WHEN** the adversarial SLM produces output that does not conform to its
  grammar
- **THEN** the system rejects that output rather than attempting to interpret
  it loosely

### Requirement: SLM opinion never counts as proof on its own
The adversarial SLM's findings SHALL feed the evidence profile as attack
results, but SHALL NOT by themselves invalidate a law or constitute proof.

#### Scenario: SLM finding recorded as evidence, not proof
- **WHEN** the adversarial SLM flags a potential violation
- **THEN** the system records it as an evidence item for review, not as an
  automatic invalidation of the hypothesis or law
