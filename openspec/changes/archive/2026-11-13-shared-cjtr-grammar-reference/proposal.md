# Proposal

## Why

`draft.go`'s `draftSystemPrompt` and the scaffolded `conjecture-
workflow` skill (`templates/conjecture-workflow-skill.md`) each
described the same subset of the `.cjtr` grammar independently - a real,
disclosed risk the skill's own design.md (block 4 of this round) named
and left open: "a future grammar change updated in one and not the
other would silently drift."

## What Changes

- The grammar structure template and its own explanatory notes
  (the "always" rule, the comparator list, the worked-example
  requirement) move into two new embedded files,
  `templates/cjtr-grammar.yaml` and `templates/cjtr-grammar-notes.md` -
  copied verbatim from `draftSystemPrompt`'s own prior text, the
  version actually exercised by `conjecture draft`'s real LLM calls.
- `draftSystemPrompt` becomes a `var`, assembled from those two embeds
  plus its own surrounding framing text - reconstructs byte-for-byte
  identical to the prior literal constant (verified directly).
- The skill template's own inline copy of the same grammar is replaced
  with two placeholders (`{{CJTR_GRAMMAR_YAML}}`/
  `{{CJTR_GRAMMAR_NOTES}}`); `ensureWorkflowSkill` (skillscaffold.go)
  now renders the skill through a new `renderWorkflowSkill` step that
  substitutes the same two embeds, each re-indented to the list item's
  own three-space continuation indent, before writing the file.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `llm-orchestration`/`cli-mcp-integration`: no requirement text
  changes - this only changes where the same already-shipped grammar
  description's own bytes come from, never what either consumer
  teaches. `skip_specs: true`.

## Impact

- Affected paths: `cmd/conjecture/draft.go`, `cmd/conjecture/
  skillscaffold.go`, `cmd/conjecture/skillscaffold_test.go` (new),
  `cmd/conjecture/templates/cjtr-grammar.yaml` (new),
  `cmd/conjecture/templates/cjtr-grammar-notes.md` (new),
  `cmd/conjecture/templates/conjecture-workflow-skill.md`.
- `draftSystemPrompt`'s own exact text is unchanged (verified
  byte-for-byte); the scaffolded skill's grammar section is now
  identically worded to `draftSystemPrompt`'s own notes (previously a
  differently-phrased, backtick-heavy paraphrase) - a real wording
  change to the skill, in exchange for the two never silently drifting
  apart again.
- Verified for real as a non-root user in WSL: the rendered skill
  contains no leftover `{{CJTR_GRAMMAR_...}}` placeholder and does
  contain the shared comparator notes, correctly re-indented.
- No new dependency.
