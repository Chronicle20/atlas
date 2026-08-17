// Package outbox contains the wire-level envelope shape used when
// atlas-configurations emits service+tenant config updates through the
// transactional outbox. The envelope is schema-versioned so subscribers
// (atlas-channel, atlas-login) can evolve independently.
package outbox

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CurrentSchemaVersion is the current envelope schema version. Bump when
// making backwards-incompatible field changes; subscribers gate on this
// value to skip messages they cannot decode.
const CurrentSchemaVersion = 1

type envelope struct {
	SchemaVersion int    `json:"schema_version"`
	Id            string `json:"id"`
	Config        any    `json:"config"`
	EmittedAt     string `json:"emitted_at"`
}

// NewServiceEnvelope serializes a service config update for the
// EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS topic. The config argument is
// the REST model the service should be reconstructed from.
func NewServiceEnvelope(id uuid.UUID, config any, emittedAt time.Time) ([]byte, error) {
	return json.Marshal(envelope{
		SchemaVersion: CurrentSchemaVersion,
		Id:            id.String(),
		Config:        config,
		EmittedAt:     emittedAt.UTC().Format(time.RFC3339),
	})
}

// NewTenantEnvelope serializes a tenant config update for the
// EVENT_TOPIC_CONFIGURATION_TENANT_STATUS topic. Tenant and service
// envelopes share a shape today; kept as a separate constructor so they
// can diverge without ricochet.
func NewTenantEnvelope(id uuid.UUID, config any, emittedAt time.Time) ([]byte, error) {
	return NewServiceEnvelope(id, config, emittedAt)
}

// NewEnvironmentEnvelope serializes an environment record for the
// EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS topic. Shares the envelope
// shape with services and tenants; kept as its own constructor so it can
// diverge without ricochet. Unlike the other two, the id is the
// environment's Name (its wire identity), not a generated uuid.
func NewEnvironmentEnvelope(name string, config any, emittedAt time.Time) ([]byte, error) {
	return json.Marshal(envelope{
		SchemaVersion: CurrentSchemaVersion,
		Id:            name,
		Config:        config,
		EmittedAt:     emittedAt.UTC().Format(time.RFC3339),
	})
}
