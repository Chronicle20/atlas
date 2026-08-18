package producer

import (
	"context"
	"encoding/binary"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type HeaderDecorator func() (map[string]string, error)

//goland:noinspection GoUnusedExportedFunction
func SpanHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func() (map[string]string, error) {
		headers := make(map[string]string)

		carrier := propagation.MapCarrier{}
		propagator := otel.GetTextMapPropagator()
		propagator.Inject(ctx, carrier)
		for _, k := range carrier.Keys() {
			headers[k] = carrier.Get(k)
		}
		return headers, nil
	}
}

//goland:noinspection GoUnusedExportedFunction
func TenantHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func() (map[string]string, error) {
		headers := make(map[string]string)
		t, err := tenant.FromContext(ctx)()
		if err != nil {
			return headers, nil
		}
		headers[tenant.ID] = t.Id().String()
		headers[tenant.Region] = t.Region()
		headers[tenant.MajorVersion] = string(binary.BigEndian.AppendUint16(make([]byte, 0), t.MajorVersion()))
		headers[tenant.MinorVersion] = string(binary.BigEndian.AppendUint16(make([]byte, 0), t.MinorVersion()))
		return headers, nil
	}
}

// EnvHeaderDecorator writes the operation's environment as a message
// header, mirroring TenantHeaderDecorator. Message BODIES are untouched —
// no domain schema changes anywhere, which is what keeps the 64-service
// migration mechanical (design §6.1).
//
//goland:noinspection GoUnusedExportedFunction
func EnvHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func() (map[string]string, error) {
		headers := make(map[string]string)
		e, _ := env.FromContext(ctx)()
		if e == "" {
			return headers, nil
		}
		headers[env.Key] = string(e)
		return headers, nil
	}
}
