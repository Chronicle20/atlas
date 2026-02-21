package saga

import (
	"atlas-reactor-actions/kafka/message/saga"
	"atlas-reactor-actions/kafka/producer"
	"context"

	sharedsaga "github.com/Chronicle20/atlas-saga"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(s sharedsaga.Saga) error
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

func (p *ProcessorImpl) Create(s sharedsaga.Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))
}
