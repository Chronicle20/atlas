package server_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

type MockSpan struct {
	trace.Span
	spanContext trace.SpanContext
}

func (ms *MockSpan) SpanContext() trace.SpanContext {
	return ms.spanContext
}

func (ms *MockSpan) IsRecording() bool {
	return true
}

func (ms *MockSpan) End(_ ...trace.SpanEndOption) {
}

func (ms *MockSpan) RecordError(_ error, _ ...trace.EventOption) {
	// You can record the error or count calls here
}

type MockTracer struct {
	trace.Tracer
	StartedSpans []*MockSpan
}

func (mt *MockTracer) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
	})
	mockSpan := &MockSpan{spanContext: spanContext}
	return trace.ContextWithSpan(ctx, mockSpan), mockSpan
}

type MockTracerProvider struct {
	trace.TracerProvider
	tracer *MockTracer
}

func (m MockTracerProvider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer {
	if m.tracer == nil {
		m.tracer = &MockTracer{}
	}
	return m.tracer
}

func TestSpanPropagation(t *testing.T) {
	l, _ := test.NewNullLogger()

	otel.SetTracerProvider(&MockTracerProvider{})
	otel.SetTextMapPropagator(propagation.TraceContext{})

	ictx, ispan := otel.GetTracerProvider().Tracer("atlas-kafka").Start(context.Background(), "test-span")

	req, err := http.NewRequest(http.MethodGet, "www.google.com", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	w := httptest.NewRecorder()

	requests.SpanHeaderDecorator(ictx)(req.Header)

	server.RetrieveSpan(l, "test-handler", context.Background(), func(l logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
		span := trace.SpanFromContext(ctx)
		if !span.SpanContext().TraceID().IsValid() {
			t.Fatal(errors.New("invalid trace id").Error())
		}
		if span.SpanContext().TraceID() != ispan.SpanContext().TraceID() {
			t.Fatal(errors.New("invalid trace id").Error())
		}
		return func(w http.ResponseWriter, r *http.Request) {
		}
	})(w, req)
}

func TestNullSpanPropagation(t *testing.T) {
	l, _ := test.NewNullLogger()

	otel.SetTracerProvider(&MockTracerProvider{})
	otel.SetTextMapPropagator(propagation.TraceContext{})

	req, err := http.NewRequest(http.MethodGet, "www.google.com", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	w := httptest.NewRecorder()

	requests.SpanHeaderDecorator(context.Background())(req.Header)

	called := false

	server.RetrieveSpan(l, "test-handler", context.Background(), func(l logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
		called = true
		span := trace.SpanFromContext(ctx)
		if !span.SpanContext().TraceID().IsValid() {
			t.Fatal(errors.New("invalid trace id").Error())
		}
		return func(w http.ResponseWriter, r *http.Request) {
		}
	})(w, req)

	if !called {
		t.Fatal(errors.New("invalid trace").Error())
	}
}

func TestTenantPropagation(t *testing.T) {
	l, _ := test.NewNullLogger()
	uuid := uuid.New()
	region := "GMS"
	majorVersion := uint16(83)
	minorVersion := uint16(1)

	it, err := tenant.Create(uuid, region, majorVersion, minorVersion)
	if err != nil {
		t.Fatal(err.Error())
	}
	ictx := tenant.WithContext(context.Background(), it)

	ctxId := ictx.Value(tenant.ID)
	if ctxId != uuid {
		t.Fatal(errors.New("invalid tenant id").Error())
	}
	ctxRegion := ictx.Value(tenant.Region)
	if ctxRegion != region {
		t.Fatal(errors.New("invalid tenant region").Error())
	}
	ctxMajorVersion := ictx.Value(tenant.MajorVersion)
	if ctxMajorVersion != majorVersion {
		t.Fatal(errors.New("invalid tenant major version").Error())
	}
	ctxMinorVersion := ictx.Value(tenant.MinorVersion)
	if ctxMinorVersion != minorVersion {
		t.Fatal(errors.New("invalid tenant minor version").Error())
	}

	req, err := http.NewRequest(http.MethodGet, "www.google.com", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	w := httptest.NewRecorder()

	requests.TenantHeaderDecorator(ictx)(req.Header)

	called := false

	server.ParseTenant(l, context.Background(), func(l logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
		called = true
		ot, err := tenant.FromContext(tctx)()
		if err != nil {
			t.Fatal(err.Error())
		}

		if !it.Is(ot) {
			t.Fatal(errors.New("invalid tenant").Error())
		}
		return func(w http.ResponseWriter, r *http.Request) {
		}
	})(w, req)

	if !called {
		t.Fatal(errors.New("invalid tenant").Error())
	}
}

func TestParseEnvironmentPutsTheHeaderOnTheContext(t *testing.T) {
	var got env.Id
	h := server.ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				got = env.MustFromContext(ctx)
				w.WriteHeader(http.StatusOK)
			}
		})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(env.Key, "pr-123")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\"", got)
	}
}

