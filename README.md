# conjecture-ai

`conjecture` is a CLI that implements the `.cjtr` language: a way to write a
business-rule **hypothesis** (a `given`/`then`/`otherwise` or `always`
statement, with worked examples) and verify it for real against a running
implementation - inside a network-isolated sandbox, not by inspecting source
code. A hypothesis that survives enough rounds of checking and adversarial
attack accumulates evidence and can be promoted (Hypothesis → Thesis →
Theory → Law), with every human promotion/review decision recorded in a
local, hash-chained ledger.

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

## Quick start

```bash
conjecture init                       # creates conjecture/phenomena/,
                                       # conjecture/materializations/, and
                                       # the local .conjecture/ (gitignored)
```

Write a hypothesis at `conjecture/phenomena/shipping/checkout.cjtr`:

```yaml
conjecture: 1.0.0

phenomenon: checkout

concepts:
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal

rules:
  shipping fee rule:
    id: shipping-rule
    hypotheses:
      fee matches:
        id: fee-matches
        always: the shipping fee is 7.50
        examples:
          - {order type: regular, shipping fee: 7.50}
```

The implementation under test is any container image + command that reads a
flat JSON object from `conjecture_input.json` (keys like `input.order_type`)
in its working directory and writes `conjecture_output.json` (keys like
`output.shipping_fee`) before exiting 0 - this project's own invented
convention, documented in `internal/conformancecycle/harness.go`. A minimal
Python adapter:

```python
import json
data = json.load(open("conjecture_input.json"))
json.dump({"output.shipping_fee": "7.50"}, open("conjecture_output.json", "w"))
```

Bake it into an image at an absolute path (the mounted workspace is a fresh
temp directory per run, not the image's own filesystem):

```dockerfile
FROM python:3-slim
WORKDIR /work
COPY adapter.py /work/adapter.py
```

```bash
podman build -t my-image:latest .
conjecture check conjecture/phenomena/shipping/checkout.cjtr my-image:latest python3 /work/adapter.py
conjecture sign --user alice --confirmed conjecture/phenomena/shipping/checkout.cjtr
```

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

This project is licensed under the [MIT License](LICENSE).
