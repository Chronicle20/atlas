package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compileSchema loads the reactor script schema from disk. The schema
// describes the bare script object (attributes), not the JSON:API envelope.
func compileSchema(path string) (*jsonschema.Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open schema: %w", err)
	}
	defer func() { _ = f.Close() }()

	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("reactor_script_schema.json", doc); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	return c.Compile("reactor_script_schema.json")
}

// validateAttributes validates one seed file's data.attributes object.
func validateAttributes(s *jsonschema.Schema, attrs json.RawMessage) error {
	var v any
	if err := json.Unmarshal(attrs, &v); err != nil {
		return fmt.Errorf("parse attributes: %w", err)
	}
	return s.Validate(v)
}
