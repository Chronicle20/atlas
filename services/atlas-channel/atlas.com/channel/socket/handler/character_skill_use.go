package handler

import (
	"atlas-channel/character"
	skill2 "atlas-channel/character/skill"
	skill3 "atlas-channel/data/skill"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/skill/handler"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/skill"
	model2 "github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"sync"
)

const CharacterUseSkillHandle = "CharacterUseSkillHandle"

var skillHandlerMap map[skill.Id]handler.Handler
var skillHandlerOnce sync.Once

func CharacterUseSkillHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		sui := &model.SkillUsageInfo{}
		sui.Decode(l, t, readerOptions)(r)

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
		if err != nil {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
			return
		}
		if c.Hp() == 0 {
			l.Warnf("Character [%d] attempting to use skill when dead.", s.CharacterId())
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
			return
		}

		var sm skill2.Model
		for _, rs := range c.Skills() {
			if rs.Id() == skill.Id(sui.SkillId()) {
				sm = rs
			}
		}
		if sm.Id() == 0 || sm.Level() == 0 || sm.Level() != sui.SkillLevel() {
			l.Debugf("Character [%d] attempting to use skill [%d] at level [%d], but they do not have it.", s.CharacterId(), sui.SkillId(), sui.SkillLevel())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		var h handler.Handler
		var ok bool
		if h, ok = GetSkillHandler(skill.Id(sui.SkillId())); !ok {
			l.Infof("Character [%d] attempting to use unhandled skill [%d].", s.CharacterId(), sui.SkillId())
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
			return
		}

		se, err := skill3.NewProcessor(l, ctx).GetEffect(sui.SkillId(), sui.SkillLevel())
		if err != nil {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
			return
		}

		l.Debugf("Character [%d] using skill [%d] at level [%d].", s.CharacterId(), sui.SkillId(), sui.SkillLevel())
		err = h(l)(ctx)(s.Map(), s.CharacterId(), *sui, se)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] failed to use skill [%d].", s.CharacterId(), sui.SkillId())
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(s.WorldId(), s.ChannelId())(s.CharacterId(), announceSkillUse(l)(ctx)(wp)(sui.SkillId(), c.Level(), sui.SkillLevel()))

		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Map(), s.CharacterId(), announceForeignSkillUse(l)(ctx)(wp)(s.CharacterId(), sui.SkillId(), c.Level(), sui.SkillLevel()))

		err = enableActions(l)(ctx)(wp)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
		}
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model.StatUpdate, 0), true))
		}
	}
}

func announceSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
			return func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterSkillUseEffectBody(l)(skillId, characterLevel, skillLevel, false, false, false))
			}
		}
	}
}

func announceForeignSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterEffectForeign)(writer.CharacterSkillUseEffectForeignBody(l)(characterId, skillId, characterLevel, skillLevel, false, false, false))
			}
		}
	}
}

