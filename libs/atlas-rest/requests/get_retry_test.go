package requests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type retryTestRestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

func (r retryTestRestModel) GetName() string { return "tests" }
func (r retryTestRestModel) GetID() string   { return r.Id }
func (r *retryTestRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func retryQuietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestGetRetries503ThenSucceeds(t *testing.T) {
	body, err := jsonapi.Marshal(retryTestRestModel{Id: "1", Name: "x"})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	res, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if err != nil {
		t.Fatalf("expected success after 503 retry, got: %v", err)
	}
	if res.Name != "x" {
		t.Fatalf("unexpected response: %+v", res)
	}
	if attempts.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestGetExhausted503ReturnsSentinel(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("expected ErrServiceUnavailable, got: %v", err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts (new GET default), got %d", attempts.Load())
	}
}

func TestGetHonorsRetryAfter(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNotFound) // terminal, ends the request quickly
	}))
	defer srv.Close()

	start := time.Now()
	_, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound terminal, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 1*time.Second {
		t.Fatalf("Retry-After not honored, elapsed %v", elapsed)
	}
}

func TestGetCapsRetryAfterAtMaxDelay(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	start := time.Now()
	_, _ = get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("Retry-After not capped at 2s MaxDelay, elapsed %v", elapsed)
	}
}

func TestDelete503IsNotRetried(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := delete(retryQuietLogger(), context.Background())(srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("non-GET must not map to ErrServiceUnavailable, got: %v", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("non-GET 503 must not be retried, got %d attempts", attempts.Load())
	}
}

// Regression matrix: every non-503 status behaves exactly as before —
// single attempt, same error identity.
func TestGetNon503StatusesUnchanged(t *testing.T) {
	tests := []struct {
		status  int
		want    error // nil means "any non-nil generic error" for 500
		attempt int32
	}{
		{http.StatusBadRequest, ErrBadRequest, 1},
		{http.StatusNotFound, ErrNotFound, 1},
		{http.StatusInternalServerError, nil, 1},
		{http.StatusBadGateway, nil, 1},
	}
	for _, tc := range tests {
		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(tc.status)
		}))
		_, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
		srv.Close()
		if err == nil {
			t.Fatalf("status %d: expected error", tc.status)
		}
		if tc.want != nil && !errors.Is(err, tc.want) {
			t.Fatalf("status %d: expected %v, got %v", tc.status, tc.want, err)
		}
		if tc.want == nil && errors.Is(err, ErrServiceUnavailable) {
			t.Fatalf("status %d: must not be ErrServiceUnavailable", tc.status)
		}
		if attempts.Load() != tc.attempt {
			t.Fatalf("status %d: expected %d attempts, got %d", tc.status, tc.attempt, attempts.Load())
		}
	}
}
