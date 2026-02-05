package monster

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	SpawnPointProvider(mapId _map.Id) model.Provider[[]SpawnPoint]
	SpawnableSpawnPointProvider(mapId _map.Id) model.Provider[[]SpawnPoint]
	GetSpawnPoints(mapId _map.Id) ([]SpawnPoint, error)
	GetSpawnableSpawnPoints(mapId _map.Id) ([]SpawnPoint, error)
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

func (p *ProcessorImpl) SpawnPointProvider(mapId _map.Id) model.Provider[[]SpawnPoint] {
	return requests.SliceProvider[RestModel, SpawnPoint](p.l, p.ctx)(requestSpawnPoints(mapId), Extract, model.Filters[SpawnPoint]())
}

func (p *ProcessorImpl) SpawnableSpawnPointProvider(mapId _map.Id) model.Provider[[]SpawnPoint] {
	return model.FilteredProvider(p.SpawnPointProvider(mapId), model.Filters(p.Spawnable))
}

func (p *ProcessorImpl) GetSpawnPoints(mapId _map.Id) ([]SpawnPoint, error) {
	return p.SpawnPointProvider(mapId)()
}

func (p *ProcessorImpl) GetSpawnableSpawnPoints(mapId _map.Id) ([]SpawnPoint, error) {
	return p.SpawnableSpawnPointProvider(mapId)()
}

func (p *ProcessorImpl) Spawnable(point SpawnPoint) bool {
	return point.MobTime >= 0
}
