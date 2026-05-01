package characterrender

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ErrorBody is the JSON:API errors-array entry shape.
type ErrorBody struct {
	Code   string         `json:"code"`
	Title  string         `json:"title"`
	Detail string         `json:"detail,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type wireError struct {
	Status string         `json:"status"`
	Code   string         `json:"code"`
	Title  string         `json:"title"`
	Detail string         `json:"detail,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

// WriteError serialises one error and sets the response status. Multiple
// errors are unusual on the render path and intentionally not supported here.
func WriteError(w http.ResponseWriter, status int, body ErrorBody) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Errors []wireError `json:"errors"`
	}{
		Errors: []wireError{{
			Status: strconv.Itoa(status),
			Code:   body.Code,
			Title:  body.Title,
			Detail: body.Detail,
			Meta:   body.Meta,
		}},
	})
}
