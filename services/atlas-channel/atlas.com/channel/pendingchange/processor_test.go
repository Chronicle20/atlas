package pendingchange

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// withCharactersServiceURL points the CHARACTERS root url at the given test
// server for the duration of the test. t.Setenv restores the previous value
// automatically on cleanup.
func withCharactersServiceURL(t *testing.T, url string) {
	t.Helper()
	t.Setenv("CHARACTERS_SERVICE_URL", url+"/")
}

func newTestProcessor() Processor {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return NewProcessor(l, context.Background())
}

func TestRequestNameChange(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       any
		rawBody    string
		wantErr    *RejectedError
		wantOK     bool
	}{
		{
			name:       "201 success",
			statusCode: http.StatusCreated,
			body: map[string]any{
				"data": map[string]any{
					"type": "pending-changes",
					"id":   "1",
					"attributes": map[string]any{
						"characterId": 1,
						"type":        TypeNameChange,
						"status":      "PENDING",
					},
				},
			},
			wantOK: true,
		},
		{
			name:       "409 already_pending",
			statusCode: http.StatusConflict,
			body:       jsonAPIErrorBody("409", "Conflict", "already_pending"),
			wantErr:    &RejectedError{Status: http.StatusConflict, Reason: "already_pending"},
		},
		{
			// This is the discriminating case: it must fail against the
			// current (unfixed) code, which hard-codes "already_pending"
			// for every 409 regardless of the upstream detail.
			name:       "409 name_reserved",
			statusCode: http.StatusConflict,
			body:       jsonAPIErrorBody("409", "Conflict", "name_reserved"),
			wantErr:    &RejectedError{Status: http.StatusConflict, Reason: "name_reserved"},
		},
		{
			name:       "409 empty errors array",
			statusCode: http.StatusConflict,
			body:       map[string]any{"errors": []any{}},
			wantErr:    &RejectedError{Status: http.StatusConflict, Reason: ""},
		},
		{
			name:       "422 reason passed through verbatim",
			statusCode: http.StatusUnprocessableEntity,
			body:       jsonAPIErrorBody("422", "Unprocessable Entity", "invalid_reason_xyz"),
			wantErr:    &RejectedError{Status: http.StatusUnprocessableEntity, Reason: "invalid_reason_xyz"},
		},
		{
			name:       "404 empty detail",
			statusCode: http.StatusNotFound,
			body:       map[string]any{"errors": []any{}},
			wantErr:    &RejectedError{Status: http.StatusNotFound, Reason: "unknown_character"},
		},
		{
			name:       "non-JSON body on non-2xx",
			statusCode: http.StatusInternalServerError,
			rawBody:    "not json at all {{{",
			wantErr:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				if tc.rawBody != "" {
					_, _ = w.Write([]byte(tc.rawBody))
					return
				}
				if tc.body != nil {
					_ = json.NewEncoder(w).Encode(tc.body)
				}
			}))
			defer server.Close()

			withCharactersServiceURL(t, server.URL)

			p := newTestProcessor()
			result, err := p.RequestNameChange(1, "newname", 5000000)

			if tc.wantOK {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if result.Type != TypeNameChange {
					t.Fatalf("expected RestModel populated, got %+v", result)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected an error, got nil")
			}

			if tc.wantErr == nil {
				// Non-JSON-body-on-error case: just confirm we get an
				// error and did not panic. It need not be a
				// *RejectedError.
				return
			}

			rejected, ok := err.(*RejectedError)
			if !ok {
				t.Fatalf("expected *RejectedError, got %T: %v", err, err)
			}
			if rejected.Status != tc.wantErr.Status || rejected.Reason != tc.wantErr.Reason {
				t.Fatalf("expected %+v, got %+v", tc.wantErr, rejected)
			}
		})
	}
}

func jsonAPIErrorBody(status, title, detail string) map[string]any {
	return map[string]any{
		"errors": []any{
			map[string]any{
				"status": status,
				"title":  title,
				"detail": detail,
			},
		},
	}
}
