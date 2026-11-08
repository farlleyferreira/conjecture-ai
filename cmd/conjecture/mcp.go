package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

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
// Honesty note: which tools this server exposes is this project's own
// reading of "Explicit permission table for LLM actions" - "the
// system SHALL maintain an explicit, documented table of actions an
// LLM may perform... (for example, requesting verification, reading
// evidence) versus actions it may not perform (for example, writing
// evidence, confirming a specification, changing weights)." The one
// tool built so far ("verify") is exactly the first named-allowed
// action; the named-disallowed ones are enforced by omission here -
// there is no tool on this server that writes evidence, confirms a
// specification, or changes a weight, not a runtime check that would
// reject such a call. "read evidence" (the other named-allowed
// action), the folder-watch channel, and the rigor-reducing-action
// rule are real, separate, disclosed gaps this first slice does not
// attempt.
func runMCP(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		_, _ = fmt.Fprintln(stderr, mcpUsage)
		return 1
	}

	server := newMCPServer()
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	return 0
}

// newMCPServer builds the server every real "conjecture mcp" run
// serves, factored out so a test can connect to it over an in-memory
// transport instead of a real stdio subprocess.
func newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "conjecture", Version: version.String()}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "verify",
		Description: "Check a .cjtr specification file for internal consistency, falsifiability, and solver-checked contradictions between unconditional statements - the same checks \"conjecture verify\" runs from the CLI, requesting verification and reading its result back through this channel.",
	}, verifyTool)

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
