package saga

import (
	"context"

	"atlas-mts/kafka/message/saga"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"
)

// Processor emits constructed sagas to the orchestrator's command topic.
type Processor interface {
	Create(s Saga) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// Create emits the saga to COMMAND_TOPIC_SAGA.
func (p *ProcessorImpl) Create(s Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))
}
