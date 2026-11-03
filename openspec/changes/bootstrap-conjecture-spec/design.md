# Design

## Context

conjecture is a local-only tool that checks, on the developer's own machine, whether
AI-written code does what a `.cjtr` specification says, without itself writing code
or promising correctness. The full product proposal and specification already exist
(see proposal.md for the "why"); this design records the technical decisions those
documents already made, so they are not re-litigated capability by capability.

Seven product principles (P1-P7) and fourteen design guidelines underlie every
capability below; they are not restated as specs because they are not themselves
testable system behavior, but every capability's requirements should be read as
instances of them:

- P1 One assertion, many projections - cases/mutations/docs are derived from the
  spec, never hand-maintained; conjecture never writes code.
- P2 Evidence with strength and independence - confidence is a measured profile,
  not a formula.
- P3 Validate intent through decisive questions - ambiguity becomes a question
  card, not a silent assumption.
- P4 Rigor scales with uncertainty x impact - cheap by default, expensive only
  where impact is high and evidence is weak.
- P5 Process derived from state - no fixed phases/personas; a single prioritized
  queue instead.
- P6 Evidence carries validity - it expires, is superseded, and becomes visible
  debt (hash chain over spec/code/harness/SLM/operators).
- P7 Observed intent != intended intent - humans promote or flag a mismatch.

Guidelines G1-G14 (format-first, composite IDs + base version, every assertion
tied to executable evidence, deterministic engine behind a JSON contract, two
execution modes by weight, graph context on three axes in one SQLite, blind-input
evidence packets for the SLM, visible token/question budgets, secure-by-default,
light and local on 6GB RAM, short prompts with evals, risk-proportional rigor,
open-standard integration, LLM outside the trust boundary) are the concrete design
constraints the Decisions below satisfy.

## Goals / Non-Goals

**Goals:**
- Establish one `openspec/specs/<capability>/spec.md` per capability so later
  implementation work has a concrete, testable contract instead of the long-form
  proposal document.
- Keep every spec capability-scoped and free of implementation/language choices,
  so a future stack decision (still open) cannot invalidate any spec.
- Carry forward every explicit constraint and non-goal from the specification
  as a requirement or an explicit exclusion, not silently drop any.

**Non-Goals:**
- Choosing the implementation language/runtime for conjecture itself (tracked
  separately; these specs are stack-agnostic by construction).
- Re-deriving or second-guessing product decisions already made in the
  specification - this change formalizes them, it does not redesign them.
- Covering the experiment's success metrics as product requirements - they
  validate the product's thesis, they are not runtime behavior of conjecture
  itself, and stay in the proposal/product documentation instead of a spec.

## Decisions

1. **`.cjtr` format: nested YAML, controlled English vocabulary, no programming
   constructs** - not a bespoke scripting language or classic BDD step
   definitions. Rationale: avoids a "programming language in disguise"; anything
   that cannot fit a closed-vocabulary sentence becomes a derived concept or a
   decision table instead of custom syntax.

2. **Expression engine: CEL favored, FEEL as fallback** - not Rego or a custom
   evaluator. Rationale: nobody reads the compiled core directly (English + the
   deterministic Portuguese translation are the human-readable surfaces), so
   FEEL's extra business-readability does not pay for itself; CEL has stronger
   embeddable implementations, faster evaluation, and an official subset that
   translates to SMT. FEEL is adopted only if CEL fails on exact-decimal
   arithmetic or on the provable subset.

3. **Persistence split: `.cjtr`/`materialization.cjtr`/`settings.cjtr` in git;
   everything else in one local, per-repo SQLite database outside the repo** -
   not a dedicated graph database or multiple stores. Rationale: explicit in the
   spec ("no second database and no dedicated graph database"); if the context
   graph outgrows memory, traversals become recursive SQLite queries instead of
   introducing a new store.

4. **Execution is always local, never CI/cloud** - not integrated into the
   project's own CI. Rationale: the product's explicit execution premise is that
   no code leaves the developer's machine; the project's own CI keeps running
   independently of conjecture.

5. **Default adversarial SLM: a small, permissively-licensed, different-vendor
   model run locally via a CPU-friendly quantized runtime** - not a larger
   same-vendor or research-only-licensed model. Rationale: a permissive license is
   a hard requirement; a research-only-licensed larger variant was considered and
   rejected on licensing grounds alone. A second, different-family model is kept
   in reserve for when the default's family would coincide with the code-writing
   model, to preserve adversarial independence.

