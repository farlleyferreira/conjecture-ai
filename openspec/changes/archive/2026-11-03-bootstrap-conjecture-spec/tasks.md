# Tasks

## 1. Specification language & authoring

- [ ] 1.1 Draft `specs/cjtr-language/spec.md` and verify it covers grammar, controlled vocabulary, YAML loading rules, composite IDs, and format versioning
- [ ] 1.2 Draft `specs/spec-authoring/spec.md` and verify it covers the three authoring modes, `???` pending markers, and the signed/confirmed flow
- [ ] 1.3 Draft `specs/expression-core/spec.md` and verify it covers the deterministic engine's required properties (no side effects, termination, exact decimal)

## 2. Judging & conformance

- [ ] 2.1 Draft `specs/spec-judge/spec.md` and verify it covers falsifiability, consistency, completeness, boundary, and concept-mutation checks
- [ ] 2.2 Draft `specs/conformance-cycle/spec.md` and verify it covers the three "for all" rigor levels and the sandboxed run
- [ ] 2.3 Draft `specs/evidence-ledger/spec.md` and verify it covers the evidence profile fields and the Hypothesis/Thesis/Theory/Law promotion criteria
- [ ] 2.4 Draft `specs/user-defined-laws/spec.md` and verify it covers declaration, precedence, status, and invalidation
- [ ] 2.5 Draft `specs/omission-detection/spec.md` and verify it covers all four omission types and their detection techniques

## 3. LLM orchestration & adversarial verification

- [ ] 3.1 Draft `specs/llm-orchestration/spec.md` and verify it covers the drafting/implementing role separation and the trust boundary
- [ ] 3.2 Draft `specs/adversarial-slm/spec.md` and verify it covers blind input, vendor/family independence, and typed output
- [ ] 3.3 Draft `specs/attacker-swarm/spec.md` and verify it covers the three modes, no inter-agent communication, and budget allocation
- [ ] 3.4 Draft `specs/bootstrap-material/spec.md` and verify it covers reference-document ingestion and the initial harness's known-answer probes

## 4. Context, knowledge & work management

- [ ] 4.1 Draft `specs/knowledge-bases/spec.md` and verify it covers attach/scope/authority and required norm metadata
- [ ] 4.2 Draft `specs/context-engine/spec.md` and verify it covers the three independent axes and the weight-gating rules
- [ ] 4.3 Draft `specs/context-composition/spec.md` and verify it covers versioning and the four composition modes
- [ ] 4.4 Draft `specs/work-queue/spec.md` and verify it covers the three-tier ordering and aging

## 5. Persistence, config & integration

- [ ] 5.1 Draft `specs/persistence/spec.md` and verify it covers the git-tracked vs. local-only split and per-OS data directories
- [ ] 5.2 Draft `specs/team-config/spec.md` and verify it covers the shared, signed, floor-not-ceiling configuration
- [ ] 5.3 Draft `specs/language-agnostic-contracts/spec.md` and verify it covers contract-only observation and standard formats
- [ ] 5.4 Draft `specs/hardware-scheduler/spec.md` and verify it covers minimum RAM and mutual-exclusion scheduling
- [ ] 5.5 Draft `specs/optimization-suite/spec.md` and verify it covers the three components and the human-approval gate
- [ ] 5.6 Draft `specs/cli-mcp-integration/spec.md` and verify it covers the integration channels and LLM permission boundaries

## 6. Validation

- [ ] 6.1 Run `openspec validate bootstrap-conjecture-spec --strict` and verify it reports no errors or warnings

## Workflow follow-up

- Archive this change once merged, so it seeds `openspec/specs/`.
