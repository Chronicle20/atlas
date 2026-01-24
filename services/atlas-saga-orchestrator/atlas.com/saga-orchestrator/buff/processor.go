package buff

import (
	"atlas-saga-orchestrator/kafka/message"
	buffMsg "atlas-saga-orchestrator/kafka/message/buff"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for buff operations
type Processor interface {
	// CancelAllAndEmit sends a command to cancel all buffs for a character
	CancelAllAndEmit(worldId world.Id, channelId channel.Id, characterId uint32) error
	// CancelAll adds a cancel all command to the message buffer
	CancelAll(mb *message.Buffer) func(worldId world.Id, channelId channel.Id, characterId uint32) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
}

// NewProcessor creates a new buff processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

// CancelAllAndEmit sends a Kafka command to atlas-buffs to cancel all buffs for a character
func (p *ProcessorImpl) CancelAllAndEmit(worldId world.Id, channelId channel.Id, characterId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.CancelAll(mb)(worldId, channelId, characterId)
	})
}

// CancelAll adds a cancel all command to the message buffer
func (p *ProcessorImpl) CancelAll(mb *message.Buffer) func(worldId world.Id, channelId channel.Id, characterId uint32) error {
	return func(worldId world.Id, channelId channel.Id, characterId uint32) error {
		return mb.Put(buffMsg.EnvCommandTopic, CancelAllCommandProvider(byte(worldId), byte(channelId), characterId))
	}
}
