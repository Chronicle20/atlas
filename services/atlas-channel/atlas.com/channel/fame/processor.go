package fame

import (
	"atlas-channel/kafka/producer"
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

func (p *Processor) RequestChange(m _map.Model, characterId uint32, targetId uint32, amount int8) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(requestChangeFameCommandProvider(m, characterId, targetId, amount))
}
