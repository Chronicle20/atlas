package definition

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Model is an immutable representation of an event definition. Configuration
// is opaque JSON — this package never interprets its contents (FR-D2).
type Model struct {
	id            uuid.UUID
	theType       string
	name          string
	enabled       bool
	configuration json.RawMessage
	createdAt     time.Time
	updatedAt     time.Time
}

func (m Model) Id() uuid.UUID                  { return m.id }
func (m Model) Type() string                   { return m.theType }
func (m Model) Name() string                   { return m.name }
func (m Model) Enabled() bool                  { return m.enabled }
func (m Model) Configuration() json.RawMessage { return m.configuration }
func (m Model) CreatedAt() time.Time           { return m.createdAt }
func (m Model) UpdatedAt() time.Time           { return m.updatedAt }
