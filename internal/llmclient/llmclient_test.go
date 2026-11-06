package llmclient

import "testing"

func TestNew_UnrecognizedProviderIsError(t *testing.T) {
	_, err := New("openai")
	if err == nil {
		t.Fatal("New(\"openai\") = nil error, want one (not a supported provider yet)")
	}
}

func TestNew_AnthropicWithNoKeySetIsError(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := New("anthropic")
	if err == nil {
		t.Fatal("New(\"anthropic\") with no API key = nil error, want one")
	}
}

func TestNew_AnthropicIsCaseInsensitiveAndReadsTheKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-123")
	client, err := New("Anthropic")
	if err != nil {
		t.Fatalf("New(\"Anthropic\") returned unexpected error: %v", err)
	}
	anthropic, ok := client.(*AnthropicClient)
	if !ok {
		t.Fatalf("New(\"Anthropic\") returned %T, want *AnthropicClient", client)
	}
	if anthropic.APIKey != "test-key-123" {
		t.Errorf("AnthropicClient.APIKey = %q, want the env var's value", anthropic.APIKey)
	}
}
