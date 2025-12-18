package expression

import (
	"atlas-expressions/kafka/message"
	expression2 "atlas-expressions/kafka/message/expression"
	"atlas-expressions/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for managing expressions
type Processor interface {
	// Change changes the expression for a character
	Change(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error)
	// ChangeAndEmit changes the expression for a character and emits an event
	ChangeAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error)
	// Clear clears the expression for a character
	Clear(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) (Model, error)
	// ClearAndEmit clears the expression for a character and emits an event
	ClearAndEmit(transactionId uuid.UUID, characterId uint32) (Model, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new Processor instance
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
	}
}

// Change changes the expression for a character
func (p *ProcessorImpl) Change(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error) {
	return func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error) {
		return func(characterId uint32) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error) {
			return func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error) {
				return func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error) {
					return func(mapId _map.Id) func(expression uint32) (Model, error) {
						return func(expression uint32) (Model, error) {
							p.l.Debugf("Changing expression to [%d] for character [%d] in map [%d].", expression, characterId, mapId)
							model := GetRegistry().add(p.t, characterId, worldId, channelId, mapId, expression)

							// Add message to buffer
							err := mb.Put(expression2.EnvExpressionEvent, expressionEventProvider(transactionId, characterId, worldId, channelId, mapId, expression))
							if err != nil {
								return Model{}, err
							}

							return model, nil
						}
					}
				}
			}
		}
	}
}

// ChangeAndEmit changes the expression for a character and emits an event
func (p *ProcessorImpl) ChangeAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error) {
	mb := message.NewBuffer()
	model, err := p.Change(mb)(transactionId)(characterId)(worldId)(channelId)(mapId)(expression)
	if err != nil {
		return model, err
	}

	for t := range mb.GetAll() {
		err = producer.ProviderImpl(p.l)(p.ctx)(t)(expressionEventProvider(transactionId, characterId, worldId, channelId, mapId, expression))
		if err != nil {
			return model, err
		}
	}
	return model, err
}

// Clear clears the expression for a character
func (p *ProcessorImpl) Clear(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) (Model, error) {
	return func(transactionId uuid.UUID) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			p.l.Debugf("Clearing expression for character [%d].", characterId)
			GetRegistry().clear(p.t, characterId)
			// Return an empty model since we're clearing
			return Model{}, nil
		}
	}
}

// ClearAndEmit clears the expression for a character and emits an event
func (p *ProcessorImpl) ClearAndEmit(transactionId uuid.UUID, characterId uint32) (Model, error) {
	mb := message.NewBuffer()
	model, err := p.Clear(mb)(transactionId)(characterId)
	if err != nil {
		return model, err
	}

	// No event to emit for clearing, but we still use the message buffer pattern for consistency
	for t, _ := range mb.GetAll() {
		// Since we don't have a specific event for clearing, we don't need to emit anything
		// This is just a placeholder to maintain the pattern
		// Use zero values for world, channel, and map IDs
		err = producer.ProviderImpl(p.l)(p.ctx)(t)(expressionEventProvider(transactionId, characterId, 0, 0, 0, 0))
		if err != nil {
			return model, err
		}
	}
	return model, err
}
