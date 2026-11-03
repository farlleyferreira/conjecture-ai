package cjtr

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Observation is a .cjtr file's top-level "observation" mapping: the
// contract operation whose call this file's rules attach to, and the
// interface that calls it. The specification's worked example declares
// one per file ("operation: the shipping fee is calculated", "interface:
// HTTP POST /orders/shipping").
type Observation struct {
	// Operation names the observation operation, exactly as declared - a
	// rule's own "when" (see [Rule.When]) names this same operation to say
	// it is called by it (grammar table: "when" -> "<observation
	// operation>" -> "Chamada pelo contrato").
	Operation string
	// Interface is the external entry point that triggers the operation
	// (e.g. "HTTP POST /orders/shipping").
	Interface string
}

// ReadObservation reads a strictly-loaded .cjtr document's top-level
// "observation" mapping, if present ("" -> nil, nil: not every file
// declares one, and nothing currently requires it).
func ReadObservation(root *yamlstrict.Node) (*Observation, error) {
	node, ok := root.Get("observation")
	if !ok {
		return nil, nil
	}
	if node.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: \"observation\" must be a mapping, got %v", node.Kind)
	}

	obs := &Observation{}

	if opNode, ok := node.Get("operation"); ok {
		if opNode.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("cjtr: \"observation\": \"operation\" must be a scalar")
		}
		obs.Operation = opNode.Value
	}

	if ifaceNode, ok := node.Get("interface"); ok {
		if ifaceNode.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("cjtr: \"observation\": \"interface\" must be a scalar")
		}
		obs.Interface = ifaceNode.Value
	}

	return obs, nil
}
