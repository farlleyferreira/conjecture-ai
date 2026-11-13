package llmclient

import "testing"

func TestNew_UnrecognizedProviderIsError(t *testing.T) {
	_, err := New("nonexistent-provider")
	if err == nil {
		t.Fatal("New(\"nonexistent-provider\") = nil error, want one (not a supported provider)")
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

func TestNew_DeepSeekWithNoKeySetIsError(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "")
	_, err := New("deepseek")
	if err == nil {
		t.Fatal("New(\"deepseek\") with no API key = nil error, want one")
	}
}

func TestNew_DeepSeekIsCaseInsensitiveAndReadsTheKey(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key-456")
	client, err := New("DeepSeek")
	if err != nil {
		t.Fatalf("New(\"DeepSeek\") returned unexpected error: %v", err)
	}
	deepseek, ok := client.(*DeepSeekClient)
	if !ok {
		t.Fatalf("New(\"DeepSeek\") returned %T, want *DeepSeekClient", client)
	}
	if deepseek.APIKey != "test-key-456" {
		t.Errorf("DeepSeekClient.APIKey = %q, want the env var's value", deepseek.APIKey)
	}
}

// openAICompatProviderCases covers every [OpenAICompatibleClient]
// preset [New] wires up: the provider name New takes, the env vars
// that construct it (set together to build it successfully), and the
// exact BaseURL that combination must produce - the one thing each
// preset exists to get right.
var openAICompatProviderCases = []struct {
	provider string
	env      map[string]string
	wantURL  string
}{
	{"openai", map[string]string{"OPENAI_API_KEY": "k", "OPENAI_MODEL": "gpt-x"}, "https://api.openai.com/v1"},
	{"gemini", map[string]string{"GEMINI_API_KEY": "k", "GEMINI_MODEL": "gemini-x"}, "https://generativelanguage.googleapis.com/v1beta/openai"},
	{"grok", map[string]string{"GROK_API_KEY": "k", "GROK_MODEL": "grok-x"}, "https://api.x.ai/v1"},
	{"llama-api", map[string]string{"LLAMA_API_KEY": "k", "LLAMA_MODEL": "llama-x"}, "https://api.llama.com/compat/v1"},
	{"mistral", map[string]string{"MISTRAL_API_KEY": "k", "MISTRAL_MODEL": "mistral-x"}, "https://api.mistral.ai/v1"},
	{"cohere", map[string]string{"COHERE_API_KEY": "k", "COHERE_MODEL": "command-x"}, "https://api.cohere.ai/compatibility/v1"},
	{"bedrock", map[string]string{"BEDROCK_API_KEY": "k", "BEDROCK_MODEL": "m", "BEDROCK_REGION": "us-east-1"}, "https://bedrock-runtime.us-east-1.amazonaws.com/openai/v1"},
	{"azure-openai", map[string]string{"AZURE_OPENAI_API_KEY": "k", "AZURE_OPENAI_MODEL": "m", "AZURE_OPENAI_RESOURCE": "my-resource"}, "https://my-resource.openai.azure.com/openai/v1"},
	{"dashscope", map[string]string{"DASHSCOPE_API_KEY": "k", "DASHSCOPE_MODEL": "qwen-x", "DASHSCOPE_REGION": "intl"}, "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
	{"dashscope", map[string]string{"DASHSCOPE_API_KEY": "k", "DASHSCOPE_MODEL": "qwen-x", "DASHSCOPE_REGION": "cn"}, "https://dashscope.aliyuncs.com/compatible-mode/v1"},
}

func TestNew_OpenAICompatibleProvidersBuildTheRightClient(t *testing.T) {
	for _, tc := range openAICompatProviderCases {
		t.Run(tc.provider+"/"+tc.wantURL, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			client, err := New(tc.provider)
			if err != nil {
				t.Fatalf("New(%q) returned unexpected error: %v", tc.provider, err)
			}
			got, ok := client.(*OpenAICompatibleClient)
			if !ok {
				t.Fatalf("New(%q) returned %T, want *OpenAICompatibleClient", tc.provider, client)
			}
			if got.BaseURL != tc.wantURL {
				t.Errorf("New(%q).BaseURL = %q, want %q", tc.provider, got.BaseURL, tc.wantURL)
			}
		})
	}
}

func TestNew_OpenAICompatibleProvidersRequireEveryEnvVar(t *testing.T) {
	for _, tc := range openAICompatProviderCases {
		for missing := range tc.env {
			t.Run(tc.provider+"/missing-"+missing, func(t *testing.T) {
				for k, v := range tc.env {
					if k == missing {
						t.Setenv(k, "")
						continue
					}
					t.Setenv(k, v)
				}
				if _, err := New(tc.provider); err == nil {
					t.Errorf("New(%q) with %s unset = nil error, want one", tc.provider, missing)
				}
			})
		}
	}
}

func TestNew_DashScopeUnrecognizedRegionIsError(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "k")
	t.Setenv("DASHSCOPE_MODEL", "qwen-x")
	t.Setenv("DASHSCOPE_REGION", "eu")
	if _, err := New("dashscope"); err == nil {
		t.Fatal("New(\"dashscope\") with DASHSCOPE_REGION=\"eu\" = nil error, want one")
	}
}

func TestNew_LlamaCppAndLlamaAPIAreDistinctProviders(t *testing.T) {
	t.Setenv("LLAMA_API_KEY", "k")
	t.Setenv("LLAMA_MODEL", "llama-x")
	client, err := New("llama-api")
	if err != nil {
		t.Fatalf("New(\"llama-api\") returned unexpected error: %v", err)
	}
	if _, ok := client.(*OpenAICompatibleClient); !ok {
		t.Fatalf("New(\"llama-api\") returned %T, want *OpenAICompatibleClient", client)
	}

	local, err := New("llamacpp")
	if err != nil {
		t.Fatalf("New(\"llamacpp\") returned unexpected error: %v", err)
	}
	if _, ok := local.(*LlamaCppClient); !ok {
		t.Fatalf("New(\"llamacpp\") returned %T, want *LlamaCppClient (distinct from the remote llama-api preset)", local)
	}
}
