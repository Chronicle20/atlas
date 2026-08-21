package portal

import (
	portalData "atlas-channel/data/portal"
	"atlas-channel/kafka/message/portal"
	"atlas-channel/movement"
	"atlas-channel/position"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// maxPortalEntryDistance bounds how far the character's last known position
// may be from the source portal for an inner-portal claim to be plausible.
// Unit: map coordinate units — the same units as portal x/y and character
// x/y. The value is DERIVED from the client's own portal collision rect, not
// chosen: see docs/tasks/task-250-inner-portal-registration/structures/gms_v95.md
// "## Threshold derivation". Do not change it without redoing that derivation.
const (
	maxPortalEntryDistance   = 81
	maxPortalEntryDistanceSq = maxPortalEntryDistance * maxPortalEntryDistance
)

type Processor interface {
	Enter(f field.Model, portalName string, characterId uint32) error
	Warp(f field.Model, characterId uint32, targetMapId _map.Id) error
	WarpToPosition(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16) error
	// EnterInner registers an inner-portal teleport: the client already moved
	// itself inside the current field and reported it. Every coordinate the
	// server adopts comes from ITS OWN portal data — the packet's claim is used
	// only for plausibility comparison and logging (PRD FR-4.5).
	//
	// A refusal is a deliberate no-op, not a handler failure: it logs at warning,
	// updates nothing, and returns nil. The character's next MOVE re-establishes
	// position exactly as it does today, so a false positive degrades to current
	// behaviour rather than to a broken portal (PRD FR-4.6). The only non-nil
	// error this returns is one from the position publish itself.
	EnterInner(f field.Model, characterId uint32, sourcePortalName string,
		claimedX int16, claimedY int16, claimedTargetX int16, claimedTargetY int16) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	pd  portalData.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		pd:  portalData.NewProcessor(l, ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// newMovementProcessor is the movement seam for EnterInner. EnterInner's only
// movement call is TeleportCharacter, which publishes a Kafka command and
// emits no clientbound packet, so the writer.Producer a movement Processor
// carries is unused on this path — hence the nil. Package-level var so portal
// tests can inject movement/mock without a live producer (precedent: warpFunc
// in socket/handler/mystic_door_enter.go, monsterByIdFn in
// movement/processor.go).
var newMovementProcessor = func(l logrus.FieldLogger, ctx context.Context) movement.Processor {
	return movement.NewProcessor(l, ctx, nil)
}

func (p *ProcessorImpl) Enter(f field.Model, portalName string, characterId uint32) error {
	pm, err := p.pd.GetInMapByName(f.MapId(), portalName)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, f.MapId())
		return err
	}
	return producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(EnterCommandProvider(f, pm.Id(), characterId))
}

func (p *ProcessorImpl) Warp(f field.Model, characterId uint32, targetMapId _map.Id) error {
	return producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(WarpCommandProvider(f, characterId, targetMapId))
}

// WarpToPosition warps the character to an exact (x, y) coordinate in the
// target map — used by Mystic Door to land the user on the linked door's
// position rather than a named portal.
func (p *ProcessorImpl) WarpToPosition(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(WarpToPositionCommandProvider(f, characterId, targetMapId, x, y))
}

// WarpToPortal warps the character to a specific portal in the target map. A
// targetPortalId of 0 falls back to the random-spawn Warp.
func (p *ProcessorImpl) WarpToPortal(f field.Model, characterId uint32, targetMapId _map.Id, targetPortalId uint32) error {
	if targetPortalId == 0 {
		return p.Warp(f, characterId, targetMapId)
	}
	return producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(WarpToPortalCommandProvider(f, characterId, targetMapId, targetPortalId))
}

// EnterInner is documented on the Processor interface.
func (p *ProcessorImpl) EnterInner(f field.Model, characterId uint32, sourcePortalName string,
	claimedX int16, claimedY int16, claimedTargetX int16, claimedTargetY int16,
) error {
	sp, err := p.pd.GetInMapByName(f.MapId(), sourcePortalName)
	if err != nil {
		p.l.WithError(err).Warnf("Character [%d] claimed inner-portal entry via unresolvable portal [%s] in map [%d].", characterId, sourcePortalName, f.MapId())
		return nil
	}

	if sp.TargetMapId().IsSentinel() || sp.TargetMapId() != f.MapId() {
		p.l.Warnf("Character [%d] claimed inner-portal entry via portal [%s] in map [%d] whose targetMapId [%d] is not the current map.", characterId, sourcePortalName, f.MapId(), sp.TargetMapId())
		return nil
	}

	if sp.Target() == "" {
		p.l.Warnf("Character [%d] claimed inner-portal entry via portal [%s] in map [%d] which has no target portal.", characterId, sourcePortalName, f.MapId())
		return nil
	}

	dp, err := p.pd.GetInMapByName(f.MapId(), sp.Target())
	if err != nil {
		p.l.WithError(err).Warnf("Character [%d] claimed inner-portal entry via portal [%s] in map [%d] whose target portal [%s] is unresolvable.", characterId, sourcePortalName, f.MapId(), sp.Target())
		return nil
	}

	if last, ok := position.GetRegistry().Lookup(tenant.MustFromContext(p.ctx), characterId); ok {
		dx, dy := int32(last.X)-int32(sp.X()), int32(last.Y)-int32(sp.Y())
		distSq := dx*dx + dy*dy
		if distSq > maxPortalEntryDistanceSq {
			p.l.Warnf("Character [%d] in map [%d] claimed inner-portal entry via portal [%s] at (%d,%d) but last known position (%d,%d) is [%d] squared units away, exceeding the threshold [%d].",
				characterId, f.MapId(), sourcePortalName, sp.X(), sp.Y(), last.X, last.Y, distSq, maxPortalEntryDistanceSq)
			return nil
		}
	}

	if claimedTargetX != dp.X() || claimedTargetY != dp.Y() {
		p.l.Warnf("Character [%d] claimed inner-portal entry via portal [%s] targeting (%d,%d) but the destination portal [%s] in map [%d] is at (%d,%d).",
			characterId, sourcePortalName, claimedTargetX, claimedTargetY, sp.Target(), f.MapId(), dp.X(), dp.Y())
		return nil
	}

	p.l.Debugf("Character [%d] entering inner portal [%s] in map [%d]; adopting server destination portal [%s] at (%d,%d) (claimed source (%d,%d)).",
		characterId, sourcePortalName, f.MapId(), sp.Target(), dp.X(), dp.Y(), claimedX, claimedY)
	return newMovementProcessor(p.l, p.ctx).TeleportCharacter(f, characterId, dp.X(), dp.Y())
}
