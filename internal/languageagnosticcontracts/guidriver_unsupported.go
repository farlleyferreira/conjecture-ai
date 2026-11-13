//go:build !windows && !linux

package languageagnosticcontracts

import (
	"context"
	"fmt"
	"runtime"
)

// NewNativeDriver on every OS without its own real backend yet (see
// guidriver.go's own doc comment) - macOS's own Accessibility API,
// most notably. Returning a [Driver] that fails clearly at Snapshot
// time, rather than refusing to compile, keeps [NewNativeDriver]
// itself genuinely OS-agnostic: a caller writes the same one call on
// every platform, and finds out at runtime, not at build time,
// whether this platform's own backend exists yet.
func NewNativeDriver(target string) Driver {
	return &unsupportedDriver{}
}

type unsupportedDriver struct{}

func (d *unsupportedDriver) Snapshot(ctx context.Context) (GUIElement, error) {
	return GUIElement{}, fmt.Errorf("languageagnosticcontracts: no native accessibility driver backend for %s yet", runtime.GOOS)
}
