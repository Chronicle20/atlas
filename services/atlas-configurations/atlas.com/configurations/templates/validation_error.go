package templates

import (
	"atlas-configurations/templates/characters/preset"
	"fmt"
)

type validationFailureError struct {
	errors []preset.ValidationError
}

func (e *validationFailureError) Error() string {
	return fmt.Sprintf("preset validation failed (%d issues)", len(e.errors))
}

type jsonapiError struct {
	Status string         `json:"status"`
	Title  string         `json:"title"`
	Detail string         `json:"detail"`
	Meta   map[string]any `json:"meta"`
}

func (e *validationFailureError) AsJSONAPIErrors() []jsonapiError {
	out := make([]jsonapiError, 0, len(e.errors))
	for _, ve := range e.errors {
		out = append(out, jsonapiError{
			Status: "400",
			Title:  "validation failed",
			Detail: ve.Message,
			Meta:   map[string]any{"path": "presets[" + ve.PresetId + "]." + ve.Field},
		})
	}
	return out
}
