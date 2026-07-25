package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// ReslotParty recomputes each affected member's door TOWN slot after a party
// membership change and reslots the door — the town-portal position only. It
// NEVER touches the area door (Reslot emits SLOT_CHANGED, which the channel
// renders as a town-portal move; it does not re-send SpawnDoor), so it cannot
// toggle the client door render. Current members reslot to their computed party
// slot; leavers reslot to solo (slot 0).
//
// This keeps each door's stored town-portal position in sync with the member's
// current party slot — which is both the in-town render position AND the warp
// destination when the door is entered. Without it, two party members' doors
// keep their cast-time (slot 0) town position and both warp to portal index 0.
// The area door is a plain map object and is unaffected.
func ReslotParty(p Processor, partyId uint32, members []character.Id, leavers []character.Id, townPortalsByMap func(_map.Id) []TownPortal) error {
	impl := p.(*ProcessorImpl)
	reslot := func(owner character.Id, slot byte) {
		doors, err := GetRegistry().GetByOwner(impl.ctx, impl.t, owner)
		if err != nil {
			impl.l.WithError(err).Warnf("ReslotParty: GetByOwner %d", uint32(owner))
			return
		}
		for _, d := range doors {
			wireId, tx, ty, _ := ResolveTownPortal(townPortalsByMap(d.TownMapId()), slot, defaultTownX, defaultTownY)
			if err := p.Reslot(d.AreaDoorId(), slot, wireId, tx, ty); err != nil {
				impl.l.WithError(err).Warnf("ReslotParty: reslot door %d", d.AreaDoorId())
			}
		}
	}
	for _, owner := range members {
		reslot(owner, ComputeSlot(partyId, members, owner))
	}
	for _, owner := range leavers {
		reslot(owner, 0)
	}
	return nil
}
