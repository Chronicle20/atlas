package effect

// Model is an immutable skill-effect value carrying the fields that the doors
// resolver needs. It mirrors the shape returned by atlas-data's skill effects
// array, restricted to the door-relevant getters.
type Model struct {
	duration    int32
	mpConsume   uint16
	itemConsume uint32
}

// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel (identical to atlas-channel's contract).
func (m Model) Duration() int32 {
	return m.duration
}

// MPConsume returns the MP cost for this skill effect level.
func (m Model) MPConsume() uint16 {
	return m.mpConsume
}

// ItemConsume returns the item id consumed when casting (WZ itemCon).
// Zero means no item is consumed.
func (m Model) ItemConsume() uint32 {
	return m.itemConsume
}
