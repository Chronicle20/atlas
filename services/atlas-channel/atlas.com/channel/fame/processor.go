package fame

import (
	fame2 "atlas-channel/kafka/message/fame"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Processor interface {
	RequestChange(f field.Model, characterId uint32, targetId uint32, amount int8) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RequestChange(f field.Model, characterId uint32, targetId uint32, amount int8) error {
	return producer.ProviderImpl(p.l)(p.ctx)(fame2.EnvCommandTopic)(RequestChangeFameCommandProvider(f, characterId, targetId, amount))
}
