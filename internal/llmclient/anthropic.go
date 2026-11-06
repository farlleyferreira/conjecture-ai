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

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
	// anthropicDefaultModel is deliberately the smallest, cheapest
	// current Claude model rather than the strongest one: this
	// package's own default has no basis to assume every caller wants
	// (and pays for) the most capable model available. A caller that
	// wants a stronger one sets AnthropicClient.Model itself.
	anthropicDefaultModel     = "claude-haiku-5-5"
	anthropicDefaultMaxTokens = 4096
)

// AnthropicClient calls Anthropic's Messages API directly over HTTPS,
// with net/http and encoding/json alone - deliberately not through the
// official SDK, which pulls in OpenTelemetry, webhook-verification, and
// streaming machinery this package has no use for, to do what is
// otherwise one authenticated JSON request and one JSON response. This
// is the same "use a library only for what's genuinely complex"
// judgment this project already made for yaml.v3 and jsonschema/v6,
// both of which wrap an actual external standard this package is not
// one of.
type AnthropicClient struct {
	APIKey string
	Model  string
	// BaseURL overrides anthropicAPIURL when set - a test hook, not a
	// configuration a real caller needs (Anthropic's API has one
	// endpoint).
	BaseURL    string
	HTTPClient *http.Client
}

// newAnthropicClient constructs an AnthropicClient reading its API key
// from the ANTHROPIC_API_KEY environment variable - never typed into
// this package directly, and never logged or included in an error
// message - failing rather than proceeding with an empty key if it is
// unset.
func newAnthropicClient() (*AnthropicClient, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("llmclient: ANTHROPIC_API_KEY is not set")
	}
	return &AnthropicClient{
		APIKey:     key,
		Model:      anthropicDefaultModel,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Error   *anthropicError         `json:"error"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete implements [Client] against Anthropic's Messages API,
// sending systemPrompt as the request's "system" field and userPrompt
// as its one "user" message, and concatenating every returned "text"
// content block into a single string.
func (c *AnthropicClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody, err := json.Marshal(anthropicRequest{
		Model:     c.Model,
		MaxTokens: anthropicDefaultMaxTokens,
		System:    systemPrompt,
		Messages:  []anthropicMessage{{Role: "user", Content: userPrompt}},
	})
	if err != nil {
		return "", fmt.Errorf("llmclient: encoding request: %w", err)
	}

	url := c.BaseURL
	if url == "" {
		url = anthropicAPIURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("llmclient: building request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llmclient: calling Anthropic: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llmclient: reading response: %w", err)
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("llmclient: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("llmclient: Anthropic API error (%s): %s", parsed.Error.Type, parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llmclient: Anthropic API returned status %d", resp.StatusCode)
	}

	var text strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	return text.String(), nil
}
