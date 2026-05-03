package movement

// basicAttackRangeLo / basicAttackRangeHi are the inclusive bounds for a
// basic mob attack action. The classification is taken from Cosmic v83's
// MoveLifeHandler.java:108 — values outside this band are not basic attacks
// (they may be movement, stand, hit, fall, or — for [42, 59] — a named
// skill, which atlas-channel handles via the existing skill-id branch).
const (
	basicAttackRangeLo int8 = 24
	basicAttackRangeHi int8 = 41
)

// basicAttackPos returns the 0-indexed attack-position derived from the
// inbound MoveLife.nActionAndDir byte, or false when the byte is outside
// the basic-attack band.
func basicAttackPos(rawActionAndDir int8) (uint8, bool) {
	if rawActionAndDir < basicAttackRangeLo || rawActionAndDir > basicAttackRangeHi {
		return 0, false
	}
	return uint8((rawActionAndDir - basicAttackRangeLo) / 2), true
}
