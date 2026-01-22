package reactor

import (
	reactor2 "atlas-channel/kafka/message/reactor"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
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

func (p *Processor) InMapModelProvider(m _map.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(m), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInMap(m _map.Model, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(m), f, model.ParallelExecute())
}

func (p *Processor) Hit(m _map.Model, reactorId uint32, stance uint16, skillId uint32) error {
	p.l.Debugf("Sending hit command for reactor [%d]. Stance [%d]. SkillId [%d].", reactorId, stance, skillId)
	return producer.ProviderImpl(p.l)(p.ctx)(reactor2.EnvCommandTopic)(HitCommandProvider(m, reactorId, stance, skillId))
}
