# Proposal

## Why

Explicit direction: the README's worked example (a free-shipping-over-
$100 / subscription-order rule) could read as resembling a food-delivery
company's own business rules, and should be replaced with something
unrelated and richer. Chose an airline ticket-change-fee policy instead:
a categorical gate (fare class) combined with a numeric threshold (hours
before departure), demonstrated across three examples instead of the
prior two-condition/no-category-contrast version.

## What Changes

- Both `README.md` (pt-BR) and `README.en.md`'s worked example changed
  from the shipping/subscription rule to: *"Economy Flex tickets changed
  at least 24 hours before departure are free; anything else pays a
  change fee."*
- The real `.cjtr` file, the buggy Python adapter, and the fixed adapter
  were all rebuilt and run for real (non-root, WSL, podman via the
  existing `docker` shim) against the compiled `conjecture` binary -
  `conjecture verify` (3/3 examples consistent), `conjecture check`
  against the buggy adapter (non-conformant, with the real counterexample
  now in the README), `conjecture check` against the fixed adapter
  (conformant), and `conjecture sign --confirmed` - the README's
  transcript is the real output of these commands, not fabricated text.

## Follow-up: too many "not"s killed the pitch

Feedback on the above: the README still read as discouraging - a "what
this is not for" section up front with six negative bullets, right before
"Status," left nothing confident for a reader to land on. Re-grounded the
value proposition directly in this project's own real `openspec/specs/*/
spec.md` Purpose statements (the authoritative product doc already in
this repo) instead of inventing marketing copy:

- Replaced the old 4-bullet "what conjecture also does" list with a
  7-bullet "Why conjecture" section placed right after the worked
  example, each bullet paraphrasing one real spec's own Purpose line
  (conformance-cycle's frozen-spec rule, llm-orchestration's trust
  boundary, adversarial-slm/attacker-swarm's independent attacker,
  evidence-ledger's maturity ladder + omission-detection's four gap
  classes, context-composition's versioning, language-agnostic-
  contracts, optimization-suite's human-gated self-tuning).
- Moved "what this is not for" and "Status" into one collapsed
  `<details>` block ("Scope and limits") near the end, right before
  License - every caveat kept, word for word, just no longer the first
  thing a reader hits after the demo.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - documentation only, no behavior change. `skip_specs: true`.

## Impact

- Affected paths: `README.md`, `README.en.md`.
- No code or test changes; `go build`/`go vet`/`go test ./...` all pass
  unchanged (docs-only).
- Every other section (philosophy bullets, Quick Start, Commands table,
  "what this is not for", Status, License) is untouched.
