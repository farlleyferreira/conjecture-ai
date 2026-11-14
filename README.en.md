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
def change_fee(inputs):
    hours = Decimal(inputs["hours_before_departure"])
    fee = "0.00" if hours >= 24 else "150.00"
    return {"change_fee": fee}
```

```
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr --lang python --entry pricing.fees:change_fee
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: fare_class=economy-basic, hours_before_departure=30.00 -> change_fee=0.00
```

An Economy Basic ticket went free: the code forgot the fare class. The
agent fixed that, but wrote `hours > 24`:

```
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: fare_class=economy-flex, hours_before_departure=24.00 -> change_fee=150.00
```

Exactly 24 hours should have been free. After the second fix:

```
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (hidden sampling (20 random cases, seed 17164254410232975056)): conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: 20/100 failure-free hidden cases against this code (low weight)

$ conjecture status conjecture/phenomena/ticketing/ticket-change-fee.cjtr
ticketing::ticket-change-fee: not signed
ticketing::ticket-change-fee::flex-fee-waiver::flex-inside-window: stage=Hypothesis attacked="hidden sampling (20 random cases, seed 17164254410232975056)" survived="every sampled case conformed"
ticketing::ticket-change-fee::flex-fee-waiver::flex-inside-window: hidden cases 20/100 (low weight); code versions meeting it: 0; model families meeting it: none yet
```

`status` does not say "approved". It says what was attacked, what
survived, how many cases are still needed to move up a stage, and that
no human has signed yet.

There is no hand-written Dockerfile, image or adapter in this demo. The
outputs above come from a real run.

<details>
<summary>The rule as a <code>.cjtr</code> file</summary>

It is YAML in a restricted grammar. Your agent writes it, you review and
sign it.

```yaml
# conjecture/phenomena/ticketing/ticket-change-fee.cjtr
conjecture: 1.0.0

phenomenon: ticket-change-fee

concepts:
  hours before departure: {variable: input.hours_before_departure, type: decimal, range: {min: 0.00, max: 720.00}}
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

Every binary carries a build provenance attestation. To check a
download:

```bash
gh attestation verify conjecture_linux_amd64.tar.gz --repo farlleyferreira/conjecture-ai
```

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
3. The agent writes the implementation and an entry function for the
   rule.
4. The agent calls `check`. If a counterexample comes back, it fixes the
   code. The signed rule does not change to make the code pass.
5. `status` sums up who signed and the state of each hypothesis.

Any stdio MCP client works with `conjecture mcp`. The skill follows the
Claude Code format.

## The entry function

`conjecture` does not read your code. You point at one function, and it
generates the adapter, picks the language's image and mounts your
project read-only, with no network.

The function has the same shape in every language: it takes the rule's
inputs as a map of name to text and returns the outputs the same way.
Values arrive as text (`"30.00"`), so a decimal never becomes a binary
float on the way.

| Language | `--lang` | `--entry` |
|---|---|---|
| Python | `python` | `package.module:function` |
| JavaScript | `javascript` | `path/file.js:exportedFunction` |
| TypeScript | `typescript` | `path/file.ts:exportedFunction` |
| Go | `go` | `./package/dir:Function` |
| Java | `java` | `com.example.Class:staticMethod` |
| C# | `csharp` | `path/Assembly.dll:Namespace.Class:StaticMethod` |
| Ruby | `ruby` | `path/file.rb:Module::Class.method` |

For another language, or for dependencies that are not in the project
tree, the original form remains: `conjecture check <file.cjtr> <image> <command...>`,
with a program that reads `conjecture_input.json` and writes
`conjecture_output.json`.

## Use it from the terminal

| Command | Does |
|---|---|
| `conjecture verify <file.cjtr>` | Checks the rule's examples against the rule itself. No container, no AI. |
| `conjecture check <file.cjtr> --lang <language> --entry <entry>` | Generates cases, runs your function in the container, gives the verdict. No AI. |
| `conjecture sign --user <name> --confirmed <file.cjtr>` | Records that a human reviewed the rule. |
| `conjecture status [file.cjtr]` | Shows signature, stage, evidence and what the next promotion still needs. |

