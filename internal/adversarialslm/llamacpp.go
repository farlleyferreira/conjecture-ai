package adversarialslm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	llamaCppDefaultBaseURL     = "http://localhost:8080"
	llamaCppCompletionEndpoint = "/completion"
)

// attackOutputGrammar is a GBNF grammar (llama.cpp's own grammar-
// constrained decoding format) for AttackOutput's exact JSON shape -
// a more literal fit for "Grammar-constrained typed output... so
// results are machine-parseable without free-form text
// interpretation" than Ollama's own "format: json" hint ([OllamaRunner]):
// llama.cpp enforces this at the token-sampling level itself, not
// just as a request to the model to try and comply.
//
// "counterexample" is always required by this grammar, even though
// [attackOutputSchema] only requires "outcome" - an empty string value
// satisfies both, and a context-free grammar has no clean way to make
// one field's presence conditional on another's value, so this does
// not attempt that.
const attackOutputGrammar = `root ::= "{" "\"outcome\":" ws outcome "," ws "\"counterexample\":" ws string "}"
outcome ::= "\"violation_found\"" | "\"no_violation_found\""
string ::= "\"" ( [^"\\] | "\\" . )* "\""
ws ::= " "?
`

// LlamaCppRunner attacks or audits a hypothesis by calling a locally
// running llama.cpp server (llama-server) - "Local execution only":
// llama-server's own HTTP API is reached over loopback only (BaseURL
// defaults to http://localhost:8080, llama-server's own standard
// default), never a remote host.
//
// Unlike [OllamaRunner], this needs no Model field: llama-server
// already has exactly one model loaded (whichever GGUF file it was
// started with), so there is nothing for a request to select between.
//
// This package does not install llama.cpp, build llama-server, or
// download a GGUF model - [LlamaCppRunner.Attack] simply fails with a
// clear connection error if it can't reach BaseURL, the same as
// [OllamaRunner] already does for Ollama.
type LlamaCppRunner struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewLlamaCppRunner returns a LlamaCppRunner with BaseURL set to
// llama-server's own standard local address.
func NewLlamaCppRunner() *LlamaCppRunner {
	return &LlamaCppRunner{
		BaseURL:    llamaCppDefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type llamaCppCompletionRequest struct {
	Prompt   string `json:"prompt"`
	Grammar  string `json:"grammar"`
	NPredict int    `json:"n_predict"`
}

type llamaCppCompletionResponse struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

// Attack sends input to the configured local llama-server and parses
// its response as a grammar-conformant [AttackOutput] via
// [ParseAttackOutput] - the same strict rejection that applies no
// matter how the output was produced; [attackOutputGrammar] makes a
// malformed response unlikely, but it is not trusted to guarantee the
// grammar on its own, the same posture [OllamaRunner.Attack] already
// takes toward Ollama's own "format: json" hint.
func (r *LlamaCppRunner) Attack(ctx context.Context, input AttackInput) (AttackOutput, error) {
	reqBody, err := json.Marshal(llamaCppCompletionRequest{
		Prompt:   buildAttackPrompt(input),
		Grammar:  attackOutputGrammar,
		NPredict: 256,
	})
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: encoding llama.cpp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+llamaCppCompletionEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: building llama.cpp request: %w", err)
	}
	req.Header.Set("content-type", "application/json")

	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: calling local llama.cpp server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: reading llama.cpp response: %w", err)
	}

	var parsed llamaCppCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return AttackOutput{}, fmt.Errorf("adversarialslm: decoding llama.cpp response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != "" {
		return AttackOutput{}, fmt.Errorf("adversarialslm: llama.cpp error: %s", parsed.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return AttackOutput{}, fmt.Errorf("adversarialslm: llama.cpp returned status %d", resp.StatusCode)
	}

	return ParseAttackOutput([]byte(parsed.Content))
}
