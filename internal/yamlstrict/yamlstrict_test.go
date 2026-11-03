package yamlstrict

import (
	"strings"
	"testing"
)

func mustLoad(t *testing.T, src string) *Node {
	t.Helper()
	n, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load(%q) returned unexpected error: %v", src, err)
	}
	return n
}

func TestLoad_ScalarPreservedAsRawText(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"exact decimal", "amount: 100.00", "100.00"},
		{"would-be bool no", "flag: no", "no"},
		{"would-be bool yes", "flag: yes", "yes"},
		{"plain string", "name: shipping", "shipping"},
		{"integer-looking", "count: 007", "007"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := mustLoad(t, tt.src)

			var got string
			for _, p := range root.Pairs {
				got = p.Value.String()
			}

			if got != tt.want {
				t.Errorf("scalar = %q, want %q (implicit YAML typing must not kick in)", got, tt.want)
			}
		})
	}
}

func TestLoad_NestedStructure(t *testing.T) {
	src := `
area: shipping
rules:
  - name: free-above
    threshold: 100.00
`
	root := mustLoad(t, src)

	area, ok := root.Get("area")
	if !ok || area.String() != "shipping" {
		t.Fatalf("area = %v, ok=%v, want %q", area, ok, "shipping")
	}

	rules, ok := root.Get("rules")
	if !ok || rules.Kind != SequenceKind || len(rules.Items) != 1 {
		t.Fatalf("rules = %+v, ok=%v, want a one-item sequence", rules, ok)
	}

	rule := rules.Items[0]
	name, _ := rule.Get("name")
	threshold, _ := rule.Get("threshold")
	if name.String() != "free-above" || threshold.String() != "100.00" {
		t.Errorf("rule = {name: %q, threshold: %q}, want {free-above, 100.00}", name.String(), threshold.String())
	}
}

func TestLoad_RejectsAnchor(t *testing.T) {
	src := "base: &b\n  x: 1\nother: *b\n"

	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatal("Load() with an anchor succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "anchor") {
		t.Errorf("Load() error = %v, want it to mention an anchor", err)
	}
}

func TestLoad_RejectsAlias(t *testing.T) {
	src := "base: &b\n  x: 1\nother: *b\n"

	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatal("Load() with an alias succeeded, want an error")
	}
}

func TestLoad_RejectsCustomTag(t *testing.T) {
	src := "value: !mytag something\n"

	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatal("Load() with a custom tag succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "custom YAML tag") {
		t.Errorf("Load() error = %v, want it to mention a custom tag", err)
	}
}

func TestLoad_RejectsDuplicateKey(t *testing.T) {
	src := "name: a\nname: b\n"

	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatal("Load() with a duplicate key succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("Load() error = %v, want it to mention a duplicate key", err)
	}
}

func TestLoad_RejectsDuplicateKeyNested(t *testing.T) {
	src := "rule:\n  threshold: 1\n  threshold: 2\n"

	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatal("Load() with a nested duplicate key succeeded, want an error")
	}
}

func TestLoad_NullScalar(t *testing.T) {
	root := mustLoad(t, "value: null")

	val, ok := root.Get("value")
	if !ok {
		t.Fatal("value not found")
	}
	if val.Kind != NullKind {
		t.Errorf("value.Kind = %v, want NullKind", val.Kind)
	}
}

func TestNode_GetMissingKey(t *testing.T) {
	root := mustLoad(t, "a: 1")

	_, ok := root.Get("missing")
	if ok {
		t.Error("Get(\"missing\") ok = true, want false")
	}
}