`verify`, `check`, `sign`, `status`, `promote` and `context` run on your
machine and call no model. `draft`, `implement`, `work`, `attack` and
`swarm` call the provider you configure. With a cloud provider, what you
send it leaves your machine; with a local `ollama` or `llama-server`, it
does not.

Full command and provider list: [docs/COMMANDS.md](docs/COMMANDS.md).

## How trust goes up

Each rule has a stage. No step takes the word of whoever asks for it:
every one is computed from what is on record.

`check` runs two kinds of case. Directed cases test the edges of the
domain and the thresholds the rule itself names (24.00, 23.99, 24.01).
Hidden cases are drawn at random from the declared domain on every run.
Only hidden cases count toward promotion: the same cases a hundred times
are one observation, not a hundred.

| Step | What has to be on record |
|---|---|
| Hypothesis to Thesis | Failure-free hidden cases against the current rule and code: 100 (low weight), 300, 1,000 or 3,000 (critical weight). Automatic in `check`. |
| Thesis to Theory | Every rule in the file at Thesis, the rule signed with `--confirmed`, the count met on implementations from two different model families, and an adversarial attack that found no violation. Through `conjecture promote theory`. |
| Theory to Law | A universal rule (`always`), at Theory, with the count met on two versions of the code. Through `conjecture promote law`. |

One counterexample demotes the rule and resets that code version's
count. Changing one byte of the code starts a new count.

## Limits

- **It does not prove the code is right.** The cases are samples. 100
  failure-free cases bound the violation rate at about 3% with 95%
  confidence, inside the declared domain and not outside it. Only a
  fully enumerable domain gets complete verification.
- **It only covers declarable business rules.** The `.cjtr` grammar
  expresses comparisons over decimals and categories
  (`given`/`then`/`otherwise`/`always`). It does not replace unit tests
  for arbitrary logic.
- **The entry function is a place to cheat.** The adapter is generated,
  but the entry function is project code and decides what the rule's
  names mean. If it re-implements the rule instead of calling the real
  code, `conjecture` approves the function, not the system. Review it.
- **The rule is worth what your review is worth.** A `.cjtr` drafted by
  AI and not signed gets a notice from `verify` and shows as `not signed`
  in `status`, but `conjecture` cannot know whether the signed rule is
  the rule you meant.
- **The model family is declared, not verified.** `--produced-by`
  records who wrote the implementation. Nothing in the code proves its
  author. The step to Theory depends on that declaration.
- **No declared domain, no hidden cases.** A decimal with no `range` or
  a text with no `one_of` cannot be drawn from, and the rule stays at
  Hypothesis. `check` names the concept that is missing one.
- **Dependencies have to be in the project tree.** The container has no
  network. `node_modules`, compiled classes and the Go module cache
  work; a global `pip install` does not. For the rest, pass `--image`.
- **TypeScript runs on Node, with no compiler.** Relative imports need
  an explicit extension. C# needs you to build the project first and
  point at the DLL.
- **Every case is one container run.** We measured about half a second
  per case (Python, Podman on Windows): 100 cases in about a minute,
  3,000 in about 25. Compiled languages and other machines were not
  measured.
- **`implement` and `work` do not change your project.** They produce a
  disposable single-file Python script. For real code, use the workflow
  with your agent.
- **`attack` and `swarm` do not run code.** The counterexample an
  adversarial model claims is not mechanically verified. Read it before
  believing it.
- **The sandbox only blocks outbound network.** There is no filesystem
  or resource isolation. Do not run code you do not trust.
- **The binaries are not code-signed.** The attestation proves where
  they came from, but Windows (SmartScreen, Smart App Control, Defender)
  may warn, block, or flag the executable as suspicious.
- **Linux and Windows binaries have no `deps check`.** That command uses
  Tree-sitter, which is C code. To use it, build with `CGO_ENABLED=1`.
- **No production history.** The project was built largely in long
  sessions with AI agents.
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
