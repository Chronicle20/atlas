package chair

import (
	chair2 "atlas-channel/kafka/message/chair"
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

func (p *Processor) Use(m _map.Model, chairType string, chairId uint32, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to use map [%d] [%s] chair [%d].", characterId, m.MapId(), chairType, chairId)
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(UseCommandProvider(m, chairType, chairId, characterId))
}

func (p *Processor) Cancel(m _map.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(CancelCommandProvider(m, characterId))
}
