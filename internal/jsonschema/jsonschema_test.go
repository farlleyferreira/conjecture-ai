package jsonschema

import "testing"

const feeSchema = `{
  "type": "object",
  "properties": {
    "shipping_fee": {"type": "number", "minimum": 0}
  },
  "required": ["shipping_fee"]
}`

func TestValidate_Valid(t *testing.T) {
	err := Validate([]byte(feeSchema), []byte(`{"shipping_fee": 7.5}`))
	if err != nil {
		t.Errorf("Validate returned unexpected error: %v", err)
	}
}

func TestValidate_MissingRequiredProperty(t *testing.T) {
	err := Validate([]byte(feeSchema), []byte(`{}`))
	if err == nil {
		t.Fatal("Validate with a missing required property = nil error, want an error")
	}
}

func TestValidate_ViolatesMinimum(t *testing.T) {
	err := Validate([]byte(feeSchema), []byte(`{"shipping_fee": -1}`))
	if err == nil {
		t.Fatal("Validate with shipping_fee below the schema's minimum = nil error, want an error")
	}
}

func TestValidate_WrongType(t *testing.T) {
	err := Validate([]byte(feeSchema), []byte(`{"shipping_fee": "not a number"}`))
	if err == nil {
		t.Fatal("Validate with shipping_fee as a string = nil error, want an error")
	}
}

func TestValidate_MalformedSchema(t *testing.T) {
	err := Validate([]byte("not json"), []byte(`{}`))
	if err == nil {
		t.Fatal("Validate with a malformed schema = nil error, want an error")
	}
}

func TestValidate_MalformedInstance(t *testing.T) {
	err := Validate([]byte(feeSchema), []byte("not json"))
	if err == nil {
		t.Fatal("Validate with a malformed instance = nil error, want an error")
	}
}

func TestValidate_SchemaWithInvalidKeywordUsage(t *testing.T) {
	// "type" must be a string or array of strings, not a number - this
	// must fail at schema-compile time, not instance-validate time.
	err := Validate([]byte(`{"type": 1}`), []byte(`{}`))
	if err == nil {
		t.Fatal("Validate with an invalid schema = nil error, want an error")
	}
}
