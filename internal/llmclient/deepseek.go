package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	deepseekAPIURL = "https://api.deepseek.com/chat/completions"
	// deepseekDefaultModel is DeepSeek's own general-purpose chat model -
	// this package's default has no basis to assume a caller wants its
	// more expensive reasoning model instead. A caller that wants a
	// different one sets DeepSeekClient.Model itself.
	deepseekDefaultModel     = "deepseek-chat"
	deepseekDefaultMaxTokens = 4096
)

// DeepSeekClient calls DeepSeek's Chat Completions API, which follows
// the same OpenAI-compatible request/response shape most providers
// besides Anthropic use - net/http and encoding/json alone, the same
// "use a library only for what's genuinely complex" judgment already
// applied to AnthropicClient.
type DeepSeekClient struct {
	APIKey string
	Model  string
	// BaseURL overrides deepseekAPIURL when set - a test hook (also
	// usable to point this client at a local stand-in server), not a
	// configuration a real caller needs.
	BaseURL    string
	HTTPClient *http.Client
}

// newDeepSeekClient constructs a DeepSeekClient reading its API key
// from the DEEPSEEK_API_KEY environment variable - never typed into
// this package directly, and never logged or included in an error
// message - failing rather than proceeding with an empty key if it is
// unset.
func newDeepSeekClient() (*DeepSeekClient, error) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("llmclient: DEEPSEEK_API_KEY is not set")
	}
	return &DeepSeekClient{
		APIKey:     key,
		Model:      deepseekDefaultModel,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type deepseekRequest struct {
	Model     string            `json:"model"`
	MaxTokens int               `json:"max_tokens"`
	Messages  []deepseekMessage `json:"messages"`
}

type deepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekResponse struct {
	Choices []deepseekChoice `json:"choices"`
	Error   *deepseekError   `json:"error"`
}

type deepseekChoice struct {
	Message deepseekMessage `json:"message"`
}

type deepseekError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete implements [Client] against DeepSeek's Chat Completions
// API, sending systemPrompt as a "system" message and userPrompt as a
// "user" message, and returning the first choice's message content -
// DeepSeek, like other OpenAI-compatible providers, returns exactly
// one choice unless a caller explicitly asks for more (this package
// never does).
func (c *DeepSeekClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := make([]deepseekMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, deepseekMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, deepseekMessage{Role: "user", Content: userPrompt})

	reqBody, err := json.Marshal(deepseekRequest{
		Model:     c.Model,
		MaxTokens: deepseekDefaultMaxTokens,
		Messages:  messages,
	})
	if err != nil {
		return "", fmt.Errorf("llmclient: encoding request: %w", err)
	}

	url := c.BaseURL
	if url == "" {
		url = deepseekAPIURL
	}
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
		return "", fmt.Errorf("llmclient: calling DeepSeek: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llmclient: reading response: %w", err)
	}

	var parsed deepseekResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("llmclient: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("llmclient: DeepSeek API error (%s): %s", parsed.Error.Type, parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llmclient: DeepSeek API returned status %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llmclient: DeepSeek API returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
