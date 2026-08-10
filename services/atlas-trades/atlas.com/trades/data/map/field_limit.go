package mapdata

import _map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

// TradeDisallowed reports whether the map's fieldLimit forbids opening a trade
// room there (FR-4.6).
//
// It is the shared mini-room bit (_map.FieldLimitNoMiniRoom), deliberately
// reused rather than given a trade-only value: a trade room IS a mini-room —
// room types 3 and 6 of the same CMiniRoomBaseDlg family the mini-game and
// merchant rooms belong to — so a map that forbids opening a mini-room forbids
// opening a trade room. No trade-specific bit is derivable from any GMS client,
// because the family has no client-side create/open gate at all (see the
// constant's provenance note in the shared library).
//
// Callers that cannot READ the field limit must refuse the trade rather than
// calling this with a zero value: a missing flag must never be read as
// "tradeable" (design §7).
func TradeDisallowed(fieldLimit uint32) bool {
	return _map.NoMiniRoom(fieldLimit)
}
