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
	llamaCppDefaultBaseURL     = "http://localhost:8080"
	llamaCppCompletionEndpoint = "/completion"
	// llamaCppNPredict is this client's own disclosed default for how
	// many tokens a completion may run - Client's own callers (e.g.
	// "conjecture implement") ask for a complete generated script, not
	// a short structured answer, so this is set well above
	// [internal/adversarialslm]'s own 256-token budget for its short
	// JSON attack verdicts.
	llamaCppNPredict = 2048
)

// LlamaCppClient implements [Client] against a locally running
// llama.cpp server (llama-server) - the same real local-inference
// server [internal/adversarialslm.LlamaCppRunner] already calls for
// attacks, reused here so "conjecture implement"/"conjecture work" can
// run their own round trip against a local model with no per-call
// cost and no outbound network call, rather than only ever being
// exercisable against a paid, remote provider.
//
// Unlike [AnthropicClient]/[DeepSeekClient], this needs no API key:
// llama-server's own HTTP API carries no authentication of its own,
// consistent with [internal/adversarialslm.LlamaCppRunner]'s own
// "Local execution only" posture (loopback by default, never a remote
// host unless BaseURL is overridden).
//
// This client has no chat template of its own: systemPrompt and
// userPrompt are concatenated into one raw prompt, since llama-server's
// "/completion" endpoint takes a single prompt string, not a
// structured messages list the way Anthropic's and DeepSeek's chat
// APIs do.
type LlamaCppClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// newLlamaCppClient constructs a LlamaCppClient pointed at
// llama-server's own standard local address, optionally overridden by
// the LLAMACPP_BASE_URL environment variable. Unlike the other
// providers' own constructors, this never fails on missing
// configuration - there is no required credential to check - a
// genuinely unreachable server surfaces as [LlamaCppClient.Complete]'s
// own connection error instead.
func newLlamaCppClient() (*LlamaCppClient, error) {
	baseURL := os.Getenv("LLAMACPP_BASE_URL")
	if baseURL == "" {
		baseURL = llamaCppDefaultBaseURL
	}
	return &LlamaCppClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

type llamaCppCompletionRequest struct {
	Prompt   string `json:"prompt"`
	NPredict int    `json:"n_predict"`
}

type llamaCppCompletionResponse struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

// Complete implements [Client] against llama-server's own
// "/completion" endpoint.
func (c *LlamaCppClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody, err := json.Marshal(llamaCppCompletionRequest{
		Prompt:   systemPrompt + "\n\n" + userPrompt,
		NPredict: llamaCppNPredict,
	})
	if err != nil {
		return "", fmt.Errorf("llmclient: encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+llamaCppCompletionEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("llmclient: building request: %w", err)
	}
	req.Header.Set("content-type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llmclient: calling local llama.cpp server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llmclient: reading response: %w", err)
	}

	var parsed llamaCppCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("llmclient: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != "" {
		return "", fmt.Errorf("llmclient: llama.cpp error: %s", parsed.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llmclient: llama.cpp server returned status %d", resp.StatusCode)
	}
	return parsed.Content, nil
}
