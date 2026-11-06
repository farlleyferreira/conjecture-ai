package adversarialslm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testRunner(server *httptest.Server) *OllamaRunner {
	return &OllamaRunner{
		BaseURL:    server.URL,
		Model:      "qwen2.5:0.5b",
		HTTPClient: server.Client(),
	}
}

func TestOllamaRunner_Attack_SuccessfulViolationFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response": "{\"outcome\": \"violation_found\", \"counterexample\": \"x = -1\"}"}`))
	}))
	defer server.Close()

	got, err := testRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err != nil {
		t.Fatalf("Attack returned unexpected error: %v", err)
	}
	if got.Outcome != ViolationFound || got.Counterexample != "x = -1" {
		t.Errorf("Attack = %+v, want ViolationFound with counterexample \"x = -1\"", got)
	}
}

func TestOllamaRunner_Attack_SuccessfulNoViolationFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response": "{\"outcome\": \"no_violation_found\"}"}`))
	}))
	defer server.Close()

	got, err := testRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err != nil {
		t.Fatalf("Attack returned unexpected error: %v", err)
	}
	if got.Outcome != NoViolationFound {
		t.Errorf("Attack.Outcome = %v, want NoViolationFound", got.Outcome)
	}
}

func TestOllamaRunner_Attack_FreeFormModelOutputIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response": "I think it might fail sometimes, hard to say"}`))
	}))
	defer server.Close()

	_, err := testRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err == nil {
		t.Fatal("Attack with free-form model output = nil error, want one (the grammar must still be enforced)")
	}
}

func TestOllamaRunner_Attack_OllamaErrorIsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error": "model not found, try pulling it first"}`))
	}))
	defer server.Close()

	_, err := testRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err == nil {
		t.Fatal("Attack with an Ollama error response = nil error, want one")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("error = %v, want it to contain Ollama's own error message", err)
	}
}

func TestOllamaRunner_Attack_ConnectionFailureIsClearError(t *testing.T) {
	runner := &OllamaRunner{BaseURL: "http://127.0.0.1:1", Model: "x", HTTPClient: &http.Client{}}
	_, err := runner.Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err == nil {
		t.Fatal("Attack against an unreachable address = nil error, want one")
	}
}

func TestOllamaRunner_Attack_SendsExpectedRequestShape(t *testing.T) {
	var gotPath string
	var gotBody struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
		Format string `json:"format"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"response": "{\"outcome\": \"no_violation_found\"}"}`))
	}))
	defer server.Close()

	runner := testRunner(server)
	_, err := runner.Attack(context.Background(), AttackInput{
		Assertion:    "the shipping fee is never negative",
		Interface:    "HTTP POST /orders/shipping",
		PriorResults: []AttackOutput{{Outcome: NoViolationFound}},
	})
	if err != nil {
		t.Fatalf("Attack returned unexpected error: %v", err)
	}

	if gotPath != ollamaGenerateEndpoint {
		t.Errorf("request path = %q, want %q", gotPath, ollamaGenerateEndpoint)
	}
	if gotBody.Model != runner.Model {
		t.Errorf("request model = %q, want %q", gotBody.Model, runner.Model)
	}
	if gotBody.Stream {
		t.Error("request Stream = true, want false")
	}
	if gotBody.Format != "json" {
		t.Errorf("request Format = %q, want %q", gotBody.Format, "json")
	}
	if !strings.Contains(gotBody.Prompt, "the shipping fee is never negative") {
		t.Errorf("prompt does not contain the assertion, got: %q", gotBody.Prompt)
	}
	if !strings.Contains(gotBody.Prompt, "HTTP POST /orders/shipping") {
		t.Errorf("prompt does not contain the interface, got: %q", gotBody.Prompt)
	}
}

func TestBuildAttackPrompt_OmitsEmptyInterfaceAndPriorResults(t *testing.T) {
	prompt := buildAttackPrompt(AttackInput{Assertion: "x is never negative"})
	if strings.Contains(prompt, "Interface:") {
		t.Errorf("prompt mentions \"Interface:\" with none given, got: %q", prompt)
	}
	if strings.Contains(prompt, "Prior attack results:") {
		t.Errorf("prompt mentions prior results with none given, got: %q", prompt)
	}
}
