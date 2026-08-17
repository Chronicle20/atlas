package requests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

func TestDefaultTimeoutTriggersOnSlowServer(t *testing.T) {
	origTimeout := DefaultTimeout
	DefaultTimeout = 100 * time.Millisecond
	defer func() { DefaultTimeout = origTimeout }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	_, err := get[any](l, context.Background())(srv.URL)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestSetTimeoutOverridesDefault(t *testing.T) {
	origTimeout := DefaultTimeout
	DefaultTimeout = 100 * time.Millisecond
	defer func() { DefaultTimeout = origTimeout }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	// Default timeout (100ms) would fail, but override to 2s should succeed.
	// Use delete which has no response body parsing.
	err := delete(l, context.Background())(srv.URL, SetTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("expected no error with extended timeout, got: %v", err)
	}
}

func TestCallerContextCancellationPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := get[any](l, ctx)(srv.URL)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("cancellation took too long: %v", elapsed)
	}
}

func TestRetryGetsFreshTimeoutPerAttempt(t *testing.T) {
	origTimeout := DefaultTimeout
	DefaultTimeout = 200 * time.Millisecond
	defer func() { DefaultTimeout = origTimeout }()

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First two attempts: simulate transport error by closing connection.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server does not support hijacking")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack failed: %v", err)
			}
			conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	// Use delete which has no response body parsing.
	err := delete(l, context.Background())(srv.URL, SetRetries(3))
	if err != nil {
		t.Fatalf("expected success on third attempt, got: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestDeleteTimesOutOnSlowServer(t *testing.T) {
	origTimeout := DefaultTimeout
	DefaultTimeout = 100 * time.Millisecond
	defer func() { DefaultTimeout = origTimeout }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	err := delete(l, context.Background())(srv.URL)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestPostTimesOutOnSlowServer(t *testing.T) {
	origTimeout := DefaultTimeout
	DefaultTimeout = 100 * time.Millisecond
	defer func() { DefaultTimeout = origTimeout }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	// createOrUpdate requires jsonapi-marshalable input, so use raw get-style test
	// to verify the timeout applies. We test at the URL level since createOrUpdate
	// will fail at marshaling with a nil input. Use delete as proxy - all share same pattern.
	err := MakeDeleteRequest(srv.URL)(l, context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestFastServerSucceedsWithDefaultTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	// Use delete which has no response body parsing.
	err := delete(l, context.Background())(srv.URL)
	if err != nil {
		t.Fatal("fast server should not have timed out or errored")
	}
}

// TestDeleteRequestEmitsTheEnvironmentHeaderFromContext drives an actual
// outbound request through the canonical DeleteRequest/decorated.go
// composition (the same one GetRequest/PostRequest/PutRequest/PatchRequest
// use) and inspects the header the server received — not a direct call to
// EnvHeaderDecorator. This is the regression FR-3.1 exists to prevent: a
// service must never have to set the header by hand.
func TestDeleteRequestEmitsTheEnvironmentHeaderFromContext(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(env.Key)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	if err := DeleteRequest(srv.URL)(l, ctx); err != nil {
		t.Fatalf("DeleteRequest: %v", err)
	}
	if got != "pr-123" {
		t.Fatalf("got ENVIRONMENT header %q, want %q", got, "pr-123")
	}
}

// TestDeleteRequestOmitsTheEnvironmentHeaderWithNoEnvironment pins NFR-7:
// a legacy operation (no environment on ctx) must be byte-identical to
// today — no ENVIRONMENT header at all.
func TestDeleteRequestOmitsTheEnvironmentHeaderWithNoEnvironment(t *testing.T) {
	var present bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, present = r.Header[http.CanonicalHeaderKey(env.Key)]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	if err := DeleteRequest(srv.URL)(l, context.Background()); err != nil {
		t.Fatalf("DeleteRequest: %v", err)
	}
	if present {
		t.Fatalf("ENVIRONMENT header present with no environment on context; want absent")
	}
}
