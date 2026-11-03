package cjtr

import "testing"

func TestValidSegment(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"shipping", true},
		{"free-shipping", true},
		{"above-threshold", true},
		{"a1-b2", true},
		{"", false},
		{"Free-Shipping", false},
		{"free_shipping", false},
		{"free shipping", false},
		{"-free-shipping", false},
		{"free-shipping-", false},
		{"free::shipping", false},
	}

	for _, tt := range tests {
		if got := ValidSegment(tt.in); got != tt.want {
			t.Errorf("ValidSegment(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestComposeID(t *testing.T) {
	got, err := ComposeID("shipping", "free-shipping", "free-above")
	if err != nil {
		t.Fatalf("ComposeID returned unexpected error: %v", err)
	}
	want := "shipping::free-shipping::free-above"
	if got != want {
		t.Errorf("ComposeID(...) = %q, want %q", got, want)
	}
}

func TestComposeID_Deterministic(t *testing.T) {
	a, err := ComposeID("shipping", "free-shipping")
	if err != nil {
		t.Fatalf("ComposeID returned unexpected error: %v", err)
	}
	b, err := ComposeID("shipping", "free-shipping")
	if err != nil {
		t.Fatalf("ComposeID returned unexpected error: %v", err)
	}
	if a != b {
		t.Errorf("ComposeID with the same names = %q and %q, want them equal", a, b)
	}
}

func TestComposeID_Errors(t *testing.T) {
	tests := [][]string{
		{},
		{"shipping", "Invalid Segment"},
		{"shipping", ""},
	}

	for _, parts := range tests {
		if _, err := ComposeID(parts...); err == nil {
			t.Errorf("ComposeID(%v) = nil error, want an error", parts)
		}
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		id     string
		prefix string
		want   bool
	}{
		{"shipping::free-shipping::free-above", "shipping::free-shipping::free-above", true},
		{"shipping::free-shipping::free-above", "shipping::free-shipping", true},
		{"shipping::free-shipping::free-above", "shipping", true},
		{"shipping::free-shipping-extra::x", "shipping::free-shipping", false},
		{"shipping::free-shipping", "billing", false},
	}

	for _, tt := range tests {
		if got := HasPrefix(tt.id, tt.prefix); got != tt.want {
			t.Errorf("HasPrefix(%q, %q) = %v, want %v", tt.id, tt.prefix, got, tt.want)
		}
	}
}
