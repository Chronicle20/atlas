// Package skill defines the semantic pet skills taught/removed by 0519 pet
// skill pouch items, and the Atlas-canonical flag bits used to persist them on
// the pet model. These bits are Atlas-internal storage semantics, deliberately
// decoupled from client wire bits (usPetSkill), which are version-dependent
// and resolve from tenant configuration (DOM-25).
package skill

// Key is the semantic identifier of a pet skill, spelled exactly as the 0519
// item WZ keys. The pet-equip family (Character.wz/PetEquip) spells DropSweep
// as "sweepForDrop"; the 0519 pouch family (Item.wz/Cash/0519.img) calls the
// same ability "dropSweep".
type Key string

const (
	PickupItem   = Key("pickupItem")
	ConsumeHP    = Key("consumeHP")
	LongRange    = Key("longRange")
	DropSweep    = Key("dropSweep")
	PickupAll    = Key("pickupAll")
	IgnorePickup = Key("ignorePickup")
	ConsumeMP    = Key("consumeMP")
	Recall       = Key("recall")
	AutoSpeaking = Key("autoSpeaking")
)

// Flag is the Atlas-canonical pet skill bitmask persisted on the pet model.
type Flag uint16

const (
	FlagPickupItem   = Flag(1 << 0)
	FlagConsumeHP    = Flag(1 << 1)
	FlagLongRange    = Flag(1 << 2)
	FlagDropSweep    = Flag(1 << 3)
	FlagPickupAll    = Flag(1 << 4)
	FlagIgnorePickup = Flag(1 << 5)
	FlagConsumeMP    = Flag(1 << 6)
	FlagRecall       = Flag(1 << 7)
	FlagAutoSpeaking = Flag(1 << 8)
)

var ordered = []Key{PickupItem, ConsumeHP, LongRange, DropSweep, PickupAll, IgnorePickup, ConsumeMP, Recall, AutoSpeaking}

var bits = map[Key]Flag{
	PickupItem:   FlagPickupItem,
	ConsumeHP:    FlagConsumeHP,
	LongRange:    FlagLongRange,
	DropSweep:    FlagDropSweep,
	PickupAll:    FlagPickupAll,
	IgnorePickup: FlagIgnorePickup,
	ConsumeMP:    FlagConsumeMP,
	Recall:       FlagRecall,
	AutoSpeaking: FlagAutoSpeaking,
}

// All returns the nine skills in canonical bit order.
func All() []Key {
	res := make([]Key, len(ordered))
	copy(res, ordered)
	return res
}

// BitFor returns the canonical flag bit for a semantic key.
func BitFor(k Key) (Flag, bool) {
	b, ok := bits[k]
	return b, ok
}

// Has reports whether the mask contains the skill.
func Has(mask uint16, k Key) bool {
	b, ok := bits[k]
	return ok && mask&uint16(b) != 0
}

// Apply sets or clears the skill on the mask. Unknown keys are a no-op.
func Apply(mask uint16, k Key, enabled bool) uint16 {
	b, ok := bits[k]
	if !ok {
		return mask
	}
	if enabled {
		return mask | uint16(b)
	}
	return mask &^ uint16(b)
}
