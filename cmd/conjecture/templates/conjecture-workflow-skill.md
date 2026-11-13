---
name: conjecture-workflow
description: Draft a business-rule specification together with the user as a .cjtr file, verify it, have the user sign it, then write the real implementation and verify it conforms via conjecture's MCP tools. Use when the user wants to specify a precise behavior/business rule and have real code checked against it, or mentions conjecture, .cjtr, or a "verifiable specification".
license: MIT
compatibility: Requires a running "conjecture mcp" server connected to this session, and (for the "check" step) a container runtime reachable from wherever that server process runs.
metadata:
  author: conjecture
---

Specify a behavior precisely with the user, then prove the real implementation matches it - never the other way around (never write code first and reverse-engineer a spec that happens to pass).

## The loop

1. **Make sure the repository is set up.** If there is no `conjecture/phenomena/` directory yet, run `conjecture init` (plain scaffolding, not a decision - safe to run on your own).

2. **Draft the specification with the user, in the chat - not with a separate tool call.** Write or edit the `.cjtr` file directly (your own file-editing ability), at `conjecture/phenomena/<area>/<phenomenon>.cjtr`. Use this grammar:

   ```yaml
   conjecture: 1.0.0
   phenomenon: <short name>
   description: "<one sentence>"

   concepts:
     <concept name>:
       variable: input.<name> OR output.<name>
       type: decimal OR text
       # optional: range: {min: X, max: Y}   (decimal only)
       # optional: one_of: [a, b, c]          (text only)
     <another concept>:
       value: 100.00   # a fixed value instead of a variable

   rules:
     <rule name>:
       id: <kebab-case-id>
       hypotheses:
         <hypothesis name>:
           id: <kebab-case-id>
           given:
             - the <concept> <comparator> <value or concept>
           then: the <concept> <comparator> <value or concept>
           otherwise: the <concept> <comparator> <value or concept>
           examples:
             - {<concept>: <value>, <concept>: <value>}
   ```

   A hypothesis with no condition (always true) uses `always:` instead of `given`/`then`/`otherwise`. Comparators: `is`/`equals`, `is not`, `is greater than`, `is less than`, `is at least`, `is at most`, `is between X and Y`, `is strictly between X and Y`, `is one of [...]`. Every hypothesis needs at least one worked example. Write `???` at any point that is genuinely unclear rather than inventing an assumption - resolve it with the user instead.

   Never write a `signed:` block yourself - see step 4.

3. **Verify the draft, every time you change it.** Call the `verify` MCP tool with the file's path. If it fails, read the report, fix the file, call `verify` again. Loop with the user - this is where you debate and refine the rule itself, before any implementation exists.

4. **Stop and have the user sign the agreed specification themselves.** Once the user is satisfied with the content, tell them to run, in their own terminal:

   ```
   conjecture sign --user <their name> --confirmed conjecture/phenomena/<area>/<phenomenon>.cjtr
   ```

   **Never run `conjecture sign --confirmed` or `conjecture promote` yourself - not through this server (there is no such tool here on purpose) and not by shelling out to the CLI either, even if you have shell access.** Confirming a specification is a decision this project reserves for a human, by design; running the command yourself on the user's behalf defeats the entire point of that boundary. Wait for the user to confirm they ran it, then call the `status` tool on the file and check `signed`/`confirmed` are both true before moving on.

5. **Write the real implementation.** Once signed, write the actual implementation code - with your own normal coding ability, in whatever language/framework the project already uses. Do not use `conjecture implement`/`work`/`attack`/`swarm`: those drive conjecture's own separate, built-in LLM loop, which is a different, already-complete feature of this tool - not what this workflow is for. Package the implementation as a container image + command that reads `conjecture_input.json` (flat JSON, keys like `input.<name>`) and writes `conjecture_output.json` the same way.

6. **Check it for real.** Call the `check` MCP tool with the file's path, the image name, and the command to run inside it. If it reports non-conformant, read the counterexample it returns, fix the implementation for real, rebuild the image, and call `check` again. Repeat until conformant - never adjust the specification to make a failing implementation pass; if the rule itself turns out to be wrong, go back to step 2 and 3 with the user, then get it re-signed.

7. **Report the final state.** Call `status` on the file and summarize it for the user: signed by whom, and each hypothesis/law's own checked state.

## Everything else this server exposes (all pure reads, call anytime)

| Tool | Reads |
|---|---|
| `read_evidence` | A hypothesis/law's full recorded evidence history, by composite id |
| `context_list` | Saved verification contexts, optionally filtered by area/date |
| `context_resume` | A context's latest content, plus the current highest-priority queue item |
| `context_open` | One exact past version of a context (`name@v<N>`) |
| `context_origin` | A context's own real provenance: its file's signed block + recorded decisions |
| `context_diff` | A real unified diff between two context versions |
| `context_resolve_conflict` | Which composition mode wins a stated conflict, by the project's own precedence rule |
| `queue_list` | Every item currently in the three-tier work queue (never removes anything) |
| `graph_inspect` | A context-engine node's weight/relevance/confidence axes |

## What you must never do through this server or by shelling out around it

- Confirm a specification (`conjecture sign --confirmed`) on the user's behalf.
- Promote a hypothesis or law (`conjecture promote`) on the user's behalf.
- Treat a hypothesis's own declared examples as sufficient proof without actually calling `check` against real running code.
