package saga

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

// Type represents the type of saga
type Type string

const (
	InventoryTransaction Type = "inventory_transaction"
)

// Status represents the status of a saga step
type Status string

const (
	Pending   Status = "pending"
	Completed Status = "completed"
	Failed    Status = "failed"
)

// Action represents an action type for saga steps
type Action string

const (
	AwardFame Action = "award_fame"
)

// Saga represents the entire saga transaction
type Saga struct {
	TransactionId uuid.UUID `json:"transactionId"`
	SagaType      Type      `json:"sagaType"`
	InitiatedBy   string    `json:"initiatedBy"`
	Steps         []Step    `json:"steps"`
}

// Step represents a single step within a saga
type Step struct {
	StepId  string `json:"stepId"`
	Status  Status `json:"status"`
	Action  Action `json:"action"`
	Payload any    `json:"payload"`
}

// AwardFamePayload is the payload for the award_fame action
type AwardFamePayload struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	Amount      int16      `json:"amount"`
}

// Builder helps construct sagas with multiple steps
type Builder struct {
	transactionId uuid.UUID
	sagaType      Type
	initiatedBy   string
	steps         []Step
	stepCounter   int
}

// NewBuilder creates a new saga builder
func NewBuilder() *Builder {
	return &Builder{
		transactionId: uuid.New(),
		steps:         make([]Step, 0),
		stepCounter:   0,
	}
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
	b.steps = append(b.steps, Step{
		StepId:  stepId,
		Status:  status,
		Action:  action,
		Payload: payload,
	})
	return b
}

// AddAwardFame adds a fame award step
func (b *Builder) AddAwardFame(characterId uint32, ch channel.Model, amount int16) *Builder {
	b.stepCounter++
	b.steps = append(b.steps, Step{
		StepId: fmt.Sprintf("step_%d", b.stepCounter),
		Status: Pending,
		Action: AwardFame,
		Payload: AwardFamePayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			Amount:      amount,
		},
	})
	return b
}

// Build creates the saga
func (b *Builder) Build() Saga {
	return Saga{
		TransactionId: b.transactionId,
		SagaType:      b.sagaType,
		InitiatedBy:   b.initiatedBy,
		Steps:         b.steps,
	}
}
