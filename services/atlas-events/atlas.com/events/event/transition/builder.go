package transition

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder provides fluent construction of occurrence transition models.
type Builder struct {
	id               uuid.UUID
	occurrenceId     uuid.UUID
	fromStage        string
	toStage          string
	occurredAt       time.Time
	triggerType      string
	triggerReference string
}

// NewBuilder creates a new builder with the required parameters. OccurredAt
// defaults to time.Now(); a trigger type must be set via SetTrigger before
// Build (FR-T3).
func NewBuilder(occurrenceId uuid.UUID, toStage string) *Builder {
	return &Builder{
		id:           uuid.New(),
		occurrenceId: occurrenceId,
		toStage:      toStage,
		occurredAt:   time.Now(),
	}
}

// SetId sets the transition id.
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetFromStage sets the prior stage. Left empty for the creation row, which
// has no prior stage (FR-T1).
func (b *Builder) SetFromStage(fromStage string) *Builder {
	b.fromStage = fromStage
	return b
}

// SetOccurredAt sets the timestamp the transition occurred at.
func (b *Builder) SetOccurredAt(occurredAt time.Time) *Builder {
	b.occurredAt = occurredAt
	return b
}

// SetTrigger sets the trigger type and reference that caused the transition.
func (b *Builder) SetTrigger(triggerType string, triggerReference string) *Builder {
	b.triggerType = triggerType
	b.triggerReference = triggerReference
	return b
}

// Build validates invariants and constructs the final immutable model.
func (b *Builder) Build() (Model, error) {
	if b.toStage == "" {
		return Model{}, errors.New("toStage is required")
	}
	if b.triggerType == "" {
		return Model{}, errors.New("triggerType is required")
	}

	return Model{
		id:               b.id,
		occurrenceId:     b.occurrenceId,
		fromStage:        b.fromStage,
		toStage:          b.toStage,
		occurredAt:       b.occurredAt,
		triggerType:      b.triggerType,
		triggerReference: b.triggerReference,
	}, nil
}

// Builder returns a builder initialized with the current model's values.
func (m Model) Builder() *Builder {
	return &Builder{
		id:               m.id,
		occurrenceId:     m.occurrenceId,
		fromStage:        m.fromStage,
		toStage:          m.toStage,
		occurredAt:       m.occurredAt,
		triggerType:      m.triggerType,
		triggerReference: m.triggerReference,
	}
}
