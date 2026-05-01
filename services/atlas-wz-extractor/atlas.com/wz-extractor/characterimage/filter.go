package characterimage

// Slot keys that are silently dropped from a render request before
// compositing. Pet/mount/cash slots are intentionally not rendered in v1.
var droppedSlots = map[int]struct{}{
	-14: {}, // pet
	-18: {}, -19: {}, -20: {}, // mount
	-21: {}, -22: {}, -23: {}, -24: {}, -25: {},
	-26: {}, -27: {}, -28: {}, -29: {}, -30: {},
}

// FilterEquipment returns a copy of `in` with mount/pet/cash slots removed.
// Cash slots (-101..-114) are dropped via numeric range so we don't have to
// enumerate them.
func FilterEquipment(in map[int]int) map[int]int {
	out := make(map[int]int, len(in))
	for slot, id := range in {
		if _, dropped := droppedSlots[slot]; dropped {
			continue
		}
		if slot <= -101 && slot >= -114 {
			continue
		}
		out[slot] = id
	}
	return out
}
