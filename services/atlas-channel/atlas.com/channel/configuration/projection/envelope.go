// Package projection is the consumer-side mirror of atlas-configurations'
// transactional outbox: it consumes the service+tenant config-status
// topics, maintains an in-memory snapshot of the desired state, gates
// readiness on a one-shot end-offset catch-up, and diffs successive
// snapshots into Add/Drain ops that the apply loop hands to
// listener.Registry.
package projection

import (
	"encoding/json"
	"errors"
)

// ServiceEnvelope is the wire shape published by atlas-configurations'
// outbox.NewServiceEnvelope. Kept in sync via the schema_version field.
type ServiceEnvelope struct {
	SchemaVersion int             `json:"schema_version"`
	Id            string          `json:"id"`
	Config        json.RawMessage `json:"config"`
	EmittedAt     string          `json:"emitted_at"`
}

// TenantEnvelope mirrors ServiceEnvelope; same wire shape today.
type TenantEnvelope = ServiceEnvelope

// ErrUnsupportedSchema is returned when the envelope's schema_version is
// higher than this projection understands. Subscribers should log and
// skip rather than crash — a forward-compatible reader is a feature.
var ErrUnsupportedSchema = errors.New("projection: unsupported envelope schema_version")

// SupportedSchemaVersion is the highest envelope schema this projection
// can decode. Bumped in lockstep with atlas-configurations/outbox.CurrentSchemaVersion
// when the wire shape changes.
const SupportedSchemaVersion = 1

// IsTombstone reports whether the kafka message is a log-compaction
// tombstone (nil value). Tombstones drive removal in the projection
// state.
func IsTombstone(value []byte) bool { return value == nil }

// DecodeServiceEnvelope decodes the wire bytes. Empty/nil bytes mean a
// tombstone — callers should check IsTombstone before calling.
func DecodeServiceEnvelope(value []byte) (ServiceEnvelope, error) {
	var env ServiceEnvelope
	if err := json.Unmarshal(value, &env); err != nil {
		return ServiceEnvelope{}, err
	}
	if env.SchemaVersion > SupportedSchemaVersion {
		return env, ErrUnsupportedSchema
	}
	return env, nil
}

// DecodeTenantEnvelope is a thin alias kept symmetrical to the
// envelope+constructor split on the producer side. Same shape today.
func DecodeTenantEnvelope(value []byte) (TenantEnvelope, error) {
	return DecodeServiceEnvelope(value)
}
