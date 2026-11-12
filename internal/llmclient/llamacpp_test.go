package llmclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testLlamaCppClient(server *httptest.Server) *LlamaCppClient {
	return &LlamaCppClient{BaseURL: server.URL, HTTPClient: server.Client()}
}

func TestLlamaCppComplete_SuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content": "print('hi')"}`))
	}))
	defer server.Close()

	got, err := testLlamaCppClient(server).Complete(context.Background(), "be helpful", "write code")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}
	if got != "print('hi')" {
		t.Errorf("Complete = %q, want %q", got, "print('hi')")
	}
}

func TestLlamaCppComplete_ConcatenatesSystemAndUserPromptIntoOnePrompt(t *testing.T) {
	var gotBody struct {
		Prompt   string `json:"prompt"`
		NPredict int    `json:"n_predict"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"content": "ok"}`))
	}))
	defer server.Close()

	_, err := testLlamaCppClient(server).Complete(context.Background(), "system instructions", "user ask")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}
	if !strings.Contains(gotBody.Prompt, "system instructions") || !strings.Contains(gotBody.Prompt, "user ask") {
		t.Errorf("request prompt = %q, want it to contain both the system and user prompt", gotBody.Prompt)
	}
	if gotBody.NPredict != llamaCppNPredict {
		t.Errorf("request n_predict = %d, want %d", gotBody.NPredict, llamaCppNPredict)
	}
}

func TestLlamaCppComplete_ServerErrorFieldIsReturnedAsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error": "model not loaded"}`))
	}))
	defer server.Close()

	_, err := testLlamaCppClient(server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with an error field in the response = nil error, want one")
	}
	if !strings.Contains(err.Error(), "model not loaded") {
		t.Errorf("error = %v, want it to contain the server's own error message", err)
	}
}

func TestLlamaCppComplete_NonOKStatusWithoutErrorFieldIsStillAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	_, err := testLlamaCppClient(server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with a 500 and no error field = nil error, want one")
	}
}

func TestLlamaCppComplete_MalformedResponseBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	_, err := testLlamaCppClient(server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with a non-JSON response body = nil error, want one")
	}
}

func TestLlamaCppComplete_UnreachableServerIsError(t *testing.T) {
	client := &LlamaCppClient{BaseURL: "http://127.0.0.1:1", HTTPClient: http.DefaultClient}
	_, err := client.Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete against an unreachable server = nil error, want one")
	}
}

func TestNewLlamaCppClient_DefaultBaseURL(t *testing.T) {
	t.Setenv("LLAMACPP_BASE_URL", "")
	client, err := newLlamaCppClient()
	if err != nil {
		t.Fatalf("newLlamaCppClient returned unexpected error: %v", err)
	}
	if client.BaseURL != llamaCppDefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL, llamaCppDefaultBaseURL)
	}
}

func TestNewLlamaCppClient_EnvOverride(t *testing.T) {
	t.Setenv("LLAMACPP_BASE_URL", "http://example.invalid:9999")
	client, err := newLlamaCppClient()
	if err != nil {
		t.Fatalf("newLlamaCppClient returned unexpected error: %v", err)
	}
	if client.BaseURL != "http://example.invalid:9999" {
		t.Errorf("BaseURL = %q, want the env override", client.BaseURL)
	}
}
