package surprise

// HasRoomForSwap reports whether a compartment holding assetCount assets at
// the given capacity can absorb a Surprise open.
//
// The reward is created while the box is consumed, so the PEAK slot count
// decides, not the net:
//   - boxQuantity > 1: the box row survives the decrement, so the reward
//     needs its own free slot.
//   - boxQuantity == 1: the box row is released, so the grant is
//     slot-neutral and an exactly-full locker is fine.
//
// An over-capacity locker (assetCount > capacity, possible through data
// drift) is rejected in both branches: the neutral case permits equality,
// not excess.
func HasRoomForSwap(assetCount uint32, capacity uint32, boxQuantity uint32) bool {
	if boxQuantity == 1 {
		return assetCount <= capacity
	}
	return assetCount < capacity
}
