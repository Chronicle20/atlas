package producer

import (
	"context"
	"encoding/binary"
	"github.com/Chronicle20/atlas-tenant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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
