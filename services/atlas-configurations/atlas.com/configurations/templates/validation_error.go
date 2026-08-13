package templates

import (
	"atlas-configurations/templates/characters/preset"
	"fmt"

	configsocket "atlas-configurations/socket"
)

// validationFailureError carries both families of blocking validation failure:
// preset issues (which need an atlas-data client and so arrive via the injected
// validator) and socket issues (pure, always run). Both render through the same
// JSON:API error shape; only the meta.path differs.
type validationFailureError struct {
	errors       []preset.ValidationError
	socketIssues []configsocket.Issue
}

func (e *validationFailureError) Error() string {
	return fmt.Sprintf("validation failed (%d preset, %d socket issues)", len(e.errors), len(e.socketIssues))
}

type jsonapiError struct {
	Status string         `json:"status"`
	Title  string         `json:"title"`
	Detail string         `json:"detail"`
	Meta   map[string]any `json:"meta"`
}

func (e *validationFailureError) AsJSONAPIErrors() []jsonapiError {
	out := make([]jsonapiError, 0, len(e.errors)+len(e.socketIssues))
	for _, ve := range e.errors {
		out = append(out, jsonapiError{
			Status: "400",
			Title:  "validation failed",
			Detail: ve.Message,
			Meta:   map[string]any{"path": "presets[" + ve.PresetId + "]." + ve.Field},
		})
	}
	for _, iss := range e.socketIssues {
		out = append(out, jsonapiError{
			Status: "400",
			Title:  "validation failed",
			Detail: iss.Message,
			Meta:   map[string]any{"path": iss.Path},
		})
	}
	return out
}
