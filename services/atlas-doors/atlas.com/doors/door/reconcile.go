package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// ReconcileParty projects every affected door's party scope from the
// authoritative post-change membership and emits the minimal status deltas.
// It replaces the per-event delta methods (Join/Leave/Disband/Show/Hide/Reslot).
//
// partyId : the party whose membership changed (never 0 here).
// members : authoritative post-change ordered list (leader index 0); empty on disband.
// joiners : ids that joined THIS event (gain visibility of existing party doors).
// leavers : ids that left THIS event (expelled/left/all former members on disband).
func ReconcileParty(
	p *ProcessorImpl,
	partyId uint32,
	members []character.Id,
	joiners []character.Id,
	leavers []character.Id,
	townPortalsByMap func(_map.Id) []TownPortal,
) error {
	inParty := make(map[character.Id]bool, len(members))
	for _, m := range members {
		inParty[m] = true
	}
	participants := dedupIds(append(append([]character.Id{}, members...), leavers...))

	for _, o := range participants {
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, o)
		if err != nil {
			p.l.WithError(err).Warnf("ReconcileParty: GetByOwner %d", uint32(o))
			continue
		}
		for _, d := range doors {
			if d.PartyId() != partyId && !inParty[o] {
				continue // door not relevant to this party
			}
			if inParty[o] {
				p.reconcileMemberDoor(partyId, members, o, d, townPortalsByMap)
			} else {
				p.dropDoorToSolo(participants, o, d, townPortalsByMap)
			}
		}
	}

	// Joiners gain visibility of every OTHER current member's party door.
	for _, j := range joiners {
		if !inParty[j] {
			continue
		}
		p.showPartyDoorsTo(partyId, members, j)
	}
	// Leavers lose visibility of every current member's party door.
	for _, o := range leavers {
		if inParty[o] {
			continue
		}
		p.hidePartyDoorsFrom(partyId, members, o)
	}
	return nil
}

func dedupIds(ids []character.Id) []character.Id {
	seen := make(map[character.Id]bool, len(ids))
	out := make([]character.Id, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (p *ProcessorImpl) reconcileMemberDoor(partyId uint32, members []character.Id, owner character.Id, d Model, townPortalsByMap func(_map.Id) []TownPortal) {
	desiredSlot := ComputeSlot(partyId, members, owner)
	wireId, tx, ty, _ := ResolveTownPortal(townPortalsByMap(d.TownMapId()), desiredSlot, defaultTownX, defaultTownY)

	if d.PartyId() == partyId {
		if d.Slot() == desiredSlot {
			return
		}
		if err := p.Reslot(d.AreaDoorId(), desiredSlot, wireId, tx, ty); err != nil {
			p.l.WithError(err).Warnf("ReconcileParty: reslot door %d", d.AreaDoorId())
		}
		return
	}

	// Adopt solo/other-party door into this party.
	oldSlot := d.Slot()
	n := Clone(d).SetPartyId(partyId).SetSlot(desiredSlot).
		SetTownPortalId(wireId).SetTownX(tx).SetTownY(ty).Build()
	if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
		p.l.WithError(err).Warnf("ReconcileParty: adopt persist door %d", d.AreaDoorId())
		return
	}
	for _, m := range members {
		if m == owner {
			continue // owner already renders the area door; no re-send (no flicker)
		}
		_ = p.emit(EnvEventTopicDoorStatus, createdEventProvider(n, uint32(m)))
	}
	// Owner: town/array-only transition (clears old slot, sets new array slot).
	_ = p.emit(EnvEventTopicDoorStatus, slotChangedEventProvider(n, oldSlot))
	p.l.WithFields(logrus.Fields{
		"door_action": "reconcile_adopt", "party_id": partyId, "owner": uint32(owner),
		"area_door_id": d.AreaDoorId(), "old_slot": oldSlot, "new_slot": desiredSlot,
	}).Infof("ReconcileParty: adopted door [%d] -> party [%d] slot [%d].", d.AreaDoorId(), partyId, desiredSlot)
}

func (p *ProcessorImpl) dropDoorToSolo(participants []character.Id, owner character.Id, d Model, townPortalsByMap func(_map.Id) []TownPortal) {
	for _, m := range participants {
		if m == owner {
			continue
		}
		_ = p.emit(EnvEventTopicDoorStatus, removedEventProvider(d, RemoveReasonPartyLeft, uint32(m)))
	}
	wireId, tx, ty, _ := ResolveTownPortal(townPortalsByMap(d.TownMapId()), 0, defaultTownX, defaultTownY)
	n := Clone(d).SetPartyId(0).SetSlot(0).SetTownPortalId(wireId).SetTownX(tx).SetTownY(ty).Build()
	if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
		p.l.WithError(err).Warnf("ReconcileParty: solo persist door %d", d.AreaDoorId())
		return
	}
	_ = p.emit(EnvEventTopicDoorStatus, createdEventProvider(n, 0))
	p.l.WithFields(logrus.Fields{
		"door_action": "reconcile_solo", "owner": uint32(owner), "area_door_id": d.AreaDoorId(),
	}).Infof("ReconcileParty: door [%d] -> solo.", d.AreaDoorId())
}

func (p *ProcessorImpl) showPartyDoorsTo(partyId uint32, members []character.Id, target character.Id) {
	for _, m := range members {
		if m == target {
			continue
		}
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, m)
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

func (p *ProcessorImpl) hidePartyDoorsFrom(partyId uint32, members []character.Id, target character.Id) {
	for _, m := range members {
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, m)
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
