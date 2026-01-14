package saga

import (
	"github.com/google/uuid"
)

// Builder is a builder for constructing Saga models
type Builder struct {
	transactionId uuid.UUID
	sagaType      Type
	initiatedBy   string
	steps         []Step[any]
}

// NewBuilder creates a new Builder instance with default values
func NewBuilder() *Builder {
	return &Builder{
		transactionId: uuid.New(),
		steps:         make([]Step[any], 0),
	}
}

// Clone creates a new Builder pre-populated with values from an existing Saga
func Clone(s Saga) *Builder {
	steps := make([]Step[any], len(s.steps))
	copy(steps, s.steps)
	return &Builder{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		steps:         steps,
	}
}

// SetTransactionId sets the transaction ID for the saga
func (b *Builder) SetTransactionId(transactionId uuid.UUID) *Builder {
	b.transactionId = transactionId
	return b
}

// SetSagaType sets the saga type
func (b *Builder) SetSagaType(sagaType Type) *Builder {
	b.sagaType = sagaType
	return b
}

// SetInitiatedBy sets who initiated the saga
func (b *Builder) SetInitiatedBy(initiatedBy string) *Builder {
	b.initiatedBy = initiatedBy
	return b
}

// SetSteps replaces all steps in the saga
func (b *Builder) SetSteps(steps []Step[any]) *Builder {
	b.steps = make([]Step[any], len(steps))
	copy(b.steps, steps)
	return b
}

// AddStep adds a step to the saga
func (b *Builder) AddStep(stepId string, status Status, action Action, payload any) *Builder {
	step := NewStep(stepId, status, action, payload)
	b.steps = append(b.steps, step)
	return b
}

// Build constructs and returns a new Saga instance with validation
func (b *Builder) Build() (Saga, error) {
	if b.transactionId == uuid.Nil {
		return Saga{}, ErrEmptyTransactionId
	}
	if b.sagaType == "" {
		return Saga{}, ErrEmptySagaType
	}

	return Saga{
		transactionId: b.transactionId,
		sagaType:      b.sagaType,
		initiatedBy:   b.initiatedBy,
		steps:         b.steps,
	}, nil
}
