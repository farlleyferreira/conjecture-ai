package llmclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(t *testing.T, server *httptest.Server) *AnthropicClient {
	t.Helper()
	return &AnthropicClient{
		APIKey:     "test-key",
		Model:      "claude-haiku-5-5",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}
}

func TestComplete_SuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "hello there"}]}`))
	}))
	defer server.Close()

	got, err := testClient(t, server).Complete(context.Background(), "be helpful", "say hi")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}
	if got != "hello there" {
		t.Errorf("Complete = %q, want %q", got, "hello there")
	}
}

func TestComplete_ConcatenatesMultipleTextBlocksAndSkipsOtherTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content": [
			{"type": "text", "text": "part one "},
			{"type": "tool_use", "text": "should be skipped"},
			{"type": "text", "text": "part two"}
		]}`))
	}))
	defer server.Close()

	got, err := testClient(t, server).Complete(context.Background(), "", "")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}
	if got != "part one part two" {
		t.Errorf("Complete = %q, want %q", got, "part one part two")
	}
}

func TestComplete_APIErrorIsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"type": "invalid_request_error", "message": "model not found"}}`))
	}))
	defer server.Close()

	_, err := testClient(t, server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with an API error response = nil error, want one")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("error = %v, want it to contain the API's own error message", err)
	}
}

func TestComplete_NonOKStatusWithoutErrorBodyIsStillAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	_, err := testClient(t, server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with a 500 and no error body = nil error, want one")
	}
}

func TestComplete_MalformedResponseBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	_, err := testClient(t, server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with a non-JSON response body = nil error, want one")
	}
}

func TestComplete_SendsExpectedRequestShapeAndHeaders(t *testing.T) {
	var gotAPIKey, gotVersion string
	var gotBody struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		System    string `json:"system"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := testClient(t, server)
	_, err := client.Complete(context.Background(), "system instructions", "user ask")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}

	if gotAPIKey != "test-key" {
		t.Errorf("x-api-key header = %q, want %q", gotAPIKey, "test-key")
	}
	if gotVersion != anthropicAPIVersion {
		t.Errorf("anthropic-version header = %q, want %q", gotVersion, anthropicAPIVersion)
	}
	if gotBody.Model != client.Model {
		t.Errorf("request model = %q, want %q", gotBody.Model, client.Model)
	}
	if gotBody.System != "system instructions" {
		t.Errorf("request system = %q, want %q", gotBody.System, "system instructions")
	}
	if len(gotBody.Messages) != 1 || gotBody.Messages[0].Content != "user ask" || gotBody.Messages[0].Role != "user" {
		t.Errorf("request messages = %+v, want one user message with content %q", gotBody.Messages, "user ask")
	}
}
