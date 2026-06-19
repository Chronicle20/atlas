package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

const (
	townPortalBase byte = 0x80
	maxPartySize        = 6
)

// default door position when a town exposes too few door-type portals (design §6.3).
const (
	defaultTownX point.X = 0
	defaultTownY point.Y = 0
)

// TownPortal is an atlas-data door-type portal position (Type==6), in load order.
type TownPortal struct {
	X point.X
	Y point.Y
}

// ComputeSlot returns the caster's 0-based party door slot (the party door-slot mapping).
// Solo (partyId==0) or non-member → slot 0.
func ComputeSlot(partyId uint32, members []character.Id, ownerCharacterId character.Id) byte {
	if partyId == 0 {
		return 0
	}
	for i, id := range members {
		if id == ownerCharacterId {
			if i >= maxPartySize {
				return maxPartySize - 1
			}
			return byte(i)
		}
	}
	return 0
}

// ResolveTownPortal maps a slot to the wire portal id (0x80+slot) and a town position.
// If the town has >slot door portals, use that portal's position; otherwise fall back to
// the provided default position (still encoding 0x80+slot on the wire). Always ok=true.
func ResolveTownPortal(doorPortals []TownPortal, slot byte, fallbackX point.X, fallbackY point.Y) (wireId uint32, x point.X, y point.Y, ok bool) {
	wireId = uint32(townPortalBase + slot)
	if int(slot) < len(doorPortals) {
		p := doorPortals[slot]
		return wireId, p.X, p.Y, true
	}
	return wireId, fallbackX, fallbackY, true
}
