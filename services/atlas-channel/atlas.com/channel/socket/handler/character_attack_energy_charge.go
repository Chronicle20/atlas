package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	buff2 "atlas-channel/kafka/message/buff"
	"context"

	"github.com/sirupsen/logrus"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	// energyChargeGainPerMob is the bar gain per attacked monster. Cosmic
	// calls handleEnergyChargeGain once per attacked mob
	// (CloseRangeDamageHandler.java:136-140); Atlas collapses the loop into
	// one emit of 102 x mobs so the attack path costs at most one Kafka
	// message (NFR-1).
	energyChargeGainPerMob = int32(102)
	// energyChargeCap is the accumulation ceiling. Reaching it promotes the
	// character to the charged state.
	energyChargeCap = int32(10000)
	// energyChargedValue is the charged-state SENTINEL, not a bar reading.
	// Nothing may treat it as "150% full" (FR-3.1).
	energyChargedValue = int32(15000)
)

// energyLine is the Energy Charge skill line a character owns: the tenant's
// wire id for that line's Energy Charge skill, plus the character's level in
// it. The level selects the WZ effect row that supplies the charged window's
// duration and its weapon-attack payoff.
type energyLine struct {
	skillId skill3.Id
	level   byte
}

// energyChargeLine resolves the character's Energy Charge line from owned
// skills, adventurer branch first. ok == false when the character owns
// neither variant at level > 0.
//
// Identities, not raw ids: set.Wire returns false for the Cygnus identity on
// gms_v61 (Thunder Breaker postdates it), so that branch degrades to a no-op
// rather than resolving a bogus id (AC-10). No job check is needed —
// owning the skill at level > 0 already implies the line, which is all
// Cosmic's isCygnus() split was ever deciding.
func energyChargeLine(set skill3.Set, skills []skill.Model) (energyLine, bool) {
	find := func(id skill3.Id) byte {
		for _, s := range skills {
			if s.Id() == id {
				return s.Level()
			}
		}
		return 0
	}
	for _, identity := range []skill3.Identity{skill3.MarauderEnergyCharge, skill3.ThunderBreakerStage2EnergyCharge} {
		wire, ok := set.Wire(identity)
		if !ok {
			continue
		}
		if lvl := find(wire); lvl > 0 {
			return energyLine{skillId: wire, level: lvl}, true
		}
	}
	return energyLine{}, false
}

// energyChargeGainAmount is the bar gain for one attack: 102 per attacked
// monster, or nothing when the attack hit nothing (FR-2.4).
func energyChargeGainAmount(mobsHit int) int32 {
	if mobsHit <= 0 {
		return 0
	}
	return energyChargeGainPerMob * int32(mobsHit)
}

// energyChargeQualifies reports whether an attack feeds the energy bar. Melee
// covers every close-range attack including the basic attack (skillId 0);
// AttackTypeEnergy is the Energy Charge aura's own touch damage, which Cosmic
// routes through the same close-range handler; on the ranged path ONLY
// Thunder Breaker Shark Wave qualifies (RangedAttackHandler.java:90-99).
func energyChargeQualifies(at packetmodel.AttackType, attackId skill3.Identity, attackIdOk bool) bool {
	switch at {
	case packetmodel.AttackTypeMelee, packetmodel.AttackTypeEnergy:
		return true
	case packetmodel.AttackTypeRanged:
		return attackIdOk && skill3.IsIdentity(attackId, skill3.ThunderBreakerStage3SharkWave)
	}
	return false
}

// isEnergyBlast reports whether a cast is the charge-gated Energy Blast.
// Energy Blast is the only skill in the family that carries no mpCon at any
// level — the sole WZ evidence that it is energy-costed rather than
// MP-costed. Shockwave (mpCon 18) and Shark Wave (mpCon 15) are NOT gated
// (design.md OQ-3).
func isEnergyBlast(attackId skill3.Identity, attackIdOk bool) bool {
	return attackIdOk && skill3.IsIdentity(attackId, skill3.MarauderEnergyBlast, skill3.ThunderBreakerStage2EnergyBlast)
}

// energyChargeDeps groups the side-effecting call energyChargeTryUpdate makes
// so tests can drive every branch without a real processor or Kafka producer.
type energyChargeDeps struct {
	emitUpsert func(sourceId int32, level byte, amount int32, capValue int32) error
}

// energyChargeProductionDeps wires energyChargeDeps to the buff
// UPDATE_STAT_VALUE emitter for one attack. CreateIfMissing is what keeps the
// attack path free of a REST read: the channel never has to ask whether the
// bar buff exists (design.md §4.2).
func energyChargeProductionDeps(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) energyChargeDeps {
	bp := buff.NewProcessor(l, ctx)
	return energyChargeDeps{
		emitUpsert: func(sourceId int32, level byte, amount int32, capValue int32) error {
			return bp.UpdateStatValue(f, characterId, buff.StatValueUpdate{
				SourceId:        sourceId,
				StatType:        string(constants.TemporaryStatTypeEnergyCharge),
				Operation:       buff2.StatOperationIncrement,
				Amount:          amount,
				Cap:             capValue,
				CreateIfMissing: true,
				Level:           level,
			})
		},
	}
}

// energyChargeTryUpdate applies Energy Charge bar bookkeeping for one
// qualifying attack: at most ONE emit, carrying 102 x mobs clamped to 10000.
//
// "No gain while charged" (FR-2.5) is structural rather than a guard here:
// at the 15000 sentinel atlas-buffs' current >= cap test makes the increment
// a no-op and emits no status event, so the channel broadcasts nothing.
//
// All failures are logged and swallowed — the attack pipeline never fails on
// energy bookkeeping (FR-2.6 / NFR-2).
func energyChargeTryUpdate(l logrus.FieldLogger, set skill3.Set, c character.Model, ai packetmodel.AttackInfo, deps energyChargeDeps) {
	line, ok := energyChargeLine(set, c.Skills())
	if !ok {
		return
	}
	amount := energyChargeGainAmount(len(ai.DamageInfo()))
	if amount == 0 {
		return
	}
	if err := deps.emitUpsert(int32(line.skillId), line.level, amount, energyChargeCap); err != nil {
		l.WithError(err).Errorf("Energy Charge: gain emit failed for character [%d] energy line [%d].", c.Id(), line.skillId)
	}
}
