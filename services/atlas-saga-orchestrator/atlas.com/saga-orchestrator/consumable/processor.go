package consumable

import (
	"atlas-saga-orchestrator/kafka/message/consumable"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for consumable operations
type Processor interface {
	// ApplyConsumableEffect sends a command to apply item effects to a character without consuming from inventory
	ApplyConsumableEffect(worldId byte, channelId byte, characterId uint32, itemId uint32) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new consumable processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// ApplyConsumableEffect sends a Kafka command to atlas-consumables to apply item effects
func (p *ProcessorImpl) ApplyConsumableEffect(worldId byte, channelId byte, characterId uint32, itemId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvCommandTopic)(ApplyConsumableEffectCommandProvider(worldId, channelId, characterId, itemId))
}
