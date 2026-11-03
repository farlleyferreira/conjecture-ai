package cjtr

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Scope is a .cjtr file's top-level "scope" mapping: what the phenomenon
// covers now (Inside), what it defers to later (Later), and what it
// excludes (Outside) - "System and boundary" in the specification's
// field table ("o que está dentro, adiado e fora"). This is declarative
// documentation, same as [Hypothesis.Decisions]: nothing here is a
// [Statement] or gets evaluated.
type Scope struct {
	Inside  []string
	Later   []string
	Outside []string
}

// ReadScope reads a strictly-loaded .cjtr document's top-level "scope"
// mapping, if present ("" -> nil, nil: not every file declares one).
func ReadScope(root *yamlstrict.Node) (*Scope, error) {
	node, ok := root.Get("scope")
	if !ok {
		return nil, nil
	}
	if node.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: \"scope\" must be a mapping, got %v", node.Kind)
	}

	scope := &Scope{}
	for _, field := range []struct {
		key string
		dst *[]string
	}{
		{"inside", &scope.Inside},
		{"later", &scope.Later},
		{"outside", &scope.Outside},
	} {
		listNode, ok := node.Get(field.key)
		if !ok {
			continue
		}
		items, err := scopeStringList(listNode, field.key)
		if err != nil {
			return nil, err
		}
		*field.dst = items
	}

	return scope, nil
}

func scopeStringList(node *yamlstrict.Node, key string) ([]string, error) {
	if node.Kind != yamlstrict.SequenceKind {
		return nil, fmt.Errorf("cjtr: \"scope\": %q must be a list, got %v", key, node.Kind)
	}
	items := make([]string, 0, len(node.Items))
	for _, item := range node.Items {
		if item.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("cjtr: \"scope\": every %q entry must be a scalar", key)
		}
		items = append(items, item.Value)
	}
	return items, nil
}
