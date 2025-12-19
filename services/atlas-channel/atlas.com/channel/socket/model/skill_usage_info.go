package model

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type SkillUsageInfo struct {
	updateTime                uint32
	skillId                   uint32
	skillLevel                byte
	castX                     int16
	castY                     int16
	spiritJavelinItemId       uint32
	affectedPartyMemberBitmap uint8
	delay                     uint16
}

func (m *SkillUsageInfo) Decode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.updateTime = r.ReadUint32()
		m.skillId = r.ReadUint32()
		m.skillLevel = r.ReadByte()
		if isAntiRepeatBuffSkill(skill.Id(m.skillId)) {
			m.castX = r.ReadInt16()
			m.castY = r.ReadInt16()
		}
		if skill.Id(m.skillId) == skill.NightLordShadowStarsId {
			m.spiritJavelinItemId = r.ReadUint32()
		}

		if isPartyBuff(skill.Id(m.skillId)) {
			m.affectedPartyMemberBitmap = r.ReadByte()
			if skill.Id(m.skillId) == skill.PriestDispelId {
				m.delay = r.ReadUint16()
			}
		}
		var mobs []uint32
		if isMobAffectingBuff(skill.Id(m.skillId)) {
			nMobCount := r.ReadByte()
			mobs = make([]uint32, 0)
			for range nMobCount {
				mobs = append(mobs, r.ReadUint32())
			}
			m.delay = r.ReadUint16()
		}
	}
}

func (m *SkillUsageInfo) SkillId() uint32 {
	return m.skillId
}

func (m *SkillUsageInfo) SkillLevel() byte {
	return m.skillLevel
}

func (m *SkillUsageInfo) AffectedPartyMemberBitmap() byte {
	return m.affectedPartyMemberBitmap
}

func isMobAffectingBuff(skillId skill.Id) bool {
	// TODO this is not all inclusive 32111004 32121007 33121007 35111013
	return skill.Is(skillId,
		skill.WarriorIronBodyId,
		skill.FighterRageId,
		skill.CrusaderArmorCrashId,
		skill.HeroMapleWarriorId,
		skill.PageThreatenId,
		skill.WhiteKnightMagicCrashId,
		skill.PaladinMapleWarriorId,
		skill.SpearmanHyperBodyId,
		skill.SpearmanIronWillId,
		skill.DragonKnightPowerCrashId,
		skill.DarkKnightMapleWarriorId,
		skill.FirePoisionWizardSlowId,
		skill.FirePoisionWizardMeditationId,
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.IceLightningWizardSlowId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericBlessId,
		skill.PriestDispelId,
		skill.PriestHolySymbolId,
		skill.BishopMapleWarriorId,
		skill.BishopHolyShieldId,
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.MarksmanMapleWarriorId,
		skill.MarksmanSharpEyesId,
		skill.AssassinHasteId,
		skill.HermitMesoUpId,
		skill.HermitShadowWebId,
		skill.NightLordMapleWarriorId,
		skill.BanditHasteId,
		skill.ShadowerMapleWarriorId,
		skill.BuccaneerMapleWarriorId,
		skill.BuccaneerSpeedInfusionId,
		skill.BuccaneerTimeLeapId,
		skill.CorsairMapleWarriorId,
		skill.CorsairSpeedInfusionId,
		skill.DawnWarriorStage1IronBodyId,
		skill.DawnWarriorStage2RageId,
		skill.BlazeWizardStage2SlowId,
		skill.BlazeWizardStage2MeditationId,
		skill.ThunderBreakerStage3SpeedInfusionId,
		skill.NightWalkerStage2HasteId,
		skill.NightWalkerStage3ShadowWebId,
		skill.AranStage4MapleWarriorId,
		skill.AranStage4ComboBarrierId,
		skill.EvanStage5MagicShieldId,
		skill.EvanStage6SlowId,
		skill.EvanStage7MagicResistanceId,
		skill.EvanStage8RecoveryAuraId,
		skill.EvanStage9MapleWarriorId,
		skill.EvanStage10BlessingOfTheOnyxId,
	)
}

