package _map

// Field limit flags - bitmask values for map restrictions
const (
	// FieldLimitNoTeleport prevents teleportation in the map
	FieldLimitNoTeleport uint32 = 0x01

	// FieldLimitNoMysticDoor prevents mystic door skill usage
	FieldLimitNoMysticDoor uint32 = 0x02

	// FieldLimitNoTeleportItem prevents teleport-rock item usage in the map
	// (client RunMapTransferItem checks fieldLimit & 0x40; design task-124 §1 Q2)
	FieldLimitNoTeleportItem uint32 = 0x40

	// FieldLimitNoSummoningBag prevents summoning bag usage
	FieldLimitNoSummoningBag uint32 = 0x04

	// FieldLimitNoMigrate prevents migration (changing channels)
	FieldLimitNoMigrate uint32 = 0x08

	// FieldLimitNoPortalScroll prevents portal scroll usage
	FieldLimitNoPortalScroll uint32 = 0x10

	// FieldLimitNoMiniRoom prevents opening a mini-room on the map — the
	// CMiniRoomBaseDlg family, which covers mini-games (omok, match cards),
	// personal/hired-merchant shops and trade rooms.
	//
	// Provenance: this bit is NOT derivable from a GMS client. CMiniRoomBaseDlg
	// carries only packet handlers, the room factory, chat and avatar decoding
	// (verified against the GMS v83 IDB) — there is no client-side create/open
	// gate, so the decision is server-side and no client code tests this bit.
	// The value comes from the emulator constant FieldLimit.CANNOTMINIGAME,
	// recorded at docs/tasks/task-133-miniroom-minigames/prd.md:53 and shipped
	// since task-133.
	FieldLimitNoMiniRoom uint32 = 0x80

	// FieldLimitNoRegularExpLoss prevents experience loss on death
	FieldLimitNoRegularExpLoss uint32 = 0x80000
)

// NoMiniRoom returns true if the field limit forbids opening a mini-room on the
// map. A caller that cannot READ the field limit must refuse rather than call
// this with a zero value: a missing flag must never be read as "permitted".
func NoMiniRoom(fieldLimit uint32) bool {
	return fieldLimit&FieldLimitNoMiniRoom != 0
}

// NoExpLossOnDeath returns true if the field limit prevents experience loss on death
func NoExpLossOnDeath(fieldLimit uint32) bool {
	return fieldLimit&FieldLimitNoRegularExpLoss != 0
}
