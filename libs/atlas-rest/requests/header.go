package requests

import (
	"context"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type HeaderDecorator func(header http.Header)

//goland:noinspection GoUnusedExportedFunction
func SpanHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func(h http.Header) {
		carrier := propagation.MapCarrier{}
		propagator := otel.GetTextMapPropagator()
		propagator.Inject(ctx, carrier)
		for _, k := range carrier.Keys() {
			h.Set(k, carrier.Get(k))
		}
	}
}

//goland:noinspection GoUnusedExportedFunction
func TenantHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func(h http.Header) {
		h.Set("Content-Type", "application/json; charset=utf-8")

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			return
		}

		h.Set(tenant.ID, t.Id().String())
		h.Set(tenant.Region, t.Region())
		h.Set(tenant.MajorVersion, strconv.Itoa(int(t.MajorVersion())))
		h.Set(tenant.MinorVersion, strconv.Itoa(int(t.MinorVersion())))
	}
}

// EnvHeaderDecorator sets the ENVIRONMENT header from the operation's
// context. Set centrally so no service sets it by hand (FR-3.1); a service
// handling a request for pr-123 emits pr-123 on every downstream call
// regardless of which deployment it is (FR-3.2).
func EnvHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func(h http.Header) {
		e, _ := env.FromContext(ctx)()
		if e == "" {
			return // legacy: no header, byte-identical to today
		}
		h.Set(env.Key, string(e))
	}
}
