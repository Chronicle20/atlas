// Package projection is the consumer-side mirror of atlas-configurations'
// transactional outbox: it consumes the tenant config-status topic,
// maintains an in-memory snapshot of the desired tenant config, and gates
// readiness on a one-shot end-offset catch-up. Unlike atlas-login's
// projection it tracks tenants only — this service runs no per-tenant
// socket listeners, so the service-config half is intentionally absent.
package projection

import (
	"encoding/json"
	"errors"
)

// TenantEnvelope is the wire shape published by atlas-configurations'
// outbox for a tenant config-status event. Kept in sync via the
// schema_version field.
type TenantEnvelope struct {
	SchemaVersion int             `json:"schema_version"`
	Id            string          `json:"id"`
	Config        json.RawMessage `json:"config"`
	EmittedAt     string          `json:"emitted_at"`
}

// ErrUnsupportedSchema is returned when the envelope's schema_version is
// higher than this projection understands. Subscribers log and skip
// rather than crash — a forward-compatible reader is a feature.
var ErrUnsupportedSchema = errors.New("projection: unsupported envelope schema_version")

// SupportedSchemaVersion is the highest envelope schema this projection
// can decode. Held in lockstep with atlas-login/atlas-channel and
// atlas-configurations/outbox.CurrentSchemaVersion.
const SupportedSchemaVersion = 1

// IsTombstone reports whether the kafka message is a log-compaction
// tombstone (nil value). Tombstones drive tenant removal.
func IsTombstone(value []byte) bool { return value == nil }

// DecodeTenantEnvelope decodes the wire bytes. Callers should check
// IsTombstone before calling. Rejects schema_version > SupportedSchemaVersion.
func DecodeTenantEnvelope(value []byte) (TenantEnvelope, error) {
	var env TenantEnvelope
	if err := json.Unmarshal(value, &env); err != nil {
		return TenantEnvelope{}, err
	}
	if env.SchemaVersion > SupportedSchemaVersion {
		return env, ErrUnsupportedSchema
	}
	return env, nil
}
