package saga

import (
	"atlas-portal-actions/kafka/message/saga"
	"atlas-portal-actions/kafka/producer"
	"context"

	"github.com/sirupsen/logrus"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Create(s sharedsaga.Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))
}
