package consumer

import (
	"context"
	"encoding/binary"
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

const (
	ID           = "TENANT_ID"
	Region       = "REGION"
	MajorVersion = "MAJOR_VERSION"
	MinorVersion = "MINOR_VERSION"
)

//goland:noinspection GoUnusedExportedFunction
func TenantHeaderParser(ctx context.Context, headers []kafka.Header) context.Context {
	var wctx = ctx
	for _, header := range headers {
		if header.Key == ID {
			if len(header.Value) == 16 {
				val, err := uuid.FromBytes(header.Value)
				if err == nil {
					wctx = context.WithValue(wctx, ID, val)
				}
			}
			continue
		}
		if header.Key == Region {
			wctx = context.WithValue(wctx, Region, string(header.Value))
			continue
		}
		if header.Key == MajorVersion {
			if len(header.Value) == 2 {
				wctx = context.WithValue(wctx, MajorVersion, binary.BigEndian.Uint16(header.Value))
				continue
			}
		}
		if header.Key == MinorVersion {
			if len(header.Value) == 2 {
				wctx = context.WithValue(wctx, MinorVersion, binary.BigEndian.Uint16(header.Value))
				continue
			}
		}
	}
	return wctx
}
