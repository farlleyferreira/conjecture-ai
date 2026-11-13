package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
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
// evidence, confirming a specification, changing weights)." "check"
// is "requesting verification" too, the same category "verify"
// already names, not a new permission: the evidence it records is the
// system's own honest account of a real sandboxed run, not the LLM
// asserting evidence directly - the disallowed "writing evidence"
// means an action that injects or edits a ledger record outright,
// which no tool on this server does. The named-disallowed actions
// stay enforced by omission - there is still no tool here that writes
// evidence directly, confirms a specification, or changes a weight,
// not a runtime check that would reject such a call. "status" is a
// pure read too - the file's own "signed" block plus the evidence
// ledger and user-defined-laws' own status table, the same stores
// "verify" and "read_evidence" already read, aggregated into one call
// instead of requiring an agent to already know a file's every
// hypothesis id. "context_list"/"context_resume"/"context_open"/
// "context_origin"/"context_diff"/"context_resolve_conflict"/
// "queue_list"/"graph_inspect" (mcp_read_tools.go) are every other
// read this codebase's CLI already exposes with no write side effect
// at all - confirmed one function at a time against their own real
// implementation before being wired here (context restore, queue
// add/next, and graph set were all confirmed to write, and stay
// CLI-only for exactly that reason). Confirming a specification
// ("conjecture sign --confirmed") and promoting it stay CLI-only,
// deliberately: those are the human-only side of this boundary, not
// gaps this slice leaves open. The folder-watch channel
// ("conjecture watch", watch.go) and the rigor-reducing-action rule
// ("conjecture optimize calibrate" marking a settings.cjtr file
// unconfirmed, see optimize.go) are both wired elsewhere in this CLI
// now, not gaps either.
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
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := contextcomposition.EnsureSchema(db); err != nil {
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
// "read_evidence" and "status" tools; a test supplies a temporary
// store instead of the real per-OS one [persistence.OpenStore] would
// otherwise open.
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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check",
		Description: "Run a hypothesis's or user-defined law's conformance cycle against a real sandboxed implementation - the same check \"conjecture check\" runs from the CLI, requesting verification and reading its result back through this channel. Requires a container runtime reachable from wherever this MCP server process runs, and an already-built image; never confirms a specification or changes a weight itself.",
	}, checkTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "status",
		Description: "Read a .cjtr file's own aggregated lifecycle state: whether it is signed (and by whom), and, per declared hypothesis or user-defined law, whether it has been checked yet and what its latest evidence and (for a law) promotion status are - the same state \"conjecture status\" reports from the CLI, gathered from the file's own \"signed\" block plus the evidence ledger and user-defined-laws' own status table. This only reads; it never writes any of them.",
	}, statusToolFor(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "context_list",
		Description: "List saved verification contexts, optionally filtered by area and/or a created-at date range - the same listing \"conjecture context list\" prints from the CLI. Pure read.",
	}, contextListToolFor(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "context_resume",
		Description: "Resume a context by name: its latest version's own content, plus the work queue's own current highest-priority item (peeked, never popped) - the same read \"conjecture context resume\" runs from the CLI. Pure read; it does not consume the queue item it reports.",
	}, contextResumeToolFor(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "context_open",
		Description: "Open one exact past version of a context (\"name@v<N>\"), read-only - the same read \"conjecture context open\" runs from the CLI.",
	}, contextOpenToolFor(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "context_origin",
		Description: "Read a context's own real provenance: its source .cjtr file's \"signed\" block (user, tool, date, confirmed) together with every decision-ledger entry recorded against that exact file path - the same read \"conjecture context origin\" runs from the CLI. Pure read.",
	}, contextOriginToolFor(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "context_diff",
		Description: "Show a real unified diff between two context versions' own stored content - the same read \"conjecture context diff\" runs from the CLI. Pure read.",
	}, contextDiffToolFor(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "context_resolve_conflict",
		Description: "Resolve a stated conflict between two composition modes (ByReference, ByEmbedding, ByInheritance, ByView) per the precedence rule \"conjecture context resolve-conflict\" already applies from the CLI - a pure computation over the two modes given, never persisted anywhere, the same posture \"conjecture law resolve-conflict\" already takes.",
	}, contextResolveConflictTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "queue_list",
		Description: "Read the three-tier work queue - by default a summary grouped by area::phenomenon (item count per tier), bounded by how many distinct phenomena have queued items rather than the raw item count, which never has a cap. Pass \"area\" to narrow the summary, or both \"area\" and \"phenomenon\" to drill into that one phenomenon's own items. The same view \"conjecture queue list\" prints from the CLI. Pure read; unlike \"conjecture queue next\", this never removes anything from the queue.",
	}, queueListTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "graph_inspect",
		Description: "Read a context-engine node's weight, relevance, and confidence, reported as three separate axes, each marked \"not set\" independently when never recorded - the same read \"conjecture graph inspect\" runs from the CLI. Pure read; it never sets any axis.",
	}, graphInspectTool)

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

// checkArgs is "check"'s own tool input - [path, image, command...],
// the same three-part shape "conjecture check" itself takes from the
// CLI ([runCheck]'s own checkUsage), with command split into its own
// field rather than left for the caller to flatten back into a single
// arg slice.
type checkArgs struct {
	Path    string   `json:"path" jsonschema:"path to the .cjtr file to check"`
	Image   string   `json:"image" jsonschema:"container image that runs the implementation under test"`
	Command []string `json:"command" jsonschema:"command (and its arguments) to run inside the container"`
}

// checkOutput mirrors verifyOutput: whether every judged hypothesis
// and law in the file came back accepted and conformant (exit code
// 0), and the same textual report "conjecture check" prints to
// stdout/stderr.
type checkOutput struct {
	Passed bool   `json:"passed"`
	Report string `json:"report"`
}

func checkTool(_ context.Context, _ *mcp.CallToolRequest, args checkArgs) (*mcp.CallToolResult, checkOutput, error) {
	var stdout, stderr bytes.Buffer
	cliArgs := append([]string{args.Path, args.Image}, args.Command...)
	code := runCheck(cliArgs, &stdout, &stderr)
	report := stdout.String() + stderr.String()

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: report}},
	}, checkOutput{Passed: code == 0, Report: report}, nil
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

// statusArgs is "status"'s own tool input - [path] alone, the one
// argument "conjecture status" itself takes from the CLI.
type statusArgs struct {
	Path string `json:"path" jsonschema:"path to the .cjtr file to report status for"`
}

// statusToolFor closes over db for the same reason
// [readEvidenceToolFor] does: the registered tool function matches
// [mcp.AddTool]'s own signature (no extra parameter for it to pass
// through) while still reading from the server's own shared store -
// [computeStatus] never opens its own connection. Its result is
// [Status] itself: unlike "verify"/"check", there is no pass/fail
// notion here to collapse it into, so this mirrors "read_evidence"'s
// own raw-error passthrough instead.
func statusToolFor(db *sql.DB) func(context.Context, *mcp.CallToolRequest, statusArgs) (*mcp.CallToolResult, Status, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, Status, error) {
		f, err := loadFicha(args.Path)
		if err != nil {
			return nil, Status{}, err
		}

		status, err := computeStatus(db, f)
		if err != nil {
			return nil, Status{}, err
		}

		var report bytes.Buffer
		printStatus(&report, status)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: report.String()}},
		}, status, nil
	}
}
