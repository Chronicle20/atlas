package door

const (
	townPortalBase byte = 0x80
	maxPartySize        = 6
	// default door position when a town exposes too few door-type portals (design §6.3).
	defaultTownX int16 = 0
	defaultTownY int16 = 0
)

// TownPortal is an atlas-data door-type portal position (Type==6), in load order.
type TownPortal struct {
	X int16
	Y int16
}

// ComputeSlot returns the caster's 0-based party door slot (Cosmic Party.getPartyDoor).
// Solo (partyId==0) or non-member → slot 0.
func ComputeSlot(partyId uint32, members []uint32, ownerCharacterId uint32) byte {
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
func ResolveTownPortal(doorPortals []TownPortal, slot byte, fallbackX, fallbackY int16) (wireId uint32, x int16, y int16, ok bool) {
	wireId = uint32(townPortalBase + slot)
	if int(slot) < len(doorPortals) {
		p := doorPortals[slot]
		return wireId, p.X, p.Y, true
	}
	return wireId, fallbackX, fallbackY, true
}
