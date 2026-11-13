# Proposal

## Why

`findHypothesisByID` (attack.go) resolved a hypothesis or law by id but
never reported which it was - `attack`/`swarm` had no way to tell a
single-opinion attack on a law (always Critical weight, per
`contextengine.GatedWeight`'s own "isUserDefinedLaw's rule wins
outright") apart from the same attack on a plain hypothesis, even though
a law's own stakes warrant the diversified coverage `swarm` gives and
`attack` alone does not.

Decided (explicit instruction): warn, do not refuse. Unlike `implement`/
`work`'s own domain-boundedness gate - where running on an unbounded
domain produces genuinely untrustworthy evidence either way - a single
local-model opinion on a law is not wrong, only weaker than its stakes
would ideally get. Refusing would block a legitimate, if weaker, signal;
warning preserves the caller's own choice while surfacing the real
information gap, the same posture check.go's own "WARNING: prior
evidence is stale" already takes.

## What Changes

- `findHypothesisByID` now also returns whether the hypothesis/law it
  found is a law.
- `attack` prints a warning to stdout, and records a corresponding note
  in the evidence's own `Unchecked` field, when attacking a law -
  recommending `conjecture swarm attack` instead. `swarm` itself needs
  no such warning (diversified coverage is already what it gives).
- `promote`'s own call site is updated for the new return signature with
  no behavior change (it never needed `isLaw` from this path - it reads
  the hypothesis's own `IsUniversal` fact separately).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `llm-orchestration`: no requirement text changes - this is a
  disclosure improvement inside an already-permitted action
  ("requesting verification" via the adversarial SLM), not a new
  boundary. `skip_specs: true`.

## Impact

- Affected paths: `cmd/conjecture/attack.go`, `cmd/conjecture/
  attack_test.go`, `cmd/conjecture/promote.go`, `cmd/conjecture/swarm.go`.
- Verified for real: ran `conjecture attack` against a real law fixture
  with an unreachable Ollama endpoint - the warning printed before the
  (expected) connection-refused failure, confirming it is not gated
  behind a successful attack call.
- No new dependency.
