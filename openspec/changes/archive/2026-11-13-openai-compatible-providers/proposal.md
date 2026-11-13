# Proposal

## Why

Switching `conjecture draft`/`implement`/`work`/`attack`/`swarm` to a
different LLM provider meant writing a new Go client - the exact
opposite of "switching provider should be as simple as breathing."
Before building anything, each candidate was individually verified
(reading each provider's own documentation, not assumed) rather than
guessed: 9 of the 10 largest LLM players confirmed to expose a
genuinely OpenAI-compatible Chat Completions endpoint (same request/
response JSON shape an OpenAI SDK pointed at a different base_url
would use) - OpenAI, Google Gemini, xAI Grok, Meta's hosted Llama API,
Mistral, Cohere, Amazon Bedrock, Azure OpenAI, and Alibaba DashScope
(Qwen). Only Anthropic's own "Messages" API is a genuinely different
wire format - conjecture-ai already has a dedicated client for exactly
that one case.

## What Changes

- `internal/llmclient` gains `OpenAICompatibleClient`, one struct
  implementing [Client] against the shared wire format all 9 providers
  above speak, instead of 9 near-duplicate hand-written clients.
- `New` gains 9 new provider names, each a thin preset (its own fixed
  or templated BaseURL, its own required env vars for API key/model,
  and - where the provider's own API key is region-bound - region):
  `openai`, `gemini`, `grok`, `llama-api` (Meta's remote hosted API,
  distinct from the existing local `llamacpp` provider), `mistral`,
  `cohere`, `bedrock`, `azure-openai`, `dashscope`.
- Every new preset requires its model explicitly (no invented default)
  - there is no single "right" default model per provider this package
  has grounds to assume, and a silently wrong default risks real cost.
- Bedrock only supports its bearer "Bedrock API key" auth variant, not
  SigV4 request signing - SigV4 would pull in the full AWS SDK, a real
  dependency every other provider here needs none of.
- README's provider list documents all 9, with their exact env var
  names and base URLs.

## Capabilities

### New Capabilities

None - this is infrastructure (`internal/llmclient`), not a new CLI
capability; every existing command that already accepted `--provider`
simply gains more valid values for it.

### Modified Capabilities

None. `skip_specs: true`.

## Impact

- Affected paths: `internal/llmclient/openaicompatible.go` (new),
  `internal/llmclient/openaicompatible_test.go` (new),
  `internal/llmclient/llmclient.go`, `internal/llmclient/llmclient_test.go`,
  `README.md`.
- No command's existing behavior changes for a caller already using
  `anthropic`, `deepseek`, or `llamacpp` - those three keep their own
  dedicated clients, untouched.
- Verified for real in WSL against the compiled `conjecture` binary:
  `conjecture draft --provider gemini ...` with no `GEMINI_API_KEY` set
  fails naming exactly that variable; with the key set but no
  `GEMINI_MODEL`, fails naming that one instead; `--provider grok` with
  no key fails the same way; `--provider bedrock` with a key and model
  but no `BEDROCK_REGION` fails naming that one - confirming the real
  CLI flag plumbs all the way through to each new provider's own
  constructor, not just unit-tested in isolation. No live network call
  against any of the 9 real providers was made (no API keys available
  in this environment) - `Complete`'s own request/response handling is
  covered by `openaicompatible_test.go` against a fake HTTP server
  instead, the same level of verification `deepseek.go` already had.
- No new dependency (`net/http`/`encoding/json` only, same as every
  other client in this package).
