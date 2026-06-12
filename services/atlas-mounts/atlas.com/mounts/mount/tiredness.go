package mount

// TickTiredness advances a mount's tiredness by one tick. Tiredness increments
// by 1 per tick and clamps at 99 (FR-6.1). The returned tooTired flag is true
// iff the value could not increase further this tick (input was already at the
// clamp), signaling the mount has become too tired (FR-6.3).
func TickTiredness(t int) (int, bool) {
	if t >= 99 {
		return 99, true
	}
	return t + 1, false
}
