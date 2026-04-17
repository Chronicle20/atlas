package saga

import (
	"time"

	"github.com/google/uuid"
)

// Builder is a builder for constructing Saga models
type Builder struct {
	transactionId uuid.UUID
	sagaType      Type
	initiatedBy   string
	timeout       time.Duration
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

// SetTimeout sets the per-saga timeout. Zero/negative means "let the
// orchestrator apply its default" (30s; see the orchestrator's DefaultSagaTimeout).
func (b *Builder) SetTimeout(timeout time.Duration) *Builder {
	b.timeout = timeout
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

// Build constructs and returns a new Saga instance
func (b *Builder) Build() Saga {
	var timeoutMs int64
	if b.timeout > 0 {
		timeoutMs = b.timeout.Milliseconds()
	}
	return Saga{
		TransactionId: b.transactionId,
		SagaType:      b.sagaType,
		InitiatedBy:   b.initiatedBy,
		Timeout:       timeoutMs,
		Steps:         b.steps,
	}
}
