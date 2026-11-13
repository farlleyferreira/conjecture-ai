// Package llmclient implements pieces of llm-orchestration's real
// infrastructure half: the actual network call a drafting or
// implementing session needs to reach an LLM provider, behind one
// provider-agnostic interface.
//
// This package does not decide anything about *what* to send a model
// (a drafting prompt's exact text, a .cjtr grammar primer, how to
// parse its response back into a candidate file) - that is a future
// caller's job (wiring this into "conjecture draft" or similar), kept
// separate from the plumbing of actually reaching a provider. It also
// does not enforce llm-orchestration's trust-boundary rules itself
// (see internal/llmorchestration's Allowed/VisibleCases) - this
// package only sends text and returns text; what the caller is
// allowed to do with the result is enforced where those rules already
// live.
package llmclient

import (
	"context"
	"fmt"
	"strings"
)

// Client sends a single system+user prompt to some configured LLM
// provider and returns its text completion. Every provider this
// package supports implements Client the same way, so a caller is
// never coupled to which one is actually configured - this exists
// because asking any one provider's API directly, hardwired into
// every call site, would make switching providers later a rewrite
// rather than a configuration change.
type Client interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// New returns the Client for the named provider (case-insensitive).
// Each provider reads its own configuration (its own API key, from its
// own environment variable) when constructed; New itself never makes a
// network call - a caller only pays for one inside Complete.
//
// "anthropic", "deepseek", and "llamacpp" each have their own
// dedicated implementation (Anthropic's own wire format, and two
// providers this package already proved out before the rest existed).
// Every other provider below is a thin, researched preset of
// [OpenAICompatibleClient] - see that type's own doc comment for which
// nine providers were actually confirmed to speak that one shared wire
// format, and why Anthropic could not simply be a tenth.
func New(provider string) (Client, error) {
	switch strings.ToLower(provider) {
	case "anthropic":
		return newAnthropicClient()
	case "deepseek":
		return newDeepSeekClient()
	case "llamacpp":
		return newLlamaCppClient()
	case "openai":
		return newOpenAIClient()
	case "gemini":
		return newGeminiClient()
	case "grok":
		return newGrokClient()
	case "llama-api":
		return newLlamaAPIClient()
	case "mistral":
		return newMistralClient()
	case "cohere":
		return newCohereClient()
	case "bedrock":
		return newBedrockClient()
	case "azure-openai":
		return newAzureOpenAIClient()
	case "dashscope":
		return newDashScopeClient()
	default:
		return nil, fmt.Errorf("llmclient: unrecognized provider %q", provider)
	}
}
