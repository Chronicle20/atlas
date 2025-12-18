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
		EvanStage7FireBreathId)
}

func Is(skillId Id, references ...Id) bool {
	for _, r := range references {
		if skillId == r {
			return true
		}
	}
	return false
}