6. **Mutation targets the concept, not the generated code** - not classical code
   mutation testing. Rationale: equivalence between two statements of the concept
   is decidable by the consistency solver in linear logic/arithmetic *before* code
   exists, so provably-equivalent mutants are discarded by proof, not by
   triage after the fact.

7. **Harness protocol has three explicit states (complete / incomplete / wrong)
   plus mandatory known-answer probes** - not trusting an integrator-supplied
   harness at face value. Rationale: the harness is "the judge's lens" - a wrong
   harness is caught by probes, out-of-schema output, or non-reproducible output
   for the same input/seed, rather than silently producing wrong verdicts.

8. **Semantic versioning applies to both the `.cjtr` format itself (`conjecture:
   MAJOR.MINOR.PATCH` key) and to the product specification** - not an
   unversioned or ad hoc format. Rationale: MAJOR changes which guarantees/
   verdicts/decisions a file depends on, MINOR adds without breaking, PATCH is a
   wording fix; the version is folded into the spec file's content hash.

9. **Composite, content-derived IDs (`parent-id::child-id`)** - not freestanding
   UUIDs. Rationale: lets a rename/move be recognized by content, supports
   prefix-based autocomplete used as a verification scope
   (`conjecture verify shipping::free-shipping::free-above`), and stays
   deterministic from the names actually written.

10. **Spec signing is attributed to the logged-in user via a content hash that
    excludes the `signed` block itself, with no dedicated signing key** -
    not a per-user cryptographic key. Rationale: "there is no key to safeguard";
    `confirmed` is a declaration, not a proof - the design accepts that an
    autonomous session could in principle run the confirming command itself,
    and treats that as a process risk rather than one this format can
    cryptographically close.

11. **Drafting and implementing are always separate LLM sessions** - never the
    same agent that both writes the spec and writes the code. Rationale: whoever
    will write the code has an incentive to shape a spec that is easy to satisfy;
    the drafting session never learns how the code will actually be written.

12. **Target systems are observed only through declared contracts (commands,
    external interfaces, standard formats, read-only Tree-sitter for dependency
    rules)** - never through per-language parsers or ASTs. Rationale: "the engine
    never understands a language's semantics; it works by contract" - this is
    what makes conjecture language-agnostic.

13. **Strict YAML loading: no anchors, no references, no tags, duplicate keys
    rejected, every value read as text and typed by its concept** - not standard
    permissive YAML parsing. Rationale: avoids YAML's classic traps (implicit
    typing turning `no` into `false`, `100.00` into a lossy float, anchors/tags
    able to execute code).

14. **Controlled vocabulary has one single source that generates the parser,
    schema, autocomplete, translation, tests, and docs** - not independently
    maintained artifacts. Rationale: "nothing is written twice, and nothing can
    end up inconsistent."

15. **Attacker swarm is optional and agents never communicate with each other** -
    not a debate/consensus protocol between small models. Rationale: avoids
    "adversarial code contagion and debate between small models"; cross-family
    auditing only votes, and agreement is a signal, never evidence by itself.

16. **The multi-objective calibrator only proposes configuration changes; a human
    must approve before anything is applied** - never automatic tuning.
    Rationale: avoids Goodharting ("efficiently optimizing the wrong thing");
    fixed floors (critical weight, negative controls, etc.) are never crossed by
    the optimizer regardless of its proposal.

17. **No telemetry of any kind** - an explicit design guideline, not an oversight.
    Rationale: "nothing leaves the machine" is a hard product constraint, not
    just a default.

## Risks / Trade-offs

- **Stack-agnostic specs vs. eventual implementation friction**: writing every
  requirement without naming a language/runtime keeps these specs valid
  regardless of the still-open stack decision, but means some design-level
  concerns (e.g., how CEL gets embedded, how the local SQLite store is accessed)
  will need their own design.md once implementation starts - deferred
  deliberately, not overlooked.
- **22 capabilities at once, before any code**: a large simultaneous baseline
  risks drifting from what gets built first. Mitigated by keeping each capability
  independently archivable later and because `openspec/changes/archive/` retains
  this change's reasoning even if a capability's spec is revised early.
- **Capability boundaries chosen now may not match the eventual module
  boundaries** of the implementation. Accepted: OpenSpec capabilities describe
  observable behavior, not code structure, so a different internal module layout
  does not by itself require renaming or re-splitting capabilities.
- **No cryptographic signing key (Decision 10)** means `confirmed: true` is
  honesty-based, not tamper-proof. Accepted explicitly in the source
  specification as a known, bounded limitation rather than solved here.