func isPartyBuff(skillId skill.Id) bool {
	// TODO this is not all inclusive 32111004 32121007 33121007 35111013
	return skill.Is(skillId,
		skill.FighterRageId,
		skill.HeroMapleWarriorId,
		skill.PaladinMapleWarriorId,
		skill.SpearmanHyperBodyId,
		skill.SpearmanIronWillId,
		skill.DarkKnightMapleWarriorId,
		skill.FirePoisionWizardMeditationId,
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericHealId,
		skill.ClericBlessId,
		skill.PriestDispelId,
		skill.PriestHolySymbolId,
		skill.BishopMapleWarriorId,
		skill.BishopHolyShieldId,
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.MarksmanMapleWarriorId,
		skill.MarksmanSharpEyesId,
		skill.AssassinHasteId,
		skill.HermitMesoUpId,
		skill.NightLordMapleWarriorId,
		skill.BanditHasteId,
		skill.ShadowerMapleWarriorId,
		skill.BuccaneerMapleWarriorId,
		skill.BuccaneerTimeLeapId,
		skill.CorsairMapleWarriorId,
		skill.DawnWarriorStage2RageId,
		skill.BlazeWizardStage2MeditationId,
		skill.NightWalkerStage2HasteId,
		skill.AranStage4MapleWarriorId,
		skill.AranStage4ComboBarrierId,
		skill.EvanStage5MagicShieldId,
		skill.EvanStage7MagicResistanceId,
		//skill.EvanStage8RecoveryAuraId,
		skill.EvanStage9MapleWarriorId,
	)
}

func isAntiRepeatBuffSkill(skillId skill.Id) bool {
	// TODO this is not all inclusive 32111004 32121007 33121007 35111013
	return skill.Is(skillId,
		skill.WarriorIronBodyId,
		skill.FighterRageId,
		skill.CrusaderArmorCrashId,
		skill.HeroMapleWarriorId,
		skill.PageThreatenId,
		skill.WhiteKnightMagicCrashId,
		skill.PaladinMapleWarriorId,
		skill.SpearmanHyperBodyId,
		skill.SpearmanIronWillId,
		skill.DragonKnightPowerCrashId,
		skill.DarkKnightMapleWarriorId,
		skill.FirePoisionWizardSlowId,
		skill.FirePoisionWizardMeditationId,
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.IceLightningWizardSlowId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericBlessId,
		skill.PriestDispelId,
		skill.PriestHolySymbolId,
		skill.BishopMapleWarriorId,
		skill.BishopHolyShieldId,
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.MarksmanMapleWarriorId,
		skill.MarksmanSharpEyesId,
		skill.AssassinHasteId,
		skill.HermitMesoUpId,
		skill.HermitShadowWebId,
		skill.NightLordMapleWarriorId,
		skill.BanditHasteId,
		skill.ShadowerMapleWarriorId,
		skill.BuccaneerMapleWarriorId,
		skill.BuccaneerSpeedInfusionId,
		skill.BuccaneerTimeLeapId,
		skill.CorsairMapleWarriorId,
		skill.CorsairSpeedInfusionId,
		skill.DawnWarriorStage1IronBodyId,
		skill.DawnWarriorStage2RageId,
		skill.BlazeWizardStage2SlowId,
		skill.BlazeWizardStage2MeditationId,
		skill.ThunderBreakerStage3SpeedInfusionId,
		skill.NightWalkerStage2HasteId,
		skill.NightWalkerStage3ShadowWebId,
		skill.AranStage4MapleWarriorId,
		skill.AranStage4ComboBarrierId,
		skill.EvanStage5MagicShieldId,
		skill.EvanStage6SlowId,
		skill.EvanStage7MagicResistanceId,
		skill.EvanStage8RecoveryAuraId,
		skill.EvanStage9MapleWarriorId,
		skill.EvanStage10BlessingOfTheOnyxId,
	)
}
