package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
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

// queueListArgs is "queue_list"'s own tool input - the same optional
// --area/--phenomenon filters "conjecture queue list" takes from the
// CLI; see [computeQueueList]'s own doc comment for what each
// combination returns.
type queueListArgs struct {
	Area       string `json:"area,omitempty" jsonschema:"only include this area's own phenomena"`
	Phenomenon string `json:"phenomenon,omitempty" jsonschema:"drill into exactly this phenomenon's own queued items instead of the summary (requires area)"`
}

// queueListOutput is "queue_list"'s own structured result: exactly
// one of Summaries or Items is populated, never both - see
// [computeQueueList]'s own doc comment for which. Warning is only
// ever set alongside Items - see [computeQueueList]'s own doc comment
// for when.
type queueListOutput struct {
	Summaries []PhenomenonQueueSummary `json:"summaries,omitempty"`
	Items     []QueueItemOut           `json:"items,omitempty"`
	Warning   string                   `json:"warning,omitempty"`
}

// queueListTool needs no db from [newMCPServer] - [workqueue.List]'s
// own store is opened directly, same as "conjecture queue list"
// itself (runQueueList). Unlike every other tool in this file, it
// does not use the shared [textReadOutput]: [computeQueueList]'s own
// per-phenomenon grouping is exactly the kind of structure an agent
// would want to read programmatically (deciding whether to drill
// further) rather than parse back out of text, the same reasoning
// "status" already applies.
func queueListTool(_ context.Context, _ *mcp.CallToolRequest, args queueListArgs) (*mcp.CallToolResult, queueListOutput, error) {
	db, err := openRepoStore()
	if err != nil {
		return nil, queueListOutput{}, err
	}
	defer func() { _ = db.Close() }()
	if err := workqueue.EnsureSchema(db); err != nil {
		return nil, queueListOutput{}, err
	}
	items, err := workqueue.List(db)
	if err != nil {
		return nil, queueListOutput{}, err
	}

	summaries, queueItems, warning, err := computeQueueList(items, args.Area, args.Phenomenon)
	if err != nil {
		return nil, queueListOutput{}, err
	}

	var report bytes.Buffer
	if args.Area != "" && args.Phenomenon != "" {
		if warning != "" {
			fmt.Fprintf(&report, "WARNING: %s\n", warning)
		}
		if len(queueItems) == 0 {
			fmt.Fprintf(&report, "%s::%s: queue is empty\n", args.Area, args.Phenomenon)
		}
		for _, it := range queueItems {
			fmt.Fprintf(&report, "%s: %s, weight %d, severity %d, queued %s\n", it.ID, it.Tier, it.Weight, it.Severity, it.Queued)
		}
	} else {
		if len(summaries) == 0 {
			fmt.Fprintln(&report, "queue is empty")
		}
		for _, s := range summaries {
			fmt.Fprintf(&report, "%s: %d items (blocker %d, current-task %d, background %d)\n", s.PhenomenonID, s.Total, s.Blocker, s.CurrentTask, s.Background)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: report.String()}},
	}, queueListOutput{Summaries: summaries, Items: queueItems, Warning: warning}, nil
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
