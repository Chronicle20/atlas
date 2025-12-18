package consumer

import (
	"context"
	"encoding/binary"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type HeaderParser func(ctx context.Context, headers []kafka.Header) context.Context

//goland:noinspection GoUnusedExportedFunction
func SpanHeaderParser(ctx context.Context, headers []kafka.Header) context.Context {
	carrier := propagation.MapCarrier{}
	for _, header := range headers {
		carrier[header.Key] = string(header.Value)
	}
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, carrier)
}

//goland:noinspection GoUnusedExportedFunction
func TenantHeaderParser(ctx context.Context, headers []kafka.Header) context.Context {
	var id uuid.UUID
	var region string
	var majorVersion uint16
	var minorVersion uint16

	for _, header := range headers {
		if header.Key == tenant.ID {
			if len(header.Value) == 36 {
				val, err := uuid.Parse(string(header.Value))
				if err == nil {
					id = val
				}
			}
			continue
		}
		if header.Key == tenant.Region {
			region = string(header.Value)
			continue
		}
		if header.Key == tenant.MajorVersion {
			if len(header.Value) == 2 {
				majorVersion = binary.BigEndian.Uint16(header.Value)
				continue
			}
		}
		if header.Key == tenant.MinorVersion {
			if len(header.Value) == 2 {
				minorVersion = binary.BigEndian.Uint16(header.Value)
				continue
			}
		}
	}
	t, err := tenant.Create(id, region, majorVersion, minorVersion)
	if err != nil {
		return ctx
	}
	return tenant.WithContext(ctx, t)
}
