package saga

import (
	msgsaga "atlas-notes/kafka/message/saga"
	"atlas-notes/kafka/producer"
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
	return producer.ProviderImpl(p.l)(p.ctx)(msgsaga.EnvCommandTopic)(CreateCommandProvider(s))
}
