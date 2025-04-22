package expression

import (
	expression2 "atlas-channel/kafka/message/expression"
	"atlas-channel/kafka/producer"
	expression3 "atlas-channel/kafka/producer/expression"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
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

func (p *Processor) Change(characterId uint32, m _map.Model, expression uint32) error {
	p.l.Debugf("Changing character [%d] expression to [%d].", characterId, m.MapId())
	return producer.ProviderImpl(p.l)(p.ctx)(expression2.EnvExpressionCommand)(expression3.SetCommandProvider(characterId, m, expression))
}
