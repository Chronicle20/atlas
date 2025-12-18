package server

import (
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"net/http"
	"strconv"
)

type SpanHandler func(logrus.FieldLogger, context.Context) http.HandlerFunc

//goland:noinspection GoUnusedExportedFunction
func RetrieveSpan(l logrus.FieldLogger, name string, ctx context.Context, next SpanHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propagator := otel.GetTextMapPropagator()
		sctx := propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))
		sctx, span := otel.GetTracerProvider().Tracer("atlas-rest").Start(sctx, name)
		sl := l.WithField("trace.id", span.SpanContext().TraceID().String()).WithField("span.id", span.SpanContext().SpanID().String())
		defer span.End()
		next(sl, sctx)(w, r)
	}
}

type TenantHandler func(logrus.FieldLogger, context.Context) http.HandlerFunc

//goland:noinspection GoUnusedExportedFunction
func ParseTenant(l logrus.FieldLogger, ctx context.Context, next TenantHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.Header.Get(tenant.ID)
		if idStr == "" {
			l.Errorf("%s is not supplied.", tenant.ID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			l.Errorf("%s is not supplied.", tenant.ID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		region := r.Header.Get(tenant.Region)
		if region == "" {
			l.Errorf("%s is not supplied.", tenant.Region)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		majorVersion := r.Header.Get(tenant.MajorVersion)
		if majorVersion == "" {
			l.Errorf("%s is not supplied.", tenant.MajorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		majorVersionVal, err := strconv.Atoi(majorVersion)
		if err != nil {
			l.Errorf("%s is not supplied.", tenant.MajorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		minorVersion := r.Header.Get(tenant.MinorVersion)
		if minorVersion == "" {
			l.Errorf("%s is not supplied.", tenant.MinorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		minorVersionVal, err := strconv.Atoi(minorVersion)
		if err != nil {
			l.Errorf("%s is not supplied.", tenant.MinorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		tl := l.
			WithField("tenant", id.String()).
			WithField("region", region).
			WithField("ms.version", fmt.Sprintf("%d.%d", majorVersionVal, minorVersionVal))

		t, err := tenant.Create(id, region, uint16(majorVersionVal), uint16(minorVersionVal))
		if err != nil {
			l.Errorf("Failed to create tenant with provided data.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		tctx := tenant.WithContext(ctx, t)
		next(tl, tctx)(w, r)
	}
}
