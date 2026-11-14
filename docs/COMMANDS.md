# Commands

[Português](COMMANDS.pt-BR.md)

Run any command with no arguments to see its usage line.

| Command | Does |
|---|---|
| `init` | Creates the repo layout (`conjecture/phenomena/`, `conjecture/materializations/`, gitignored `.conjecture/`). |
| `draft` | Asks an LLM to draft a `.cjtr` spec from a plain-language request. |
| `sign` | Records authorship/confirmation on a `.cjtr` file; `--confirmed` also records a real decision-ledger entry. |
| `verify` | Checks a spec's own worked examples against its own hypotheses - no sandbox, no implementation. |
| `check` | Runs a hypothesis's directed-sampling conformance cycle against a real sandboxed implementation. |
| `status` | Shows signature, stage and recorded evidence for one `.cjtr` file, or project-wide totals with no argument. |
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

`deps` only exists in binaries built with `CGO_ENABLED=1` (the macOS
release binaries, or one you build yourself with a C compiler). In the
others it reports that it needs a CGO build.

## Which commands call a model

`verify`, `check`, `sign`, `status`, `promote` and `context` call no
model. `draft`, `implement`, `work`, `attack` and `swarm` do.

## Providers

Pick one with `--provider`. The default is `anthropic`.

| Provider | Setup |
|---|---|
| `anthropic` | `ANTHROPIC_API_KEY` |
| `deepseek` | `DEEPSEEK_API_KEY` |
| `llamacpp` | A local `ollama serve` or `llama-server`. No key. |
| `openai`, `gemini`, `grok`, `llama-api`, `mistral`, `cohere`, `bedrock`, `azure-openai`, `dashscope` | OpenAI's Chat Completions format. Each one's environment variables are in [`internal/llmclient/openaicompatible.go`](../internal/llmclient/openaicompatible.go). |

## Container runtime

`check`, `implement` and `work` run the implementation in a container
with `--network none`. `conjecture` uses `docker` when it is on `PATH`,
otherwise `podman`. `CONJECTURE_CONTAINER_RUNTIME` picks another
executable.
