package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
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

func (p *Processor) GetById(uniqueId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(uniqueId), Extract)()
}

func (p *Processor) InMapModelProvider(m _map.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(m), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInMap(m _map.Model, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(m), f, model.ParallelExecute())
}

func (p *Processor) GetInMap(m _map.Model) ([]Model, error) {
	return p.InMapModelProvider(m)()
}

func (p *Processor) Damage(m _map.Model, monsterId uint32, characterId uint32, damage uint32) error {
	p.l.Debugf("Applying damage to monster [%d]. Character [%d]. Damage [%d].", monsterId, characterId, damage)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DamageCommandProvider(m, monsterId, characterId, damage))
}
