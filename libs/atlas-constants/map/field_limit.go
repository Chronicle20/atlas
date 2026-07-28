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

	// FieldLimitNoRegularExpLoss prevents experience loss on death
	FieldLimitNoRegularExpLoss uint32 = 0x80000
)

// NoExpLossOnDeath returns true if the field limit prevents experience loss on death
func NoExpLossOnDeath(fieldLimit uint32) bool {
	return fieldLimit&FieldLimitNoRegularExpLoss != 0
}
