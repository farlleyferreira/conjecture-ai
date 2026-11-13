package main

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connectMCPTestClient connects a real client to a fresh server over
// an in-memory transport pair, so a test can call a tool through the
// actual MCP protocol machinery without a real stdio subprocess. The
// returned db is the server's own "read_evidence" store - a fresh
// temporary one (openCheckTestStore), never the real per-OS one
// [persistence.OpenStore] would otherwise open - so a test can seed it
// directly with [evidenceledger.Append]. contextcomposition's own
// schema is ensured here too, since openCheckTestStore does not: the
// real "conjecture mcp" (runMCP) ensures it before ever building the
// server, for the context_* tools (mcp_read_tools.go) this test
// harness also exercises.
func connectMCPTestClient(t *testing.T) (*mcp.ClientSession, *sql.DB) {
	t.Helper()
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	db := openCheckTestStore(t)
	if err := contextcomposition.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	server := newMCPServer(db)
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server.Connect returned unexpected error: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, db
}

func TestMCPVerifyTool_ConsistentFilePasses(t *testing.T) {
	path := writeFixture(t, fixtureCjtr)
	session, _ := connectMCPTestClient(t)

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
	session, _ := connectMCPTestClient(t)

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
	session, _ := connectMCPTestClient(t)

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

func TestMCPServer_CheckToolIsRegistered(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned unexpected error: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "check" {
			return
		}
	}
	t.Errorf("ListTools = %+v, want a \"check\" tool", res.Tools)
}

func TestMCPCheckTool_MissingFileReportsFailure(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "check",
		Arguments: map[string]any{
			"path":    "this-file-does-not-exist.cjtr",
			"image":   "irrelevant:latest",
			"command": []string{"irrelevant"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}

	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent = %T, want map[string]any: %+v", res.StructuredContent, res.StructuredContent)
	}
	if passed, _ := out["passed"].(bool); passed {
		t.Error("checkOutput.Passed = true for a missing file, want false")
	}
	if report, _ := out["report"].(string); report == "" {
		t.Error("checkOutput.Report is empty, want it to explain the failure")
	}
}

func TestMCPServer_StatusToolIsRegistered(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned unexpected error: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "status" {
			return
		}
	}
	t.Errorf("ListTools = %+v, want a \"status\" tool", res.Tools)
}

func TestMCPStatusTool_ReportsAggregatedState(t *testing.T) {
	path := writeStatusFixture(t)
	session, db := connectMCPTestClient(t)

	hypothesisID := "demo-area::demo::shipping-rule::fee-matches"
	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Hypothesis,
		ProducedBy: "conjecture check test", Attacked: "directed sampling",
		Survived: "every generated case conformed", ValidityHash: []byte("x"),
	}); err != nil {
		t.Fatalf("evidenceledger.Append returned unexpected error: %v", err)
	}

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "status",
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
	if signed, _ := out["signed"].(bool); signed {
		t.Error("status.signed = true for an unsigned fixture, want false")
	}
	entries, ok := out["entries"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("status.entries = %+v, want exactly 2 entries", out["entries"])
	}
}

func TestMCPStatusTool_MissingFileReportsError(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "status",
		Arguments: map[string]any{"path": "this-file-does-not-exist.cjtr"},
	})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	if !res.IsError {
		t.Error("CallTool result IsError = false for a missing file, want true")
	}
}

func TestMCPServer_ProjectStatusToolIsRegistered(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned unexpected error: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "project_status" {
			return
		}
	}
	t.Errorf("ListTools = %+v, want a \"project_status\" tool", res.Tools)
}

func TestMCPProjectStatusTool_AggregatesAcrossFiles(t *testing.T) {
	session, db := connectMCPTestClient(t)

	repoRoot := t.TempDir()
	writeProjectStatusFixture(t, repoRoot, "demo-area", "demo", statusFixture)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned unexpected error: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	hypothesisID := "demo-area::demo::shipping-rule::fee-matches"
	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Hypothesis,
		ProducedBy: "test", ValidityHash: []byte("x"),
	}); err != nil {
		t.Fatalf("evidenceledger.Append returned unexpected error: %v", err)
	}

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "project_status", Arguments: map[string]any{}})
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
	if out["files"] != float64(1) {
		t.Errorf("project_status.files = %v, want 1", out["files"])
	}
}

func TestMCPServer_ReadEvidenceToolIsRegistered(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned unexpected error: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "read_evidence" {
			return
		}
	}
	t.Errorf("ListTools = %+v, want a \"read_evidence\" tool", res.Tools)
}

func TestMCPReadEvidenceTool_NoRecordedEvidenceIsEmptyHistory(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_evidence",
		Arguments: map[string]any{"hypothesis_id": "demo-area::demo::some-rule::never-checked"},
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
	history, _ := out["history"].([]any)
	if len(history) != 0 {
		t.Errorf("history = %+v, want empty for a hypothesis with no recorded evidence", history)
	}
}

func TestMCPReadEvidenceTool_ReturnsRecordedHistory(t *testing.T) {
	session, db := connectMCPTestClient(t)
	hypothesisID := "demo-area::demo::some-rule::seeded"

	if _, err := evidenceledger.Append(db, evidenceledger.Evidence{
		HypothesisID: hypothesisID, Stage: evidenceledger.Hypothesis,
		ProducedBy: "conjecture check test", Attacked: "directed sampling",
		Survived: "every generated case conformed", ValidityHash: []byte("x"),
	}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_evidence",
		Arguments: map[string]any{"hypothesis_id": hypothesisID},
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
	history, ok := out["history"].([]any)
	if !ok || len(history) != 1 {
		t.Fatalf("history = %+v, want exactly one record", out["history"])
	}
	record, ok := history[0].(map[string]any)
	if !ok {
		t.Fatalf("history[0] = %T, want map[string]any", history[0])
	}
	if record["attacked"] != "directed sampling" {
		t.Errorf("record[\"attacked\"] = %v, want %q", record["attacked"], "directed sampling")
	}
	if record["stage"] != evidenceledger.Hypothesis.String() {
		t.Errorf("record[\"stage\"] = %v, want %q", record["stage"], evidenceledger.Hypothesis.String())
	}
}
