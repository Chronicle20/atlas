package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
)

type errorObject struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type errorDocument struct {
	Errors []errorObject `json:"errors"`
}

// WriteError writes a JSON:API error document with an arbitrary status.
//
// server.WriteErrorResponse covers 500/503 and server.WriteBadRequest covers
// 400; the coupon admin surface additionally needs 404, 409 (duplicate code,
// delete with redemptions) and 422 (every malformed-bundle rejection), which
// are ordinary outcomes of a well-formed request rather than server faults.
// The document shape matches WriteBadRequest's so a client parses one form.
func WriteError(l logrus.FieldLogger, w http.ResponseWriter, status int, detail string) {
	body, err := json.Marshal(errorDocument{Errors: []errorObject{{
		Status: strconv.Itoa(status),
		Title:  http.StatusText(status),
		Detail: detail,
	}}})
	if err != nil {
		l.WithError(err).Errorf("Unable to marshal error response.")
		body = []byte(`{"errors":[{"status":"500","title":"Internal Server Error"}]}`)
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		l.WithError(err).Errorf("Unable to write error response.")
	}
}
