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

// NeedsMasterLevel reports whether a skill entry in GW_CharacterData carries the
// trailing 4-byte master level. It is a direct port of the client's
// `is_skill_need_master_level(nSkillID)`, which is the ONLY authority: the field
// is per-SKILL, and approximating it with a per-JOB test is what produced the
// task-218 field report (a preset Evan closed the client with error 38 while a
// level-1 Evan logged in fine, because the length only diverges once the
// character owns skills).
//
// The client rule, read per version:
//
//	job = skillId / 10000
//	if job/100 == 22 || job == 2001:          // Evan (and the Evan beginner)
//	    jobLevel(job) in {9, 10}              // == job 2217 or 2218, see below
//	      [ || skillId in {22111001, 22141002, 22140000} — GMS only ]
//	else if job/10 == 43:                     // Dual Blade — see the note below
//	    ...
//	else:
//	    job%100 != 0 && job%10 == 2           // 4th job, excluding job roots
//
// jobLevel is the client's own helper (v87 @0x508fbe): for an Evan job it
// returns job%10 + 2, so 2210->2 … 2217->9, 2218->10. "jobLevel in {9,10}"
// is therefore exactly "job is 2217 or 2218", which is what this implements —
// NOT job.GetSkillBook, whose Evan indexing is offset by one (2210->1 … 2218->9)
// and would select the wrong two growths.
//
// evanExceptions carries the one genuine version divergence. The three-skill
// exception list is present on GMS v84 (@0x4f0ad2), v87 (@0x508f33), v92
// (@0x4792f0) and v95 (@0x47ccb0) but ABSENT on JMS v185 (@0x47d2a8). GMS v83
// (@0x4e8f04) also lacks it, which is moot: Evan launched at v84, so no v83
// character can reach the Evan branch at all. Callers pass region == "GMS".
//
// Dual Blade (job 430-434) is deliberately not modelled: the arm diverges four
// ways across the same six clients (v83/v84 have no arm, v87 returns false,
// v92/v95/jms return jobLevel==4 plus a version-specific id list), and
// atlas-constants defines no 430-434 job, so no Atlas character can reach it.
// Add it WITH its per-version id lists if Dual Blade is ever introduced; do not
// let it fall through to the 4th-job rule, which is what v83 did.
func NeedsMasterLevel(skillId Id, evanExceptions bool) bool {
	jobId := uint32(skillId) / 10000
	if jobId/100 == 22 || jobId == 2001 {
		if jobId == 2217 || jobId == 2218 {
			return true
		}
		if !evanExceptions {
			return false
		}
		return Is(skillId, Id(22111001), Id(22141002), Id(22140000))
	}
	if jobId%100 == 0 {
		return false
	}
	return jobId%10 == 2
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
