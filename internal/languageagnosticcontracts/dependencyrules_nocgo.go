//go:build !cgo

package languageagnosticcontracts

// DependencyAnalysisAvailable reports whether this binary was built
// with CGO, and so carries Tree-sitter's own C parsing engine - see
// dependencyrules.go for the other half of this pair.
const DependencyAnalysisAvailable = false

// CheckGoForbiddenImports is the CGO-free build's stand-in for the
// real Tree-sitter-backed check in dependencyrules.go: it always
// returns [ErrDependencyAnalysisUnavailable], never a guessed or
// partial answer - a check that silently reported "no forbidden
// imports" without having parsed anything would be worse than one
// that says plainly it cannot run here.
func CheckGoForbiddenImports(source []byte, forbidden []string) ([]ForbiddenImportViolation, error) {
	return nil, ErrDependencyAnalysisUnavailable
}
