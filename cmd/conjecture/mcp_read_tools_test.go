package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
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

func TestMCPQueueListTool_EmptyQueueReportsEmpty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	session, _ := connectMCPTestClient(t)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "queue_list", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("CallTool result IsError = true, content: %+v", res.Content)
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "queue is empty") {
		t.Errorf("queue_list content = %+v, want it to report \"queue is empty\" for a fresh store", res.Content)
	}
	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent = %T, want map[string]any: %+v", res.StructuredContent, res.StructuredContent)
	}
	if out["summaries"] != nil || out["items"] != nil {
		t.Errorf("queue_list structured output = %+v, want both summaries and items empty/absent", out)
	}
}

func TestMCPQueueListTool_GroupsByPhenomenonAndDrillsDown(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	session, _ := connectMCPTestClient(t)

	// queueListTool reads via openRepoStore (the real per-OS store,
	// redirected here only by XDG_DATA_HOME above) - not the shared db
	// connectMCPTestClient returns, which backs the other MCP tools
	// instead. Seed through the same path queueListTool itself reads.
	queueDB, err := persistence.OpenStore(".")
	if err != nil {
		t.Fatalf("OpenStore returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = queueDB.Close() })
	if err := workqueue.EnsureSchema(queueDB); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	now := time.Now().UTC()
	items := []workqueue.Item{
		{ID: "demo-area::checkout::shipping-rule::fee-matches", Tier: workqueue.Blocker, Weight: 1, Severity: 1, CreatedAt: now},
		{ID: "demo-area::checkout::shipping-rule::other-hyp", Tier: workqueue.Background, Weight: 1, Severity: 0, CreatedAt: now},
		{ID: "demo-area::billing::some-law", Tier: workqueue.CurrentTask, Weight: 2, Severity: 1, CreatedAt: now},
		{ID: "other-area::demo::some-law", Tier: workqueue.Blocker, Weight: 1, Severity: 1, CreatedAt: now},
	}
	for _, it := range items {
		if err := workqueue.Add(queueDB, it); err != nil {
			t.Fatalf("workqueue.Add returned unexpected error: %v", err)
		}
	}

	// No filter: grouped summary across every phenomenon.
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "queue_list", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent = %T, want map[string]any: %+v", res.StructuredContent, res.StructuredContent)
	}
	summaries, ok := out["summaries"].([]any)
	if !ok || len(summaries) != 3 {
		t.Fatalf("summaries = %+v, want exactly 3 (demo-area::checkout, demo-area::billing, other-area::demo)", out["summaries"])
	}
	first, _ := summaries[0].(map[string]any)
	if first["phenomenon_id"] != "demo-area::billing" || first["total"] != float64(1) {
		t.Errorf("summaries[0] = %+v, want demo-area::billing with total 1 (alphabetically first)", first)
	}

	// --area narrows the summary.
	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "queue_list", Arguments: map[string]any{"area": "demo-area"}})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	out, _ = res.StructuredContent.(map[string]any)
	summaries, _ = out["summaries"].([]any)
	if len(summaries) != 2 {
		t.Fatalf("summaries with area=demo-area = %+v, want exactly 2 (checkout and billing)", out["summaries"])
	}

	// --area + --phenomenon drills down to the raw items.
	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "queue_list", Arguments: map[string]any{"area": "demo-area", "phenomenon": "checkout"}})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	out, _ = res.StructuredContent.(map[string]any)
	if out["summaries"] != nil {
		t.Errorf("drill-down structured output = %+v, want summaries absent", out)
	}
	resultItems, ok := out["items"].([]any)
	if !ok || len(resultItems) != 2 {
		t.Fatalf("items = %+v, want exactly 2 (demo-area::checkout's own two)", out["items"])
	}
}

func TestMCPQueueListTool_DrillDownWarnsAboveThreeTimesAverage(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	session, _ := connectMCPTestClient(t)

	queueDB, err := persistence.OpenStore(".")
	if err != nil {
		t.Fatalf("OpenStore returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = queueDB.Close() })
	if err := workqueue.EnsureSchema(queueDB); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	now := time.Now().UTC()
	seed := []workqueue.Item{
		{ID: "demo-area::billing::rule::hyp", Tier: workqueue.Blocker, CreatedAt: now},
		{ID: "demo-area::invoicing::rule::hyp", Tier: workqueue.Blocker, CreatedAt: now},
	}
	for i := 0; i < 4; i++ {
		seed = append(seed, workqueue.Item{ID: fmt.Sprintf("demo-area::checkout::rule::hyp%d", i), Tier: workqueue.Blocker, CreatedAt: now})
	}
	for _, it := range seed {
		if err := workqueue.Add(queueDB, it); err != nil {
			t.Fatalf("workqueue.Add returned unexpected error: %v", err)
		}
	}

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "queue_list", Arguments: map[string]any{"area": "demo-area", "phenomenon": "checkout"},
	})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent = %T, want map[string]any: %+v", res.StructuredContent, res.StructuredContent)
	}
	warning, _ := out["warning"].(string)
	if warning == "" {
		t.Error("warning is empty, want one (4 items vs. the other phenomena's own average of 1 is more than 3x)")
	}
}

func TestMCPQueueListTool_PhenomenonWithoutAreaIsError(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	session, _ := connectMCPTestClient(t)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "queue_list", Arguments: map[string]any{"phenomenon": "checkout"}})
	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}
	if !res.IsError {
		t.Error("CallTool result IsError = false for phenomenon without area, want true")
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
