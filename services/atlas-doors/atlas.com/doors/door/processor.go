package door

import (
	"context"
	"time"

	doorproducer "atlas-doors/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
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
	GetByOwner(ownerCharacterId character.Id) ([]Model, error)
	Spawn(f field.Model, ownerCharacterId character.Id, skillId skill.Id, skillLevel byte, x point.X, y point.Y) (Model, error)
	RemoveByOwner(ownerCharacterId character.Id, reason string) error
	RemoveByOwnerIfLeftField(ownerCharacterId character.Id, newField field.Model) error
	Reslot(areaDoorId uint32, newSlot byte, townPortalId uint32, townX point.X, townY point.Y) error
}

// spawnPlan is the resolver's verdict for a single spawn: where the town side
// lands, which party slot/portal the caster occupies, and how long the door lives.
type spawnPlan struct {
	townMapId    _map.Id
	slot         byte
	townPortalId uint32
	townX        point.X
	townY        point.Y
	durationMs   int32
}

// resolver computes the spawnPlan from external data (map/skill/party). Injected
// so tests can supply canned inputs.
type resolver interface {
	ResolveSpawn(ctx context.Context, f field.Model, ownerCharacterId character.Id, partyId uint32, skillId skill.Id, level byte) (spawnPlan, error)
	PartyIdFor(ctx context.Context, ownerCharacterId character.Id) (uint32, error)
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

func (p *ProcessorImpl) GetByOwner(ownerCharacterId character.Id) ([]Model, error) {
	return GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
}

func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId character.Id, skillId skill.Id, skillLevel byte, x point.X, y point.Y) (Model, error) {
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
	if err := p.emit(EnvEventTopicDoorStatus, createdEventProvider(m, 0)); err != nil {
		p.l.WithError(err).Errorf("failed emitting CREATED for door %d", areaId)
	}
	return m, nil
}

func (p *ProcessorImpl) RemoveByOwner(ownerCharacterId character.Id, reason string) error {
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
		if err := p.emit(EnvEventTopicDoorStatus, removedEventProvider(m, reason, 0)); err != nil {
			p.l.WithError(err).Errorf("failed emitting REMOVED for door %d", m.AreaDoorId())
		}
	}
	return nil
}

// RemoveByOwnerIfLeftField removes the owner's door only when newField is neither the
// door's source field nor its town map (walking into the town the door spans is a warp,
// not abandonment — FR-6.2 / design §5.3).
func (p *ProcessorImpl) RemoveByOwnerIfLeftField(ownerCharacterId character.Id, newField field.Model) error {
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
		_ = p.emit(EnvEventTopicDoorStatus, removedEventProvider(m, RemoveReasonLeftField, 0))
	}
	return nil
}

// JoinPartyDoor adopts a joining member's SOLO door into the party — the inverse
// of LeavePartyDoor. A member who left then rejoined carries a solo door
// (partyId 0, slot 0); on rejoin it must be re-keyed into the party at the
// member's slot and broadcast so the EXISTING members see it added and the joiner
// renders it at the correct slot. Without this the joiner's solo door is skipped
// by ReslotParty and ShowPartyDoorsToCharacter (both filter d.PartyId()==partyId),
// so it never reslots off slot 0 and is never shown to the rest of the party.
//
// members is the post-join ordered member list (used to compute the joiner's
// slot). Only doors NOT already in this party are adopted.
func (p *ProcessorImpl) JoinPartyDoor(partyId uint32, members []character.Id, joinerCharacterId character.Id, townPortalsByMap func(_map.Id) []TownPortal) {
	slot := ComputeSlot(partyId, members, joinerCharacterId)
	doors, err := GetRegistry().GetByOwner(p.ctx, p.t, joinerCharacterId)
	if err != nil {
		p.l.WithError(err).Warnf("JoinPartyDoor: unable to load doors for owner %d", joinerCharacterId)
		return
	}
	for _, d := range doors {
		if d.PartyId() == partyId {
			continue // already a party door (e.g. duplicate event)
		}
		wireId, tx, ty, _ := ResolveTownPortal(townPortalsByMap(d.TownMapId()), slot, defaultTownX, defaultTownY)
		n := Clone(d).SetPartyId(partyId).SetSlot(slot).SetTownPortalId(wireId).SetTownX(tx).SetTownY(ty).Build()
		if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
			p.l.WithError(err).Warnf("JoinPartyDoor: persist failed for door %d", d.AreaDoorId())
			continue
		}
		// Broadcast CREATED for the now party-scoped door — reaches every member
		// (existing + joiner) and sets the joiner's town-portal slot for them.
		if err := p.emit(EnvEventTopicDoorStatus, createdEventProvider(n, 0)); err != nil {
			p.l.WithError(err).Warnf("JoinPartyDoor: broadcast failed for door %d", d.AreaDoorId())
		}
	}
}

