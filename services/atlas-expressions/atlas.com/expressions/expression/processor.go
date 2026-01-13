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
	Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error)
	// ChangeAndEmit changes the expression for a character and emits an event
	ChangeAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error)
	// Clear clears the expression for a character
	Clear(mb *message.Buffer, transactionId uuid.UUID, characterId uint32) (Model, error)
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
func (p *ProcessorImpl) Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error) {
	p.l.Debugf("Changing expression to [%d] for character [%d] in map [%d].", expression, characterId, mapId)
	m := GetRegistry().add(p.t, characterId, worldId, channelId, mapId, expression)

	// Add message to buffer
	err := mb.Put(expression2.EnvExpressionEvent, expressionEventProvider(transactionId, characterId, worldId, channelId, mapId, expression))
	if err != nil {
		return Model{}, err
	}

	return m, nil
}

// ChangeAndEmit changes the expression for a character and emits an event
func (p *ProcessorImpl) ChangeAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error) {
	return message.EmitWithResult[Model, changeInput](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(input changeInput) (Model, error) {
		return func(input changeInput) (Model, error) {
			return p.Change(mb, input.transactionId, input.characterId, input.worldId, input.channelId, input.mapId, input.expression)
		}
	})(changeInput{
		transactionId: transactionId,
		characterId:   characterId,
		worldId:       worldId,
		channelId:     channelId,
		mapId:         mapId,
		expression:    expression,
	})
}

// changeInput holds the parameters for Change operation
type changeInput struct {
	transactionId uuid.UUID
	characterId   uint32
	worldId       world.Id
	channelId     channel.Id
	mapId         _map.Id
	expression    uint32
}

// Clear clears the expression for a character
func (p *ProcessorImpl) Clear(mb *message.Buffer, transactionId uuid.UUID, characterId uint32) (Model, error) {
	p.l.Debugf("Clearing expression for character [%d].", characterId)
	GetRegistry().clear(p.t, characterId)
	// Return an empty model since we're clearing
	return Model{}, nil
}

// ClearAndEmit clears the expression for a character and emits an event
func (p *ProcessorImpl) ClearAndEmit(transactionId uuid.UUID, characterId uint32) (Model, error) {
	return message.EmitWithResult[Model, clearInput](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(input clearInput) (Model, error) {
		return func(input clearInput) (Model, error) {
			return p.Clear(mb, input.transactionId, input.characterId)
		}
	})(clearInput{
		transactionId: transactionId,
		characterId:   characterId,
	})
}

// clearInput holds the parameters for Clear operation
type clearInput struct {
	transactionId uuid.UUID
	characterId   uint32
}

// EmitExpressionEvent emits an expression event for the given model
func EmitExpressionEvent(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, m Model) error {
	return producer.ProviderImpl(l)(ctx)(expression2.EnvExpressionEvent)(expressionEventProvider(transactionId, m.CharacterId(), m.WorldId(), m.ChannelId(), m.MapId(), m.Expression()))
}
