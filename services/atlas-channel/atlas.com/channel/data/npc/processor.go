package npc

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// notImitateTemplate filters out the WZ Player NPC imitate-pool placeholders
// (design §4.2: 9901000-9906599) from a map's NPC list. Those life entries
// exist so the client's CNpcPool overlay (CNpcPool::OnNpcImitateData) can
// paint a deployed Player NPC's look onto them, but the channel spawns its
// own SPAWN_NPC for a deployed Player NPC (a different oid outside this
// pool), so leaving the placeholders in this list would double-spawn and
// double-elect a controller for every deployed Player NPC (task-251 bug
// report §2).
func notImitateTemplate(m Model) bool {
	return !objectid.IsPlayerNpcImitateTemplate(m.Template())
}

type Processor interface {
	ForEachInMap(mapId _map.Id, f model.Operator[Model]) error
	InMapModelProvider(mapId _map.Id) model.Provider[[]Model]
	InMapByObjectIdModelProvider(mapId _map.Id, objectId uint32) model.Provider[[]Model]
	GetInMapByObjectId(mapId _map.Id, objectId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ForEachInMap(mapId _map.Id, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(mapId), f, model.ParallelExecute())
}

// InMapModelProvider fetches every NPC spawned on a map. atlas-data's GET
// /data/maps/{id}/npcs is now paginated (task-117), so this drains every
// page rather than fetching one.
func (p *ProcessorImpl) InMapModelProvider(mapId _map.Id) model.Provider[[]Model] {
	url, err := npcsInMapUrl(p.ctx, mapId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model](notImitateTemplate))
}

func (p *ProcessorImpl) InMapByObjectIdModelProvider(mapId _map.Id, objectId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestNPCsInMapByObjectId(p.ctx, mapId, objectId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetInMapByObjectId(mapId _map.Id, objectId uint32) (Model, error) {
	mp := p.InMapByObjectIdModelProvider(mapId, objectId)
	return model.First[Model](mp, model.Filters[Model]())
}
