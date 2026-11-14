package languageagnosticcontracts

import "errors"

// ForbiddenImportViolation is one declared import in a source file
// whose path matches a caller-named forbidden import.
type ForbiddenImportViolation struct {
	ImportPath string
	// Line is the import declaration's own 1-based source line.
	Line int
}

// ErrDependencyAnalysisUnavailable is what [CheckGoForbiddenImports]
// returns from a binary built without CGO. Tree-sitter's parsing
// engine and its grammars are C code, so the prebuilt release
// binaries (built with CGO_ENABLED=0, so that one build cross-compiles
// to every supported OS and installs without a C toolchain) do not
// carry it. Every other command is unaffected.
var ErrDependencyAnalysisUnavailable = errors.New(
	"languageagnosticcontracts: dependency analysis needs a binary built with CGO " +
		"(Tree-sitter is C code); build from source with CGO_ENABLED=1 and a C compiler to use it")
