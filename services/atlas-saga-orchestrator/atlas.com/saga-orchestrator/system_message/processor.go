package system_message

import (
	"atlas-saga-orchestrator/kafka/message/system_message"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for system message operations
type Processor interface {
	// SendMessage sends a system message to a character
	SendMessage(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, messageType string, message string) error
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

// SendMessage sends a Kafka command to atlas-channel to display a system message
func (p *ProcessorImpl) SendMessage(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, messageType string, message string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(SendMessageCommandProvider(transactionId, worldId, channelId, characterId, messageType, message))
}
