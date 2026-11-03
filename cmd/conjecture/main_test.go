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