// ShowPartyDoorsToCharacter re-emits a CREATED status for every door owned by a
// party member, targeted at `target` (a member who just joined). The channel
// delivers the spawn only to that character, so a player who joins a party with
// an active door starts seeing it without waiting for a recast (the door state
// itself is unchanged — this is a visibility-only refresh).
func (p *ProcessorImpl) ShowPartyDoorsToCharacter(partyId uint32, ownerIds []character.Id, target character.Id) {
	for _, owner := range ownerIds {
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, owner)
		if err != nil {
			continue
		}
		for _, d := range doors {
			if d.PartyId() != partyId {
				continue
			}
			_ = p.emit(EnvEventTopicDoorStatus, createdEventProvider(d, uint32(target)))
		}
	}
}

// HidePartyDoorsFromCharacter emits a REMOVED status for every door owned by a
// REMAINING party member, targeted at `target` (a member who just left), so the
// leaver stops seeing the party's doors. The leaver's own door is left alone (it
// reslots to solo via ReslotParty and stays visible to them).
func (p *ProcessorImpl) HidePartyDoorsFromCharacter(partyId uint32, ownerIds []character.Id, target character.Id) {
	for _, owner := range ownerIds {
		if owner == target {
			continue
		}
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, owner)
		if err != nil {
			continue
		}
		for _, d := range doors {
			if d.PartyId() != partyId {
				continue
			}
			_ = p.emit(EnvEventTopicDoorStatus, removedEventProvider(d, RemoveReasonPartyLeft, uint32(target)))
		}
	}
}

// LeavePartyDoor transitions a departed member's door OUT of the party and into
// solo scope. It is the counterpart to HidePartyDoorsFromCharacter: that hides the
// REMAINING members' doors from the LEAVER; this hides the LEAVER's door from the
// REMAINING members while the leaver keeps it as a solo door.
//
// For each door owned by the leaver in this party it:
//  1. emits a broadcast REMOVED (still party-scoped) — the remaining members drop
//     it AND clear its town-portal slot (a member who stays in the party renders
//     town doors from the party town-portal array, so the slot must be cleared);
//  2. re-keys the door to solo (partyId 0, slot 0, slot-0 town portal);
//  3. emits a CREATED for the now-solo door, which reaches only the owner.
//
// Without this the leaver's door lingered on every former party member's client,
// and its solo reslot (emitted with the stale partyId) was broadcast back to the
// party — collapsing both members' town portals onto slot 0. A party DISBAND does
// not need this: disband flips the remaining client to solo, which renders from
// its own town portal and ignores the party array entirely.
func (p *ProcessorImpl) LeavePartyDoor(partyId uint32, ownerCharacterId character.Id, townPortalsByMap func(_map.Id) []TownPortal) {
	doors, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		p.l.WithError(err).Warnf("LeavePartyDoor: unable to load doors for owner %d", ownerCharacterId)
		return
	}
	for _, d := range doors {
		if d.PartyId() != partyId {
			continue
		}
		// 1. Remove the still-party-scoped door from the remaining members.
		if err := p.emit(EnvEventTopicDoorStatus, removedEventProvider(d, RemoveReasonPartyLeft, 0)); err != nil {
			p.l.WithError(err).Warnf("LeavePartyDoor: remove failed for door %d", d.AreaDoorId())
		}
		// 2. Re-key to solo: party 0, slot 0, slot-0 town portal.
		wireId, tx, ty, _ := ResolveTownPortal(townPortalsByMap(d.TownMapId()), 0, defaultTownX, defaultTownY)
		n := Clone(d).SetPartyId(0).SetSlot(0).SetTownPortalId(wireId).SetTownX(tx).SetTownY(ty).Build()
		if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
			p.l.WithError(err).Warnf("LeavePartyDoor: persist failed for door %d", d.AreaDoorId())
			continue
		}
		// 3. Re-create as a solo door — reaches only the owner.
		if err := p.emit(EnvEventTopicDoorStatus, createdEventProvider(n, 0)); err != nil {
			p.l.WithError(err).Warnf("LeavePartyDoor: re-create failed for door %d", d.AreaDoorId())
		}
	}
}

func (p *ProcessorImpl) Reslot(areaDoorId uint32, newSlot byte, townPortalId uint32, townX point.X, townY point.Y) error {
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
