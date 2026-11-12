package adversarialslm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testLlamaCppRunner(server *httptest.Server) *LlamaCppRunner {
	return &LlamaCppRunner{BaseURL: server.URL, HTTPClient: server.Client()}
}

func TestLlamaCppRunner_Attack_SuccessfulViolationFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content": "{\"outcome\": \"violation_found\", \"counterexample\": \"x = -1\"}"}`))
	}))
	defer server.Close()

	got, err := testLlamaCppRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err != nil {
		t.Fatalf("Attack returned unexpected error: %v", err)
	}
	if got.Outcome != ViolationFound || got.Counterexample != "x = -1" {
		t.Errorf("Attack = %+v, want ViolationFound with counterexample \"x = -1\"", got)
	}
}

func TestLlamaCppRunner_Attack_SuccessfulNoViolationFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content": "{\"outcome\": \"no_violation_found\", \"counterexample\": \"\"}"}`))
	}))
	defer server.Close()

	got, err := testLlamaCppRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err != nil {
		t.Fatalf("Attack returned unexpected error: %v", err)
	}
	if got.Outcome != NoViolationFound {
		t.Errorf("Attack.Outcome = %v, want NoViolationFound", got.Outcome)
	}
}

func TestLlamaCppRunner_Attack_FreeFormModelOutputIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content": "I think it might fail sometimes, hard to say"}`))
	}))
	defer server.Close()

	_, err := testLlamaCppRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err == nil {
		t.Fatal("Attack with free-form model output = nil error, want one (the grammar must still be enforced)")
	}
}

func TestLlamaCppRunner_Attack_LlamaCppErrorIsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error": "no slots available"}`))
	}))
	defer server.Close()

	_, err := testLlamaCppRunner(server).Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err == nil {
		t.Fatal("Attack with a llama.cpp error response = nil error, want one")
	}
	if !strings.Contains(err.Error(), "no slots available") {
		t.Errorf("error = %v, want it to contain llama.cpp's own error message", err)
	}
}

func TestLlamaCppRunner_Attack_ConnectionFailureIsClearError(t *testing.T) {
	runner := &LlamaCppRunner{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{}}
	_, err := runner.Attack(context.Background(), AttackInput{Assertion: "x is never negative"})
	if err == nil {
		t.Fatal("Attack against an unreachable address = nil error, want one")
	}
}

func TestLlamaCppRunner_Attack_SendsExpectedRequestShape(t *testing.T) {
	var gotPath string
	var gotBody struct {
		Prompt   string `json:"prompt"`
		Grammar  string `json:"grammar"`
		NPredict int    `json:"n_predict"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"content": "{\"outcome\": \"no_violation_found\", \"counterexample\": \"\"}"}`))
	}))
	defer server.Close()

	_, err := testLlamaCppRunner(server).Attack(context.Background(), AttackInput{
		Assertion: "the shipping fee is never negative",
		Interface: "HTTP POST /orders/shipping",
	})
	if err != nil {
		t.Fatalf("Attack returned unexpected error: %v", err)
	}

	if gotPath != llamaCppCompletionEndpoint {
		t.Errorf("request path = %q, want %q", gotPath, llamaCppCompletionEndpoint)
	}
	if gotBody.Grammar != attackOutputGrammar {
		t.Error("request Grammar does not match attackOutputGrammar")
	}
	if gotBody.NPredict <= 0 {
		t.Errorf("request NPredict = %d, want a positive value", gotBody.NPredict)
	}
	if !strings.Contains(gotBody.Prompt, "the shipping fee is never negative") {
		t.Errorf("prompt does not contain the assertion, got: %q", gotBody.Prompt)
	}
	if !strings.Contains(gotBody.Prompt, "HTTP POST /orders/shipping") {
		t.Errorf("prompt does not contain the interface, got: %q", gotBody.Prompt)
	}
}

func TestNewLlamaCppRunner_DefaultBaseURL(t *testing.T) {
	r := NewLlamaCppRunner()
	if r.BaseURL != llamaCppDefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", r.BaseURL, llamaCppDefaultBaseURL)
	}
}
