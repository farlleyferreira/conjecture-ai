package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "no arguments",
			args:       []string{},
			wantCode:   1,
			wantStderr: "usage:",
		},
		{
			name:       "version command",
			args:       []string{"version"},
			wantCode:   0,
			wantStdout: "dev",
		},
		{
			name:       "unknown command",
			args:       []string{"bogus"},
			wantCode:   1,
			wantStderr: "unknown command: bogus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			gotCode := run(tt.args, &stdout, &stderr)

			if gotCode != tt.wantCode {
				t.Errorf("run(%v) code = %d, want %d", tt.args, gotCode, tt.wantCode)
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("run(%v) stdout = %q, want it to contain %q", tt.args, stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("run(%v) stderr = %q, want it to contain %q", tt.args, stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestRunRecovered_RecoversPanicAsExitCode1WithAPlainErrorLine(t *testing.T) {
	var stderr bytes.Buffer
	code := runRecovered(&stderr, func() int { panic("boom") })
	if code != 1 {
		t.Errorf("runRecovered with a panic = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "conjecture: internal error: boom") {
		t.Errorf("stderr = %q, want a plain internal-error line, not a raw stack trace", stderr.String())
	}
}

func TestRunRecovered_NoPanicPassesTheExitCodeThrough(t *testing.T) {
	var stderr bytes.Buffer
	code := runRecovered(&stderr, func() int { return 3 })
	if code != 3 {
		t.Errorf("runRecovered with no panic = %d, want 3 (fn's own exit code, untouched)", code)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty when fn does not panic", stderr.String())
	}
}
