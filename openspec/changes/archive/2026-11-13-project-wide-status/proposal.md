# Proposal

## Why

"conjecture status" (and the "status" MCP tool) only ever read one
file. A usability review asked directly how to know how many theses or
queued work items a project has - "trabalhos" (work items) already had
a real answer in "queue_list"'s own per-phenomenon summary; "teses" (and
every other maturity stage, signed/confirmed counts, and law statuses)
did not: answering "how many theses does this project have" meant
calling "status" on every file by hand and adding the counts yourself.
A real, disclosed gap, not a hypothetical one.

## What Changes

- "conjecture status" with no path argument now reports a project-wide
  aggregate instead of a usage error: every ".cjtr" file under
  "conjecture/phenomena/", summed by signed/confirmed state and by each
  judged hypothesis/law's own maturity stage (Hypothesis/Thesis/Theory/
  Law) and, for a law, its own status. "conjecture status <file.cjtr>"
  is unchanged.
- Adds the "project_status" MCP tool (no input) for the same aggregate.
- A repository with no "conjecture/phenomena/" directory yet reports an
  empty summary, not an error - the same "legitimate, honestly-
  reportable state" posture a hypothesis with no recorded evidence
  already gets from "status" itself.
- A file that fails to load (malformed, unparseable) is still counted
  in the total and disclosed by path in the summary's own "errors"
  field, never silently excluded from the counts.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cli-mcp-integration`: no requirement text changes - "project_status"
  is "reading evidence"/"requesting verification"'s own category,
  already established for "status" itself; aggregating across files
  reads nothing "status" could not already read one file at a time.
  `skip_specs: true`.

## Impact

- Affected paths: `cmd/conjecture/status.go`, `cmd/conjecture/
  status_test.go`, `cmd/conjecture/mcp.go`, `cmd/conjecture/mcp_test.go`.
- "conjecture status" with no argument used to be a usage error; it now
  succeeds with the project-wide aggregate. No internal caller of this
  command relied on the old error behavior.
- No new dependency.
