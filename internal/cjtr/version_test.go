package cjtr

import (
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Version
		wantErr bool
	}{
		{"simple", "1.0.0", Version{1, 0, 0}, false},
		{"all nonzero", "1.2.3", Version{1, 2, 3}, false},
		{"missing component", "1.2", Version{}, true},
		{"extra component", "1.2.3.4", Version{}, true},
		{"non-numeric", "1.a.3", Version{}, true},
		{"empty", "", Version{}, true},
		{"negative", "1.-2.3", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.in)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseVersion(%q) = %v, want an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseVersion(%q) returned unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseVersion(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	got := Version{Major: 1, Minor: 2, Patch: 3}.String()
	want := "1.2.3"
	if got != want {
		t.Errorf("Version{1,2,3}.String() = %q, want %q", got, want)
	}
}

func TestCheckSupported(t *testing.T) {
	tests := []struct {
		name    string
		v       Version
		wantErr bool
	}{
		{"matching major, same minor/patch", Version{SupportedMajor, 0, 0}, false},
		{"matching major, higher minor", Version{SupportedMajor, 9, 9}, false},
		{"higher major", Version{SupportedMajor + 1, 0, 0}, true},
		{"lower major", Version{SupportedMajor - 1, 0, 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSupported(tt.v)
			if tt.wantErr && err == nil {
				t.Errorf("CheckSupported(%v) = nil, want an error", tt.v)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("CheckSupported(%v) = %v, want nil", tt.v, err)
			}
		})
	}
}

func TestFormatVersion(t *testing.T) {
	root, err := yamlstrict.Load(strings.NewReader("conjecture: 1.0.0\narea: shipping\n"))
	if err != nil {
		t.Fatalf("yamlstrict.Load returned unexpected error: %v", err)
	}

	got, err := FormatVersion(root)
	if err != nil {
		t.Fatalf("FormatVersion returned unexpected error: %v", err)
	}
	if want := (Version{1, 0, 0}); got != want {
		t.Errorf("FormatVersion = %+v, want %+v", got, want)
	}
}

func TestFormatVersion_Missing(t *testing.T) {
	root, err := yamlstrict.Load(strings.NewReader("area: shipping\n"))
	if err != nil {
		t.Fatalf("yamlstrict.Load returned unexpected error: %v", err)
	}

	_, err = FormatVersion(root)
	if err == nil {
		t.Fatal("FormatVersion with no \"conjecture\" key = nil error, want an error")
	}
}
