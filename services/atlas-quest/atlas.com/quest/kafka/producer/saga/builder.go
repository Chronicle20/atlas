package saga

import (
	"atlas-quest/kafka/message/saga"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

// Builder helps construct sagas with multiple steps
type Builder struct {
	transactionId uuid.UUID
	sagaType      saga.Type
	initiatedBy   string
	steps         []saga.Step
	stepCounter   int
}

// NewBuilder creates a new saga builder
func NewBuilder(sagaType saga.Type, initiatedBy string) *Builder {
	return &Builder{
		transactionId: uuid.New(),
		sagaType:      sagaType,
		initiatedBy:   initiatedBy,
		steps:         make([]saga.Step, 0),
		stepCounter:   0,
	}
}

// AddAwardItem adds an item award step
func (b *Builder) AddAwardItem(characterId uint32, templateId uint32, quantity uint32) *Builder {
	b.stepCounter++
	b.steps = append(b.steps, saga.Step{
		StepId: stepId(b.stepCounter),
		Status: saga.Pending,
		Action: saga.AwardAsset,
		Payload: saga.AwardItemPayload{
			CharacterId: characterId,
			Item: saga.ItemDetail{
				TemplateId: templateId,
				Quantity:   quantity,
			},
		},
	})
	return b
}

// AddAwardMesos adds a meso award step
func (b *Builder) AddAwardMesos(characterId uint32, ch channel.Model, amount int32, actorId uint32) *Builder {
	b.stepCounter++
	b.steps = append(b.steps, saga.Step{
		StepId: stepId(b.stepCounter),
		Status: saga.Pending,
		Action: saga.AwardMesos,
		Payload: saga.AwardMesosPayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			ActorId:     actorId,
			ActorType:   "quest",
			Amount:      amount,
			ShowEffect:  true,
		},
	})
	return b
}

// AddAwardExperience adds an experience award step
func (b *Builder) AddAwardExperience(characterId uint32, ch channel.Model, amount int32) *Builder {
	b.stepCounter++
	b.steps = append(b.steps, saga.Step{
		StepId: stepId(b.stepCounter),
		Status: saga.Pending,
		Action: saga.AwardExperience,
		Payload: saga.AwardExperiencePayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			Distributions: []saga.ExperienceDistribution{
				{
					ExperienceType: "WHITE",
					Amount:         uint32(amount),
				},
			},
			ShowEffect: true,
		},
	})
	return b
}

// AddAwardFame adds a fame award step
func (b *Builder) AddAwardFame(characterId uint32, ch channel.Model, amount int16, actorId uint32) *Builder {
	b.stepCounter++
	b.steps = append(b.steps, saga.Step{
		StepId: stepId(b.stepCounter),
		Status: saga.Pending,
		Action: saga.AwardFame,
		Payload: saga.AwardFamePayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			ActorId:     actorId,
			ActorType:   "quest",
			Amount:      amount,
		},
	})
	return b
}

// AddCreateSkill adds a skill creation step
func (b *Builder) AddCreateSkill(characterId uint32, skillId uint32, level byte, masterLevel byte) *Builder {
	b.stepCounter++
	b.steps = append(b.steps, saga.Step{
		StepId: stepId(b.stepCounter),
		Status: saga.Pending,
		Action: saga.CreateSkill,
		Payload: saga.CreateSkillPayload{
			CharacterId: characterId,
			SkillId:     skillId,
			Level:       level,
			MasterLevel: masterLevel,
		},
	})
	return b
}

// AddConsumeItem adds an item consumption step
func (b *Builder) AddConsumeItem(characterId uint32, templateId uint32, quantity uint32) *Builder {
	b.stepCounter++
	b.steps = append(b.steps, saga.Step{
		StepId: stepId(b.stepCounter),
		Status: saga.Pending,
		Action: saga.ConsumeItem,
		Payload: saga.ConsumeItemPayload{
			CharacterId: characterId,
			TemplateId:  templateId,
			Quantity:    quantity,
		},
	})
	return b
}

// Build creates the saga
func (b *Builder) Build() saga.Saga {
	return saga.Saga{
		TransactionId: b.transactionId,
		SagaType:      b.sagaType,
		InitiatedBy:   b.initiatedBy,
		Steps:         b.steps,
	}
}

// HasSteps returns true if the builder has any steps
func (b *Builder) HasSteps() bool {
	return len(b.steps) > 0
}