func GetSkillHandler(id skill.Id) (handler.Handler, bool) {
	skillHandlerOnce.Do(func() {
		skillHandlerMap = make(map[skill.Id]handler.Handler)
		skillHandlerMap[skill.BeginnerRecoveryId] = handler.UseBeginnerRecovery
		skillHandlerMap[skill.BeginnerNimbleFeetId] = handler.UseNimbleFeet
		skillHandlerMap[skill.WarriorIronBodyId] = handler.UseIronBody
		skillHandlerMap[skill.FighterSwordBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.FighterAxeBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.FighterRageId] = handler.UseRage
		skillHandlerMap[skill.FighterPowerGuardId] = handler.UsePowerGuard
		skillHandlerMap[skill.CrusaderComboAttackId] = handler.UseComboAttack
		skillHandlerMap[skill.HeroMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.HeroMonsterMagnetId] = handler.UseMonsterMagnet
		skillHandlerMap[skill.HeroPowerStanceId] = handler.UsePowerStance
		skillHandlerMap[skill.HeroEnrageId] = handler.UseEnrage
		skillHandlerMap[skill.HeroHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.PageSwordBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.PageBluntWeaponBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.PagePowerGuardId] = handler.UsePowerGuard
		skillHandlerMap[skill.WhiteKnightFireChargeSwordId] = handler.UseWeaponCharge
		skillHandlerMap[skill.WhiteKnightFlameChargeBluntWeaponId] = handler.UseWeaponCharge
		skillHandlerMap[skill.WhiteKnightIceChargeSwordId] = handler.UseWeaponCharge
		skillHandlerMap[skill.WhiteKnightBlizzardChargeBluntWeaponId] = handler.UseWeaponCharge
		skillHandlerMap[skill.WhiteKnightThunderChargeSwordId] = handler.UseWeaponCharge
		skillHandlerMap[skill.WhiteKnightLightningChargeBluntWeaponId] = handler.UseWeaponCharge
		skillHandlerMap[skill.WhiteKnightMagicCrashId] = handler.UseMagicCrash
		skillHandlerMap[skill.PaladinMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.PaladinMonsterMagnetId] = handler.UseMonsterMagnet
		skillHandlerMap[skill.PaladinPowerStanceId] = handler.UsePowerStance
		skillHandlerMap[skill.PaladinHolyChargeSwordId] = handler.UseWeaponCharge
		skillHandlerMap[skill.PaladinDivineChargeBluntWeaponId] = handler.UseWeaponCharge
		skillHandlerMap[skill.PaladinHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.SpearmanSpearBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.SpearmanPolearmBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.SpearmanIronWillId] = handler.UseIronWill
		skillHandlerMap[skill.SpearmanHyperBodyId] = handler.UseHyperBody
		skillHandlerMap[skill.DragonKnightPowerCrashId] = handler.UserPowerCrash
		skillHandlerMap[skill.DragonKnightDragonBloodId] = handler.UseDragonBlood
		skillHandlerMap[skill.DarkKnightMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.DarkKnightMonsterMagnetId] = handler.UseMonsterMagnet
		skillHandlerMap[skill.DarkKnightPowerStanceId] = handler.UsePowerStance
		skillHandlerMap[skill.DarkKnightBeholderId] = handler.UseSummon
		// TODO HexOfBeholder?
		skillHandlerMap[skill.DarkKnightHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.MagicianMagicGuardId] = handler.UseMagicGuard
		skillHandlerMap[skill.MagicianMagicArmorId] = handler.UseMagicArmor
		skillHandlerMap[skill.FirePoisionWizardMeditationId] = handler.UseMeditation
		skillHandlerMap[skill.FirePoisionWizardSlowId] = handler.UseSlow
		skillHandlerMap[skill.FirePoisonMagicianSpellBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.FirePoisonMagicianSealId] = handler.UseSeal
		skillHandlerMap[skill.FirePoisonArchMagicianMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.FirePoisonArchMagicianManaReflectionId] = handler.UseManaReflection
		skillHandlerMap[skill.FirePoisonArchMagicianInfinityId] = handler.UseInfinity
		skillHandlerMap[skill.FirePoisonArchMagicianElquinesId] = handler.UseSummon
		skillHandlerMap[skill.FirePoisonArchMagicianHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.IceLightningWizardMeditationId] = handler.UseMeditation
		skillHandlerMap[skill.IceLightningWizardSlowId] = handler.UseSlow
		skillHandlerMap[skill.IceLightningMagicianSpellBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.IceLightningMagicianSealId] = handler.UseSeal
		skillHandlerMap[skill.IceLightningArchMagicianMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.IceLightningArchMagicianManaReflectionId] = handler.UseManaReflection
		skillHandlerMap[skill.IceLightningArchMagicianInfinityId] = handler.UseInfinity
		skillHandlerMap[skill.IceLightningArchMagicianIfritId] = handler.UseSummon
		skillHandlerMap[skill.IceLightningArchMagicianHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.ClericInvincibleId] = handler.UseInvincible
		skillHandlerMap[skill.ClericBlessId] = handler.UseBless
		skillHandlerMap[skill.PriestDispelId] = handler.UseDispel
		skillHandlerMap[skill.PriestMysticDoorId] = handler.UseMysticDoor
		skillHandlerMap[skill.PriestHolySymbolId] = handler.UseHolySymbol
		skillHandlerMap[skill.PriestSummonDragonId] = handler.UseSummon
		skillHandlerMap[skill.BishopMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.BishopManaReflectionId] = handler.UseManaReflection
		skillHandlerMap[skill.BishopBahamutId] = handler.UseSummon
		skillHandlerMap[skill.BishopInfinityId] = handler.UseInfinity
		skillHandlerMap[skill.BishopHolyShieldId] = handler.UseHolyShield
		skillHandlerMap[skill.BishopResurrectionId] = handler.UseResurrection
		skillHandlerMap[skill.BishopHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.BowmanFocusId] = handler.UseFocus
		skillHandlerMap[skill.HunterBowBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.HunterSoulArrowBowId] = handler.UseSoulArrow
		skillHandlerMap[skill.RangerPuppetId] = handler.UseSummon
		skillHandlerMap[skill.RangerSilverHawkId] = handler.UseSummon
		skillHandlerMap[skill.BowmasterMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.BowmasterSharpEyesId] = handler.UseSharpEyes
		skillHandlerMap[skill.BowmasterPhoenixId] = handler.UseSummon
		skillHandlerMap[skill.BowmasterConcentrateId] = handler.UseConcentrate
		skillHandlerMap[skill.BowmasterHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.CrossbowmanCrossbowBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.CrossbowmanSoulArrowCrossbowId] = handler.UseSoulArrow
		skillHandlerMap[skill.SniperPuppetId] = handler.UseSummon
		skillHandlerMap[skill.SniperGoldenEagleId] = handler.UseSummon
		skillHandlerMap[skill.MarksmanMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.MarksmanSharpEyesId] = handler.UseSharpEyes
		skillHandlerMap[skill.MarksmanFrostpreyId] = handler.UseSummon
		skillHandlerMap[skill.MarksmanHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.RogueDarkSightId] = handler.UseDarkSight
		skillHandlerMap[skill.AssassinClawBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.AssassinHasteId] = handler.UseSkillHaste
		skillHandlerMap[skill.HermitMesoUpId] = handler.UseMesoUp
		skillHandlerMap[skill.HermitShadowPartnerId] = handler.UseShadowPartner
		skillHandlerMap[skill.HermitShadowMesoId] = handler.UseShadowMeso
		skillHandlerMap[skill.NightLordMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.NightLordTauntId] = handler.UseTaunt
		skillHandlerMap[skill.NightLordNinjaAmbushId] = handler.UseSummon
		skillHandlerMap[skill.NightLordShadowStarsId] = handler.UseShadowStars
		skillHandlerMap[skill.NightLordHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.BanditDaggerBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.BanditHasteId] = handler.UseSkillHaste
		skillHandlerMap[skill.ChiefBanditChakraId] = handler.UseChakra
		skillHandlerMap[skill.ChiefBanditPickpocketId] = handler.UsePickPocket
		skillHandlerMap[skill.ChiefBanditMesoGuardId] = handler.UseMesoGuard
		skillHandlerMap[skill.ShadowerMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.ShadowerTauntId] = handler.UseTaunt
		skillHandlerMap[skill.ShadowerNinjaAmbushId] = handler.UseSummon
		skillHandlerMap[skill.ShadowerHerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.PirateDashId] = handler.UseDash
		skillHandlerMap[skill.BrawlerMPRecoveryId] = handler.UseMPRecovery
		skillHandlerMap[skill.BrawlerKnucklerBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.BrawlerOakBarrelId] = handler.UseOakBarrel
		skillHandlerMap[skill.MarauderEnergyChargeId] = handler.UseEnergyCharge
		skillHandlerMap[skill.MarauderTransformationId] = handler.UseTransformation
		skillHandlerMap[skill.BuccaneerMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.BuccaneerSuperTransformationId] = handler.UseSuperTransformation
		skillHandlerMap[skill.BuccaneerSnatchId] = handler.UseSnatch
		skillHandlerMap[skill.BuccaneerSpeedInfusionId] = handler.UseSpeedInfusion
		skillHandlerMap[skill.BuccaneerTimeLeapId] = handler.UseTimeLeap
		skillHandlerMap[skill.GunslingerGunBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.GunslingerWingsId] = handler.UseWings
		skillHandlerMap[skill.OutlawOctopusId] = handler.UseSummon
		skillHandlerMap[skill.OutlawGaviotaId] = handler.UseSummon
		skillHandlerMap[skill.OutlawHomingBeaconId] = handler.UseHomingBeacon
		skillHandlerMap[skill.CorsairMapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.CorsairWrathOfTheOctopiId] = handler.UseSummon
		skillHandlerMap[skill.CorsairBattleshipId] = handler.UseBattleship
		skillHandlerMap[skill.CorsairSpeedInfusionId] = handler.UseSpeedInfusion
		skillHandlerMap[skill.NoblesseRecoveryId] = handler.UseBeginnerRecovery
		skillHandlerMap[skill.NoblesseNimbleFeetId] = handler.UseNimbleFeet
		skillHandlerMap[skill.ThunderBreakerStage2KnuckleBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.ThunderBreakerStage3SpeedInfusionId] = handler.UseSpeedInfusion
		skillHandlerMap[skill.DawnWarriorStage2SwordBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.DawnWarriorStage2RageId] = handler.UseRage
		skillHandlerMap[skill.BlazeWizardStage1MagicGuardId] = handler.UseMagicGuard
		skillHandlerMap[skill.BlazeWizardStage1MagicArmorId] = handler.UseMagicArmor
		skillHandlerMap[skill.BlazeWizardStage2MeditationId] = handler.UseMeditation
		skillHandlerMap[skill.BlazeWizardStage2SpellBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.BlazeWizardStage2SlowId] = handler.UseSlow
		skillHandlerMap[skill.BlazeWizardStage3SealId] = handler.UseSeal
		skillHandlerMap[skill.WindArcherStage1FocusId] = handler.UseFocus
		skillHandlerMap[skill.WindArcherStage2BowBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.WindArcherStage2SoulArrowId] = handler.UseSoulArrow
		skillHandlerMap[skill.WindArcherStage3PuppetId] = handler.UseSummon
		skillHandlerMap[skill.NightWalkerStage1DarkSightId] = handler.UseDarkSight
		skillHandlerMap[skill.NightWalkerStage2HasteId] = handler.UseSkillHaste
		skillHandlerMap[skill.NightWalkerStage3ShadowPartnerId] = handler.UseShadowPartner
		skillHandlerMap[skill.LegendRecoveryId] = handler.UseBeginnerRecovery
		skillHandlerMap[skill.LegendNimbleFeetId] = handler.UseNimbleFeet
		skillHandlerMap[skill.AranStage1PolearmBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.AranStage4MapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.AranStage4HerosWillId] = handler.UseHerosWill
		skillHandlerMap[skill.EvanRecoveryId] = handler.UseBeginnerRecovery
		skillHandlerMap[skill.EvanNimbleFeetId] = handler.UseNimbleFeet
		skillHandlerMap[skill.EvanStage3MagicGuardId] = handler.UseMagicGuard
		skillHandlerMap[skill.EvanStage6MagicBoosterId] = handler.UseWeaponBooster
		skillHandlerMap[skill.EvanStage6SlowId] = handler.UseSlow
		skillHandlerMap[skill.EvanStage9MapleWarriorId] = handler.UseMapleWarrior
		skillHandlerMap[skill.EvanStage9HerosWillId] = handler.UseHerosWill
	})
	h, ok := skillHandlerMap[id]
	return h, ok
}
