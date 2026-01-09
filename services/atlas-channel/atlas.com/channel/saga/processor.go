package saga

import (
	"atlas-channel/kafka/message/saga"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/sirupsen/logrus"
)

// Processor interface defines operations for saga processing
type Processor interface {
	Create(s Saga) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new saga processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// Create initiates a new saga by emitting it to Kafka
func (p *ProcessorImpl) Create(s Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))
}
