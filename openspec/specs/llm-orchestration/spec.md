# llm-orchestration Specification

## Purpose
Defines the separation between the drafting and implementing LLM roles and
the trust boundary that keeps any LLM from reaching into the judge or the
evidence store directly.

## Requirements

### Requirement: Drafting and implementing are always separate sessions
The system SHALL always open a new, separate session for drafting a `.cjtr`
specification, distinct from the session that later implements code against
it. The drafting session SHALL NOT be the same session, nor have visibility
into how the code will be written.

#### Scenario: Drafting session isolated from implementation
- **WHEN** a specification is drafted and later implemented
- **THEN** the drafting session and the implementing session are distinct,
  and the drafting session has no visibility into the implementation
  approach

### Requirement: Whoever writes code does not judge
Code produced by an implementing session SHALL be checked by a judge
component the implementing LLM cannot influence, using test cases the
implementing LLM never sees.

#### Scenario: Hidden cases withheld from the implementer
- **WHEN** an implementing session requests the cases used to judge its code
- **THEN** the system does not provide the hidden cases it is being judged
  against

### Requirement: LLM stays outside the trust boundary
No LLM involved in drafting or implementing SHALL be able to write evidence
records directly, call the adversarial SLM directly, sign a `.cjtr` file on
behalf of a different user, or change weights, rules, operators, or
configuration through the integration channel.

#### Scenario: Direct SLM call blocked
- **WHEN** an LLM attempts to invoke the adversarial SLM directly rather than
  through the system's own verification flow
- **THEN** the system blocks the call

#### Scenario: Config change by LLM blocked
- **WHEN** an LLM attempts to change a weight, rule, operator, or
  configuration value through the integration channel
- **THEN** the system rejects the change

### Requirement: Only proposals cross the boundary
An LLM on either side of the trust boundary SHALL only be able to submit
proposals (a draft, a diff, a code change) for the system to evaluate; it
SHALL NOT be able to directly mark a specification confirmed, a hypothesis
promoted, or a verdict conformant.

#### Scenario: LLM cannot self-confirm a specification
- **WHEN** an LLM finishes drafting a `.cjtr` file
- **THEN** the file's `confirmed` flag remains false until a human
  explicitly confirms it
