# Spec Delta

## Purpose

Defines the single, three-tier work queue that drives what conjecture works
on next, replacing fixed phases or personas with a queue derived from current
state.

## ADDED Requirements

### Requirement: Single queue with three tiers
The system SHALL maintain exactly one work queue with three tiers: blockers,
current task, and background work. The system SHALL NOT maintain separate
per-phase or per-persona queues.

#### Scenario: Blocker preempts background work
- **WHEN** a blocker-tier item exists
- **THEN** the system works on it before any background-tier item

### Requirement: Ordering by weight, then severity, then age
Within a tier, the system SHALL order items by weight first, then severity,
then age (oldest first).

#### Scenario: Higher weight item ordered first
- **WHEN** two items in the same tier have different weights
- **THEN** the system orders the higher-weight item first regardless of
  their relative age

### Requirement: Aging prevents starvation
The system SHALL increase an item's effective priority the longer it remains
unaddressed, so a low-weight item is not starved indefinitely.

#### Scenario: Long-waiting item eventually surfaces
- **WHEN** a low-weight item has remained in the queue far longer than
  typical items of its weight
- **THEN** its effective priority increases so it eventually surfaces for
  work

### Requirement: Loop runs to completeness, not to a fixed step count
The system SHALL keep checking the queue in a loop until no actionable item
remains, rather than stopping after a fixed number of steps.

#### Scenario: Loop continues while actionable items remain
- **WHEN** the queue still has at least one actionable item after
  processing one
- **THEN** the system continues processing rather than stopping early
