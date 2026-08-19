package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

type EnvironmentHandler func(logrus.FieldLogger, context.Context) http.HandlerFunc

// ParseEnvironment reads the ENVIRONMENT header onto the context. An absent
// header is the legacy value and passes through unchanged (FR-1.8). A
// present header naming an environment the registry does not know, or knows
// as DEACTIVATING or DELETED, is rejected with 400 — never served by the
// baseline. A known environment in PROVISIONING or ACTIVE is admitted: this
// gate only puts the id on the context, it does not grant broad access.
// PROVISIONING must be admitted so an environment can write its own rows
// while it is still being set up (e.g. atlas-pr-bootstrap's service-config
// self-writes). Confinement to the caller's own data is enforced downstream
// by the scope layer (scope.Strict on reads, scope.AuthorizeWrite on
// writes), not by this handler. Traffic ownership is separate and still
// governed by Registry.IsOwner, which keeps requiring ACTIVE (FR-5.2).
//
//goland:noinspection GoUnusedExportedFunction
func ParseEnvironment(l logrus.FieldLogger, ctx context.Context, next EnvironmentHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := env.Id(r.Header.Get(env.Key))
		if id != "" && !env.CurrentRegistry().IsProvisionable(id) {
			l.WithField(env.Key, string(id)).Error("Request names an unknown or inactive environment.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		el := l
		if id != "" {
			el = l.WithField("environment", string(id))
		}
		next(el, env.WithContext(ctx, id))(w, r)
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

		resolved, err := env.Reconcile(env.CurrentRegistry(), env.MustFromContext(tctx), t.Id().String())
		if err != nil {
			l.WithError(err).Error("Environment header disagrees with the tenant's environment.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tctx = env.WithContext(tctx, resolved)
		next(tl, tctx)(w, r)
	}
}
