package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// ReslotParty recomputes every affected party member's door slot after a
// membership change and calls Reslot on each whose slot actually changed
// (FR-4.3 / FR-6.4).
//
// Parameters:
//   - p: the door processor (carries ctx/tenant/emit/registry seams).
//   - partyId: the party whose membership changed.
//   - newMembers: ordered member list AFTER the change (leader at index 0).
//   - formerMembers: character ids that LEFT the party in this event.
//   - townPortalsByMap: closure that returns the door-type town portals for a
//     given town map id; used to resolve the new TownX/TownY/TownPortalId.
//
// Algorithm:
//  1. For each owner in newMembers: newSlot = ComputeSlot(partyId, newMembers, owner).
//  2. For each owner in formerMembers: newSlot = 0 (solo scope).
//  3. For each candidate, load their doors via GetByOwner filtered to this partyId.
//  4. Resolve the new town portal from the door's TownMapId.
//  5. Call processor.Reslot — which no-ops when slot unchanged, else
//     persists and emits SLOT_CHANGED.
func ReslotParty(
	p *ProcessorImpl,
	partyId uint32,
	newMembers []character.Id,
	formerMembers []character.Id,
	townPortalsByMap func(_map.Id) []TownPortal,
) error {
	process := func(ownerCharacterId character.Id, newSlot byte) error {
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
		if err != nil {
			return err
		}
		for _, d := range doors {
			if d.PartyId() != partyId {
				continue
			}
			portals := townPortalsByMap(d.TownMapId())
			wireId, tx, ty, _ := ResolveTownPortal(portals, newSlot, defaultTownX, defaultTownY)
			if err := p.Reslot(d.AreaDoorId(), newSlot, wireId, tx, ty); err != nil {
				p.l.WithError(err).Warnf("reslot failed for door %d owner %d", d.AreaDoorId(), ownerCharacterId)
			}
		}
		return nil
	}

	// Remaining members: recompute slot against new ordering.
	for _, owner := range newMembers {
		newSlot := ComputeSlot(partyId, newMembers, owner)
		if err := process(owner, newSlot); err != nil {
			p.l.WithError(err).Warnf("ReslotParty: error processing remaining member %d", owner)
		}
	}

	// Former members: drop to solo slot 0.
	for _, owner := range formerMembers {
		if err := process(owner, 0); err != nil {
			p.l.WithError(err).Warnf("ReslotParty: error processing former member %d", owner)
		}
	}

	return nil
}
