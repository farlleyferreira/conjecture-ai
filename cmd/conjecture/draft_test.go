package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeLLMClient struct {
	response string
	err      error
}

func (f *fakeLLMClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return f.response, f.err
}

func TestRunDraft_NoArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDraft(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runDraft with no arguments: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunDraft_MissingAsk(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDraft([]string{"shipping.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runDraft with a path but no ask: exit code = %d, want 1", code)
	}
}

func TestRunDraft_UnrecognizedProvider(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDraft([]string{"--provider", "nope", "shipping.cjtr", "free shipping over R$100"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runDraft with an unrecognized provider: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "nope") {
		t.Errorf("stderr should mention the unrecognized provider, got %q", stderr.String())
	}
}

func TestDraftContent_ReturnsPlainOutputUnchanged(t *testing.T) {
	client := &fakeLLMClient{response: "conjecture: 1.0.0\n\nphenomenon: x\n"}
	got, err := draftContent(context.Background(), client, "draft something")
	if err != nil {
		t.Fatalf("draftContent returned unexpected error: %v", err)
	}
	if string(got) != client.response {
		t.Errorf("draftContent = %q, want %q", got, client.response)
	}
}

func TestDraftContent_StripsMarkdownCodeFence(t *testing.T) {
	client := &fakeLLMClient{response: "```yaml\nconjecture: 1.0.0\n\nphenomenon: x\n```"}
	got, err := draftContent(context.Background(), client, "draft something")
	if err != nil {
		t.Fatalf("draftContent returned unexpected error: %v", err)
	}
	if strings.Contains(string(got), "```") {
		t.Errorf("draftContent left a code fence in, got:\n%s", got)
	}
	if !strings.Contains(string(got), "phenomenon: x") {
		t.Errorf("draftContent stripped real content along with the fence, got:\n%s", got)
	}
}

func TestDraftContent_StripsSignedBlockRegardlessOfPromptCompliance(t *testing.T) {
	client := &fakeLLMClient{response: "conjecture: 1.0.0\n\nphenomenon: x\n\nsigned:\n  user: an-llm\n  confirmed: true\n"}
	got, err := draftContent(context.Background(), client, "draft something")
	if err != nil {
		t.Fatalf("draftContent returned unexpected error: %v", err)
	}
	if strings.Contains(string(got), "signed:") {
		t.Errorf("draftContent left a \"signed\" block in the model's output, got:\n%s", got)
	}
}

func TestDraftContent_PropagatesClientError(t *testing.T) {
	client := &fakeLLMClient{err: errors.New("simulated client failure")}
	_, err := draftContent(context.Background(), client, "draft something")
	if err == nil {
		t.Fatal("draftContent with a failing client = nil error, want one")
	}
}

func TestStripCodeFence_StripsWhenFenceSpansWholeText(t *testing.T) {
	got := stripCodeFence("```yaml\nfoo: bar\n```")
	if got != "foo: bar" {
		t.Errorf("stripCodeFence = %q, want %q", got, "foo: bar")
	}
}

func TestStripCodeFence_LeavesUnfencedTextAlone(t *testing.T) {
	text := "foo: bar\n"
	if got := stripCodeFence(text); got != text {
		t.Errorf("stripCodeFence changed unfenced text: got %q, want %q", got, text)
	}
}

func TestStripCodeFence_LeavesPartialBackticksAlone(t *testing.T) {
	// A real description that happens to contain a triple-backtick
	// somewhere in the middle must not be treated as a fence.
	text := "description: \"see the ``` marker in docs\"\nfoo: bar"
	if got := stripCodeFence(text); got != text {
		t.Errorf("stripCodeFence changed text with backticks mid-content: got %q, want %q", got, text)
	}
}
