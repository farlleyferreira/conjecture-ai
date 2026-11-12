package main

import (
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/languageagnosticcontracts"
)

const depsUsage = "usage: conjecture deps check <file.go> --forbid <import-path> [--forbid <import-path> ...]"

// runDeps implements "conjecture deps <subcommand>" - language-
// agnostic-contracts' own "Read-only dependency analysis" wired into
// the CLI for the first time: "check" is this project's own invented
// verb (the specification names no concrete command for this, unlike
// context-composition's or cli-mcp-integration's own worked-example
// tables), scoped to Go source only for now (see [languageagnostic
// contracts]'s own package doc comment).
func runDeps(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, depsUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "check":
		return runDepsCheck(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, depsUsage)
		return 1
	}
}

// runDepsCheck parses "check"'s own arguments and reads the real
// file - the thin, real-path wrapper around [checkGoForbiddenImports],
// which a test calls directly with in-memory source instead of a
// real file on disk.
func runDepsCheck(args []string, stdout, stderr io.Writer) int {
	var path string
	var forbidden []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--forbid" {
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, depsUsage)
				return 1
			}
			forbidden = append(forbidden, args[i])
			continue
		}
		if path != "" {
			_, _ = fmt.Fprintln(stderr, depsUsage)
			return 1
		}
		path = args[i]
	}
	if path == "" || len(forbidden) == 0 {
		_, _ = fmt.Fprintln(stderr, depsUsage)
		return 1
	}

	source, err := os.ReadFile(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	return checkGoForbiddenImports(path, source, forbidden, stdout, stderr)
}

// checkGoForbiddenImports implements "conjecture deps check"'s own
// real logic against source: [languageagnosticcontracts.
// CheckGoForbiddenImports] applied for real, reporting every
// violation found (and exiting 1 if any), or reporting plainly that
// none were found.
func checkGoForbiddenImports(path string, source []byte, forbidden []string, stdout, stderr io.Writer) int {
	violations, err := languageagnosticcontracts.CheckGoForbiddenImports(source, forbidden)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if len(violations) == 0 {
		_, _ = fmt.Fprintln(stdout, "no forbidden imports found")
		return 0
	}
	for _, v := range violations {
		_, _ = fmt.Fprintf(stdout, "%s:%d: forbidden import %q\n", path, v.Line, v.ImportPath)
	}
	return 1
}
