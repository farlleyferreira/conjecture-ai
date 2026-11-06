package adversarialslm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ollamaDefaultBaseURL   = "http://localhost:11434"
	ollamaGenerateEndpoint = "/api/generate"
)

// OllamaRunner attacks or audits a hypothesis by calling a locally
// running Ollama server - "Local execution only": Ollama's own API is
// reached over loopback only (BaseURL defaults to
// http://localhost:11434), never a remote host, so the attack/audit
// function itself makes no network call outside this machine, even
// though pulling a model into Ollama in the first place did (a one-
// time setup step, not something [Attack] does).
//
// This package does not install Ollama, download a model, or check
// Ollama is even running - [OllamaRunner.Attack] simply fails with a
// clear connection error if it can't reach BaseURL, the same as any
// other unavailable local service.
type OllamaRunner struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// NewOllamaRunner returns an OllamaRunner for model, with BaseURL set
// to Ollama's own standard local address.
func NewOllamaRunner(model string) *OllamaRunner {
	return &OllamaRunner{
		BaseURL:    ollamaDefaultBaseURL,
		Model:      model,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// Attack sends input to the configured local model and parses its
// response as a grammar-conformant [AttackOutput] via
// [ParseAttackOutput] - the same strict rejection that applies no
// matter how the output was produced, model output included; Ollama's
// "format: json" option only asks the model to try, it is not trusted
// to guarantee the grammar on its own.
func (r *OllamaRunner) Attack(ctx context.Context, input AttackInput) (AttackOutput, error) {
	reqBody, err := json.Marshal(ollamaGenerateRequest{
		Model:  r.Model,
		Prompt: buildAttackPrompt(input),
		Stream: false,
		Format: "json",
	})
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: encoding Ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+ollamaGenerateEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: building Ollama request: %w", err)
	}
	req.Header.Set("content-type", "application/json")

	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: calling local Ollama server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: reading Ollama response: %w", err)
	}

	var parsed ollamaGenerateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: decoding Ollama response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != "" {
		return AttackOutput{}, fmt.Errorf("adversarialslm: Ollama error: %s", parsed.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return AttackOutput{}, fmt.Errorf("adversarialslm: Ollama returned status %d", resp.StatusCode)
	}

	return ParseAttackOutput([]byte(parsed.Response))
}

// buildAttackPrompt renders input into the text sent to the model -
// "Blind input": only input's own fields (Assertion, Interface,
// PriorResults), never a reasoning trace or chat history, since
// [AttackInput] has no field for either to come from.
func buildAttackPrompt(input AttackInput) string {
	var b strings.Builder
	b.WriteString("You are an adversarial auditor. Try to find a concrete case where the following assertion is false.\n\n")
	fmt.Fprintf(&b, "Assertion: %s\n", input.Assertion)
	if input.Interface != "" {
		fmt.Fprintf(&b, "Interface: %s\n", input.Interface)
	}
	if len(input.PriorResults) > 0 {
		b.WriteString("Prior attack results:\n")
		for _, prior := range input.PriorResults {
			fmt.Fprintf(&b, "- %s", prior.Outcome)
			if prior.Counterexample != "" {
				fmt.Fprintf(&b, " (counterexample: %s)", prior.Counterexample)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\nRespond with ONLY a JSON object matching this exact shape: {\"outcome\": \"violation_found\" or \"no_violation_found\", \"counterexample\": \"<only if violation_found>\"}. No other text.\n")
	return b.String()
}
