package door

import (
	"context"
	"time"

	doorproducer "atlas-doors/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Processor is the door engine: spawn (with FR-1.4 recast replace), remove,
// query, and reslot. It is field-injected (emit + resolver + allocator) so it
// unit-tests without Kafka/REST/Redis, mirroring the monsters processor seam.
type Processor interface {
	GetById(areaDoorId uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	GetByOwner(ownerCharacterId uint32) ([]Model, error)
	Spawn(f field.Model, ownerCharacterId, skillId uint32, skillLevel byte, x, y int16) (Model, error)
	RemoveByOwner(ownerCharacterId uint32, reason string) error
	RemoveByOwnerIfLeftField(ownerCharacterId uint32, newField field.Model) error
	Reslot(areaDoorId uint32, newSlot byte, townPortalId uint32, townX, townY int16) error
}

// spawnPlan is the resolver's verdict for a single spawn: where the town side
// lands, which party slot/portal the caster occupies, and how long the door lives.
type spawnPlan struct {
	townMapId    _map.Id
	slot         byte
	townPortalId uint32
	townX        int16
	townY        int16
	durationMs   int32
}

// resolver computes the spawnPlan from external data (map/skill/party). Injected
// so tests can supply canned inputs.
type resolver interface {
	ResolveSpawn(ctx context.Context, f field.Model, ownerCharacterId, partyId, skillId uint32, level byte) (spawnPlan, error)
	PartyIdFor(ctx context.Context, ownerCharacterId uint32) (uint32, error)
}

// allocator is the object-id allocation seam. *IdAllocator satisfies it; tests
// inject a counter-based stub that can force the second allocation to fail.
type allocator interface {
	Allocate(ctx context.Context, t tenant.Model) (uint32, error)
	Release(ctx context.Context, t tenant.Model, id uint32)
}

type emitter func(topic string, p model.Provider[[]kafka.Message]) error

type ProcessorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	t     tenant.Model
	emit  emitter
	res   resolver
	alloc allocator
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(topic string, p model.Provider[[]kafka.Message]) error {
			return doorproducer.ProviderImpl(l)(ctx)(topic)(p)
		},
		res:   newRestResolver(l, ctx),
		alloc: GetIdAllocator(),
	}
}

func (p *ProcessorImpl) GetById(areaDoorId uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, areaDoorId)
}

func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

func (p *ProcessorImpl) GetByOwner(ownerCharacterId uint32) ([]Model, error) {
	return GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
}

func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId, skillId uint32, skillLevel byte, x, y int16) (Model, error) {
	// FR-1.4 recast: remove any existing owner door (and emit REMOVED/RECAST)
	// BEFORE deploying the replacement.
	if err := p.RemoveByOwner(ownerCharacterId, RemoveReasonRecast); err != nil {
		p.l.WithError(err).Warnf("recast cleanup failed for character %d", ownerCharacterId)
	}

	partyId, err := p.res.PartyIdFor(p.ctx, ownerCharacterId)
	if err != nil {
		partyId = 0
	}
	plan, err := p.res.ResolveSpawn(p.ctx, f, ownerCharacterId, partyId, skillId, skillLevel)
	if err != nil {
		p.l.WithError(err).Warnf("door spawn rejected (resolve) for character %d", ownerCharacterId)
		return Model{}, err
	}

	// Allocate the area oid first, then the town oid. On town-alloc failure we
	// release the area oid and persist/emit nothing.
	areaId, err := p.alloc.Allocate(p.ctx, p.t)
	if err != nil {
		p.l.WithError(err).Errorf("door area oid alloc failed")
		return Model{}, err
	}
	townId, err := p.alloc.Allocate(p.ctx, p.t)
	if err != nil {
		p.alloc.Release(p.ctx, p.t, areaId)
		p.l.WithError(err).Errorf("door town oid alloc failed")
		return Model{}, err
	}

	now := time.Now()
	expires := now
	if plan.durationMs > 0 {
		expires = now.Add(time.Duration(plan.durationMs) * time.Millisecond)
	}
	m := NewBuilder().
		SetAreaDoorId(areaId).SetTownDoorId(townId).
		SetOwnerCharacterId(ownerCharacterId).SetPartyId(partyId).
		SetSkillId(skillId).SetSkillLevel(skillLevel).SetField(f).
		SetTownMapId(plan.townMapId).SetSlot(plan.slot).SetTownPortalId(plan.townPortalId).
		SetAreaX(x).SetAreaY(y).SetTownX(plan.townX).SetTownY(plan.townY).
		SetDeployTime(now).SetExpiresAt(expires).Build()

	if err := GetRegistry().Put(p.ctx, p.t, m); err != nil {
		p.alloc.Release(p.ctx, p.t, areaId)
		p.alloc.Release(p.ctx, p.t, townId)
		return Model{}, err
	}
	if err := p.emit(EnvEventTopicDoorStatus, createdEventProvider(m)); err != nil {
		p.l.WithError(err).Errorf("failed emitting CREATED for door %d", areaId)
	}
	return m, nil
}

func (p *ProcessorImpl) RemoveByOwner(ownerCharacterId uint32, reason string) error {
	doors, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		return err
	}
	for _, m := range doors {
		if err := GetRegistry().Remove(p.ctx, p.t, m.AreaDoorId()); err != nil {
			p.l.WithError(err).Warnf("failed removing door %d", m.AreaDoorId())
			continue
		}
		p.alloc.Release(p.ctx, p.t, m.AreaDoorId())
		p.alloc.Release(p.ctx, p.t, m.TownDoorId())
		if err := p.emit(EnvEventTopicDoorStatus, removedEventProvider(m, reason)); err != nil {
			p.l.WithError(err).Errorf("failed emitting REMOVED for door %d", m.AreaDoorId())
		}
	}
	return nil
}

// RemoveByOwnerIfLeftField removes the owner's door only when newField is neither the
// door's source field nor its town map (walking into the town the door spans is a warp,
// not abandonment — FR-6.2 / design §5.3).
func (p *ProcessorImpl) RemoveByOwnerIfLeftField(ownerCharacterId uint32, newField field.Model) error {
	doors, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		return err
	}
	for _, m := range doors {
		src := m.Field()
		sameSource := src.WorldId() == newField.WorldId() && src.ChannelId() == newField.ChannelId() &&
			src.MapId() == newField.MapId() && src.Instance() == newField.Instance()
		intoTown := newField.MapId() == m.TownMapId()
		if sameSource || intoTown {
			continue
		}
		if err := GetRegistry().Remove(p.ctx, p.t, m.AreaDoorId()); err != nil {
			continue
		}
		p.alloc.Release(p.ctx, p.t, m.AreaDoorId())
		p.alloc.Release(p.ctx, p.t, m.TownDoorId())
		_ = p.emit(EnvEventTopicDoorStatus, removedEventProvider(m, RemoveReasonLeftField))
	}
	return nil
}

func (p *ProcessorImpl) Reslot(areaDoorId uint32, newSlot byte, townPortalId uint32, townX, townY int16) error {
	m, err := GetRegistry().Get(p.ctx, p.t, areaDoorId)
	if err != nil {
		return err
	}
	oldSlot := m.Slot()
	if oldSlot == newSlot {
		return nil
	}
	n := m.Reslot(newSlot, townPortalId, townX, townY)
	if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
		return err
	}
	return p.emit(EnvEventTopicDoorStatus, slotChangedEventProvider(n, oldSlot))
}
