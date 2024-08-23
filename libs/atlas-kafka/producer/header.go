package producer

import (
	"context"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"strconv"
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

const (
	ID           = "TENANT_ID"
	Region       = "REGION"
	MajorVersion = "MAJOR_VERSION"
	MinorVersion = "MINOR_VERSION"
)

//goland:noinspection GoUnusedExportedFunction
func TenantHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func() (map[string]string, error) {
		headers := make(map[string]string)
		if val, ok := ctx.Value(ID).(uuid.UUID); ok {
			headers[ID] = val.String()
		}
		if val, ok := ctx.Value(Region).(string); ok {
			headers[Region] = val
		}
		if val, ok := ctx.Value(MajorVersion).(uint16); ok {
			headers[MajorVersion] = strconv.Itoa(int(val))
		}
		if val, ok := ctx.Value(MinorVersion).(uint16); ok {
			headers[MinorVersion] = strconv.Itoa(int(val))
		}
		return headers, nil
	}
}
