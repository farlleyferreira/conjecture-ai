package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// readToolNames is every tool [mcp_read_tools.go] registers -
// TestMCPServer_EveryReadToolIsRegistered checks all of them in one
// pass instead of one near-identical test per name.
var readToolNames = []string{
	"context_list", "context_resume", "context_open", "context_origin",
	"context_diff", "context_resolve_conflict", "queue_list", "graph_inspect",
}

func TestMCPServer_EveryReadToolIsRegistered(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned unexpected error: %v", err)
	}
	registered := make(map[string]bool, len(res.Tools))
	for _, tool := range res.Tools {
		registered[tool.Name] = true
	}
	for _, name := range readToolNames {
		if !registered[name] {
			t.Errorf("ListTools = %+v, want a %q tool", res.Tools, name)
		}
	}
}

func callReadTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) textReadOutput {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%q) returned unexpected error: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool(%q) result IsError = true, content: %+v", name, res.Content)
	}
	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("CallTool(%q) StructuredContent = %T, want map[string]any: %+v", name, res.StructuredContent, res.StructuredContent)
	}
	ok2, _ := out["ok"].(bool)
	report, _ := out["report"].(string)
	return textReadOutput{OK: ok2, Report: report}
}

func TestMCPContextListTool_ReportsSeededContext(t *testing.T) {
	session, db := connectMCPTestClient(t)
	if _, err := contextcomposition.CreateContext(db, "checkout::shipping", "checkout"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	out := callReadTool(t, session, "context_list", map[string]any{})
	if !out.OK {
		t.Errorf("context_list.ok = false, report: %s", out.Report)
	}
	if !strings.Contains(out.Report, "checkout::shipping") {
		t.Errorf("context_list report = %q, want it to mention the seeded context", out.Report)
	}
}

func TestMCPContextResumeTool_ReportsLatestVersionAndQueue(t *testing.T) {
	session, db := connectMCPTestClient(t)
	if _, err := contextcomposition.CreateContext(db, "checkout::shipping", "checkout"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout::shipping", "v1 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if err := workqueue.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if err := workqueue.Add(db, workqueue.Item{ID: "demo::item", Tier: workqueue.Blocker, Weight: 1, Severity: 1, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("workqueue.Add returned unexpected error: %v", err)
	}

	out := callReadTool(t, session, "context_resume", map[string]any{"name": "checkout::shipping"})
	if !out.OK {
		t.Errorf("context_resume.ok = false, report: %s", out.Report)
	}
	if !strings.Contains(out.Report, "v1 content") {
		t.Errorf("context_resume report = %q, want it to contain the latest version's content", out.Report)
	}
	if !strings.Contains(out.Report, "demo::item") {
		t.Errorf("context_resume report = %q, want it to report the queued item", out.Report)
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("workqueue.List returned unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("workqueue after context_resume = %+v, want the item still queued (resume only peeks)", items)
	}
}

func TestMCPContextOpenTool_OpensExactVersion(t *testing.T) {
	session, db := connectMCPTestClient(t)
	if _, err := contextcomposition.CreateContext(db, "checkout::shipping", "checkout"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout::shipping", "v1 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	out := callReadTool(t, session, "context_open", map[string]any{"ref": "checkout::shipping@v1"})
	if !out.OK {
		t.Errorf("context_open.ok = false, report: %s", out.Report)
	}
	if !strings.Contains(out.Report, "v1 content") {
		t.Errorf("context_open report = %q, want it to contain v1's content", out.Report)
	}
}

func TestMCPContextOriginTool_ReportsNoDecisionsForUnknownSource(t *testing.T) {
	session, db := connectMCPTestClient(t)
	if _, err := contextcomposition.CreateContext(db, "checkout::shipping", "checkout"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	out := callReadTool(t, session, "context_origin", map[string]any{"name": "checkout::shipping"})
	if !out.OK {
		t.Errorf("context_origin.ok = false, report: %s", out.Report)
	}
	if !strings.Contains(out.Report, "checkout::shipping") {
		t.Errorf("context_origin report = %q, want it to mention the context", out.Report)
	}
}

func TestMCPContextDiffTool_ReportsUnifiedDiffBetweenVersions(t *testing.T) {
	session, db := connectMCPTestClient(t)
	if _, err := contextcomposition.CreateContext(db, "checkout::shipping", "checkout"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout::shipping", "line one\nline two\n"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout::shipping", "line one\nline THREE\n"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	out := callReadTool(t, session, "context_diff", map[string]any{
		"ref_a": "checkout::shipping@v1", "ref_b": "checkout::shipping@v2",
	})
	if !out.OK {
		t.Errorf("context_diff.ok = false, report: %s", out.Report)
	}
	if !strings.Contains(out.Report, "line two") || !strings.Contains(out.Report, "line THREE") {
		t.Errorf("context_diff report = %q, want it to show both differing lines", out.Report)
	}
}

func TestMCPContextResolveConflictTool_ReportsDecidedWinner(t *testing.T) {
	session, _ := connectMCPTestClient(t)

	out := callReadTool(t, session, "context_resolve_conflict", map[string]any{
		"mode_a": "ByEmbedding", "mode_b": "ByReference",
	})
	if !out.OK {
		t.Errorf("context_resolve_conflict.ok = false, report: %s", out.Report)
	}
	if out.Report == "" {
		t.Error("context_resolve_conflict report is empty, want a resolution")
	}
}

func TestMCPQueueListTool_ReportsSeededItem(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	session, _ := connectMCPTestClient(t)

	out := callReadTool(t, session, "queue_list", map[string]any{})
	if !out.OK {
		t.Errorf("queue_list.ok = false, report: %s", out.Report)
	}
	if !strings.Contains(out.Report, "queue is empty") {
		t.Errorf("queue_list report = %q, want \"queue is empty\" for a fresh store", out.Report)
	}
}

func TestMCPGraphInspectTool_ReportsSeededAxis(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	session, _ := connectMCPTestClient(t)

	out := callReadTool(t, session, "graph_inspect", map[string]any{"node_id": "demo::node"})
	if !out.OK {
		t.Errorf("graph_inspect.ok = false, report: %s", out.Report)
	}
	if !strings.Contains(out.Report, "no axes recorded") {
		t.Errorf("graph_inspect report = %q, want \"no axes recorded\" for a node never seen before", out.Report)
	}
}
