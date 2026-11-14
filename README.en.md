[🇧🇷 Português](README.md) | 🇺🇸 **English**

# conjecture-ai

**Write a rule in plain words. `conjecture` proves, for real, whether your
code obeys it.**

[![CI](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## The simplest way to think about it

Imagine a rule for a game: *"if you roll a 6, you get another turn."* You
don't have to read the whole rulebook to find out if that's true - you just
play the game, roll a 6, and watch what happens.

`conjecture` does that for software rules. You write the rule once, in
plain words, in a small file - no code. `conjecture` then plays the game
for you: it builds real test cases from your rule, runs your *actual*
program against them inside a locked-down sandbox, and tells you
**conformant** (it holds) or **non-conformant** (here's the exact input
that broke it). It never takes the code's word for it. It always watches
what the code actually does.

## Say it once, in plain words

Here's a real airline policy: *"Economy Flex tickets changed at least 24
hours before departure are free to change; anything else pays a change
fee."* Normally the only way to know whether your code actually does that
is to read the code, or write a unit test by hand and hope you thought of
the right cases.

```yaml
# conjecture/phenomena/ticketing/ticket-change-fee.cjtr
conjecture: 1.0.0

phenomenon: ticket-change-fee

concepts:
  hours before departure: {variable: input.hours_before_departure, type: decimal}
  fare class: {variable: input.fare_class, one_of: [economy-basic, economy-flex]}
  change fee: {variable: output.change_fee, type: decimal}
  free change window: {value: 24.00}

rules:
  flex fares waive the fee inside the window:
    id: flex-fee-waiver
    hypotheses:
      flex fare inside window is free:
        id: flex-inside-window
        given:
          - the fare class is economy-flex
          - the hours before departure is at least the free change window
        then: the change fee is 0.00
        otherwise: the change fee is at least 75.00
        examples:
          - {hours before departure: 30.00, fare class: economy-flex, change fee: 0.00}
          - {hours before departure: 10.00, fare class: economy-flex, change fee: 150.00}
          - {hours before departure: 30.00, fare class: economy-basic, change fee: 150.00}
```

That's it. No code. Just the rule, in words a non-programmer could read
out loud.

## See it catch a real bug

Here's an "implementation" with a real bug: it forgets to check the fare
class at all, and only looks at how many hours are left.

```python
# adapter.py
import json
data = json.load(open("conjecture_input.json"))
hours = float(data["input.hours_before_departure"])
fee = "0.00" if hours >= 24.00 else "150.00"
json.dump({"output.change_fee": fee}, open("conjecture_output.json", "w"))
```

```dockerfile
FROM python:3-slim
WORKDIR /work
COPY adapter.py /work/adapter.py
```

```
$ podman build -t ticket-fee-demo:latest .
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: map[input.fare_class:economy-basic input.hours_before_departure:30.00 output.change_fee:0.00]
```

Caught for real - an Economy Basic ticket changed 30 hours out should
still pay the fee, and this implementation waives it just because enough
hours are left. Fix the bug for real:

```python
# adapter.py
import json
data = json.load(open("conjecture_input.json"))
hours = float(data["input.hours_before_departure"])
fare_class = data["input.fare_class"]
fee = "0.00" if (fare_class == "economy-flex" and hours >= 24.00) else "150.00"
json.dump({"output.change_fee": fee}, open("conjecture_output.json", "w"))
```

```
$ podman build -t ticket-fee-demo:latest .
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): conformant

$ conjecture sign --user alice --confirmed conjecture/phenomena/ticketing/ticket-change-fee.cjtr
```

Write the rule once, point it at real code, get a real verdict - and a
signed, attributable record that a human looked at it.

## Why `conjecture`

- **A signed spec is frozen before code is judged.** Once you sign a
  `.cjtr` file, the judge checks implementation code only against that
  exact frozen rule - nobody can quietly move the goalposts mid-check.
- **The model that writes the code never grades its own homework.**
  Drafting and implementing always run in separate sessions. Neither one
  can see the hidden cases the judge uses, call the judge directly, or
  sign anything on your behalf.
- **An independent model actively tries to break your rule.** One local
  adversarial SLM by default, or a swarm of at least five distinct
  attack strategies, with budget allocated between them by a bandit -
  hunting for the counterexample you didn't think of, never colluding
  or debating with each other.
- **Climbing the maturity ladder takes more than "it passed once".**
  Thesis to Theory needs a second implementation, from a different model
  family than the original, reproducing the result, plus robustness
  against the whole swarm. Theory to Law needs conformance against
  hidden samples and across new code versions. Evidence climbs a real
  ladder (Hypothesis → Thesis → Theory → Law), never a binary pass/fail.
- **Weight, relevance, and confidence are three independent axes, never
  multiplied together.** Money, identity, permissions, and personal data
  automatically come in as high weight - and that weight is never
  auto-tuned by the system itself, on purpose.
- **Knowledge bases cite real authority.** Norm, definition, reference,
  or historical - four levels, not a flat list of links - so a decision
  can point at something beyond the `.cjtr` file itself.
- **Every verification context is versioned and resumable.** List it,
  reopen an exact past version, diff two versions, or restore one -
  nothing about where you left off has to be rebuilt from memory.
- **It works with whatever language you already have.** `conjecture`
  observes your real program only through a declared input/output
  contract, never by understanding your language's own semantics.
- **Self-tuning stays on a leash.** The only things `conjecture` can
  optimize about itself - input generation, swarm budget, calibration -
  all require your explicit `--approved` before anything changes.

## Quick start

> [!NOTE]
> There's no pre-built download yet - you build the one binary yourself.
> It's one command, and it's the same command on every OS.

This repo has exactly one CGO dependency
(`github.com/smacker/go-tree-sitter`, used for dependency analysis), so the
binary needs a real C compiler for whatever OS you build it for:

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

You also need **a container runtime on `PATH` named `docker`** -
`conjecture check`/`implement`/`work` run the implementation under test
inside a container with `--network none`. Real Docker works as-is; real
Podman works behind a one-line shim (`docker` isn't aliased to it by
default):

```bash
printf '#!/bin/sh\nexec podman "$@"\n' | sudo tee /usr/local/bin/docker
sudo chmod +x /usr/local/bin/docker
```

Then:

```bash
conjecture init   # creates conjecture/phenomena/, conjecture/materializations/, .conjecture/
```

Write a rule, run `conjecture check` against your real code - that's the
whole worked example above, start to finish.

> [!NOTE]
> `conjecture check`/`verify`/`sign`/`promote`/`context` need no LLM at all
> - they only run already-written rules against already-written code.
> Only `draft`/`implement`/`work`/`attack`/`swarm` need one, and switching
> providers is just a `--provider` flag plus that provider's own API key -
> no code change, ever. Set one of: `ANTHROPIC_API_KEY` (provider
> `anthropic`, the default), `DEEPSEEK_API_KEY` (provider `deepseek`), a
> local `ollama serve`/`llama-server` (provider `llamacpp`, no key needed),
> or any provider speaking OpenAI's own Chat Completions format - verified
> against nine: `openai`, `gemini`, `grok`, `llama-api` (Meta's hosted API,
> distinct from local `llamacpp`), `mistral`, `cohere`, `bedrock`,
> `azure-openai`, `dashscope` (serves Qwen) - each just its own env vars,
> listed in [`internal/llmclient/openaicompatible.go`](internal/llmclient/openaicompatible.go).

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
| `queue` | The three-tier (blocker/current-task/background) work queue `work` consumes; `review` lists items it couldn't act on automatically. |
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

<details>
<summary><strong>Scope and limits</strong> - read before you rely on this for anything that matters</summary>

The verification core (check → attack/swarm → promote, sandboxed execution,
the decision ledger, context-composition's full command set) is implemented
and has been run for real: against a real constraint solver (z3), a real
local LLM (llama.cpp/Ollama), a real genuine third-party library
(python-slugify - this actually found and fixed a real bug in this repo),
and real network-isolation verification.

What it's not, on purpose:

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
- **Not a sandboxing/security product.** The sandbox blocks outbound
  network access and nothing else - no filesystem or resource hardening.
  Don't point it at code you don't already trust to run on your machine.
- **Not a static analyzer for a whole codebase.** `deps check` is a narrow
  forbidden-import checker for one Go file at a time, not a linter.
- **Not something with a production track record yet.** This is an
  experimental project, built mostly in one extended session. Nobody has
  used it yet to make a real decision about real code they depend on -
  that's the next real test.

One more specific thing to know:

- **The GUI-accessibility driver has real Windows (UI Automation) and Linux
  (AT-SPI) backends, but the `.cjtr` grammar has no way to declare a GUI
  target yet** - the capability exists and is tested on its own, but nothing
  in the CLI's main verification flow can reach it. macOS is not supported
  (no backend, by deliberate decision - see
  [issue #144](https://github.com/farlleyferreira/conjecture-ai/issues/144)).

</details>

## License

This project is licensed under the [MIT License](LICENSE).
