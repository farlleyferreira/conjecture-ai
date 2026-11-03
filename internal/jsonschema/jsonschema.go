// Package jsonschema validates a JSON document against a JSON Schema
// document - the other "standard result format" the language-agnostic-
// contracts capability names alongside JUnit XML and TAP (openspec/specs/
// language-agnostic-contracts/spec.md: "The system SHALL accept target
// output in standard formats, including JSON validated against a
// schema").
//
// JSON Schema itself is a large, pre-existing specification (keywords,
// multiple drafts, $ref resolution, format/content assertions) that this
// package does not reimplement. It wraps
// github.com/santhosh-tekuri/jsonschema/v6, a third-party library that
// passes the official JSON-Schema-Test-Suite across every current draft -
// the same choice this project already made for YAML (gopkg.in/yaml.v3,
// underneath internal/yamlstrict's own strict-profile logic): use a
// conformant implementation of an external standard rather than hand-roll
// one.
package jsonschema

import (
	"bytes"
	"fmt"

	upstream "github.com/santhosh-tekuri/jsonschema/v6"
)

// Validate reports whether instance (a JSON document's raw bytes)
// conforms to schema (a JSON Schema document's raw bytes).
//
// It returns nil when instance conforms. Otherwise it returns the
// underlying library's own *jsonschema.ValidationError (wrapped error
// values use a different kind for a malformed schema or instance, so a
// caller distinguishing "the target's output didn't validate" from "the
// schema or output wasn't even valid JSON" can tell them apart), whose
// Error() gives a human-readable description and whose fields (schema
// location, instance location, nested causes) a caller may inspect
// further.
func Validate(schema, instance []byte) error {
	compiler := upstream.NewCompiler()

	schemaDoc, err := upstream.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return fmt.Errorf("jsonschema: schema is not valid JSON: %w", err)
	}
	if err := compiler.AddResource("schema.json", schemaDoc); err != nil {
		return fmt.Errorf("jsonschema: schema: %w", err)
	}

	sch, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("jsonschema: compiling schema: %w", err)
	}

	instanceDoc, err := upstream.UnmarshalJSON(bytes.NewReader(instance))
	if err != nil {
		return fmt.Errorf("jsonschema: instance is not valid JSON: %w", err)
	}

	return sch.Validate(instanceDoc)
}
