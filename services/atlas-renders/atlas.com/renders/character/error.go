package character

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// ErrorBody is the JSON:API errors-array entry shape, ported from the donor
// service so error responses match the original characterrender contract.
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
// errors are unusual on the render path and intentionally not supported.
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

// Sentinel error types surfaced by the compositor. These mirror the donor's
// characterimage errors so callers can branch on them when crafting
// user-facing responses.
var (
	ErrInvalidStance   = errors.New("character: invalid stance")
	ErrUnknownSkin     = errors.New("character: unknown skin id")
	ErrFrameOutOfRange = errors.New("character: frame out of range")
	ErrAssetMissing    = errors.New("character: asset missing")
)