func TestParseEnvironmentWithNoHeaderIsTheLegacyPath(t *testing.T) {
	// FR-1.8 / NFR-7: an unheadered request is exactly today's request.
	called := false
	h := server.ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				called = true
				if got := env.MustFromContext(ctx); got != env.Id("") {
					t.Errorf("environment = %q, want the empty id", got)
				}
				w.WriteHeader(http.StatusOK)
			}
		})

	h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("handler not reached; an unheadered request must pass through")
	}
}

func TestParseEnvironmentRejectsAnUnknownEnvironment(t *testing.T) {
	// FR-3.6: a request naming an unknown or inactive environment is
	// rejected. Never served by the baseline (G4).
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive,
	})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	h := server.ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, _ context.Context) http.HandlerFunc {
			return func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler reached for an unknown environment")
			}
		})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(env.Key, "pr-999")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestParseEnvironmentAdmitsAProvisioningEnvironment(t *testing.T) {
	// A known environment still in PROVISIONING must be admitted so it can
	// perform self-writes (e.g. atlas-pr-bootstrap's service-config rows)
	// before the ACTIVE flip. Confinement to its own data is enforced
	// downstream by the scope layer, not by this gate.
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive,
	})
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseProvisioning,
	})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	called := false
	h := server.ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				called = true
				if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
					t.Errorf("environment = %q, want \"pr-123\"", got)
				}
				w.WriteHeader(http.StatusOK)
			}
		})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(env.Key, "pr-123")
	w := httptest.NewRecorder()
	h(w, r)

	if !called {
		t.Fatal("handler not reached for a PROVISIONING environment")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestParseEnvironmentRejectsADeactivatingEnvironment(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseDeactivating,
	})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	h := server.ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, _ context.Context) http.HandlerFunc {
			return func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler reached for a DEACTIVATING environment")
			}
		})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(env.Key, "pr-123")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestParseEnvironmentRejectsADeletedEnvironment(t *testing.T) {
	// Apply with PhaseDeleted removes the record, so a DELETED environment
	// is indistinguishable from unknown to the registry — both are rejected.
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseDeleted,
	})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	h := server.ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, _ context.Context) http.HandlerFunc {
			return func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler reached for a DELETED environment")
			}
		})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(env.Key, "pr-123")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestParseTenantRejectsAMismatchedEnvironment(t *testing.T) {
	// FR-7.7: a header environment that disagrees with the tenant's
	// registered environment is rejected outright; the handler is never
	// reached.
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive,
	})
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseActive,
	})
	id := uuid.New()
	reg.ApplyTenant(id.String(), env.Id("pr-123"))
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	it, err := tenant.Create(id, "GMS", uint16(83), uint16(1))
	if err != nil {
		t.Fatal(err.Error())
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	requests.TenantHeaderDecorator(tenant.WithContext(context.Background(), it))(r.Header)
	w := httptest.NewRecorder()

	// Simulate ParseEnvironment having already put the (mismatched) header
	// environment on the context, as register.go composes it.
	ctx := env.WithContext(context.Background(), env.Id("not-pr-123"))

	server.ParseTenant(testLogger(t), ctx, func(_ logrus.FieldLogger, _ context.Context) http.HandlerFunc {
		return func(http.ResponseWriter, *http.Request) {
			t.Fatal("handler reached for a mismatched environment")
		}
	})(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
