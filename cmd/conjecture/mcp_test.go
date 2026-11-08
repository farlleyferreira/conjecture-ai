package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connectMCPTestClient connects a real client to a fresh server over
// an in-memory transport pair, so a test can call a tool through the
// actual MCP protocol machinery without a real stdio subprocess.
func connectMCPTestClient(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	server := newMCPServer()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server.Connect returned unexpected error: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestMCPVerifyTool_ConsistentFilePasses(t *testing.T) {
	path := writeFixture(t, fixtureCjtr)
	session := connectMCPTestClient(t)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "verify",
		Arguments: map[string]any{"path": path},
	})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("CallTool result IsError = true, content: %+v", res.Content)
	}

	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent = %T, want map[string]any: %+v", res.StructuredContent, res.StructuredContent)
	}
	if passed, _ := out["passed"].(bool); !passed {
		t.Errorf("verifyOutput.Passed = %v, want true; report: %v", out["passed"], out["report"])
	}
}

func TestMCPVerifyTool_MissingFileReportsFailure(t *testing.T) {
	session := connectMCPTestClient(t)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "verify",
		Arguments: map[string]any{"path": "this-file-does-not-exist.cjtr"},
	})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}

	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent = %T, want map[string]any: %+v", res.StructuredContent, res.StructuredContent)
	}
	if passed, _ := out["passed"].(bool); passed {
		t.Error("verifyOutput.Passed = true for a missing file, want false")
	}
	if report, _ := out["report"].(string); report == "" {
		t.Error("verifyOutput.Report is empty, want it to explain the failure")
	}
}

func TestMCPServer_VerifyToolIsRegistered(t *testing.T) {
	session := connectMCPTestClient(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned unexpected error: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "verify" {
			return
		}
	}
	t.Errorf("ListTools = %+v, want a \"verify\" tool", res.Tools)
}
