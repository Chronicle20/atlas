package fame

import (
	fame2 "atlas-channel/kafka/message/fame"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) RequestChange(f field.Model, characterId uint32, targetId uint32, amount int8) error {
	return producer.ProviderImpl(p.l)(p.ctx)(fame2.EnvCommandTopic)(RequestChangeFameCommandProvider(f, characterId, targetId, amount))
}
