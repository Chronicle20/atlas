package portal

import (
	"atlas-portals/blocked"
	"atlas-portals/character"
	"atlas-portals/portal_actions"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"math/rand"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapByNameProvider(mapId _map.Id, name string) model.Provider[[]Model]
	InMapByIdProvider(mapId _map.Id, id uint32) model.Provider[Model]
	GetInMapByName(mapId _map.Id, name string) (Model, error)
	GetInMapById(mapId _map.Id, id uint32) (Model, error)
	InMapProvider(mapId _map.Id) model.Provider[[]Model]
	Warp(f field.Model, characterId uint32, targetMapId _map.Id)
	Enter(f field.Model, portalId uint32, characterId uint32)
	WarpById(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32)
	WarpToPosition(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16)
	WarpToPortal(f field.Model, characterId uint32, targetMapId _map.Id, portalProvider model.Provider[uint32])
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

func (p *ProcessorImpl) InMapByNameProvider(mapId _map.Id, name string) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMapByName(mapId, name), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) InMapByIdProvider(mapId _map.Id, id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestInMapById(mapId, id), Extract)
}

func (p *ProcessorImpl) GetInMapByName(mapId _map.Id, name string) (Model, error) {
	return model.First(p.InMapByNameProvider(mapId, name), model.Filters[Model]())
}

func (p *ProcessorImpl) GetInMapById(mapId _map.Id, id uint32) (Model, error) {
	return p.InMapByIdProvider(mapId, id)()
}

// InMapProvider fetches every portal in a map. atlas-data's GET
// /data/maps/{id}/portals is now paginated (task-117), so this drains
// every page rather than fetching one.
func (p *ProcessorImpl) InMapProvider(mapId _map.Id) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inMapUrl(mapId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) Warp(f field.Model, characterId uint32, targetMapId _map.Id) {
	p.l.Debugf("Character [%d] warping to map [%d].", characterId, targetMapId)

	portals, err := p.InMapProvider(targetMapId)()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve portals for map [%d].", targetMapId)
		character.NewProcessor(p.l, p.ctx).EnableActions(f, characterId)
		return
	}

	if len(portals) == 0 {
		p.l.Warnf("No portals found in target map [%d]. Defaulting to portal 0.", targetMapId)
		p.WarpById(f, characterId, targetMapId, 0)
		return
	}

	tp := portals[rand.Intn(len(portals))]
	p.WarpById(f, characterId, targetMapId, tp.Id())
}

func (p *ProcessorImpl) Enter(f field.Model, portalId uint32, characterId uint32) {
	p.l.Debugf("Character [%d] entering portal [%d] in map [%d].", characterId, portalId, f.MapId())

	// Check if the portal is blocked for this character
	if blocked.GetRegistry().IsBlocked(p.ctx, characterId, f.MapId(), portalId) {
		p.l.Debugf("Portal [%d] in map [%d] is blocked for character [%d]. Enabling actions and returning.", portalId, f.MapId(), characterId)
		character.NewProcessor(p.l, p.ctx).EnableActions(f, characterId)
		return
	}

	pt, err := p.GetInMapById(f.MapId(), portalId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate portal [%d] in map [%d] character [%d] is trying to enter.", portalId, f.MapId(), characterId)
		return
	}

	if pt.HasScript() {
		p.l.Debugf("Portal [%s] has script. Executing [%s] for character [%d].", pt.String(), pt.ScriptName(), characterId)
		portal_actions.ExecuteScript(p.l)(p.ctx)(f, portalId, characterId, pt.ScriptName())
		return
	}

	if pt.HasTargetMap() {
		p.l.Debugf("Portal [%s] has target. Transferring character [%d] to [%d].", pt.String(), characterId, pt.TargetMapId())

		var tp Model
		tp, err = p.GetInMapByName(pt.TargetMapId(), pt.Target())
		if err != nil {
			p.l.WithError(err).Warnf("Unable to locate portal target [%s] for map [%d]. Defaulting to portal 0.", pt.Target(), pt.TargetMapId())
			tp, err = p.GetInMapById(pt.TargetMapId(), 0)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to locate portal 0 for map [%d]. Is there invalid wz data?", pt.TargetMapId())
				character.NewProcessor(p.l, p.ctx).EnableActions(f, characterId)
				return
			}
		}
		p.WarpById(f, characterId, pt.TargetMapId(), tp.Id())
		return
	}

	character.NewProcessor(p.l, p.ctx).EnableActions(f, characterId)
}

func (p *ProcessorImpl) WarpById(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32) {
	p.WarpToPortal(f, characterId, targetMapId, model.FixedProvider(portalId))
}

// WarpToPosition warps the character to an exact (x, y) coordinate in the
// target map rather than a named portal — used by Mystic Door to land the user
// on the linked door's position. The CHANGE_MAP command carries the position;
// atlas-maps relays it on MAP_CHANGED and atlas-channel applies it via the
// SET_FIELD chase mechanism.
func (p *ProcessorImpl) WarpToPosition(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16) {
	_ = producer.ProviderImpl(p.l)(p.ctx)(character.EnvCommandTopic)(character.ChangeToPositionProvider(f, characterId, targetMapId, x, y))
}

func (p *ProcessorImpl) WarpToPortal(f field.Model, characterId uint32, targetMapId _map.Id, portalProvider model.Provider[uint32]) {
	id, err := portalProvider()
	if err == nil {
		_ = producer.ProviderImpl(p.l)(p.ctx)(character.EnvCommandTopic)(character.ChangeMapProvider(f, characterId, targetMapId, id))
	}
}
