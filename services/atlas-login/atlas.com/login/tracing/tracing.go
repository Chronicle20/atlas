package tracing

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"os"
	"time"
)

func InitTracer(l logrus.FieldLogger) func(serviceName string) (*trace.TracerProvider, error) {
	return func(serviceName string) (*trace.TracerProvider, error) {
		exporter, err := otlptrace.New(
			context.Background(),
			otlptracegrpc.NewClient(
				otlptracegrpc.WithInsecure(),
				otlptracegrpc.WithEndpoint(os.Getenv("JAEGER_HOST_PORT")),
			),
		)
		if err != nil {
			return nil, err
		}

		tp := trace.NewTracerProvider(
			trace.WithBatcher(exporter),
			trace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			)),
		)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.TraceContext{})
		return tp, nil
	}
}

func Teardown(l logrus.FieldLogger) func(tp *trace.TracerProvider) func() {
	return func(tp *trace.TracerProvider) func() {
		return func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				l.WithError(err).Errorf("Unable to close tracer.")
			}
		}
	}
}

type LogrusAdapter struct {
	logger logrus.FieldLogger
}

func (l LogrusAdapter) Error(msg string) {
	l.logger.Error(msg)
}

func (l LogrusAdapter) Infof(msg string, args ...interface{}) {
	l.logger.Infof(msg, args)
}
