package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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
	affectedMobIds            []uint32
	delay                     uint16
}

func (m *SkillUsageInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
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
		if isMobAffectingBuff(skill.Id(m.skillId)) {
			nMobCount := r.ReadByte()
			m.affectedMobIds = make([]uint32, 0, nMobCount)
			for range nMobCount {
				m.affectedMobIds = append(m.affectedMobIds, r.ReadUint32())
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

func (m *SkillUsageInfo) AffectedMobIds() []uint32 {
	return m.affectedMobIds
}

func (m *SkillUsageInfo) Delay() uint16 {
	return m.delay
}

// SkillUsageInfoBuilder fluently constructs SkillUsageInfo values for
// callers that don't go through Decode (today: tests). The wire decoder
// remains the canonical production path.
type SkillUsageInfoBuilder struct {
	info SkillUsageInfo
}

func NewSkillUsageInfoBuilder() *SkillUsageInfoBuilder {
	return &SkillUsageInfoBuilder{}
}

func (b *SkillUsageInfoBuilder) SetUpdateTime(v uint32) *SkillUsageInfoBuilder {
	b.info.updateTime = v
	return b
}

func (b *SkillUsageInfoBuilder) SetSkillId(v uint32) *SkillUsageInfoBuilder {
	b.info.skillId = v
	return b
}

func (b *SkillUsageInfoBuilder) SetSkillLevel(v byte) *SkillUsageInfoBuilder {
	b.info.skillLevel = v
	return b
}

func (b *SkillUsageInfoBuilder) SetCastX(v int16) *SkillUsageInfoBuilder {
	b.info.castX = v
	return b
}

func (b *SkillUsageInfoBuilder) SetCastY(v int16) *SkillUsageInfoBuilder {
	b.info.castY = v
	return b
}

func (b *SkillUsageInfoBuilder) SetSpiritJavelinItemId(v uint32) *SkillUsageInfoBuilder {
	b.info.spiritJavelinItemId = v
	return b
}

func (b *SkillUsageInfoBuilder) SetAffectedPartyMemberBitmap(v uint8) *SkillUsageInfoBuilder {
	b.info.affectedPartyMemberBitmap = v
	return b
}

func (b *SkillUsageInfoBuilder) SetAffectedMobIds(v []uint32) *SkillUsageInfoBuilder {
	b.info.affectedMobIds = v
	return b
}

func (b *SkillUsageInfoBuilder) SetDelay(v uint16) *SkillUsageInfoBuilder {
	b.info.delay = v
	return b
}

func (b *SkillUsageInfoBuilder) Build() SkillUsageInfo {
	return b.info
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
		skill.FirePoisonWizardSlowId,
		skill.FirePoisonWizardMeditationId,
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.IceLightningWizardSlowId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericBlessId,
		skill.PriestDoomId,
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
		skill.FirePoisonWizardMeditationId,
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
		skill.FirePoisonWizardSlowId,
		skill.FirePoisonWizardMeditationId,
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
