# conjecture-ai

## What is this?

Say you have a business rule: *"orders over $100 ship free, unless it's a
subscription order."* Normally, the only way to know whether your code
actually does that is to read the code, or write a unit test by hand and
hope you covered the right cases.

`conjecture` flips that around: you write the rule itself, once, in a small
structured file (no code) - and `conjecture` generates test cases from it,
runs your *real* implementation inside a sandboxed container, and tells you
**conformant** or **non-conformant**, with the actual input that broke it if
it didn't hold. It never trusts the code's own claims about itself - it
always runs it for real and checks the real output.

On top of that one core loop, it can also: ask an LLM to draft the rule or
write/fix the implementation for you, run adversarial LLM "attacks" that try
to find a case where the rule breaks, and promote a rule through a maturity
ladder (Hypothesis → Thesis → Theory → Law) as more evidence accumulates -
with every human decision along the way recorded in a local, tamper-evident
ledger, not just printed and forgotten.

## Build

This repo has exactly one CGO dependency (`github.com/smacker/go-tree-sitter`,
used for dependency analysis), which means the whole binary needs a real C
compiler for whatever OS you build it for:

```bash
CGO_ENABLED=1 go build -o conjecture ./cmd/conjecture
```

- **Linux/macOS**: works directly if `gcc`/`clang` is installed.
- **Windows**: there is no native gcc. Build it inside WSL2 (install
  `golang-go` and `gcc` there), or inside a `golang` container via
  Podman/Docker:
  ```bash
  podman run --rm -v "$(pwd):/src" -w /src golang:1.25 \
    go build -o conjecture ./cmd/conjecture
  ```

## Runtime requirements

