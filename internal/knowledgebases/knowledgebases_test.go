package knowledgebases

import "testing"

func TestParseAuthority(t *testing.T) {
	tests := []struct {
		s    string
		want Authority
	}{
		{"norm", Norm},
		{"definition", Definition},
		{"reference", Reference},
		{"history", History},
	}
	for _, tt := range tests {
		got, err := ParseAuthority(tt.s)
		if err != nil {
			t.Errorf("ParseAuthority(%q) returned unexpected error: %v", tt.s, err)
		}
		if got != tt.want {
			t.Errorf("ParseAuthority(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestParseAuthority_Unrecognized(t *testing.T) {
	_, err := ParseAuthority("opinion")
	if err == nil {
		t.Fatal("ParseAuthority(\"opinion\") = nil error, want an error")
	}
}

func TestAuthority_String(t *testing.T) {
	for _, a := range []Authority{Norm, Definition, Reference, History} {
		s := a.String()
		parsed, err := ParseAuthority(s)
		if err != nil {
			t.Fatalf("ParseAuthority(%q) returned unexpected error: %v", s, err)
		}
		if parsed != a {
			t.Errorf("Authority(%d).String() = %q, which parses back to %v", int(a), s, parsed)
		}
	}
}

func TestAuthority_Outranks_NormOverReference(t *testing.T) {
	// The specification's own one explicit relationship.
	if !Norm.Outranks(Reference) {
		t.Error("Norm.Outranks(Reference) = false, want true")
	}
	if Reference.Outranks(Norm) {
		t.Error("Reference.Outranks(Norm) = true, want false")
	}
}

func TestAuthority_Outranks_FullOrder(t *testing.T) {
	order := []Authority{Norm, Definition, Reference, History}
	for i, higher := range order {
		for j, lower := range order {
			want := i < j
			if got := higher.Outranks(lower); got != want {
				t.Errorf("%v.Outranks(%v) = %v, want %v", higher, lower, got, want)
			}
		}
	}
}
