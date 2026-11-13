package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// openAICompatDefaultMaxTokens is this file's own disclosed default
// for every provider built on [OpenAICompatibleClient] - the same
// value [DeepSeekClient] (already proven to use this exact wire shape)
// already defaults to, not a value researched per provider.
const openAICompatDefaultMaxTokens = 4096

// OpenAICompatibleClient calls any provider that implements OpenAI's
// own Chat Completions wire format: POST <BaseURL>/chat/completions,
// bearer auth, a messages array in, "choices[0].message.content" out.
// Verified (by reading each provider's own documentation, not assumed)
// against OpenAI itself, Google Gemini, xAI Grok, Meta's hosted Llama
// API, Mistral, Cohere, Amazon Bedrock, Azure OpenAI, and Alibaba
// DashScope/Qwen - nine distinct providers exposing exactly this one
// shape behind their own BaseURL, API key, and model name, which is
// why one struct serves all of them rather than nine near-duplicates
// of [DeepSeekClient]. Anthropic is the one major provider confirmed
// NOT to fit this shape (its own "Messages" API) - [AnthropicClient]
// stays its own dedicated implementation for that reason, not an
// oversight.
type OpenAICompatibleClient struct {
	// Name is the provider's own display name, used only in error
	// messages so a failure names which provider rejected the call.
	Name string
	// BaseURL is the provider's own "v1" root; "/chat/completions" is
	// appended the same way an OpenAI SDK pointed at a different
	// base_url would.
	BaseURL    string
	APIKey     string
	Model      string
	MaxTokens  int
	HTTPClient *http.Client
}

type openAICompatRequest struct {
	Model     string                `json:"model"`
	MaxTokens int                   `json:"max_tokens"`
	Messages  []openAICompatMessage `json:"messages"`
}

type openAICompatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAICompatResponse struct {
	Choices []openAICompatChoice `json:"choices"`
	Error   *openAICompatError   `json:"error"`
}

type openAICompatChoice struct {
	Message openAICompatMessage `json:"message"`
}

type openAICompatError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete implements [Client] against any provider's own OpenAI-
// compatible Chat Completions endpoint, sending systemPrompt as a
// "system" message and userPrompt as a "user" message, and returning
// the first choice's message content - the same shape [DeepSeekClient]
// already proved out for one such provider.
func (c *OpenAICompatibleClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := make([]openAICompatMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, openAICompatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, openAICompatMessage{Role: "user", Content: userPrompt})

	reqBody, err := json.Marshal(openAICompatRequest{
		Model:     c.Model,
		MaxTokens: c.MaxTokens,
		Messages:  messages,
	})
	if err != nil {
		return "", fmt.Errorf("llmclient: encoding request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("llmclient: building request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.APIKey)

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llmclient: calling %s: %w", c.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llmclient: reading response: %w", err)
	}

	var parsed openAICompatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("llmclient: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("llmclient: %s API error (%s): %s", c.Name, parsed.Error.Type, parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llmclient: %s API returned status %d", c.Name, resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llmclient: %s API returned no choices", c.Name)
	}
	return parsed.Choices[0].Message.Content, nil
}

// requiredEnv returns the named environment variable's value, or an
// error naming which variable a caller must set - the same "fail
// rather than proceed with an empty credential" posture every other
// constructor in this package already takes.
func requiredEnv(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("llmclient: %s is not set", name)
	}
	return v, nil
}

func newOpenAICompatHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

