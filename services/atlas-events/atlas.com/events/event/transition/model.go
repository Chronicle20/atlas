package transition

import (
	"time"

	"github.com/google/uuid"
)

// Model is an immutable representation of a single occurrence stage
// transition — one row in the occurrence's history.
type Model struct {
	id               uuid.UUID
	occurrenceId     uuid.UUID
	fromStage        string
	toStage          string
	occurredAt       time.Time
	triggerType      string
	triggerReference string
}

func (m Model) Id() uuid.UUID            { return m.id }
func (m Model) OccurrenceId() uuid.UUID  { return m.occurrenceId }
func (m Model) FromStage() string        { return m.fromStage }
func (m Model) ToStage() string          { return m.toStage }
func (m Model) OccurredAt() time.Time    { return m.occurredAt }
func (m Model) TriggerType() string      { return m.triggerType }
func (m Model) TriggerReference() string { return m.triggerReference }
