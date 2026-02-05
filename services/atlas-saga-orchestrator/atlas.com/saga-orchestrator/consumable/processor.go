package consumable

import (
	"context"

	"atlas-saga-orchestrator/kafka/message/consumable"
	"atlas-saga-orchestrator/kafka/producer"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for consumable operations
type Processor interface {
	// ApplyConsumableEffect sends a command to apply item effects to a character without consuming from inventory
	ApplyConsumableEffect(transactionId uuid.UUID, ch channel.Model, characterId character.Id, itemId item.Id) error
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
func (p *ProcessorImpl) ApplyConsumableEffect(transactionId uuid.UUID, ch channel.Model, characterId character.Id, itemId item.Id) error {
	return producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvCommandTopic)(ApplyConsumableEffectCommandProvider(transactionId, ch, characterId, itemId))
}
