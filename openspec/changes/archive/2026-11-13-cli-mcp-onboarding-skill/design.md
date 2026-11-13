# Design

## Context

See proposal.md - Why. Blocks 0-3 (persistence fix, "check"/"status"
MCP tools, eight read-only MCP tools) are already merged; this is the
fourth and final block from that same usability discussion.

## Goals / Non-Goals

**Goals:**
- Teach any agent picking up a conjecture-using repository the real
  loop this codebase supports, without requiring it to read Go source
  first.
- Ship it the way an agent's own tooling already expects a skill to
  arrive - `.claude/skills/<name>/SKILL.md` - rather than a bespoke
  format only this project understands.
- Make the human-only boundary (sign --confirmed, promote) impossible
  to miss: stated as an explicit rule the skill text repeats, not
  left implicit.

**Non-Goals:**
- Multi-agent-tool formats (AGENTS.md, Cursor's `.cursor/rules/`,
  Copilot's instructions file). This round of work only grounds the
  one convention this repository's own OpenSpec dependency already
  demonstrates working end to end; adding others without a concrete
  need would be guessing at formats this project has not verified
  anything against.
- A general "skill installer" command independent of `init`.
  `conjecture init` is already the one, sole place this codebase
  scaffolds anything into a repository (init.go's own doc comment);
  adding a second entry point for one more file would duplicate that
  decision rather than reuse it.
- Keeping the skill's own prose in sync with `draft.go`'s
  `draftSystemPrompt` grammar reference automatically. They currently
  describe the same grammar independently; a future drift between
  them is a real, disclosed risk (see Risks below), not solved here.

## Decisions

1. **The skill file is embedded at build time (`//go:embed`), not
   generated from a template with substituted values.** The content
   is static prose - there is nothing per-repository to substitute
   (no version placeholder: this project has no release-version
   tracking spanning builds, so claiming one in a "generatedBy" field
   the way the OpenSpec-generated skills in this same checkout do
   would overstate what this project can actually promise to keep in
   sync).

2. **Idempotent, write-once.** `conjecture init` already treats every
   other piece of the layout this way (persistence.EnsureWorkingLayout's
   own doc comment: "an existing directory is left alone"). A user who
   edits the skill file - to adapt it to their own project's
   conventions, for instance - never has that edit silently discarded
   by a later `init` re-run.

3. **The skill explicitly repeats the human-only boundary, by name.**
   "Never run `conjecture sign --confirmed` or `conjecture promote`
   yourself... even if you have shell access" is stated as its own
   rule, not left to be inferred from the absence of a corresponding
   MCP tool. An agent with a general-purpose shell tool could
   otherwise trivially run the CLI command itself and defeat the
   entire point of cli-mcp-integration's own permission table - this
   skill is the one place that risk is named for the agent reading it,
   directly.

4. **`conjecture implement`/`work`/`attack`/`swarm` are named and
   explicitly excluded from the primary loop.** Those commands already
   drive conjecture's own separate, internal LLM call - exactly the
   "hidden second LLM" pattern this round of work started by rejecting
   in favor of the user's own already-present coding agent writing the
   implementation directly. They remain real, working, documented
   features of this CLI; this skill just does not route through them.

## Risks / Trade-offs

- **Grammar drift.** The skill's own condensed grammar reference and
  `draft.go`'s `draftSystemPrompt` describe the same `.cjtr` subset
  independently. A future grammar change updated in one and not the
  other would silently drift. Not solved here (a shared constant
  would need both commands to agree on one literal block of prose,
  which the two current read audiences - a human-invoked drafting LLM
  call vs. an agent reading a skill file - do not obviously need
  verbatim-identical wording for); disclosed as a real, open gap
  instead of claimed as handled.
- **No automatic update path.** A repository's own already-scaffolded
  skill file does not pick up a later improvement to the bundled
  template; only a fresh `conjecture init` on a repository that never
  had the file would. Acceptable for a first version; an explicit
  `conjecture init --force`-style re-scaffold is a real, separate
  future option, not attempted here.
- **Claude Code only.** Correct for this project's own actual,
  demonstrated distribution precedent; an agent running a different
  tool that reads a different convention (AGENTS.md, etc.) gets
  nothing from this change. Named as a Non-Goal above, not hidden.
