package server_test

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func (ms *MockSpan) End(options ...trace.SpanEndOption) {
}

func (ms *MockSpan) RecordError(err error, options ...trace.EventOption) {
	// You can record the error or count calls here
}

type MockTracer struct {
	trace.Tracer
	StartedSpans []*MockSpan
}

func (mt *MockTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
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

func (m MockTracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
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

	var called = false

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

	var called = false

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
