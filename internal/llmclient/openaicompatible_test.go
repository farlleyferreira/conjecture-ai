package llmclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testOpenAICompatClient(t *testing.T, server *httptest.Server) *OpenAICompatibleClient {
	t.Helper()
	return &OpenAICompatibleClient{
		Name:       "TestProvider",
		BaseURL:    server.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxTokens:  openAICompatDefaultMaxTokens,
		HTTPClient: server.Client(),
	}
}

func TestOpenAICompatComplete_SuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices": [{"message": {"role": "assistant", "content": "hello there"}}]}`))
	}))
	defer server.Close()

	got, err := testOpenAICompatClient(t, server).Complete(context.Background(), "be helpful", "say hi")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}
	if got != "hello there" {
		t.Errorf("Complete = %q, want %q", got, "hello there")
	}
}

func TestOpenAICompatComplete_APIErrorNamesTheProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"type": "invalid_request_error", "message": "model not found"}}`))
	}))
	defer server.Close()

	_, err := testOpenAICompatClient(t, server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with an API error response = nil error, want one")
	}
	if !strings.Contains(err.Error(), "model not found") || !strings.Contains(err.Error(), "TestProvider") {
		t.Errorf("error = %v, want it to contain the provider's name and its own error message", err)
	}
}

func TestOpenAICompatComplete_NonOKStatusWithoutErrorBodyIsStillAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	_, err := testOpenAICompatClient(t, server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with a 500 and no error body = nil error, want one")
	}
}

func TestOpenAICompatComplete_MalformedResponseBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	_, err := testOpenAICompatClient(t, server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with a non-JSON response body = nil error, want one")
	}
}

func TestOpenAICompatComplete_NoChoicesIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices": []}`))
	}))
	defer server.Close()

	_, err := testOpenAICompatClient(t, server).Complete(context.Background(), "", "")
	if err == nil {
		t.Fatal("Complete with no choices = nil error, want one")
	}
}

func TestOpenAICompatComplete_SendsExpectedRequestShapeAndHeaders(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody struct {
		Model    string                `json:"model"`
		Messages []openAICompatMessage `json:"messages"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("authorization")
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"choices": [{"message": {"role": "assistant", "content": "ok"}}]}`))
	}))
	defer server.Close()

	client := testOpenAICompatClient(t, server)
	_, err := client.Complete(context.Background(), "system instructions", "user ask")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}

	if gotAuth != "Bearer test-key" {
		t.Errorf("authorization header = %q, want %q", gotAuth, "Bearer test-key")
	}
	if gotPath != "/chat/completions" {
		t.Errorf("request path = %q, want %q", gotPath, "/chat/completions")
	}
	if gotBody.Model != client.Model {
		t.Errorf("request model = %q, want %q", gotBody.Model, client.Model)
	}
	if len(gotBody.Messages) != 2 || gotBody.Messages[0].Role != "system" || gotBody.Messages[0].Content != "system instructions" {
		t.Errorf("request messages = %+v, want a system message first", gotBody.Messages)
	}
	if gotBody.Messages[1].Role != "user" || gotBody.Messages[1].Content != "user ask" {
		t.Errorf("request messages[1] = %+v, want the user message", gotBody.Messages[1])
	}
}

func TestOpenAICompatComplete_OmitsSystemMessageWhenEmpty(t *testing.T) {
	var gotBody struct {
		Messages []openAICompatMessage `json:"messages"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"choices": [{"message": {"role": "assistant", "content": "ok"}}]}`))
	}))
	defer server.Close()

	_, err := testOpenAICompatClient(t, server).Complete(context.Background(), "", "user ask")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}
	if len(gotBody.Messages) != 1 || gotBody.Messages[0].Role != "user" {
		t.Errorf("request messages = %+v, want only the user message", gotBody.Messages)
	}
}

func TestOpenAICompatComplete_BaseURLWithTrailingSlashStillResolvesCorrectly(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"choices": [{"message": {"role": "assistant", "content": "ok"}}]}`))
	}))
	defer server.Close()

	client := testOpenAICompatClient(t, server)
	client.BaseURL += "/"
	_, err := client.Complete(context.Background(), "", "ask")
	if err != nil {
		t.Fatalf("Complete returned unexpected error: %v", err)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("request path = %q, want %q (no double slash)", gotPath, "/chat/completions")
	}
}
