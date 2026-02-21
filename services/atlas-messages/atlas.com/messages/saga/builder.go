package saga

import (
	"errors"
	"fmt"

	sharedsaga "github.com/Chronicle20/atlas-saga"
	"github.com/google/uuid"
)

// Builder wraps the shared saga builder and adds validation on Build().
type Builder struct {
	b *sharedsaga.Builder
}

// NewBuilder creates a new Builder instance with default values
func NewBuilder() *Builder {
	return &Builder{
		b: sharedsaga.NewBuilder(),
	}
}

// SetTransactionId sets the transaction ID for the saga
func (b *Builder) SetTransactionId(transactionId uuid.UUID) *Builder {
	b.b.SetTransactionId(transactionId)
	return b
}

// SetSagaType sets the saga type
func (b *Builder) SetSagaType(sagaType Type) *Builder {
	b.b.SetSagaType(sagaType)
	return b
}

// SetInitiatedBy sets who initiated the saga
func (b *Builder) SetInitiatedBy(initiatedBy string) *Builder {
	b.b.SetInitiatedBy(initiatedBy)
	return b
}

// AddStep adds a step to the saga
func (b *Builder) AddStep(stepId string, status Status, action Action, payload any) *Builder {
	b.b.AddStep(stepId, status, action, payload)
	return b
}

// Build constructs and returns a new Saga instance with validation
func (b *Builder) Build() (Saga, error) {
	s := b.b.Build()
	if len(s.Steps) == 0 {
		return Saga{}, errors.New("saga must have at least one step")
	}
	if s.SagaType == "" {
		return Saga{}, errors.New("saga type is required")
	}
	if s.InitiatedBy == "" {
		return Saga{}, errors.New("initiatedBy is required")
	}
	for i, step := range s.Steps {
		if step.Action == "" {
			return Saga{}, fmt.Errorf("step %d has invalid action", i)
		}
	}
	return s, nil
}
