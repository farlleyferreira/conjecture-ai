package main

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// textReadOutput is the shared structured result for every read-only
// MCP tool in this file: each one just wraps an existing CLI function
// that already prints its own report to stdout/stderr and returns an
// exit code, the same {bool, text} shape [verifyOutput] and
// [checkOutput] already use. None of these commands has a richer
// per-field structure worth a bespoke type the way "status" does -
// only a report and whether it completed without error.
type textReadOutput struct {
	OK     bool   `json:"ok"`
	Report string `json:"report"`
}

// textReadTool runs an existing CLI command function (one that writes
// its own report to the stdout/stderr it is given and returns an exit
// code) and collects both into one [textReadOutput] - the shared
// plumbing every tool below calls with its own closure, so each one
// only has to build cliArgs from its own typed input.
func textReadTool(run func(stdout, stderr io.Writer) int) (*mcp.CallToolResult, textReadOutput, error) {
	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr)
	report := stdout.String() + stderr.String()
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: report}},
	}, textReadOutput{OK: code == 0, Report: report}, nil
}

// contextListArgs is "context_list"'s own tool input - the same
// optional --area/--since/--until filters "conjecture context list"
// takes from the CLI (contextList's own doc comment), each left empty
// to mean "no filter" rather than the CLI's flag-present/flag-absent
// distinction, which a JSON tool call has no equivalent for.
type contextListArgs struct {
	Area  string `json:"area,omitempty" jsonschema:"only list contexts in this area"`
	Since string `json:"since,omitempty" jsonschema:"only list contexts created on or after this date (YYYY-MM-DD)"`
	Until string `json:"until,omitempty" jsonschema:"only list contexts created on or before this date (YYYY-MM-DD)"`
}

func contextListToolFor(db *sql.DB) func(context.Context, *mcp.CallToolRequest, contextListArgs) (*mcp.CallToolResult, textReadOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, args contextListArgs) (*mcp.CallToolResult, textReadOutput, error) {
		var cliArgs []string
		if args.Area != "" {
			cliArgs = append(cliArgs, "--area", args.Area)
		}
		if args.Since != "" {
			cliArgs = append(cliArgs, "--since", args.Since)
		}
		if args.Until != "" {
			cliArgs = append(cliArgs, "--until", args.Until)
		}
		return textReadTool(func(stdout, stderr io.Writer) int {
			return contextList(db, cliArgs, stdout, stderr)
		})
	}
}

// contextResumeArgs is "context_resume"'s own tool input - [name]
// alone, the one argument "conjecture context resume" itself takes.
type contextResumeArgs struct {
	Name string `json:"name" jsonschema:"the context's name"`
}

func contextResumeToolFor(db *sql.DB) func(context.Context, *mcp.CallToolRequest, contextResumeArgs) (*mcp.CallToolResult, textReadOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, args contextResumeArgs) (*mcp.CallToolResult, textReadOutput, error) {
		return textReadTool(func(stdout, stderr io.Writer) int {
			return contextResume(db, []string{args.Name}, stdout, stderr)
		})
	}
}

// contextOpenArgs is "context_open"'s own tool input - [ref] alone,
// the one argument "conjecture context open" itself takes: a context
// name with an explicit version ("name@v<N>") - [contextcomposition.
// Open] always requires one, unlike "resume".
type contextOpenArgs struct {
	Ref string `json:"ref" jsonschema:"the context's name and an explicit version, e.g. \"checkout@v4\""`
}

func contextOpenToolFor(db *sql.DB) func(context.Context, *mcp.CallToolRequest, contextOpenArgs) (*mcp.CallToolResult, textReadOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, args contextOpenArgs) (*mcp.CallToolResult, textReadOutput, error) {
		return textReadTool(func(stdout, stderr io.Writer) int {
			return contextOpen(db, []string{args.Ref}, stdout, stderr)
		})
	}
}

// contextOriginArgs is "context_origin"'s own tool input - [name]
// alone, the one argument "conjecture context origin" itself takes.
type contextOriginArgs struct {
	Name string `json:"name" jsonschema:"the context's name (an area::phenomenon composite id)"`
}

func contextOriginToolFor(db *sql.DB) func(context.Context, *mcp.CallToolRequest, contextOriginArgs) (*mcp.CallToolResult, textReadOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, args contextOriginArgs) (*mcp.CallToolResult, textReadOutput, error) {
		return textReadTool(func(stdout, stderr io.Writer) int {
			repoRoot, err := os.Getwd()
			if err != nil {
				_, _ = io.WriteString(stderr, err.Error()+"\n")
				return 1
			}
			return contextOrigin(db, repoRoot, []string{args.Name}, stdout, stderr)
		})
	}
}

// contextDiffArgs is "context_diff"'s own tool input - the same two
// refs "conjecture context diff" takes, each either a bare name
// (latest version) or "name@v<N>" (one exact past version).
type contextDiffArgs struct {
	RefA string `json:"ref_a" jsonschema:"the first context's name, optionally with an explicit version (e.g. \"checkout\" or \"checkout@v2\")"`
	RefB string `json:"ref_b" jsonschema:"the second context's name, same form as ref_a"`
}

func contextDiffToolFor(db *sql.DB) func(context.Context, *mcp.CallToolRequest, contextDiffArgs) (*mcp.CallToolResult, textReadOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, args contextDiffArgs) (*mcp.CallToolResult, textReadOutput, error) {
		return textReadTool(func(stdout, stderr io.Writer) int {
			return contextDiff(db, []string{args.RefA, args.RefB}, stdout, stderr)
		})
	}
}

// contextResolveConflictArgs is "context_resolve_conflict"'s own tool
// input - the two composition modes in conflict, the operator's own
// facts about what is actually conflicting (contextResolveConflict's
// own doc comment), never derived or persisted by this tool itself.
type contextResolveConflictArgs struct {
	ModeA string `json:"mode_a" jsonschema:"the first composition mode in conflict (ByReference, ByEmbedding, ByInheritance, or ByView)"`
	ModeB string `json:"mode_b" jsonschema:"the second composition mode in conflict, same set as mode_a"`
}

func contextResolveConflictTool(_ context.Context, _ *mcp.CallToolRequest, args contextResolveConflictArgs) (*mcp.CallToolResult, textReadOutput, error) {
	return textReadTool(func(stdout, stderr io.Writer) int {
		return contextResolveConflict([]string{"--mode-a", args.ModeA, "--mode-b", args.ModeB}, stdout, stderr)
	})
}

// queueListTool takes no input: "conjecture queue list" itself takes
// none either. Unlike the context tools above, it needs no db from
// [newMCPServer] - [runQueueList] already opens and closes its own
// connection, same as "verify"/"check".
func queueListTool(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, textReadOutput, error) {
	return textReadTool(func(stdout, stderr io.Writer) int {
		return runQueueList(nil, stdout, stderr)
	})
}

// graphInspectArgs is "graph_inspect"'s own tool input - [node-id]
// alone, the one argument "conjecture graph inspect" itself takes.
type graphInspectArgs struct {
	NodeID string `json:"node_id" jsonschema:"the context-engine node's own id"`
}

// graphInspectTool needs no db either, for the same reason
// [queueListTool] does not - [runGraphInspect] opens its own
// connection via openGraphStore.
func graphInspectTool(_ context.Context, _ *mcp.CallToolRequest, args graphInspectArgs) (*mcp.CallToolResult, textReadOutput, error) {
	return textReadTool(func(stdout, stderr io.Writer) int {
		return runGraphInspect([]string{args.NodeID}, stdout, stderr)
	})
}
