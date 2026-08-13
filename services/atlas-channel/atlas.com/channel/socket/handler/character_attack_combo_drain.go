package handler

import (
	"atlas-channel/character/buff"
	"math"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// buffStatAmount returns the Amount of the first stat change of statType
// carried by a non-expired buff, mirroring hasBuff's matching rules
// (character_attack_projectile.go).
func buffStatAmount(buffs []buff.Model, statType charconst.TemporaryStatType) (int32, bool) {
	for _, b := range buffs {
		if b.Expired() {
			continue
		}
		for _, c := range b.Changes() {
			if c.Type() == string(statType) {
				return c.Amount(), true
			}
		}
	}
	return 0, false
}

// attackTotalDamage sums every damage line across every DamageInfo entry.
// uint64 so a full multi-target attack (15 targets x 15 lines x MaxUint32)
// cannot overflow the sum.
func attackTotalDamage(ai packetmodel.AttackInfo) uint64 {
	total := uint64(0)
	for _, di := range ai.DamageInfo() {
		for _, d := range di.Damages() {
			total += uint64(d)
		}
	}
	return total
}

// comboDrainHealAmount computes totalDamage * percent / 100 in integer
// arithmetic, returning 0 when percent <= 0 or totalDamage == 0, and
// clamping to math.MaxInt16 before narrowing. For any percent >= 1,
// totalDamage >= MaxInt16*100 already guarantees the clamp, so saturate
// early — below that bound totalDamage*percent fits uint64 for any int32
// percent.
func comboDrainHealAmount(totalDamage uint64, percent int32) int16 {
	if percent <= 0 || totalDamage == 0 {
		return 0
	}
	if totalDamage >= uint64(math.MaxInt16)*100 {
		return math.MaxInt16
	}
	heal := totalDamage * uint64(percent) / 100
	if heal > uint64(math.MaxInt16) {
		return math.MaxInt16
	}
	return int16(heal)
}
