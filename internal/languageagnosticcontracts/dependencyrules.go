// Package languageagnosticcontracts implements pieces of the
// language-agnostic-contracts capability (openspec/specs/language-
// agnostic-contracts/spec.md): "Read-only dependency analysis" (this
// file - "The system SHALL use a read-only, language-agnostic parsing
// approach (such as Tree-sitter) only for dependency-rule checks, and
// SHALL NOT use it to infer business semantics") and "Generic
// accessibility driver for GUI targets" - the latter as a contract
// only, with no per-OS backend yet (see guidriver.go's own doc
// comment for why).
//
// "Observation only through declared contracts" and "Standard result
// formats" are already satisfied elsewhere in this codebase without a
// dedicated package under this name: sandbox invokes declared
// commands/interfaces directly, and internal/jsonschema, internal/
// junit, and internal/tap already parse the standard result formats
// the specification names. "Optional runtime adapters for
// introspection" remains a separate, still-disclosed gap this package
// does not touch.
//
// CGO honesty note: Tree-sitter's own parsing engine and every
// language grammar for it are C code - there is no pure-Go
// implementation to depend on instead, unlike every other dependency
// this codebase otherwise keeps CGO-free (modernc.org/sqlite,
// github.com/pbnjay/memory, github.com/mackerelio/go-osstat). This
// package is a deliberate, scoped exception to that posture: CGO is
// isolated to this one package, which nothing else in this codebase
// depends on, rather than spreading it anywhere else.
//
// Language coverage: only Go, via github.com/smacker/go-tree-sitter's
// own golang grammar, is wired so far - "language-agnostic" describes
// Tree-sitter's own parsing approach in general (a grammar exists for
// most languages), not a claim that every language is already
// supported here. Adding another language means adding its own
// grammar sub-package and query, not a redesign of this one.
package languageagnosticcontracts

import (
	"context"
	"fmt"
	"strconv"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// ForbiddenImportViolation is one declared import in a source file
// whose path matches a caller-named forbidden import.
type ForbiddenImportViolation struct {
	ImportPath string
	// Line is the import declaration's own 1-based source line.
	Line int
}

// goImportPathQuery is a Tree-sitter query (S-expression syntax) over
// the Go grammar's own parse tree, capturing every import_spec's own
// path string literal - single imports ("fmt") and grouped ones
// (inside "import (...)") alike, since the grammar represents both
// the same way: an import_spec node per path, grouped or not.
const goImportPathQuery = `(import_spec path: (interpreted_string_literal) @path)`

// CheckGoForbiddenImports parses source - a single Go source file's
// content - with Tree-sitter's own Go grammar and reports every
// declared import whose path exactly matches one of forbidden.
//
// This only ever reads an import declaration's own path string - the
// one fact this check needs - never anything about what an imported
// package's code actually does, which would be "business semantics,"
// the one thing "Read-only dependency analysis" rules out.
//
// forbidden is matched by exact import path (e.g.
// "github.com/some/internal/pkg"), not by prefix or pattern glob - a
// caller wanting to forbid a whole subtree names each of its own
// known import paths explicitly, the same "caller supplies the fact"
// posture this codebase already takes for every other configurable
// rule (omission-detection's own catalog, context-engine's own
// categories).
func CheckGoForbiddenImports(source []byte, forbidden []string) ([]ForbiddenImportViolation, error) {
	forbiddenSet := make(map[string]bool, len(forbidden))
	for _, f := range forbidden {
		forbiddenSet[f] = true
	}

	lang := golang.GetLanguage()
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, fmt.Errorf("languageagnosticcontracts: parsing Go source: %w", err)
	}
	defer tree.Close()

	query, err := sitter.NewQuery([]byte(goImportPathQuery), lang)
	if err != nil {
		return nil, fmt.Errorf("languageagnosticcontracts: building import query: %w", err)
	}
	defer query.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	cursor.Exec(query, tree.RootNode())

	var violations []ForbiddenImportViolation
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			path, err := strconv.Unquote(capture.Node.Content(source))
			if err != nil {
				// A string literal Tree-sitter's own grammar accepted
				// but Go's own quoting rules reject is a malformed
				// file - a compile error elsewhere, not something
				// this read-only check tries to diagnose itself.
				continue
			}
			if forbiddenSet[path] {
				violations = append(violations, ForbiddenImportViolation{
					ImportPath: path,
					Line:       int(capture.Node.StartPoint().Row) + 1,
				})
			}
		}
	}
	return violations, nil
}
