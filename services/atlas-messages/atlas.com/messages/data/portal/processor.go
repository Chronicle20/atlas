package portal

import (
	"context"
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

func (p *Processor) InMapProvider(mapId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAll(mapId), Extract, model.Filters[Model]())
}

func (p *Processor) RandomSpawnPointProvider(mapId uint32) model.Provider[Model] {
	return func() (Model, error) {
		sps, err := model.FilteredProvider(p.InMapProvider(mapId), model.Filters(SpawnPoint, NoTarget))()
		if err != nil {
			return Model{}, err
		}
		return model.RandomPreciselyOneFilter(sps)
	}
}

func (p *Processor) RandomSpawnPointIdProvider(mapId uint32) model.Provider[uint32] {
	return model.Map(getId)(p.RandomSpawnPointProvider(mapId))
}

func getId(m Model) (uint32, error) {
	return m.Id(), nil
}
