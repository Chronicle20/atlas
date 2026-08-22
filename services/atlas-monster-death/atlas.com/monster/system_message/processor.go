package system_message

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

// Processor is the interface for system message operations
type Processor interface {
	// ShowHint sends a command to show a hint box for a character
	ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new system message processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// ShowHint sends a Kafka command to atlas-channel to show a hint box
func (p *ProcessorImpl) ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(ShowHintCommandProvider(transactionId, ch, characterId, hint, width, height))
}
