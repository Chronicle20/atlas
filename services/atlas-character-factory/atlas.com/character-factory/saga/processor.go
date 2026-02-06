package saga

import (
	"atlas-character-factory/kafka/message/saga"
	"atlas-character-factory/kafka/producer"
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(s Saga) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) Create(s Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))
}
