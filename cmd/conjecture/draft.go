package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/llmclient"
	"github.com/farlleyferreira/conjecture-ai/internal/llmorchestration"
)

const draftUsage = "usage: conjecture draft [--provider <name>] <file.cjtr> <plain-language ask>"

// cjtrGrammarYAML and cjtrGrammarNotes are the one source of truth for
// the subset of the cjtr-language grammar this codebase's own parser
// (internal/cjtr) is actually verified to accept - embedded here and
// in skillscaffold.go's own scaffolded skill (which substitutes them,
// re-indented, into templates/conjecture-workflow-skill.md in place
// of "{{CJTR_GRAMMAR_YAML}}"/"{{CJTR_GRAMMAR_NOTES}}") so the two
// descriptions of the same grammar - one read by an LLM via this
// prompt, the other read by an agent via the skill - cannot drift
// apart the way two independently-worded copies already had.
//
//go:embed templates/cjtr-grammar.yaml
var cjtrGrammarYAML string

//go:embed templates/cjtr-grammar-notes.md
var cjtrGrammarNotes string

// draftSystemPrompt teaches the model only the subset of the cjtr-
// language grammar this codebase's own parser (internal/cjtr) is
// actually verified to accept - the same core shape every worked
// fixture in this project's own tests already exercises (concepts,
// rules, hypotheses, given/then/otherwise/always, examples) - rather
// than the full grammar's rarer corners (unchanged, decisions,
// user_defined_laws nesting, table concepts, axioms/assumes) that
// would widen the surface a drafted file could get wrong. It tells the
// model explicitly to prefer a "???" marker over inventing a field
// outside this, and never to include a "signed" block itself - this
// codebase's own trust-boundary rule (llmorchestration.Allowed), not
// just a style preference, since draftContent below also enforces it
// structurally regardless of what the model actually does.
//
// The grammar description itself (cjtrGrammarYAML/cjtrGrammarNotes)
// is built once, from the same embedded files the scaffolded skill
// reads - see their own doc comment above for why.
var draftSystemPrompt = fmt.Sprintf(`You draft a .cjtr specification file from a plain-language request. Output ONLY the YAML document - no markdown code fences, no commentary before or after it.

Use exactly this structure (omit a section if it doesn't apply):

%s

%s

If something about the request is genuinely unclear or undecided, write ??? at that exact point rather than inventing an assumption - a human will resolve it later.

Do not include a "signed" block. Whether a specification is confirmed is a decision for a human, never for you.`,
	strings.TrimRight(cjtrGrammarYAML, "\n"),
	strings.TrimRight(cjtrGrammarNotes, "\n"),
)

// runDraft implements "conjecture draft [--provider <name>]
// <file.cjtr> <ask>" - llm-orchestration's drafting side: a plain-
// language request becomes a candidate .cjtr file, which this command
// then runs through the same checks "conjecture verify" always does,
// since a draft is exactly the kind of unconfirmed, possibly-wrong
// specification that command's own NOTICE already exists for.
//
// This plays llm-orchestration's Drafter role on its own, in
// isolation - there is no separate "implement" command yet for
// RequireSeparateSessions to check this against.
func runDraft(args []string, stdout, stderr io.Writer) int {
	provider := "anthropic"
	var path string
	var askWords []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--provider" {
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, draftUsage)
				return 1
			}
			provider = args[i]
			continue
		}
		if path == "" {
			path = args[i]
			continue
		}
		askWords = append(askWords, args[i])
	}
	ask := strings.Join(askWords, " ")
	if path == "" || ask == "" {
		_, _ = fmt.Fprintln(stderr, draftUsage)
		return 1
	}

	client, err := llmclient.New(provider)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	content, err := draftContent(context.Background(), client, ask)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "drafted %s (unconfirmed - review before signing)\n\n", path)
	return runVerify([]string{path}, stdout, stderr)
}

// draftContent asks client to draft ask into a .cjtr document and
// returns its content, with a markdown code fence stripped if the
// model added one despite being told not to, and any "signed" block
// stripped unconditionally - llm-orchestration's own trust-boundary
// gate (llmorchestration.Allowed) never permits an LLM actor to
// confirm a specification, so this does not trust the prompt alone to
// keep one out.
func draftContent(ctx context.Context, client llmclient.Client, ask string) ([]byte, error) {
	text, err := client.Complete(ctx, draftSystemPrompt, ask)
	if err != nil {
		return nil, fmt.Errorf("drafting failed: %w", err)
	}

	text = stripCodeFence(text)

	content := []byte(text)
	if !llmorchestration.Allowed(llmorchestration.LLM, llmorchestration.ConfirmSpecification) {
		stripped, err := cjtr.StripSignedBlock(content)
		if err != nil {
			return nil, fmt.Errorf("stripping a \"signed\" block from the draft: %w", err)
		}
		content = stripped
	}

	return content, nil
}

// stripCodeFence removes a single markdown code fence (e.g.
// "```yaml\n...\n```") wrapping the entirety of text, if there is one -
// models often add one despite being told not to. It is deliberately
// conservative: it only strips a fence that spans text's whole trimmed
// content, starting at its very first line and ending at its very
// last, never one found only partway through, so it cannot
// accidentally eat real content that happens to contain triple
// backticks elsewhere (text is returned unchanged in that case).
func stripCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return text
	}
	firstNewline := strings.IndexByte(trimmed, '\n')
	if firstNewline == -1 {
		return text
	}
	body := trimmed[firstNewline+1:]
	if !strings.HasSuffix(body, "```") {
		return text
	}
	return strings.TrimSpace(strings.TrimSuffix(body, "```"))
}
