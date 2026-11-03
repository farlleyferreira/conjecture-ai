package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	got := String()

	if !strings.Contains(got, version) {
		t.Errorf("String() = %q, want it to contain version %q", got, version)
	}
	if !strings.Contains(got, commit) {
		t.Errorf("String() = %q, want it to contain commit %q", got, commit)
	}
}
