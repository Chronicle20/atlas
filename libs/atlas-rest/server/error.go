package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// TransientRetryAfterSeconds is the Retry-After value (seconds) sent with 503
// responses produced by WriteErrorResponse for transient errors.
const TransientRetryAfterSeconds = 1

var transientClassifier atomic.Pointer[func(error) bool]

// RegisterTransientErrorClassifier installs the process-wide predicate used
// by WriteErrorResponse to map errors to 503 Service Unavailable. Typically
// called once from main.go:
//
//	server.RegisterTransientErrorClassifier(func(err error) bool {
//		if database.IsTransientConnectionError(err) {
//			database.CountTransient(err)
//			return true
//		}
//		return false
//	})
//
// Passing nil clears the classifier (everything maps to 500).
func RegisterTransientErrorClassifier(f func(error) bool) {
	transientClassifier.Store(&f)
}

type errorObject struct {
	Status string `json:"status"`
	Title  string `json:"title"`
}

type errorDocument struct {
	Errors []errorObject `json:"errors"`
}

// WriteErrorResponse maps err to a JSON:API error response. Errors the
// registered classifier reports as transient produce
// 503 + Retry-After: TransientRetryAfterSeconds; everything else produces
// 500. With no classifier registered, every error maps to 500.
func WriteErrorResponse(l logrus.FieldLogger) func(w http.ResponseWriter) func(err error) {
	return func(w http.ResponseWriter) func(err error) {
		return func(err error) {
			status := http.StatusInternalServerError
			title := "internal server error"
			if fp := transientClassifier.Load(); fp != nil && *fp != nil && (*fp)(err) {
				status = http.StatusServiceUnavailable
				title = "temporarily unavailable"
				w.Header().Set("Retry-After", strconv.Itoa(TransientRetryAfterSeconds))
			}
			l.WithError(err).Warnf("Writing [%d] error response.", status)
			w.WriteHeader(status)
			doc := errorDocument{Errors: []errorObject{{Status: strconv.Itoa(status), Title: title}}}
			if encodeErr := json.NewEncoder(w).Encode(doc); encodeErr != nil {
				l.WithError(encodeErr).Errorf("Encoding error response body.")
			}
		}
	}
}

// badRequestError is the JSON:API error object shape emitted by
// WriteBadRequest: {"errors":[{"status":"400","title":"Bad Request","detail":"..."}]}.
type badRequestError struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type badRequestBody struct {
	Errors []badRequestError `json:"errors"`
}

// WriteBadRequest writes a JSON:API error object with HTTP 400.
func WriteBadRequest(l logrus.FieldLogger, w http.ResponseWriter, detail string) {
	writeBadRequest(l, w, detail, "application/json")
}

// writeBadRequestJSONAPI is writeBadRequest's vnd.api+json-negotiated
// sibling, used by ParseInput's failure paths (context.go), whose endpoints
// otherwise emit application/vnd.api+json for every response, success or
// error.
func writeBadRequestJSONAPI(l logrus.FieldLogger, w http.ResponseWriter, detail string) {
	writeBadRequest(l, w, detail, "application/vnd.api+json")
}

// writeBadRequest writes the shared JSON:API error object shape used by
// WriteBadRequest and writeBadRequestJSONAPI, differing only in the
// Content-Type header they negotiate.
func writeBadRequest(l logrus.FieldLogger, w http.ResponseWriter, detail string, contentType string) {
	body, err := json.Marshal(badRequestBody{Errors: []badRequestError{{
		Status: "400",
		Title:  "Bad Request",
		Detail: detail,
	}}})
	if err != nil {
		l.WithError(err).Errorf("Unable to marshal error response.")
		body = []byte(`{"errors":[{"status":"400","title":"Bad Request","detail":"invalid request"}]}`)
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusBadRequest)
	if _, err := w.Write(body); err != nil {
		l.WithError(err).Errorf("Unable to write error response.")
	}
}
