package mapdata

// fieldLimitNoMiniRoom is the WZ `fieldLimit` bit that forbids opening a
// mini-room on a map.
//
// PROVENANCE, and the limit of it. `fieldLimit` never reaches the client: the
// GMS v83 IDB contains no "fieldLimit" string at all (verified with a whole-image
// string search of MapleStory_dump.exe.i64), so no client read order can be
// decompiled to name a trade-specific bit. The only in-repo, already-shipped
// reading of this bitmask is atlas-mini-games' create ladder
// (services/atlas-mini-games/atlas.com/mini-games/game/processor.go:305), which
// treats bit 0x80 as "this map forbids opening a mini-room" (task-133).
//
// A trade room IS a mini-room — the same CMiniRoomBaseDlg family, room types 3
// and 6 — so a map that forbids opening a mini-room forbids opening a trade
// room. This constant therefore deliberately reuses that one verified bit
// rather than inventing a trade-only bit that nothing in the client, the WZ
// schema or the repo can corroborate.
const fieldLimitNoMiniRoom uint32 = 0x80

// TradeDisallowed reports whether the map's fieldLimit forbids opening a trade
// room there (FR-4.6). Callers that cannot READ the field limit must refuse the
// trade rather than calling this with a zero value — a missing flag must never
// be read as "allowed" (design §7).
func TradeDisallowed(fieldLimit uint32) bool {
	return fieldLimit&fieldLimitNoMiniRoom != 0
}
