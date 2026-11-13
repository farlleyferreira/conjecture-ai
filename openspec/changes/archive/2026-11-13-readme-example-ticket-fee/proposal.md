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
