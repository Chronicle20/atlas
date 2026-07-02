package server

import (
	"encoding/json"
	"fmt"
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

// WriteBadRequest writes a JSON:API error object with HTTP 400.
func WriteBadRequest(l logrus.FieldLogger, w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	body := fmt.Sprintf(`{"errors":[{"status":"400","title":"Bad Request","detail":%q}]}`, detail)
	if _, err := w.Write([]byte(body)); err != nil {
		l.WithError(err).Errorf("Unable to write error response.")
	}
}
