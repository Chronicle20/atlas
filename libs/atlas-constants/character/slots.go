package character

// Character-slot bounds shared across atlas-account (owns persistence),
// atlas-login (reads for the world character list), and atlas-channel
// (reads and increments for Maple Life B-Type creation, task-246
// bug-b-type-must-add-a-slot.md).
//
// A per-(account, world) slot count replaces the previous flat,
// account-scoped hardcoded value of 4. MaxCharacterSlotsPerWorld (12) is the
// hard cap String.wz Cash.img gives for Maple Life (B-Type)'s "*Warning : If
// all 12 character slots are full, it cannot be used." -- this is the ruled
// cap, not the "6 per server world" figure from the standalone Extra
// Character Slot Coupon string, which does not apply here.
const (
	// DefaultCharacterSlotsPerWorld is what every (account, world) pair
	// starts at before any explicit slot row exists, matching the value
	// every service hardcoded before task-246.
	DefaultCharacterSlotsPerWorld = int16(4)

	// MaxCharacterSlotsPerWorld is the hard cap a slot increment may never
	// cross.
	MaxCharacterSlotsPerWorld = int16(12)
)
