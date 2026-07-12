package saga

import (
	"time"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/google/uuid"
)

// Builder wraps the shared saga builder. The list flow constructs its saga
// through this builder so the timeout is always set explicitly (the orchestrator
// processes steps serially over Kafka and a missing/flat timeout rolls back
// legitimate multi-step sagas — see the preset-creation timeout bug).
type Builder struct {
	b *sharedsaga.Builder
}

// NewBuilder creates a new Builder instance with default values.
func NewBuilder() *Builder {
	return &Builder{b: sharedsaga.NewBuilder()}
}

// SetTransactionId sets the transaction ID for the saga.
func (b *Builder) SetTransactionId(transactionId uuid.UUID) *Builder {
	b.b.SetTransactionId(transactionId)
	return b
}

// SetSagaType sets the saga type.
func (b *Builder) SetSagaType(sagaType Type) *Builder {
	b.b.SetSagaType(sagaType)
	return b
}

// SetInitiatedBy sets who initiated the saga.
func (b *Builder) SetInitiatedBy(initiatedBy string) *Builder {
	b.b.SetInitiatedBy(initiatedBy)
	return b
}

// SetTimeout sets the per-saga timeout.
func (b *Builder) SetTimeout(timeout time.Duration) *Builder {
	b.b.SetTimeout(timeout)
	return b
}

// AddStep adds a step to the saga.
func (b *Builder) AddStep(stepId string, status Status, action Action, payload any) *Builder {
	b.b.AddStep(stepId, status, action, payload)
	return b
}

// Build constructs and returns the Saga.
func (b *Builder) Build() Saga {
	return b.b.Build()
}
