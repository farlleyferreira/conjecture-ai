package cjtr

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// ReadSimplifications reads a strictly-loaded .cjtr document's top-level
// "simplifications" list, if present (nil, nil if absent: not every file
// declares one). The field-reference table calls these "Model
// simplifications" - declarative documentation, like [Scope] and
// [Hypothesis.Decisions]: plain declared text, never a [Statement].
func ReadSimplifications(root *yamlstrict.Node) ([]string, error) {
	node, ok := root.Get("simplifications")
	if !ok {
		return nil, nil
	}
	if node.Kind != yamlstrict.SequenceKind {
		return nil, fmt.Errorf("cjtr: \"simplifications\" must be a list, got %v", node.Kind)
	}

	items := make([]string, 0, len(node.Items))
	for _, item := range node.Items {
		if item.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("cjtr: every \"simplifications\" entry must be a scalar")
		}
		items = append(items, item.Value)
	}
	return items, nil
}
