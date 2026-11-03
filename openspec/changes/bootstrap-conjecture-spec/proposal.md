# Proposal

## Why

conjecture does not have a single line of product code yet, but it already has a
detailed product proposal and full specification written up front. The product's
own first principle is that every claim needs evidence and every decision needs to
stay auditable - so the project should hold itself to that before writing any code.
This change turns the existing proposal/specification into a formal, versioned
OpenSpec baseline: a durable, reviewable record of what the system must do, split
into cohesive capabilities, so that every future implementation change has a spec
to conform to and a design record to extend rather than re-deriving the product's
intent from a long-form document each time.

## What Changes

- Add `cjtr-language`: the `.cjtr` declarative spec format - closed grammar, controlled
  vocabulary, strict YAML loading, composite IDs, semantic versioning of the format.
- Add `spec-authoring`: the three authoring modes, `???` pending markers, bilingual
  `# pt:` comments, and the `signed`/`confirmed` authorship flow for a `.cjtr` file.
- Add `expression-core`: the deterministic expression engine (CEL favored, FEEL
  fallback) that statements compile down to.
- Add `spec-judge`: judging a spec before any code exists - falsifiability,
  consistency (SMT), completeness, boundaries, side effects, concept mutation.
- Add `conformance-cycle`: the judge-LLM loop that checks implementation code
  against a frozen spec, with three "for all" rigor levels and a sandboxed run.
- Add `evidence-ledger`: the evidence profile per hypothesis and the
  Hypothesis -> Thesis -> Theory -> Law maturity scale with numeric promotion criteria.
- Add `user-defined-laws`: laws declared directly by the user, their precedence and
  their own status/invalidation lifecycle.
- Add `omission-detection`: the four omission types and the techniques used to surface
  each one.
- Add `llm-orchestration`: the separated drafting/implementing roles and the trust
  boundary between the LLM and the judge.
- Add `adversarial-slm`: the local, blind-input small language model that attacks and
  audits hypotheses.
- Add `attacker-swarm`: the optional multi-agent adversarial mode and its budget
  allocation.
- Add `bootstrap-material`: ingesting a reference architecture document and an
  initial observation harness before any first-party harness exists.
- Add `knowledge-bases`: attaching external norm/definition/reference/history bases
  via the CLI, with required hash and review-date metadata for norms.
- Add `context-engine`: the three-axis (weight, relevance, confidence) context graph
  and the gating rules that compute weight.
- Add `context-composition`: versioned, composable contexts and their four
  composition modes.
- Add `work-queue`: the single three-tier work queue that drives what the system
  works on next.
- Add `persistence`: the split between git-tracked `.cjtr`/materialization/settings
  files and the local, per-repo, append-only SQLite store outside the repo.
- Add `team-config`: the versioned, signed `settings.cjtr` team-wide floor
  configuration.
- Add `language-agnostic-contracts`: observing target systems only through declared
  contracts (commands, interfaces, standard formats), never language-specific parsing.
- Add `hardware-scheduler`: minimum hardware requirements and the scheduling rules
  that keep the SLM, the system under test, and idle-only heavy work from colliding.
- Add `optimization-suite`: the three optimization components (input generator,
  swarm bandit, multi-objective calibrator) and the human-approval gate on all of them.
- Add `cli-mcp-integration`: the MCP server, CLI, and folder-watch channels, and what
  an LLM is and is not allowed to do through them.

## Capabilities

### New Capabilities

- `cjtr-language`: the `.cjtr` declarative specification format - grammar, controlled vocabulary, YAML loading rules, composite IDs, format versioning.
- `spec-authoring`: creating and signing a `.cjtr` file through the three authoring modes.
- `expression-core`: the deterministic expression engine statements compile to (CEL/FEEL).
- `spec-judge`: judging a specification for correctness before any code exists.
- `conformance-cycle`: checking that implementation code conforms to a frozen spec.
- `evidence-ledger`: per-hypothesis evidence profiles and the maturity promotion scale.
- `user-defined-laws`: laws declared directly rather than promoted from a hypothesis.
- `omission-detection`: surfacing the four classes of missing specification.
- `llm-orchestration`: separating drafting and implementing LLM roles across the trust boundary.
- `adversarial-slm`: the local small-model adversary that attacks and audits hypotheses.
- `attacker-swarm`: optional multi-agent adversarial attack mode.
- `bootstrap-material`: ingesting reference docs and an initial observation harness pre-first-harness.
- `knowledge-bases`: attaching and using external norm/definition/reference/history bases.
- `context-engine`: the weight/relevance/confidence context graph and its gating rules.
- `context-composition`: versioned, composable contexts.
- `work-queue`: the single prioritized queue driving what gets verified next.
- `persistence`: where and how conjecture state is stored, in-repo vs. local-only.
- `team-config`: the shared, versioned, signed team configuration floor.
- `language-agnostic-contracts`: observing target systems through contracts, not language parsing.
- `hardware-scheduler`: hardware requirements and local resource scheduling.
- `optimization-suite`: human-gated optimization of inputs, swarm budget, and calibration.
- `cli-mcp-integration`: the MCP/CLI/folder-watch integration channels and their permission boundaries.

### Modified Capabilities

(none - there is no prior spec baseline)

## Impact

- No product code exists yet; this change creates the `openspec/specs/` baseline
  that every future implementation change will add deltas against.
- Affected paths: `openspec/changes/bootstrap-conjecture-spec/specs/<capability>/spec.md`
  for all 22 capabilities above, plus this proposal and its design/tasks.
- Establishes capability boundaries that future changes must reuse (exact paths)
  rather than re-deriving or renaming ad hoc.
