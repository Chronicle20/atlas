package rest

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"net/http"
)

type SpanHandler func(logrus.FieldLogger, context.Context) http.HandlerFunc

func RetrieveSpan(l logrus.FieldLogger, name string, next SpanHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propagator := otel.GetTextMapPropagator()
		sctx := propagator.Extract(context.Background(), propagation.HeaderCarrier(r.Header))
		sctx, span := otel.GetTracerProvider().Tracer("atlas-rest").Start(sctx, name)
		sl := l.WithField("trace.id", span.SpanContext().TraceID().String()).WithField("span.id", span.SpanContext().SpanID().String())
		defer span.End()
		next(sl, sctx)(w, r)
	}
}
