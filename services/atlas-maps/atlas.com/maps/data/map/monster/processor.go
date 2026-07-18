package monster

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

var _ Processor = (*ProcessorImpl)(nil)

// SpawnPointProvider fetches every monster spawn point on a map. atlas-data's
// GET /data/maps/{id}/monsters is now paginated (task-117) and busy
// grinding/party-quest maps commonly exceed the default page size, so this
// drains every page rather than fetching one.
func (p *ProcessorImpl) SpawnPointProvider(mapId _map.Id) model.Provider[[]SpawnPoint] {
	return requests.DrainProvider[RestModel, SpawnPoint](p.l, p.ctx)(spawnPointsUrl(mapId), 250, Extract, model.Filters[SpawnPoint]())
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
