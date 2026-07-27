package mprecovery

import "math"

// Amounts returns (hpLost, mpGain) for MP Recovery (5101005), per the
// WZ-verified v83 formula: hpLost = floor(maxHp / x), mpGain =
// floor(hpLost * y / 100), integer floor division at each step.
// mpGain is computed from the full intended loss, not any post-clamp delta.
// x <= 0 returns (0, 0) — the caller treats that as "skip, warn" (bad tenant
// data). Computation is int32 then narrowed with a MaxInt16 clamp so
// pathological tenant data can never wrap negative; a negative y floors
// mpGain at zero.
func Amounts(maxHp uint16, x int16, y int16) (int16, int16) {
	if x <= 0 {
		return 0, 0
	}
	hpLost := int32(maxHp) / int32(x)
	mpGain := hpLost * int32(y) / 100
	if hpLost > math.MaxInt16 {
		hpLost = math.MaxInt16
	}
	if mpGain > math.MaxInt16 {
		mpGain = math.MaxInt16
	}
	if mpGain < 0 {
		mpGain = 0
	}
	return int16(hpLost), int16(mpGain)
}
