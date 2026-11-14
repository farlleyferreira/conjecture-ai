//go:build cgo

package languageagnosticcontracts

import "testing"

func TestCheckGoForbiddenImports_FlagsForbiddenImportInGroupedBlock(t *testing.T) {
	source := []byte(`package demo

import (
	"fmt"
	"github.com/example/internal/secret"
)

func main() {
	fmt.Println(secret.Value)
}
`)
	violations, err := CheckGoForbiddenImports(source, []string{"github.com/example/internal/secret"})
	if err != nil {
		t.Fatalf("CheckGoForbiddenImports returned unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("len(violations) = %d, want 1; got %+v", len(violations), violations)
	}
	if violations[0].ImportPath != "github.com/example/internal/secret" {
		t.Errorf("ImportPath = %q, want %q", violations[0].ImportPath, "github.com/example/internal/secret")
	}
	if violations[0].Line != 5 {
		t.Errorf("Line = %d, want 5", violations[0].Line)
	}
}

func TestCheckGoForbiddenImports_FlagsForbiddenSingleLineImport(t *testing.T) {
	source := []byte(`package demo

import "github.com/example/forbidden"

func main() {}
`)
	violations, err := CheckGoForbiddenImports(source, []string{"github.com/example/forbidden"})
	if err != nil {
		t.Fatalf("CheckGoForbiddenImports returned unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("len(violations) = %d, want 1", len(violations))
	}
	if violations[0].Line != 3 {
		t.Errorf("Line = %d, want 3", violations[0].Line)
	}
}

func TestCheckGoForbiddenImports_AllowedImportsNotFlagged(t *testing.T) {
	source := []byte(`package demo

import "fmt"

func main() { fmt.Println("hi") }
`)
	violations, err := CheckGoForbiddenImports(source, []string{"os"})
	if err != nil {
		t.Fatalf("CheckGoForbiddenImports returned unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("violations = %+v, want none", violations)
	}
}

func TestCheckGoForbiddenImports_NoForbiddenConfiguredFindsNothing(t *testing.T) {
	source := []byte(`package demo

import "fmt"
`)
	violations, err := CheckGoForbiddenImports(source, nil)
	if err != nil {
		t.Fatalf("CheckGoForbiddenImports returned unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("violations = %+v, want none", violations)
	}
}

func TestCheckGoForbiddenImports_MultipleForbiddenImportsAllFlagged(t *testing.T) {
	source := []byte(`package demo

import (
	"fmt"
	"github.com/example/a"
	"github.com/example/b"
)
`)
	violations, err := CheckGoForbiddenImports(source, []string{"github.com/example/a", "github.com/example/b"})
	if err != nil {
		t.Fatalf("CheckGoForbiddenImports returned unexpected error: %v", err)
	}
	if len(violations) != 2 {
		t.Fatalf("len(violations) = %d, want 2; got %+v", len(violations), violations)
	}
}

func TestCheckGoForbiddenImports_DoesNotInspectImportedPackageBehavior(t *testing.T) {
	// "SHALL NOT use it to infer business semantics": a file that
	// merely imports a forbidden path, without even calling anything
	// from it, is still flagged - this check never looks past the
	// import declaration itself to decide whether the import is
	// "actually used" in some semantic sense.
	source := []byte(`package demo

import "github.com/example/unused"
`)
	violations, err := CheckGoForbiddenImports(source, []string{"github.com/example/unused"})
	if err != nil {
		t.Fatalf("CheckGoForbiddenImports returned unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Errorf("violations = %+v, want 1 (flagged by its declaration alone)", violations)
	}
}
