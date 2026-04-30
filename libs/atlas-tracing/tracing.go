package tracing

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer configures and registers the global TracerProvider for the calling
// service. It reads TRACE_ENDPOINT (OTLP gRPC target) and TRACE_SAMPLING_RATIO
// from the environment.
//
// The returned *sdktrace.TracerProvider must be passed to Teardown for clean
// shutdown.
func InitTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(os.Getenv("TRACE_ENDPOINT")),
		),
	)
	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	ratio := parseSamplingRatio(logger)
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp, nil
}

// Teardown returns a curried shutdown closure compatible with the existing
// per-service main.go call sites:
//
//	tdm.TeardownFunc(tracing.Teardown(l)(tc))
func Teardown(l logrus.FieldLogger) func(tp *sdktrace.TracerProvider) func() {
	return func(tp *sdktrace.TracerProvider) func() {
		return func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				l.WithError(err).Errorf("Unable to close tracer.")
			}
		}
	}
}
