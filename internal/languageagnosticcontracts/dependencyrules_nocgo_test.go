//go:build !cgo

package languageagnosticcontracts

import (
	"errors"
	"testing"
)

func TestCheckGoForbiddenImports_WithoutCGOReportsUnavailable(t *testing.T) {
	violations, err := CheckGoForbiddenImports([]byte("package demo\n\nimport \"os\"\n"), []string{"os"})
	if !errors.Is(err, ErrDependencyAnalysisUnavailable) {
		t.Fatalf("CheckGoForbiddenImports without CGO: err = %v, want ErrDependencyAnalysisUnavailable", err)
	}
	if violations != nil {
		t.Errorf("CheckGoForbiddenImports without CGO: violations = %v, want nil", violations)
	}
}
