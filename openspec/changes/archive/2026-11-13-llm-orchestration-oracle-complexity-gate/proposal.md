# Proposal

## Why

The `conjecture-workflow` skill (block 4 of this round) keeps `conjecture
implement`/`work` out of its own recommended loop, since the agent driving
conjecture already does better: real production code, no separate
credential, full visibility. A follow-up discussion asked the honest
question directly - is there a principled reason `implement`/`work` cannot
be a private/local LLM the same way `attack`/`swarm` already are? Reading
`attack`'s own real implementation settled it: `attack` records whatever
the adversarial SLM claims with no mechanical check at all, while
`implement`'s generated code is always judged by real execution
(`conformancecycle.CheckHypothesisWithSelectedRigor`) - so `implement` is
already *more* grounded than `attack`, not less. There is no trust-based
reason to forbid a private LLM for `implement`/`work`.

What is a real, groundable reason: `implement`'s own round-by-round
revision (write code, run it, get the one counterexample that round found,
revise) is exactly the setup where a disposable, isolated oracle can pass
by special-casing the revealed counterexample rather than fixing the
general logic - and that risk grows with how much of the input domain the
oracle was ever actually exercised against. A hypothesis or law whose
domain is fully or boundedly enumerable (`conformancecycle.
CompleteProofInputs`/`BoundedProofInputs` already classify exactly this,
today only to pick a law's own rigor *level* for `check`) was tested
against its whole domain or a sound bound of it - a clean pass there is
honest. A hypothesis whose domain can never be fully bounded was only ever
tested against a directed sample; a pass there proves much less, and the
agent driving conjecture - with full project context the oracle never
has - is the one that should implement it instead.

## What Changes

- `runImplementLoop` (shared by "conjecture implement" and "conjecture
  work") now classifies the hypothesis/law's own input domain via
  `conformancecycle.CompleteProofInputs`/`BoundedProofInputs` before
  starting any round, reusing the same classifier `check.go`'s own rigor-
  level selection already uses for a different decision - and refuses
  (`ErrDomainTooComplexForOracle`) rather than running the oracle at all
  when neither finds a bounded domain.
- This is a hard refusal in the code, not documentation an agent could
  skip: even a human running `conjecture implement`/`work` directly from
  the CLI, without ever reading the skill, is stopped.
- Adds llm-orchestration's "The built-in implementer oracle refuses an
  unbounded domain" requirement.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `llm-orchestration`: adds one requirement, grounding this project's own
  "drafting and implementing are separate roles" purpose statement in a
  concrete rule for when the built-in implementing role may run at all.
  Edited directly in this change (`skip_specs: true`).

## Impact

- Affected paths: `cmd/conjecture/implement.go`, `cmd/conjecture/
  implement_test.go`, `cmd/conjecture/work_test.go`, `openspec/specs/
  llm-orchestration/spec.md`.
- A work-queue item that only ever resolves to an unbounded-domain
  hypothesis/law is dropped by "conjecture work" on first attempt (the
  same "cannot be processed, do not retry" path already used for a
  malformed or unresolvable item), rather than retried up to
  `workMaxAttemptsPerItem` times for a refusal that can never change - the
  refusal is deterministic, so retrying would only waste attempts.
- No new dependency.
