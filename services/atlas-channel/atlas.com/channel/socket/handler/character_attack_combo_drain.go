package handler

import (
	"atlas-channel/character/buff"
	"math"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
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

// newAttackBuffLoader returns a per-attack memoized buff loader: fetch is
// invoked at most once, on the first call, regardless of how many consumers
// (projectile consumption gate, Pick Pocket, Combo Drain) call the returned
// closure. A failed first fetch is logged once here and cached as "no buffs
// active" (nil, nil) for every subsequent call — callers see the error only
// on that first call and must apply their own degraded posture; the loader
// never re-fetches after either a success or a failure.
func newAttackBuffLoader(l logrus.FieldLogger, fetch func(characterId uint32) ([]buff.Model, error)) func(characterId uint32) ([]buff.Model, error) {
	var buffs []buff.Model
	loaded := false
	return func(characterId uint32) ([]buff.Model, error) {
		if loaded {
			return buffs, nil
		}
		loaded = true
		bs, err := fetch(characterId)
		if err != nil {
			l.WithError(err).Warnf("Unable to load buffs for character [%d] attack; assuming none active.", characterId)
			buffs = nil
			return nil, err
		}
		buffs = bs
		return buffs, nil
	}
}

// comboDrainTryProc evaluates Combo Drain for one accepted attack and emits
// at most one ChangeHP via the injected changeHP: once per attack, computed
// from the plain damage total across all monsters and hit lines (no
// per-monster running total). The gate is the COMBO_DRAIN stat alone — no
// job, skill-ownership, attack-type, or version check. A version whose
// client has no Aran simply never carries the stat, so this is correct on
// every supported version without a branch (design §6.1). Failures are
// logged and swallowed — never abort the attack pipeline. Downstream
// max-HP clamping is owned by atlas-character.
func comboDrainTryProc(
	l logrus.FieldLogger,
	getBuffs func(characterId uint32) ([]buff.Model, error),
	changeHP func(f field.Model, characterId uint32, amount int16) error,
	f field.Model,
	characterId uint32,
	ai packetmodel.AttackInfo,
) {
	buffs, err := getBuffs(characterId)
	if err != nil {
		// The loader already logged the failure at Warn level once per
		// attack; skipping the heal is the FR-1 degraded posture.
		l.WithError(err).Debugf("Combo Drain: buff snapshot unavailable for character [%d]; skipping heal.", characterId)
		return
	}
	percent, ok := buffStatAmount(buffs, charconst.TemporaryStatTypeComboDrain)
	if !ok {
		return
	}
	total := attackTotalDamage(ai)
	heal := comboDrainHealAmount(total, percent)
	if heal <= 0 {
		return
	}
	l.Debugf("Combo Drain heal: caster=[%d] totalDamage=[%d] percent=[%d] heal=[%d].", characterId, total, percent, heal)
	if err := changeHP(f, characterId, heal); err != nil {
		l.WithError(err).Errorf("Combo Drain: CHANGE_HP emit failed for character [%d].", characterId)
	}
}
