package consumer

import (
	"context"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// EnvHeaderParser reads the ENVIRONMENT message header onto the context and
// reconciles it against the tenant already on the context (FR-7.7). It must
// be registered AFTER TenantHeaderParser so the tenant is present when it
// reconciles.
//
// A mismatch cannot be returned — the HeaderParser signature is
// context-in/context-out — so it is recorded on the context and the
// ownership gate drops the message with the alertable counter.
//
//goland:noinspection GoUnusedExportedFunction
func EnvHeaderParser(ctx context.Context, headers []kafka.Header) context.Context {
	var id env.Id
	for _, h := range headers {
		if h.Key == env.Key {
			id = env.Id(h.Value)
			break
		}
	}

	tenantId := ""
	if t, err := tenant.FromContext(ctx)(); err == nil {
		tenantId = t.Id().String()
	}

	resolved, err := env.Reconcile(env.CurrentRegistry(), id, tenantId)
	if err != nil {
		return env.WithMismatch(env.WithContext(ctx, id))
	}
	return env.WithContext(ctx, resolved)
}
