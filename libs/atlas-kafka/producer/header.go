package producer

import (
	"context"
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
