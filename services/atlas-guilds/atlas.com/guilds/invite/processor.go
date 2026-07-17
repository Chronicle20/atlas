package invite

import (
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Processor interface {
	Create(actorId uint32, worldId world.Id, referenceId uint32, targetId uint32) error
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

func (p *ProcessorImpl) Create(actorId uint32, worldId world.Id, referenceId uint32, targetId uint32) error {
	p.l.Debugf("Creating guild [%d] invitation for [%d] from [%d].", referenceId, targetId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(createInviteCommandProvider(actorId, referenceId, worldId, targetId))
}
