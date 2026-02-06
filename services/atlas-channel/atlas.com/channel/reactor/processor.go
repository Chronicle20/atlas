package reactor

import (
	reactor2 "atlas-channel/kafka/message/reactor"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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

func (p *Processor) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(f), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *Processor) Hit(f field.Model, reactorId uint32, characterId uint32, stance uint16, skillId uint32) error {
	p.l.Debugf("Sending hit command for reactor [%d]. CharacterId [%d]. Stance [%d]. SkillId [%d].", reactorId, characterId, stance, skillId)
	return producer.ProviderImpl(p.l)(p.ctx)(reactor2.EnvCommandTopic)(HitCommandProvider(f, reactorId, characterId, stance, skillId))
}
