package saga

import (
	"atlas-channel/kafka/message/saga"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

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

var _ Processor = (*ProcessorImpl)(nil)

// Create initiates a new saga by emitting it to Kafka
func (p *ProcessorImpl) Create(s Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))
}
