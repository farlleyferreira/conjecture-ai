[🇧🇷 Português](README.md) | 🇺🇸 **English**

# conjecture-ai

**Your agent said "done". `conjecture` tries to break what it wrote,
using a rule you approved, and shows the exact input that broke it.**

[![CI](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> **Experimental project.** Nobody has used this yet to decide anything
> about code they depend on. Read [Limits](#limits) before trusting it.

## In 30 seconds

The rule: *"Economy Flex changed 24 hours or more before departure pays
no fee. Everything else pays."*

The agent delivered this and said it was finished:

```python
hours = float(data["input.hours_before_departure"])
fee = "0.00" if hours >= 24.00 else "150.00"
```

`conjecture` builds cases from the rule, runs the program in a container
with no network, and answers:

```
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: map[input.fare_class:economy-basic input.hours_before_departure:30.00 output.change_fee:0.00]
```

An Economy Basic ticket changed 30 hours ahead went free: the code
forgot the fare class. After the fix:

```
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): conformant

$ conjecture status conjecture/phenomena/ticketing/ticket-change-fee.cjtr
ticketing::ticket-change-fee: not signed
ticketing::ticket-change-fee::flex-fee-waiver::flex-inside-window: stage=Hypothesis attacked="directed sampling" survived="every generated case conformed"
```

`status` does not say "approved". It says what was attacked, what
survived, which stage the rule is at, and that no human has signed it
yet.

<details>
<summary>The rule as a <code>.cjtr</code> file</summary>

It is YAML in a restricted grammar. Your agent writes it, you review and
sign it.

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

</details>

<details>
<summary>The full program and the demo container</summary>

`conjecture` does not read your code. It writes `conjecture_input.json`,
runs the command you name inside the image, and reads
`conjecture_output.json`.

```python
# adapter.py - fixed version
import json
data = json.load(open("conjecture_input.json"))
hours = float(data["input.hours_before_departure"])
fare_class = data["input.fare_class"]
fee = "0.00" if (fare_class == "economy-flex" and hours >= 24.00) else "150.00"
json.dump({"output.change_fee": fee}, open("conjecture_output.json", "w"))
```

```dockerfile
FROM python:3-slim
WORKDIR /work
COPY adapter.py /work/adapter.py
```

```bash
podman build -t ticket-fee-demo:latest .
```

</details>

## The problem

The same model that writes the code writes the tests and declares it
finished. A second agent of the same model reviewing it inherits the
same blind spots. And "passed" does not say what was tested or what was
left out.

`conjecture` separates the roles: the rule is signed by you, the cases
are generated from it by a verifier with no AI in it, and the verdict
comes from running the program.

## Install

Linux and macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/farlleyferreira/conjecture-ai/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/farlleyferreira/conjecture-ai/main/install.ps1 | iex
```

The scripts download the binary from the [latest release](https://github.com/farlleyferreira/conjecture-ai/releases/latest)
and check its SHA-256. With Go installed, this also works:

```bash
go install github.com/farlleyferreira/conjecture-ai/cmd/conjecture@latest
```

You need Docker or Podman for `check`. `conjecture` uses `docker` when
it is on `PATH`, otherwise `podman`. Set `CONJECTURE_CONTAINER_RUNTIME`
to pick another.

The binaries are not signed. Windows (SmartScreen or Smart App Control)
may warn or block on first run.

## Use it with your agent

`conjecture` runs as an MCP server and installs a skill that teaches the
workflow to the agent. With Claude Code:

```bash
conjecture init
```

```bash
claude mcp add conjecture -- conjecture mcp
```

`init` creates `conjecture/phenomena/` and
`.claude/skills/conjecture-workflow/SKILL.md`. Then ask for the rule in
plain language. The workflow the skill enforces:

1. The agent writes the `.cjtr` with you and calls `verify` on every
   change.
2. You sign in your own terminal: `conjecture sign --user <name> --confirmed <file.cjtr>`.
   The MCP server exposes neither `sign` nor `promote`: the agent cannot
   approve the rule for you.
3. The agent writes the implementation and calls `check`.
4. If a counterexample comes back, it fixes the code. The signed rule
   does not change to make the code pass.
5. `status` sums up who signed and the state of each hypothesis.

Any stdio MCP client works with `conjecture mcp`. The skill follows the
Claude Code format.

## Use it from the terminal

| Command | Does |
|---|---|
| `conjecture verify <file.cjtr>` | Checks the rule's examples against the rule itself. No container, no AI. |
| `conjecture check <file.cjtr> <image> <command...>` | Generates cases, runs the program in the container, gives the verdict. No AI. |
| `conjecture sign --user <name> --confirmed <file.cjtr>` | Records that a human reviewed the rule. |
| `conjecture status [file.cjtr]` | Shows signature, stage and evidence for each hypothesis. |

`verify`, `check`, `sign`, `status`, `promote` and `context` run on your
machine and call no model. `draft`, `implement`, `work`, `attack` and
`swarm` call the provider you configure. With a cloud provider, what you
send it leaves your machine; with a local `ollama` or `llama-server`, it
does not.

Full command and provider list: [docs/COMMANDS.md](docs/COMMANDS.md).

## How trust goes up

Each rule has a stage: Hypothesis, Thesis, Theory, Law. The idea is that
moving up takes new evidence, not the same test again. What exists today
is less than that:

| Step | How it happens today |
|---|---|
| Hypothesis to Thesis | Automatic in `check`, but only for rules declared as `laws`, after 3,000 accumulated failure-free cases. One counterexample resets the count. An ordinary hypothesis stays at Hypothesis even when `check` is conformant. |
| Thesis to Theory | Manual, with `conjecture promote theory`. The conditions (mechanism confirmed, reproduction by a different model family) are flags you attest. |
| Theory to Law | Manual, with `conjecture promote law`. The conditions (hidden samples, new code versions) are also flags you attest. |

Every promotion is recorded with who decided it.

## Limits

- **It does not prove the code is right.** `check` tries boundary and
  partition values, not every possible input. Only a fully enumerable
  domain gets complete verification.
- **It only covers declarable business rules.** The `.cjtr` grammar
  expresses comparisons over decimals and categories
  (`given`/`then`/`otherwise`/`always`). It does not replace unit tests
  for arbitrary logic.
- **You have to package the program.** Each implementation becomes a
  container image with an adapter that reads and writes JSON. That is
  extra work per rule.
- **The adapter is a place to cheat.** If the same agent writes the code
  and the adapter, it can make the adapter answer what the rule expects.
  Review the adapter.
- **The rule is worth what your review is worth.** A `.cjtr` drafted by
  AI and not signed gets a notice from `verify` and shows as `not signed`
  in `status`, but `conjecture` cannot know whether the signed rule is the rule you meant.
- **The stage ladder is incomplete.** Ordinary hypotheses do not move up,
  and Theory and Law depend on flags a human attests, not on
  measurement. See [How trust goes up](#how-trust-goes-up).
- **`implement` and `work` do not change your project.** They produce a
  disposable single-file Python script. For real code, use the workflow
  with your agent.
- **`attack` and `swarm` do not run code.** The counterexample an
  adversarial model claims is not mechanically verified. Read it before
  believing it.
- **The sandbox only blocks outbound network.** There is no filesystem
  or resource isolation. Do not run code you do not trust.
- **Linux and Windows binaries have no `deps check`.** That command uses
  Tree-sitter, which is C code. To use it, build with `CGO_ENABLED=1`.
- **No production history.** The project was built largely in a single
  long session, with AI agents.
- **GUI targets do not work through the main flow.** Accessibility
  backends exist for Windows and Linux, but the grammar cannot declare a
  GUI target yet. macOS is not supported
  ([issue #144](https://github.com/farlleyferreira/conjecture-ai/issues/144)).

## How we will know it works

We do not know yet. The deciding target was fixed before the experiment:
at least 30% fewer violations than the same AI without `conjecture`,
across 50 tasks in each of two projects in different languages. If that
target fails, no other one saves the proposal.

Result: not published yet.

## Build from source

```bash
go build -o conjecture ./cmd/conjecture
```

No C compiler needed. With `CGO_ENABLED=1` and `gcc` or `clang`, the
binary also includes `deps check`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT License](LICENSE).
