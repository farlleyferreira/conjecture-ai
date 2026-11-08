package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const mcpUsage = "usage: conjecture mcp"

// runMCP implements "conjecture mcp" - cli-mcp-integration's own "MCP
// server as the primary channel" requirement: "The system SHALL
// expose an MCP server as its primary integration channel for an
// agent, alongside a CLI and automatic folder-watch observation...
// WHEN an agent is configured with conjecture's MCP server THEN it
// can request verification and read results through that channel."
// It serves github.com/modelcontextprotocol/go-sdk's own stdio
// transport - the standard way an MCP client launches a local server
// (as a subprocess, communicating over its stdin/stdout), not a
// network listener this specification never asks for.
//
// Like "conjecture base" and every other command that reads the
// local per-repository store, this assumes the current working
// directory is the repository root - this codebase has no
// repository-root-discovery mechanism of its own yet.
//
// Honesty note: which tools this server exposes is this project's own
// reading of "Explicit permission table for LLM actions" - "the
// system SHALL maintain an explicit, documented table of actions an
// LLM may perform... (for example, requesting verification, reading
// evidence) versus actions it may not perform (for example, writing
// evidence, confirming a specification, changing weights)." Both
// named-allowed actions are built now ("verify" and "read_evidence");
// the named-disallowed ones are enforced by omission - there is no
// tool on this server that writes evidence, confirms a specification,
// or changes a weight, not a runtime check that would reject such a
// call. The folder-watch channel and the rigor-reducing-action rule
// are real, separate, disclosed gaps this slice does not attempt.
func runMCP(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		_, _ = fmt.Fprintln(stderr, mcpUsage)
		return 1
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := evidenceledger.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	server := newMCPServer(db)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	return 0
}

// newMCPServer builds the server every real "conjecture mcp" run
// serves, factored out so a test can connect to it over an in-memory
// transport instead of a real stdio subprocess. db backs the
// "read_evidence" tool; a test supplies a temporary store instead of
// the real per-OS one [persistence.OpenStore] would otherwise open.
func newMCPServer(db *sql.DB) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "conjecture", Version: version.String()}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "verify",
		Description: "Check a .cjtr specification file for internal consistency, falsifiability, and solver-checked contradictions between unconditional statements - the same checks \"conjecture verify\" runs from the CLI, requesting verification and reading its result back through this channel.",
	}, verifyTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_evidence",
		Description: "Read the recorded evidence-ledger history for a hypothesis or user-defined law by its composite id (area::phenomenon::rule::hypothesis, or area::phenomenon::hypothesis for a file-level law) - the other named-allowed action in the explicit permission table (\"reading evidence\"), alongside \"verify\". This only reads; it never writes evidence, confirms a specification, or changes a weight.",
	}, readEvidenceToolFor(db))

	return server
}

// verifyArgs is "verify"'s own tool input - [path] alone, the one
// argument "conjecture verify" itself takes from the CLI.
type verifyArgs struct {
	Path string `json:"path" jsonschema:"path to the .cjtr file to verify"`
}

// verifyOutput is "verify"'s own structured result: whether the file
// passed (exit code 0), and the same textual report "conjecture
// verify" prints to stdout/stderr - "the resulting verdict is
// delivered back through" this channel, per "Verdicts delivered back
// through the same channel."
type verifyOutput struct {
	Passed bool   `json:"passed"`
	Report string `json:"report"`
}

func verifyTool(_ context.Context, _ *mcp.CallToolRequest, args verifyArgs) (*mcp.CallToolResult, verifyOutput, error) {
	var stdout, stderr bytes.Buffer
	code := runVerify([]string{args.Path}, &stdout, &stderr)
	report := stdout.String() + stderr.String()

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: report}},
	}, verifyOutput{Passed: code == 0, Report: report}, nil
}

// readEvidenceArgs is "read_evidence"'s own tool input: the
// hypothesis or law's composite id, the same scheme
// [evidenceledger.Evidence.HypothesisID] already uses.
type readEvidenceArgs struct {
	HypothesisID string `json:"hypothesis_id" jsonschema:"the hypothesis or law's composite id (area::phenomenon::rule::hypothesis, or area::phenomenon::hypothesis for a file-level law)"`
}

// evidenceRecord is one evidence-ledger record's own MCP-facing shape
// - the same fields [recordEvidence] itself writes, read back here
// rather than written.
type evidenceRecord struct {
	Stage      string `json:"stage"`
	Attacked   string `json:"attacked"`
	Survived   string `json:"survived"`
	Open       string `json:"open"`
	Unchecked  string `json:"unchecked"`
	ProducedBy string `json:"produced_by"`
	CreatedAt  string `json:"created_at"`
}

// readEvidenceOutput is "read_evidence"'s own structured result: the
// hypothesis's full recorded history, oldest first (the same order
// [evidenceledger.History] already returns). An empty History is not
// an error - a hypothesis with no recorded evidence yet is a
// legitimate, honestly-reportable state, not a tool failure.
type readEvidenceOutput struct {
	History []evidenceRecord `json:"history"`
}

// readEvidenceToolFor closes over db so the registered tool function
// matches [mcp.AddTool]'s own signature (no extra parameter for it to
// pass through) while still reading from a real, caller-supplied
// store - the same reason [newMCPServer] takes db as its own
// parameter.
func readEvidenceToolFor(db *sql.DB) func(context.Context, *mcp.CallToolRequest, readEvidenceArgs) (*mcp.CallToolResult, readEvidenceOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, args readEvidenceArgs) (*mcp.CallToolResult, readEvidenceOutput, error) {
		history, err := evidenceledger.History(db, args.HypothesisID)
		if err != nil {
			return nil, readEvidenceOutput{}, err
		}

		out := readEvidenceOutput{History: make([]evidenceRecord, len(history))}
		var report bytes.Buffer
		if len(history) == 0 {
			fmt.Fprintf(&report, "%s: no evidence recorded\n", args.HypothesisID)
		}
		for i, e := range history {
			out.History[i] = evidenceRecord{
				Stage: e.Stage.String(), Attacked: e.Attacked, Survived: e.Survived,
				Open: e.Open, Unchecked: e.Unchecked, ProducedBy: e.ProducedBy,
				CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
			fmt.Fprintf(&report, "%s: #%d stage=%s producedBy=%q", args.HypothesisID, i+1, e.Stage, e.ProducedBy)
			if e.Attacked != "" {
				fmt.Fprintf(&report, " attacked=%q", e.Attacked)
			}
			if e.Survived != "" {
				fmt.Fprintf(&report, " survived=%q", e.Survived)
			}
			if e.Open != "" {
				fmt.Fprintf(&report, " open=%q", e.Open)
			}
			if e.Unchecked != "" {
				fmt.Fprintf(&report, " unchecked=%q", e.Unchecked)
			}
			report.WriteByte('\n')
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: report.String()}},
		}, out, nil
	}
}
