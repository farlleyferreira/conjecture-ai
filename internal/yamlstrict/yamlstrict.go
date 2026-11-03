// Package yamlstrict parses a restricted subset of YAML: no anchors, no
// aliases (references), no custom tags, and no duplicate mapping keys.
// Every scalar is returned as its raw source text, never converted through
// YAML's own implicit type resolution (so "100.00" stays the string
// "100.00", not a lossy binary float). Callers are expected to type each
// scalar themselves, based on what they expect at that position.
package yamlstrict

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kind identifies what a [Node] holds.
type Kind int

const (
	// ScalarKind is a single value, held as raw text in Node.Value.
	ScalarKind Kind = iota
	// MappingKind is an ordered set of key/value pairs, held in Node.Pairs.
	MappingKind
	// SequenceKind is an ordered list of nodes, held in Node.Items.
	SequenceKind
	// NullKind is an explicit YAML null (`~`, `null`, or an empty value).
	NullKind
)

// Node is one parsed YAML node under the strict profile.
type Node struct {
	Kind   Kind
	Value  string  // set when Kind == ScalarKind
	Pairs  []Pair  // set when Kind == MappingKind, in source order
	Items  []*Node // set when Kind == SequenceKind
	Line   int
	Column int
}

// Pair is one key/value entry of a [MappingKind] node, in source order.
type Pair struct {
	Key    string
	Value  *Node
	Line   int
	Column int
}

// Error reports a position in the source document that violates the strict
// profile (an anchor, an alias, a custom tag, or a duplicate mapping key).
type Error struct {
	Reason string
	Line   int
	Column int
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Reason)
}

// Load parses r under the strict profile described in the package doc.
func Load(r io.Reader) (*Node, error) {
	var doc yaml.Node
	if err := yaml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, &Error{Reason: "empty document", Line: doc.Line, Column: doc.Column}
	}

	return convert(doc.Content[0])
}

func convert(n *yaml.Node) (*Node, error) {
	if err := reject(n); err != nil {
		return nil, err
	}

	switch n.Kind {
	case yaml.ScalarNode:
		return convertScalar(n)
	case yaml.MappingNode:
		return convertMapping(n)
	case yaml.SequenceNode:
		return convertSequence(n)
	default:
		return nil, &Error{
			Reason: fmt.Sprintf("unsupported node kind %d", n.Kind),
			Line:   n.Line,
			Column: n.Column,
		}
	}
}

// reject fails the node if it uses a feature the strict profile disallows:
// an anchor definition, an alias/reference, or a non-standard tag.
func reject(n *yaml.Node) error {
	if n.Anchor != "" {
		return &Error{
			Reason: fmt.Sprintf("YAML anchors are not allowed (found &%s)", n.Anchor),
			Line:   n.Line,
			Column: n.Column,
		}
	}
	if n.Kind == yaml.AliasNode {
		return &Error{Reason: "YAML aliases/references are not allowed", Line: n.Line, Column: n.Column}
	}
	if !isStandardTag(n.Tag) {
		return &Error{
			Reason: fmt.Sprintf("custom YAML tags are not allowed (found %s)", n.Tag),
			Line:   n.Line,
			Column: n.Column,
		}
	}
	return nil
}

// standardTags are the resolved tags yaml.v3 assigns without any custom tag
// in the source. Anything else (e.g. "!mytag") is a custom tag.
var standardTags = map[string]bool{
	"!!str":       true,
	"!!bool":      true,
	"!!int":       true,
	"!!float":     true,
	"!!null":      true,
	"!!timestamp": true,
	"!!binary":    true,
	"!!map":       true,
	"!!seq":       true,
}

func isStandardTag(tag string) bool {
	return standardTags[tag]
}

func convertScalar(n *yaml.Node) (*Node, error) {
	if n.Tag == "!!null" {
		return &Node{Kind: NullKind, Line: n.Line, Column: n.Column}, nil
	}
	return &Node{Kind: ScalarKind, Value: n.Value, Line: n.Line, Column: n.Column}, nil
}

func convertSequence(n *yaml.Node) (*Node, error) {
	items := make([]*Node, 0, len(n.Content))
	for _, c := range n.Content {
		item, err := convert(c)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &Node{Kind: SequenceKind, Items: items, Line: n.Line, Column: n.Column}, nil
}

func convertMapping(n *yaml.Node) (*Node, error) {
	if len(n.Content)%2 != 0 {
		return nil, &Error{Reason: "malformed mapping", Line: n.Line, Column: n.Column}
	}

	pairs := make([]Pair, 0, len(n.Content)/2)
	seen := make(map[string]int, len(n.Content)/2) // key -> line it was first seen on

	for i := 0; i < len(n.Content); i += 2 {
		keyNode := n.Content[i]
		valNode := n.Content[i+1]

		if err := reject(keyNode); err != nil {
			return nil, err
		}
		if keyNode.Kind != yaml.ScalarNode {
			return nil, &Error{Reason: "mapping keys must be scalars", Line: keyNode.Line, Column: keyNode.Column}
		}

		key := keyNode.Value
		if firstLine, ok := seen[key]; ok {
			return nil, &Error{
				Reason: fmt.Sprintf("duplicate key %q (first seen on line %d)", key, firstLine),
				Line:   keyNode.Line,
				Column: keyNode.Column,
			}
		}
		seen[key] = keyNode.Line

		val, err := convert(valNode)
		if err != nil {
			return nil, err
		}

		pairs = append(pairs, Pair{Key: key, Value: val, Line: keyNode.Line, Column: keyNode.Column})
	}

	return &Node{Kind: MappingKind, Pairs: pairs, Line: n.Line, Column: n.Column}, nil
}

// Get returns the value of the first pair in a [MappingKind] node whose key
// equals key, and reports whether it was found.
func (n *Node) Get(key string) (*Node, bool) {
	if n == nil || n.Kind != MappingKind {
		return nil, false
	}
	for _, p := range n.Pairs {
		if p.Key == key {
			return p.Value, true
		}
	}
	return nil, false
}

// String renders a scalar node's raw value, or "" for anything else -
// mainly useful in tests and error messages.
func (n *Node) String() string {
	if n == nil {
		return ""
	}
	if n.Kind == ScalarKind {
		return n.Value
	}
	return strings.ToUpper(fmt.Sprintf("<%v>", n.Kind))
}