- **A container runtime on `PATH` named `docker`.** `conjecture check`/
  `implement`/`work` run the implementation under test inside a container
  with `--network none`. Real Docker works as-is; real Podman works behind
  a one-line shim (`docker` isn't aliased to it by default):
  ```bash
  printf '#!/bin/sh\nexec podman "$@"\n' | sudo tee /usr/local/bin/docker
  sudo chmod +x /usr/local/bin/docker
  ```
- **An LLM provider, only for `draft`/`implement`/`work`/`attack`/`swarm`**
  (writing a spec from a plain-language ask, writing/fixing implementation
  code, or attacking a hypothesis). Set one of:
  - `ANTHROPIC_API_KEY` (provider `anthropic`, the default)
  - `DEEPSEEK_API_KEY` (provider `deepseek`)
  - a local `ollama serve` or `llama-server` (provider `llamacpp` for
    `implement`/`work`; `--runtime ollama`/`--runtime llamacpp` for
    `attack`/`swarm`) - no key needed, reachable at `http://localhost:8080`
    (llama.cpp) or `http://localhost:11434` (Ollama) by default.
  `conjecture check`/`verify`/`sign`/`promote`/`context` need none of this -
  they only run already-written specs against already-written code.

## A worked example

```bash
conjecture init   # conjecture/phenomena/, conjecture/materializations/, .conjecture/
```

Write the rule at `conjecture/phenomena/checkout/shipping.cjtr`:

```yaml
conjecture: 1.0.0

phenomenon: shipping

concepts:
  discounted subtotal: {variable: input.discounted_subtotal, type: decimal}
  order type: {variable: input.order_type, one_of: [regular, subscription]}
  shipping fee: {variable: output.shipping_fee, type: decimal}
  free shipping threshold: {value: 100.00}

rules:
  shipping is free above the threshold:
    id: free-above
    hypotheses:
      above the threshold shipping is free:
        id: above-threshold
        given:
          - the discounted subtotal is greater than the free shipping threshold
          - the order type is not subscription
        then: the shipping fee is 0.00
        otherwise: the shipping fee is at least 5.00
        examples:
          - {discounted subtotal: 150.00, order type: regular, shipping fee: 0.00}
          - {discounted subtotal: 50.00, order type: regular, shipping fee: 7.50}
          - {discounted subtotal: 150.00, order type: subscription, shipping fee: 7.50}
```

The "implementation" is any container image + command that reads
`conjecture_input.json` (flat JSON, keys like `input.order_type`) and writes
`conjecture_output.json` the same way. Here's a real one - with a real bug,
flat $7.50 no matter what:

```python
# adapter.py
import json
data = json.load(open("conjecture_input.json"))
json.dump({"output.shipping_fee": "7.50"}, open("conjecture_output.json", "w"))
```

```dockerfile
FROM python:3-slim
WORKDIR /work
COPY adapter.py /work/adapter.py
```

```
$ podman build -t shipping-demo:latest .
$ conjecture check conjecture/phenomena/checkout/shipping.cjtr shipping-demo:latest python3 /work/adapter.py
checkout::shipping::free-above: above the threshold shipping is free: judge: accepted
checkout::shipping::free-above: above the threshold shipping is free: conformance (directed sampling): non-conformant
checkout::shipping::free-above: above the threshold shipping is free:   counterexample: map[input.discounted_subtotal:150.00 input.order_type:regular output.shipping_fee:0.00]
```

Caught for real - a $150 regular order should ship free, and this
implementation never does. Fix the bug for real:

```python
# adapter.py
import json
from decimal import Decimal

data = json.load(open("conjecture_input.json"))
subtotal = Decimal(data["input.discounted_subtotal"])
order_type = data["input.order_type"]

fee = Decimal("0.00") if subtotal > Decimal("100.00") and order_type != "subscription" else Decimal("7.50")
json.dump({"output.shipping_fee": str(fee)}, open("conjecture_output.json", "w"))
```

```
$ podman build -t shipping-demo:latest .
$ conjecture check conjecture/phenomena/checkout/shipping.cjtr shipping-demo:latest python3 /work/adapter.py
checkout::shipping::free-above: above the threshold shipping is free: judge: accepted
checkout::shipping::free-above: above the threshold shipping is free: conformance (directed sampling): conformant

$ conjecture sign --user alice --confirmed conjecture/phenomena/checkout/shipping.cjtr
```

That's the whole core loop: write the rule once, point it at real code,
get a real verdict - and a signed, attributable record that a human looked
at it.

## What this is *not* for

- **Not a general test framework.** It only checks whether a running
  implementation matches a rule you've written in `.cjtr`'s own narrow
  grammar (comparisons over decimals/categories, `given`/`then`/
  `otherwise`/`always`). It can't express arbitrary logic, and it won't
  replace your unit tests for anything that isn't shaped like a declared
  business rule.
- **Not exhaustive proof, by default.** `check`'s directed-sampling mode
  tries boundary and partition values, not every possible input. Only a
  hypothesis over a fully bounded, enumerable domain gets this codebase's
  own "complete proof" rigor level - most checks are strong evidence, not
  a mathematical guarantee for every possible input.
- **Not a GUI or UI testing tool yet.** There's a real accessibility-driver
  backend for Windows and Linux, but nothing in the `.cjtr` grammar can
  target a GUI - it's unreachable from the actual CLI workflow today.
- **Not a sandboxing/security product.** The sandbox blocks outbound
  network access and nothing else - no filesystem or resource hardening.
  Don't point it at code you don't already trust to run on your machine.
- **Not a static analyzer for a whole codebase.** `deps check` is a narrow
  forbidden-import checker for one Go file at a time, not a linter.
- **Not something with a production track record.** This is an
  experimental project, built mostly in one extended session. Nobody has
  used it yet to make a real decision about real code they depend on -
  that's the next real test, not something already behind it.

## Status

The verification core (check → attack/swarm → promote, sandboxed execution,
the decision ledger, context-composition's full command set) is implemented
and has been run for real: against a real constraint solver (z3), a real
local LLM (llama.cpp/Ollama), a real genuine third-party library
(python-slugify - this actually found and fixed a real bug in this repo),
and real network-isolation verification. Two things to know before relying
on it:

- **The GUI-accessibility driver has real Windows (UI Automation) and Linux
  (AT-SPI) backends, but the `.cjtr` grammar has no way to declare a GUI
  target yet** - the capability exists and is tested on its own, but nothing
  in the CLI's main verification flow can reach it. macOS is not supported
  (no backend, by deliberate decision - see
  [issue #144](https://github.com/farlleyferreira/conjecture-ai/issues/144)).
- **Materialization content generation** (a `conjecture/materializations/`
  path exists after `conjecture init`, but nothing writes anything there
  yet) is explicitly out of scope of the persistence work done so far.

## Commands

| Command | Does |
|---|---|
| `init` | Creates the repo layout (`conjecture/phenomena/`, `conjecture/materializations/`, gitignored `.conjecture/`). |
| `draft` | Asks an LLM to draft a `.cjtr` spec from a plain-language request. |
| `sign` | Records authorship/confirmation on a `.cjtr` file; `--confirmed` also records a real decision-ledger entry. |
| `verify` | Checks a spec's own worked examples against its own hypotheses - no sandbox, no implementation. |
| `check` | Runs a hypothesis's directed-sampling conformance cycle against a real sandboxed implementation. |
| `implement` / `work` | Asks an LLM to write/fix implementation code until it conforms (or the round budget runs out); `work` pulls from the queue instead of taking a file directly. |
| `attack` / `swarm` | Runs one or several adversarial-LLM attacks against a hypothesis, looking for a counterexample. |
| `promote` | Promotes a hypothesis/law to Theory or Law once its real evidence conditions hold; records a decision-ledger entry. |
| `law` | Resolves a conflict between a user-defined law and a formal proof or another law. |
| `context` | `list`/`resume`/`open`/`origin`/`diff`/`restore`/`resolve-conflict` over a saved verification context. |
| `base` | Attaches/searches a knowledge base (norms, definitions, references) scoped to a project or area. |
| `queue` | The three-tier (blocker/current-task/background) work queue `work` consumes. |
| `results` | Imports a target system's own already-produced JUnit XML or TAP results. |
| `graph` | Reads/sets a context-engine node's weight/relevance/confidence. |
| `optimize` | Applies a calibration proposal - always requires explicit `--approved`. |
| `settings` | Checks a value against a team-configured floor (never a ceiling). |
| `translate` | Generates a hypothesis's deterministic `# pt:` comment. |
| `deps` | Checks Go source for forbidden imports. |
| `watch` | Polls a directory and re-verifies any `.cjtr` file that changed. |
| `mcp` | Runs conjecture as an MCP server. |

Run any command with no arguments (or `--help`-shaped nonsense) to see its
own real usage string.

## License

This project is licensed under the [Apache License, Version 2.0](LICENSE).
