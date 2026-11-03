# Spec Delta

## Purpose

Defines the integration channels (MCP server, CLI, folder watch) through
which an agent or developer interacts with conjecture, and the explicit
boundary on what an LLM may and may not do through them.

## ADDED Requirements

### Requirement: MCP server as the primary channel
The system SHALL expose an MCP server as its primary integration channel for
an agent, alongside a CLI and automatic folder-watch observation.

#### Scenario: Agent interacts via MCP
- **WHEN** an agent is configured with conjecture's MCP server
- **THEN** it can request verification and read results through that
  channel

### Requirement: Explicit permission table for LLM actions
The system SHALL maintain an explicit, documented table of actions an LLM
may perform through any integration channel (for example, requesting
verification, reading evidence) versus actions it may not perform (for
example, writing evidence, confirming a specification, changing weights).

#### Scenario: Disallowed action rejected regardless of channel
- **WHEN** an LLM attempts an action marked disallowed in the permission
  table, through any channel
- **THEN** the system rejects the action

### Requirement: Rigor-reducing CLI actions stay unconfirmed
When an action taken through the CLI would reduce verification rigor (for
example, lowering a weight or skipping a check), the system SHALL mark the
affected specification as `confirmed: false` until a human reviews it.

#### Scenario: Rigor-reducing CLI action marks spec unconfirmed
- **WHEN** a CLI action reduces rigor for a specification
- **THEN** that specification's `confirmed` flag is set to false until
  reviewed

### Requirement: Folder-watch observation is read-only
Automatic folder-watch observation SHALL only read files to detect relevant
changes; it SHALL NOT write to any file in the watched folder as part of
observation.

#### Scenario: Folder watch does not modify watched files
- **WHEN** the folder-watch mechanism detects a change
- **THEN** it reads the change but does not modify any file in the watched
  folder

### Requirement: Verdicts delivered back through the same channel
The system SHALL return a verification verdict through the same channel
(MCP, CLI, or folder-watch-triggered notification) that the corresponding
request came through.

#### Scenario: CLI request receives CLI-delivered verdict
- **WHEN** a verification request is made through the CLI
- **THEN** the resulting verdict is delivered back through the CLI
