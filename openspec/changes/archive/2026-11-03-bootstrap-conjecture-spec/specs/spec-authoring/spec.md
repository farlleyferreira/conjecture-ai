# Spec Delta

## Purpose

Defines how a `.cjtr` file gets created and signed off, across three equally
valid authoring modes, so every mode produces a file that goes through the
same loader, judge, and signature flow.

## ADDED Requirements

### Requirement: Three equivalent authoring modes
The system SHALL support creating a `.cjtr` file by example/pattern-driven
drafting, by hand, or without a model at all. All three modes SHALL produce a
file that passes through the same loader, judge, and signature flow with no
mode-specific exception.

#### Scenario: Hand-authored file treated identically
- **WHEN** a user writes a `.cjtr` file by hand without using the drafting
  assistant
- **THEN** the system loads, judges, and signs it through the same pipeline
  used for a drafted file

### Requirement: Pending markers block signature
The system SHALL represent an open question inside a `.cjtr` file as a `???`
marker. A file containing any unresolved `???` marker SHALL NOT be eligible
for signature.

#### Scenario: Unresolved marker blocks signing
- **WHEN** a user attempts to sign a `.cjtr` file that still contains a `???`
  marker
- **THEN** the system refuses to sign the file and reports which marker is
  unresolved

### Requirement: Bilingual comments carry deterministic translation
The system SHALL support a `# pt:` comment attached to a statement, holding a
deterministic Portuguese translation of that statement, generated from the
same controlled vocabulary source used for the grammar.

#### Scenario: Translation regenerated from vocabulary
- **WHEN** a statement using controlled-vocabulary terms is written in English
- **THEN** the system can generate its `# pt:` translation deterministically
  from the vocabulary source, without invoking a model

### Requirement: Signature records authorship without a dedicated key
The system SHALL let the signed-in user sign a `.cjtr` file. The signature
SHALL be a `signed` block containing a content hash of the file excluding the
`signed` block itself, and SHALL NOT require or depend on a per-user
cryptographic key.

#### Scenario: Signature invalidated by content change
- **WHEN** any part of a `.cjtr` file outside the `signed` block changes after
  signing
- **THEN** the recorded signature no longer matches the file's content hash

### Requirement: Confirmed flag gates downstream use
A `.cjtr` file SHALL carry a `confirmed` flag. The system SHALL treat an
unconfirmed file's verdicts as carrying a visible caveat that the
specification lacked human review and may have been shaped by the drafting
model.

#### Scenario: Verdict flags unreviewed specification
- **WHEN** verification runs against a `.cjtr` file whose `confirmed` flag is
  not `true`
- **THEN** the verdict output states that the specification has not had human
  review and may have been shaped by the model
