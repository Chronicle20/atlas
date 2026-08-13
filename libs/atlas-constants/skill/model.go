package skill

type Skill struct {
	id     Id
	buff   bool
	charge bool
}

func (s Skill) Id() Id {
	return s.id
}

func (s Skill) Buff() bool {
	return s.buff
}

func (s Skill) Charge() bool {
	return s.charge
}

func IsBuff(skillId Id) bool {
	s, ok := Skills[skillId]
	if !ok {
		return false
	}
	return s.Buff()
}

func NeedsCharging(skillId Id) bool {
	s, ok := Skills[skillId]
	if !ok {
		return false
	}
	return s.Charge()
}

func IsShootSkillNotUsingShootingWeapon(skillId Id) bool {
	switch skillId {
	case NightLordTauntId, ShadowerTauntId, BuccaneerEnergyOrbId, DawnWarriorStage2SoulBladeId, ThunderBreakerStage3SparkId, ThunderBreakerStage3SharkWaveId, AranStage2ComboSmashId, AranStage3ComboFenrirId, AranStage4ComboTempestId:
		return true
	default:
		return false
	}
}

func IsShootSkillNotConsumingBullet(skillId Id) bool {
	if IsShootSkillNotUsingShootingWeapon(skillId) {
		return true
	}
	switch skillId {
	case HunterPowerKnockBackId, CrossbowmanPowerKnockBackId, HermitShadowMesoId, WindArcherStage2StormBreakId, NightWalkerStage2VampireId:
		return true
	default:
		return false
	}
}

func IsKeyDownSkill(skillId Id) bool {
	return Is(skillId,
		FirePoisonArchMagicianBigBangId,
		IceLightningArchMagicianBigBangId,
		BishopBigBangId,
		HeroMonsterMagnetId,
		PaladinMonsterMagnetId,
		DarkKnightMonsterMagnetId,
		BowmasterHurricaneId,
		MarksmanPiercingArrowId,
		CorsairRapidFireId,
		NightWalkerStage3PoisonBombId,
		WindArcherStage3HurricaneId,
		ThunderBreakerStage2CorkscrewBlowId,
		EvanStage4IceBreathId,
		EvanStage7FireBreathId,
		BrawlerCorkscrewBlowId, // 5101004 — IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185 (task-161)
		GunslingerGrenadeId)    // 5201002 — IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185 (task-161)
}

// IsGrenadeSkill reports whether an attack by this skill carries the trailing
// grenade landing-point coordinate pair (AttackInfo.GrenadeX/GrenadeY).
//
// This is the single authority for that block: the attack codec reads it here
// and callers that want the landing point gate on the same predicate, so the
// "does this packet have grenade coords" question cannot be answered two ways.
//
// The membership is exactly the set the codec has an IDA-derived read order
// for. Poison Bomb is the only skill Atlas has verified writes the pair; adding
// another grenade-style skill needs its own client read, not an inference from
// the name or from IsKeyDownSkill membership (Gunslinger Grenade is keydown but
// is NOT verified to carry this block, so it is deliberately absent).
func IsGrenadeSkill(skillId Id) bool {
	return Is(skillId, NightWalkerStage3PoisonBombId)
}

func Is(skillId Id, references ...Id) bool {
	for _, r := range references {
		if skillId == r {
			return true
		}
	}
	return false
}
