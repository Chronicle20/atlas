package saga

import (
	"errors"
	"fmt"
	"time"

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

// AddStep adds a step to the saga
func (b *Builder) AddStep(stepId string, status Status, action Action, payload any) *Builder {
	now := time.Now()
	step := Step[any]{
		StepId:    stepId,
		Status:    status,
		Action:    action,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}
	b.steps = append(b.steps, step)
	return b
}

// validate checks builder invariants before construction
func (b *Builder) validate() error {
	if len(b.steps) == 0 {
		return errors.New("saga must have at least one step")
	}
	if b.sagaType == "" {
		return errors.New("saga type is required")
	}
	if b.initiatedBy == "" {
		return errors.New("initiatedBy is required")
	}
	for i, step := range b.steps {
		if step.Action == "" {
			return fmt.Errorf("step %d has invalid action", i)
		}
	}
	return nil
}

// Build constructs and returns a new Saga instance
func (b *Builder) Build() (Saga, error) {
	if err := b.validate(); err != nil {
		return Saga{}, err
	}
	return Saga{
		TransactionId: b.transactionId,
		SagaType:      b.sagaType,
		InitiatedBy:   b.initiatedBy,
		Steps:         b.steps,
	}, nil
}
