package saga

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	sharedsaga "github.com/Chronicle20/atlas-saga"
)

// Re-export types from atlas-saga shared library
type (
	Type   = sharedsaga.Type
	Saga   = sharedsaga.Saga
	Status = sharedsaga.Status
	Action = sharedsaga.Action

	// Payload types
	AwardFamePayload = sharedsaga.AwardFamePayload
)

// Re-export constants from atlas-saga shared library
const (
	InventoryTransaction = sharedsaga.InventoryTransaction

	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	AwardFame = sharedsaga.AwardFame
)

// Builder helps construct sagas with multiple steps
type Builder struct {
	b           *sharedsaga.Builder
	stepCounter int
}

// NewBuilder creates a new saga builder
func NewBuilder() *Builder {
	return &Builder{
		b:           sharedsaga.NewBuilder(),
		stepCounter: 0,
	}
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

// AddAwardFame adds a fame award step
func (b *Builder) AddAwardFame(characterId uint32, ch channel.Model, amount int16) *Builder {
	b.stepCounter++
	b.b.AddStep(
		fmt.Sprintf("step_%d", b.stepCounter),
		Pending,
		AwardFame,
		AwardFamePayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			Amount:      amount,
		},
	)
	return b
}

// Build creates the saga
func (b *Builder) Build() Saga {
	return b.b.Build()
}