// newOpenAIClient reads OPENAI_API_KEY and OPENAI_MODEL - there is no
// safe default model to assume on a caller's behalf (OpenAI offers
// many, at very different price points), so both are required rather
// than guessed.
func newOpenAIClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("OPENAI_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("OPENAI_MODEL")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name: "OpenAI", BaseURL: "https://api.openai.com/v1",
		APIKey: key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newGeminiClient reads GEMINI_API_KEY and GEMINI_MODEL, against
// Google's own explicit OpenAI-compatibility endpoint (a separate
// opt-in path alongside Gemini's native API, not Gemini's default
// wire format).
func newGeminiClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("GEMINI_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("GEMINI_MODEL")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name: "Gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		APIKey: key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newGrokClient reads GROK_API_KEY and GROK_MODEL, against xAI's own
// OpenAI-compatible Chat Completions endpoint.
func newGrokClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("GROK_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("GROK_MODEL")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name: "Grok", BaseURL: "https://api.x.ai/v1",
		APIKey: key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newLlamaAPIClient reads LLAMA_API_KEY and LLAMA_MODEL, against
// Meta's own hosted Llama API's compatibility endpoint - distinct from
// this package's own "llamacpp" provider, which calls a *locally run*
// llama.cpp server over its own non-OpenAI wire format, not this
// remote, paid, OpenAI-compatible one.
func newLlamaAPIClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("LLAMA_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("LLAMA_MODEL")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name: "Llama API", BaseURL: "https://api.llama.com/compat/v1",
		APIKey: key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newMistralClient reads MISTRAL_API_KEY and MISTRAL_MODEL.
func newMistralClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("MISTRAL_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("MISTRAL_MODEL")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name: "Mistral", BaseURL: "https://api.mistral.ai/v1",
		APIKey: key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newCohereClient reads COHERE_API_KEY and COHERE_MODEL, against
// Cohere's own "Compatibility API" - a reduced parameter set (model,
// messages, stream only) compared to OpenAI's full surface, per
// Cohere's own documentation, but the same request/response shape
// this client already sends.
func newCohereClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("COHERE_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("COHERE_MODEL")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name: "Cohere", BaseURL: "https://api.cohere.ai/compatibility/v1",
		APIKey: key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newBedrockClient reads BEDROCK_API_KEY, BEDROCK_MODEL, and
// BEDROCK_REGION. Amazon documents two ways to authenticate against
// Bedrock's own OpenAI-compatible endpoint - SigV4 request signing, or
// a bearer "Bedrock API key" - and this client only ever does the
// latter: SigV4 would need the full AWS SDK (a real dependency, not a
// config change) for a posture every other provider here needs none
// of, which is exactly the complexity "simple as breathing" set out to
// avoid. There is no sane default region to assume on a caller's
// behalf, so it is required like the key and model.
func newBedrockClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("BEDROCK_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("BEDROCK_MODEL")
	if err != nil {
		return nil, err
	}
	region, err := requiredEnv("BEDROCK_REGION")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name:    "Bedrock",
		BaseURL: fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/openai/v1", region),
		APIKey:  key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newAzureOpenAIClient reads AZURE_OPENAI_API_KEY, AZURE_OPENAI_MODEL,
// and AZURE_OPENAI_RESOURCE. Azure's own "model" is really the
// caller's own deployment name, not an OpenAI model id, and its
// endpoint is resource-specific - both are a caller's own Azure
// account's own values, never a default this package could guess.
func newAzureOpenAIClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("AZURE_OPENAI_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("AZURE_OPENAI_MODEL")
	if err != nil {
		return nil, err
	}
	resource, err := requiredEnv("AZURE_OPENAI_RESOURCE")
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{
		Name:    "Azure OpenAI",
		BaseURL: fmt.Sprintf("https://%s.openai.azure.com/openai/v1", resource),
		APIKey:  key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}

// newDashScopeClient reads DASHSCOPE_API_KEY, DASHSCOPE_MODEL, and
// DASHSCOPE_REGION ("intl" or "cn") - Alibaba's DashScope (which
// serves Qwen among other models) issues region-bound API keys, per
// its own documentation, so the region is required rather than
// defaulted to either one.
func newDashScopeClient() (*OpenAICompatibleClient, error) {
	key, err := requiredEnv("DASHSCOPE_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("DASHSCOPE_MODEL")
	if err != nil {
		return nil, err
	}
	region, err := requiredEnv("DASHSCOPE_REGION")
	if err != nil {
		return nil, err
	}
	var baseURL string
	switch region {
	case "intl":
		baseURL = "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
	case "cn":
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	default:
		return nil, fmt.Errorf("llmclient: DASHSCOPE_REGION must be \"intl\" or \"cn\", got %q", region)
	}
	return &OpenAICompatibleClient{
		Name: "DashScope", BaseURL: baseURL,
		APIKey: key, Model: model, MaxTokens: openAICompatDefaultMaxTokens,
		HTTPClient: newOpenAICompatHTTPClient(),
	}, nil
}
