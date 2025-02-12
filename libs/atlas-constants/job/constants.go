package job

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"math"
)

type Id uint16

type Job struct {
	id        Id
	skills    []skill.Id
	fourthJob bool
	buffs     []skill.Id
}

func (j Job) Id() Id {
	return j.id
}

func (j Job) Skills() []skill.Id {
	return j.skills
}

func (j Job) IsFourthJob() bool {
	return j.fourthJob
}

var Beginner = Job{
	id: BeginnerId,
	skills: []skill.Id{
		skill.BeginnerThreeSnailsId,
		skill.BeginnerRecoveryId,
		skill.BeginnerNimbleFeetId,
		skill.BeginnerSoulOfCraftsman,
		skill.BeginnerMonsterRiding,
		skill.BeginnerEchoOfHeroId,
		skill.BeginnerJumpDownId,
		skill.BeginnerMakerId,
		skill.BeginnerMultiPetId,
		skill.BeginnerBambooId,
		skill.BeginnerInvincibleId,
		skill.BeginnerBerserkId,
		skill.BeginnerBlessOfNymph,
	},
	buffs: []skill.Id{
		skill.BeginnerRecoveryId,
		skill.BeginnerNimbleFeetId,
		skill.BeginnerMonsterRiding,
		skill.BeginnerEchoOfHeroId,
	},
}

var Warrior = Job{
	id: WarriorId,
	skills: []skill.Id{
		skill.WarriorImprovedHpRecoveryId,
		skill.WarriorImprovedMaxHpIncreaseId,
		skill.WarriorEndureId,
		skill.WarriorIronBodyId,
		skill.WarriorPowerStrikeId,
		skill.WarriorSlashBlastId,
	},
	buffs: []skill.Id{
		skill.WarriorIronBodyId,
	},
}

var Fighter = Job{
	id: FighterId,
	skills: []skill.Id{
		skill.FighterSwordMasteryId,
		skill.FighterAxeMasteryId,
		skill.FighterFinalAttackSwordId,
		skill.FighterFinalAttackAxeId,
		skill.FighterSwordBoosterId,
		skill.FighterAxeBoosterId,
		skill.FighterRageId,
		skill.FighterPowerGuardId,
	},
	buffs: []skill.Id{
		skill.FighterSwordBoosterId,
		skill.FighterAxeBoosterId,
		skill.FighterRageId,
		skill.FighterPowerGuardId,
	},
}

var Crusader = Job{
	id: CrusaderId,
	skills: []skill.Id{
		skill.CrusaderImprovingMpRecoveryId,
		skill.CrusaderShieldMasteryId,
		skill.CrusaderComboAttackId,
		skill.CrusaderPanicSwordId,
		skill.CrusaderPanicAxeId,
		skill.CrusaderComaSwordId,
		skill.CrusaderComaAxeId,
		skill.CrusaderArmorCrashId,
		skill.CrusaderShoutId,
	},
	buffs: []skill.Id{
		skill.CrusaderComboAttackId,
		skill.CrusaderArmorCrashId,
	},
}

var Hero = Job{
	id: HeroId,
	skills: []skill.Id{
		skill.HeroMapleWarriorId,
		skill.HeroMonsterMagnetId,
		skill.HeroPowerStanceId,
		skill.HeroAdvancedComboAttackId,
		skill.HeroAchillesId,
		skill.HeroGuardianId,
		skill.HeroRushId,
		skill.HeroBrandishId,
		skill.HeroEnrageId,
		skill.HeroHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.HeroMapleWarriorId,
		skill.HeroPowerStanceId,
		skill.HeroEnrageId,
		skill.HeroHerosWillId,
	},
}

var Page = Job{
	id: PageId,
	skills: []skill.Id{
		skill.PageSwordMasteryId,
		skill.PageBluntWeaponMasteryId,
		skill.PageSwordFinalAttackId,
		skill.PageBluntFinalAttackId,
		skill.PageSwordBoosterId,
		skill.PageBluntWeaponBoosterId,
		skill.PageThreatenId,
		skill.PagePowerGuardId,
	},
	buffs: []skill.Id{
		skill.PageSwordBoosterId,
		skill.PageBluntWeaponBoosterId,
		skill.PageThreatenId,
		skill.PagePowerGuardId,
	},
}

var WhiteKnight = Job{
	id: WhiteKnightId,
	skills: []skill.Id{
		skill.WhiteKnightImprovingMpRecoveryId,
		skill.WhiteKnightShieldMasteryId,
		skill.WhiteKnightChargedBlowId,
		skill.WhiteKnightFireChargeSwordId,
		skill.WhiteKnightFlameChargeBluntWeaponId,
		skill.WhiteKnightIceChargeSwordId,
		skill.WhiteKnightBlizzardChargeBluntWeaponId,
		skill.WhiteKnightThunderChargeSwordId,
		skill.WhiteKnightLightningChargeBluntWeaponId,
		skill.WhiteKnightMagicCrashId,
	},
	buffs: []skill.Id{
		skill.WhiteKnightFireChargeSwordId,
		skill.WhiteKnightFlameChargeBluntWeaponId,
		skill.WhiteKnightIceChargeSwordId,
		skill.WhiteKnightBlizzardChargeBluntWeaponId,
		skill.WhiteKnightThunderChargeSwordId,
		skill.WhiteKnightLightningChargeBluntWeaponId,
		skill.WhiteKnightMagicCrashId,
	},
}

var Paladin = Job{
	id: PaladinId,
	skills: []skill.Id{
		skill.PaladinMapleWarriorId,
		skill.PaladinMonsterMagnetId,
		skill.PaladinPowerStanceId,
		skill.PaladinHolyChargeSwordId,
		skill.PaladinDivineChargeBluntWeaponId,
		skill.PaladinAchillesId,
		skill.PaladinGuardianId,
		skill.PaladinRushId,
		skill.PaladinBlastId,
		skill.PaladinAdvancedChargeId,
		skill.PaladinHeavensHammerId,
		skill.PaladinHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.PaladinMapleWarriorId,
		skill.PaladinPowerStanceId,
		skill.PaladinHolyChargeSwordId,
		skill.PaladinDivineChargeBluntWeaponId,
		skill.PaladinHerosWillId,
	},
}

var Spearman = Job{
	id: SpearmanId,
	skills: []skill.Id{
		skill.SpearmanSpearMasteryId,
		skill.SpearmanPolearmMasteryId,
		skill.SpearmanFinalAttackSpearId,
		skill.SpearmanFinalAttackPolearmId,
		skill.SpearmanSpearBoosterId,
		skill.SpearmanPolearmBoosterId,
		skill.SpearmanIronWillId,
		skill.SpearmanHyperBodyId,
	},
	buffs: []skill.Id{
		skill.SpearmanSpearBoosterId,
		skill.SpearmanPolearmBoosterId,
		skill.SpearmanIronWillId,
		skill.SpearmanHyperBodyId,
	},
}

var DragonKnight = Job{
	id: DragonKnightId,
	skills: []skill.Id{
		skill.DragonKnightElementalResistanceId,
		skill.DragonKnightSpearCrusherId,
		skill.DragonKnightPolearmCrusherId,
		skill.DragonKnightDragonFurySpearId,
		skill.DragonKnightDragonFuryPolearmId,
		skill.DragonKnightSacrificeId,
		skill.DragonKnightDragonRoarId,
		skill.DragonKnightPowerCrashId,
		skill.DragonKnightDragonBloodId,
	},
	buffs: []skill.Id{
		skill.DragonKnightPowerCrashId,
		skill.DragonKnightDragonBloodId,
	},
}

var DarkKnight = Job{
	id: DarkKnightId,
	skills: []skill.Id{
		skill.DarkKnightMapleWarriorId,
		skill.DarkKnightMonsterMagnetId,
		skill.DarkKnightPowerStanceId,
		skill.DarkKnightRushId,
		skill.DarkKnightAchillesId,
		skill.DarkKnightBerserkId,
		skill.DarkKnightBeholderId,
		skill.DarkKnightAuraOfTheBeholderId,
		skill.DarkKnightHexOfTheBeholderId,
		skill.DarkKnightHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.DarkKnightMapleWarriorId,
		skill.DarkKnightBeholderId,
		skill.DarkKnightAuraOfTheBeholderId,
		skill.DarkKnightHexOfTheBeholderId,
		skill.DarkKnightHerosWillId,
	},
}

var Magician = Job{
	id: MagicianId,
	skills: []skill.Id{
		skill.MagicianImprovedMpRecoveryId,
		skill.MagicianImprovedMaxMpIncreaseId,
		skill.MagicianMagicGuardId,
		skill.MagicianMagicArmorId,
		skill.MagicianEnergyBoltId,
		skill.MagicianMagicClawId,
	},
	buffs: []skill.Id{
		skill.MagicianMagicGuardId,
		skill.MagicianMagicArmorId,
	},
}

var FirePoisonWizard = Job{
	id: FirePoisonWizardId,
	skills: []skill.Id{
		skill.FirePoisionWizardMpEaterId,
		skill.FirePoisionWizardMeditationId,
		skill.FirePoisionWizardTeleportId,
		skill.FirePoisionWizardSlowId,
		skill.FirePoisionWizardFireArrowId,
		skill.FirePoisionWizardPoisonBreathId,
	},
	buffs: []skill.Id{
		skill.FirePoisionWizardMeditationId,
		skill.FirePoisionWizardSlowId,
	},
}

var FirePoisonMagician = Job{
	id: FirePoisonMagicianId,
	skills: []skill.Id{
		skill.FirePoisonMagicianPartialResistanceId,
		skill.FirePoisonMagicianElementAmplificationId,
		skill.FirePoisonMagicianExplosionId,
		skill.FirePoisonMagicianPoisonMistId,
		skill.FirePoisonMagicianSealId,
		skill.FirePoisonMagicianSpellBoosterId,
		skill.FirePoisonMagicianElementCompositionId,
	},
	buffs: []skill.Id{
		skill.FirePoisonMagicianSealId,
		skill.FirePoisonMagicianSpellBoosterId,
	},
}

var FirePoisonArchMagician = Job{
	id: FirePoisonArchMagicianId,
	skills: []skill.Id{
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.FirePoisonArchMagicianBigBangId,
		skill.FirePoisonArchMagicianManaReflectionId,
		skill.FirePoisonArchMagicianFireDemonId,
		skill.FirePoisonArchMagicianInfinityId,
		skill.FirePoisonArchMagicianElquinesId,
		skill.FirePoisonArchMagicianParalyzeId,
		skill.FirePoisonArchMagicianMeteorShowerId,
		skill.FirePoisonArchMagicianHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.FirePoisonArchMagicianManaReflectionId,
		skill.FirePoisonArchMagicianInfinityId,
		skill.FirePoisonArchMagicianHerosWillId,
	},
}

var IceLightningWizard = Job{
	id: IceLightningWizardId,
	skills: []skill.Id{
		skill.IceLightningWizardMpEaterId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningWizardTeleportId,
		skill.IceLightningWizardSlowId,
		skill.IceLightningWizardColdBeamId,
		skill.IceLightningWizardThunderBoltId,
	},
	buffs: []skill.Id{
		skill.IceLightningWizardMeditationId,
		skill.IceLightningWizardSlowId,
	},
}

var IceLightningMagician = Job{
	id: IceLightningMagicianId,
	skills: []skill.Id{
		skill.IceLightningMagicianPartialResistanceId,
		skill.IceLightningMagicianElementAmplificationId,
		skill.IceLightningMagicianIceStrikeId,
		skill.IceLightningMagicianThunderSpearId,
		skill.IceLightningMagicianSealId,
		skill.IceLightningMagicianSpellBoosterId,
		skill.IceLightningMagicianElementCompositionId,
	},
	buffs: []skill.Id{
		skill.IceLightningMagicianSealId,
		skill.IceLightningMagicianSpellBoosterId,
	},
}

var IceLightningArchMagician = Job{
	id: IceLightningArchMagicianId,
	skills: []skill.Id{
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.IceLightningArchMagicianBigBangId,
		skill.IceLightningArchMagicianManaReflectionId,
		skill.IceLightningArchMagicianIceDemonId,
		skill.IceLightningArchMagicianInfinityId,
		skill.IceLightningArchMagicianIfritId,
		skill.IceLightningArchMagicianChainLightningId,
		skill.IceLightningArchMagicianBlizzardId,
		skill.IceLightningArchMagicianHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.IceLightningArchMagicianManaReflectionId,
		skill.IceLightningArchMagicianInfinityId,
		skill.IceLightningArchMagicianHerosWillId,
	},
}

var Cleric = Job{
	id: ClericId,
	skills: []skill.Id{
		skill.ClericMpEaterId,
		skill.ClericTeleportId,
		skill.ClericHealId,
		skill.ClericInvincibleId,
		skill.ClericBlessId,
		skill.ClericHolyArrowId,
	},
	buffs: []skill.Id{
		skill.ClericInvincibleId,
		skill.ClericBlessId,
	},
}

var Priest = Job{
	id: PriestId,
	skills: []skill.Id{
		skill.PriestElementalResistanceId,
		skill.PriestDispelId,
		skill.PriestMysticDoorId,
		skill.PriestHolySymbolId,
		skill.PriestShiningRayId,
		skill.PriestDoomId,
		skill.PriestSummonDragonId,
	},
	buffs: []skill.Id{
		skill.PriestDispelId,
		skill.PriestMysticDoorId,
		skill.PriestHolySymbolId,
		skill.PriestDoomId,
	},
}

var Bishop = Job{
	id: BishopId,
	skills: []skill.Id{
		skill.BishopMapleWarriorId,
		skill.BishopBigBangId,
		skill.BishopManaReflectionId,
		skill.BishopBahamutId,
		skill.BishopInfinityId,
		skill.BishopHolyShieldId,
		skill.BishopResurrectionId,
		skill.BishopAngelRayId,
		skill.BishopGenesisId,
		skill.BishopHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.BishopMapleWarriorId,
		skill.BishopManaReflectionId,
		skill.BishopInfinityId,
		skill.BishopHolyShieldId,
		skill.BishopHerosWillId,
	},
}

var Bowman = Job{
	id: BowmanId,
	skills: []skill.Id{
		skill.BowmanTheBlessingOfAmazonId,
		skill.BowmanCriticalShotId,
		skill.BowmanTheEyeOfAmazonId,
		skill.BowmanFocusId,
		skill.BowmanArrowBlowId,
		skill.BowmanDoubleShotId,
	},
	buffs: []skill.Id{
		skill.BowmanFocusId,
	},
}

var Hunter = Job{
	id: HunterId,
	skills: []skill.Id{
		skill.HunterBowMasteryId,
		skill.HunterFinalAttackBowId,
		skill.HunterBowBoosterId,
		skill.HunterPowerKnockBackId,
		skill.HunterSoulArrowBowId,
		skill.HunterArrowBombBowId,
	},
	buffs: []skill.Id{
		skill.HunterBowBoosterId,
		skill.HunterSoulArrowBowId,
	},
}

var Ranger = Job{
	id: RangerId,
	skills: []skill.Id{
		skill.RangerThrustId,
		skill.RangerMortalBlowId,
		skill.RangerPuppetId,
		skill.RangerInfernoId,
		skill.RangerArrowRainId,
		skill.RangerSilverHawkId,
		skill.RangerStrafeId,
	},
	buffs: []skill.Id{
		skill.RangerPuppetId,
	},
}

var Bowmaster = Job{
	id: BowmasterId,
	skills: []skill.Id{
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.BowmasterDragonsBreathId,
		skill.BowmasterHurricaneId,
		skill.BowmasterBowExpertId,
		skill.BowmasterPhoenixId,
		skill.BowmasterHamstringId,
		skill.BowmasterConcentrateId,
		skill.BowmasterHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.BowmasterConcentrateId,
		skill.BowmasterHerosWillId,
	},
}

var Crossbowman = Job{
	id: CrossbowmanId,
	skills: []skill.Id{
		skill.CrossbowmanCrossbowMasteryId,
		skill.CrossbowmanFinalAttackCrossbowId,
		skill.CrossbowmanCrossbowBoosterId,
		skill.CrossbowmanPowerKnockBackId,
		skill.CrossbowmanSoulArrowCrossbowId,
		skill.CrossbowmanIronArrowCrossbowId,
	},
	buffs: []skill.Id{
		skill.CrossbowmanCrossbowBoosterId,
		skill.CrossbowmanSoulArrowCrossbowId,
	},
}

var Sniper = Job{
	id: SniperId,
	skills: []skill.Id{
		skill.SniperThrustId,
		skill.SniperMortalBlowId,
		skill.SniperPuppetId,
		skill.SniperBlizzardId,
		skill.SniperArrowEruptionId,
		skill.SniperGoldenEagleId,
		skill.SniperStrafeId,
	},
	buffs: []skill.Id{
		skill.SniperPuppetId,
	},
}

var Marksman = Job{
	id: MarksmanId,
	skills: []skill.Id{
		skill.MarksmanMapleWarriorId,
		skill.MarksmanPiercingArrowId,
		skill.MarksmanSharpEyesId,
		skill.MarksmanDragonsBreathId,
		skill.MarksmanMarksmanBoostId,
		skill.MarksmanFrostpreyId,
		skill.MarksmanBlindId,
		skill.MarksmanSnipeId,
		skill.MarksmanHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.MarksmanMapleWarriorId,
		skill.MarksmanSharpEyesId,
		skill.MarksmanBlindId,
		skill.MarksmanHerosWillId,
	},
}

var Rogue = Job{
	id: RogueId,
	skills: []skill.Id{
		skill.RogueNimbleBodyId,
		skill.RogueKeenEyesId,
		skill.RogueDisorderId,
		skill.RogueDarkSightId,
		skill.RogueDoubleStabId,
		skill.RogueLuckySevenId,
	},
	buffs: []skill.Id{
		skill.RogueDarkSightId,
	},
}

var Assassin = Job{
	id: AssassinId,
	skills: []skill.Id{
		skill.AssassinClawMasteryId,
		skill.AssassinCriticalThrowId,
		skill.AssassinEndureId,
		skill.AssassinClawBoosterId,
		skill.AssassinHasteId,
		skill.AssassinDrainId,
	},
	buffs: []skill.Id{
		skill.AssassinClawBoosterId,
		skill.AssassinHasteId,
	},
}

var Hermit = Job{
	id: HermitId,
	skills: []skill.Id{
		skill.HermitAlchemistId,
		skill.HermitMesoUpId,
		skill.HermitShadowPartnerId,
		skill.HermitShadowWebId,
		skill.HermitShadowMesoId,
		skill.HermitAvengerId,
		skill.HermitFlashJumpId,
	},
	buffs: []skill.Id{
		skill.HermitMesoUpId,
		skill.HermitShadowPartnerId,
	},
}

var NightLord = Job{
	id: NightLordId,
	skills: []skill.Id{
		skill.NightLordMapleWarriorId,
		skill.NightLordShadowShifterId,
		skill.NightLordTauntId,
		skill.NightLordNinjaAmbushId,
		skill.NightLordVenomousStarId,
		skill.NightLordShadowStarsId,
		skill.NightLordTripleThrowId,
		skill.NightLordNinjaStormId,
		skill.NightLordHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.NightLordMapleWarriorId,
		skill.NightLordNinjaAmbushId,
		skill.NightLordShadowStarsId,
		skill.NightLordHerosWillId,
	},
}

var Bandit = Job{
	id: BanditId,
	skills: []skill.Id{
		skill.BanditDaggerMasteryId,
		skill.BanditEndureId,
		skill.BanditDaggerBoosterId,
		skill.BanditHasteId,
		skill.BanditStealId,
		skill.BanditSavageBlowId,
	},
	buffs: []skill.Id{
		skill.BanditDaggerBoosterId,
		skill.BanditHasteId,
	},
}

var ChiefBandit = Job{
	id: ChiefBanditId,
	skills: []skill.Id{
		skill.ChiefBanditShieldMasteryId,
		skill.ChiefBanditChakraId,
		skill.ChiefBanditAssaulterId,
		skill.ChiefBanditPickpocketId,
		skill.ChiefBanditBandOfThievesId,
		skill.ChiefBanditMesoGuardId,
		skill.ChiefBanditMesoExplosionId,
	},
	buffs: []skill.Id{
		skill.ChiefBanditPickpocketId,
		skill.ChiefBanditMesoGuardId,
	},
}

var Shadower = Job{
	id: ShadowerId,
	skills: []skill.Id{
		skill.ShadowerMapleWarriorId,
		skill.ShadowerAssassinateId,
		skill.ShadowerShadowShifterId,
		skill.ShadowerTauntId,
		skill.ShadowerNinjaAmbushId,
		skill.ShadowerVenomousStabId,
		skill.ShadowerSmokescreenId,
		skill.ShadowerBoomerangStepId,
		skill.ShadowerHerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.ShadowerMapleWarriorId,
		skill.ShadowerNinjaAmbushId,
		skill.ShadowerHerosWillId,
	},
}

var Pirate = Job{
	id: PirateId,
	skills: []skill.Id{
		skill.PirateBulletTimeId,
		skill.PirateFlashFistId,
		skill.PirateSommersaultKickId,
		skill.PirateDoubleShotId,
		skill.PirateDashId,
	},
	buffs: []skill.Id{
		skill.PirateDashId,
	},
}

var Brawler = Job{
	id: BrawlerId,
	skills: []skill.Id{
		skill.BrawlerImproveMaxHpId,
		skill.BrawlerKnucklerMasteryId,
		skill.BrawlerBackspinBlowId,
		skill.BrawlerDoubleUppercutId,
		skill.BrawlerCorkscrewBlowId,
		skill.BrawlerMPRecoveryId,
		skill.BrawlerKnucklerBoosterId,
		skill.BrawlerOakBarrelId,
	},
}

var Marauder = Job{
	id: MarauderId,
	skills: []skill.Id{
		skill.MarauderStunMasteryId,
		skill.MarauderEnergyChargeId,
		skill.MarauderEnergyBlastId,
		skill.MarauderEnergyDrainId,
		skill.MarauderTransformationId,
		skill.MarauderShockwaveId,
	},
	buffs: []skill.Id{
		skill.MarauderTransformationId,
	},
}

var Buccaneer = Job{
	id: BuccaneerId,
	skills: []skill.Id{
		skill.BuccaneerMapleWarriorId,
		skill.BuccaneerDragonStrikeId,
		skill.BuccaneerEnergyOrbId,
		skill.BuccaneerSuperTransformationId,
		skill.BuccaneerDemolitionId,
		skill.BuccaneerSnatchId,
		skill.BuccaneerBarrageId,
		skill.BuccaneerPiratesRageId,
		skill.BuccaneerSpeedInfusionId,
		skill.BuccaneerTimeLeapId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.BuccaneerSuperTransformationId,
	},
}

var Gunslinger = Job{
	id: GunslingerId,
	skills: []skill.Id{
		skill.GunslingerGunMasteryId,
		skill.GunslingerInvisibleShotId,
		skill.GunslingerGrenadeId,
		skill.GunslingerGunBoosterId,
		skill.GunslingerBlankShotId,
		skill.GunslingerWingsId,
		skill.GunslingerRecoilShotId,
	},
}

var Outlaw = Job{
	id: OutlawId,
	skills: []skill.Id{
		skill.OutlawBurstFireId,
		skill.OutlawOctopusId,
		skill.OutlawGaviotaId,
		skill.OutlawFlamethrowerId,
		skill.OutlawIceSplitterId,
		skill.OutlawHomingBeaconId,
	},
}

var Corsair = Job{
	id: CorsairId,
	skills: []skill.Id{
		skill.CorsairMapleWarriorId,
		skill.CorsairElementalBoostId,
		skill.CorsairWrathOfTheOctopiId,
		skill.CorsairAerialStrikeId,
		skill.CorsairRapidFireId,
		skill.CorsairBattleshipId,
		skill.CorsairBattleshipCannonId,
		skill.CorsairBattleshipTorpedoId,
		skill.CorsairHypnotizeId,
		skill.CorsairSpeedInfusionId,
		skill.CorsairBullseyeId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.CorsairBattleshipId,
	},
}

var MapleLeafBrigadier = Job{
	id: MapleLeafBrigadierId,
	skills: []skill.Id{
		skill.MapleLeafBrigadierAntiMacroId,
		skill.MapleLeafBrigadierTeleportId,
	},
}

var Gm = Job{
	id: GmId,
	skills: []skill.Id{
		skill.GmHasteId,
		skill.GmSuperDragonRoarId,
		skill.GmTeleportId,
		skill.GmHide,
	},
	buffs: []skill.Id{
		skill.GmHide,
	},
}

var SuperGm = Job{
	id: SuperGmId,
	skills: []skill.Id{
		skill.SuperGmHealDispelId,
		skill.SuperGmHasteId,
		skill.SuperGmHolySymbolId,
		skill.SuperGmBlessId,
		skill.SuperGmHideId,
		skill.SuperGmResurrectionId,
		skill.SuperGmDragonRoarId,
		skill.SuperGmTeleportId,
		skill.SuperGmHyperBodyId,
	},
	buffs: []skill.Id{
		skill.SuperGmHasteId,
		skill.SuperGmHolySymbolId,
		skill.SuperGmBlessId,
		skill.SuperGmHideId,
		skill.SuperGmHyperBodyId,
	},
}

var Noblesse = Job{
	id: NoblesseId,
	skills: []skill.Id{
		skill.NoblesseThreeSnailsId,
		skill.NoblesseRecoveryId,
		skill.NoblesseNimbleFeetId,
		skill.NoblesseSoulOfCraftsman,
		skill.NoblesseMonsterRiding,
		skill.NoblesseEchoOfHeroId,
		skill.NoblesseJumpDownId,
		skill.NoblesseMakerId,
		skill.NoblesseMultiPetId,
		skill.NoblesseBambooId,
		skill.NoblesseInvincibleId,
		skill.NoblesseBerserkId,
		skill.NoblesseBlessOfNymphId,
	},
	buffs: []skill.Id{
		skill.NoblesseRecoveryId,
		skill.NoblesseNimbleFeetId,
		skill.NoblesseMonsterRiding,
		skill.NoblesseEchoOfHeroId,
		skill.NoblesseBlessOfNymphId,
	},
}

var DawnWarriorStage1 = Job{
	id: DawnWarriorStage1Id,
	skills: []skill.Id{
		skill.DawnWarriorStage1IronBodyId,
		skill.DawnWarriorStage1PowerStrikeId,
		skill.DawnWarriorStage1SlashBlastId,
		skill.DawnWarriorStage1SoulId,
		skill.DawnWarriorStage1ImprovedMaxHpIncreaseId,
	},
	buffs: []skill.Id{
		skill.DawnWarriorStage1IronBodyId,
		skill.DawnWarriorStage1SoulId,
	},
}

var DawnWarriorStage2 = Job{
	id: DawnWarriorStage2Id,
	skills: []skill.Id{
		skill.DawnWarriorStage2SwordMasteryId,
		skill.DawnWarriorStage2SwordBoosterId,
		skill.DawnWarriorStage2FinalAttackSwordId,
		skill.DawnWarriorStage2RageId,
		skill.DawnWarriorStage2SoulBladeId,
		skill.DawnWarriorStage2SoulRushId,
	},
	buffs: []skill.Id{
		skill.DawnWarriorStage2SwordBoosterId,
		skill.DawnWarriorStage2FinalAttackSwordId,
		skill.DawnWarriorStage2RageId,
	},
}

var DawnWarriorStage3 = Job{
	id: DawnWarriorStage3Id,
	skills: []skill.Id{
		skill.DawnWarriorStage3ImprovedMpRecoveryId,
		skill.DawnWarriorStage3ComboAttackId,
		skill.DawnWarriorStage3PanicId,
		skill.DawnWarriorStage3ComaId,
		skill.DawnWarriorStage3BrandishId,
		skill.DawnWarriorStage3AdvancedComboId,
		skill.DawnWarriorStage3SoulDriverId,
		skill.DawnWarriorStage3SoulChargeId,
	},
	buffs: []skill.Id{
		skill.DawnWarriorStage3ComboAttackId,
		skill.DawnWarriorStage3SoulChargeId,
	},
}

var DawnWarriorStage4 = Job{
	id:        DawnWarriorStage4Id,
	skills:    []skill.Id{},
	fourthJob: true,
}

var BlazeWizardStage1 = Job{
	id: BlazeWizardStage1Id,
	skills: []skill.Id{
		skill.BlazeWizardStage1MagicGuardId,
		skill.BlazeWizardStage1MagicArmorId,
		skill.BlazeWizardStage1MagicClawId,
		skill.BlazeWizardStage1FlameId,
		skill.BlazeWizardStage1ImprovedMaxMpIncreaseId,
	},
	buffs: []skill.Id{
		skill.BlazeWizardStage1MagicGuardId,
		skill.BlazeWizardStage1MagicArmorId,
		skill.BlazeWizardStage1FlameId,
	},
}

var BlazeWizardStage2 = Job{
	id: BlazeWizardStage2Id,
	skills: []skill.Id{
		skill.BlazeWizardStage2MeditationId,
		skill.BlazeWizardStage2SlowId,
		skill.BlazeWizardStage2FireArrowId,
		skill.BlazeWizardStage2TeleportId,
		skill.BlazeWizardStage2SpellBoosterId,
		skill.BlazeWizardStage2ElementalResetId,
		skill.BlazeWizardStage2FirePillarId,
	},
	buffs: []skill.Id{
		skill.BlazeWizardStage2MeditationId,
		skill.BlazeWizardStage2SlowId,
		skill.BlazeWizardStage2SpellBoosterId,
		skill.BlazeWizardStage2ElementalResetId,
	},
}

var BlazeWizardStage3 = Job{
	id: BlazeWizardStage3Id,
	skills: []skill.Id{
		skill.BlazeWizardStage3SpellMasteryId,
		skill.BlazeWizardStage3ElementResistanceId,
		skill.BlazeWizardStage3ElementAmplificationId,
		skill.BlazeWizardStage3SealId,
		skill.BlazeWizardStage3MeteorShowerId,
		skill.BlazeWizardStage3IfritId,
		skill.BlazeWizardStage3FlameGearId,
		skill.BlazeWizardStage3FireStrikeId,
	},
	buffs: []skill.Id{
		skill.BlazeWizardStage3SealId,
		skill.BlazeWizardStage3IfritId,
	},
}

var BlazeWizardStage4 = Job{
	id:        BlazeWizardStage4Id,
	skills:    []skill.Id{},
	fourthJob: true,
}

var WindArcherStage1 = Job{
	id: WindArcherStage1Id,
	skills: []skill.Id{
		skill.WindArcherStage1CriticalShotId,
		skill.WindArcherStage1TheEyeOfAmazonId,
		skill.WindArcherStage1FocusId,
		skill.WindArcherStage1DoubleShotId,
		skill.WindArcherStage1StormId,
	},
	buffs: []skill.Id{
		skill.WindArcherStage1FocusId,
		skill.WindArcherStage1StormId,
	},
}

var WindArcherStage2 = Job{
	id: WindArcherStage2Id,
	skills: []skill.Id{
		skill.WindArcherStage2BowMasteryId,
		skill.WindArcherStage2BowBoosterId,
		skill.WindArcherStage2FinalAttackBowId,
		skill.WindArcherStage2SoulArrowId,
		skill.WindArcherStage2ThrustId,
		skill.WindArcherStage2StormBreakId,
		skill.WindArcherStage2WindWalkId,
	},
	buffs: []skill.Id{
		skill.WindArcherStage2BowBoosterId,
		skill.WindArcherStage2FinalAttackBowId,
		skill.WindArcherStage2SoulArrowId,
		skill.WindArcherStage2WindWalkId,
	},
}

var WindArcherStage3 = Job{
	id: WindArcherStage3Id,
	skills: []skill.Id{
		skill.WindArcherStage3ArrowRainId,
		skill.WindArcherStage3StrafeId,
		skill.WindArcherStage3HurricaneId,
		skill.WindArcherStage3BowExpertId,
		skill.WindArcherStage3PuppetId,
		skill.WindArcherStage3EagleEyeId,
		skill.WindArcherStage3WindPiercingId,
		skill.WindArcherStage3WindShotId,
	},
	buffs: []skill.Id{
		skill.WindArcherStage3EagleEyeId,
		skill.WindArcherStage3PuppetId,
	},
}

var WindArcherStage4 = Job{
	id:        WindArcherStage4Id,
	skills:    []skill.Id{},
	fourthJob: true,
}

var NightWalkerStage1 = Job{
	id: NightWalkerStage1Id,
	skills: []skill.Id{
		skill.NightWalkerStage1NimbleBodyId,
		skill.NightWalkerStage1KeenEyesId,
		skill.NightWalkerStage1DisorderId,
		skill.NightWalkerStage1DarkSightId,
		skill.NightWalkerStage1LuckySevenId,
		skill.NightWalkerStage1DarknessId,
	},
	buffs: []skill.Id{
		skill.NightWalkerStage1DarkSightId,
		skill.NightWalkerStage1DarknessId,
	},
}

var NightWalkerStage2 = Job{
	id: NightWalkerStage2Id,
	skills: []skill.Id{
		skill.NightWalkerStage2ClawMasteryId,
		skill.NightWalkerStage2CriticalThrowId,
		skill.NightWalkerStage2ClawBoosterId,
		skill.NightWalkerStage2HasteId,
		skill.NightWalkerStage2FlashJumpId,
		skill.NightWalkerStage2VanishId,
		skill.NightWalkerStage2VampireId,
	},
	buffs: []skill.Id{
		skill.NightWalkerStage2ClawBoosterId,
		skill.NightWalkerStage2HasteId,
	},
}

var NightWalkerStage3 = Job{
	id: NightWalkerStage3Id,
	skills: []skill.Id{
		skill.NightWalkerStage3ShadowPartnerId,
		skill.NightWalkerStage3ShadowWebId,
		skill.NightWalkerStage3AvengerId,
		skill.NightWalkerStage3AlchemistId,
		skill.NightWalkerStage3VenomId,
		skill.NightWalkerStage3TripleThrowId,
		skill.NightWalkerStage3PoisonBombId,
	},
	buffs: []skill.Id{
		skill.NightWalkerStage3ShadowPartnerId,
	},
}

var NightWalkerStage4 = Job{
	id:        NightWalkerStage4Id,
	skills:    []skill.Id{},
	fourthJob: true,
}

var ThunderBreakerStage1 = Job{
	id: ThunderBreakerStage1Id,
	skills: []skill.Id{
		skill.ThunderBreakerStage1QuickMotionId,
		skill.ThunderBreakerStage1FirstStrikeId,
		skill.ThunderBreakerStage1SomersaultKickId,
		skill.ThunderBreakerStage1DashId,
		skill.ThunderBreakerStage1LightningSpriteId,
	},
	buffs: []skill.Id{
		skill.ThunderBreakerStage1DashId,
		skill.ThunderBreakerStage1LightningSpriteId,
	},
}

var ThunderBreakerStage2 = Job{
	id: ThunderBreakerStage2Id,
	skills: []skill.Id{
		skill.ThunderBreakerStage2KnuckleMasteryId,
		skill.ThunderBreakerStage2KnuckleBoosterId,
		skill.ThunderBreakerStage2CorkscrewBlowId,
		skill.ThunderBreakerStage2EnergyChargeId,
		skill.ThunderBreakerStage2EnergyBlastId,
		skill.ThunderBreakerStage2LightningChargeId,
		skill.ThunderBreakerStage2ImprovedMaxHpIncreaseId,
	},
	buffs: []skill.Id{
		skill.ThunderBreakerStage2KnuckleBoosterId,
		skill.ThunderBreakerStage2EnergyChargeId,
		skill.ThunderBreakerStage2LightningChargeId,
	},
}

var ThunderBreakerStage3 = Job{
	id: ThunderBreakerStage3Id,
	skills: []skill.Id{
		skill.ThunderBreakerStage3CriticalPunchId,
		skill.ThunderBreakerStage3EnergyDrainId,
		skill.ThunderBreakerStage3TransformationId,
		skill.ThunderBreakerStage3ShockWaveId,
		skill.ThunderBreakerStage3BarrageId,
		skill.ThunderBreakerStage3SpeedInfusionId,
		skill.ThunderBreakerStage3SparkId,
		skill.ThunderBreakerStage3SharkWaveId,
	},
	buffs: []skill.Id{
		skill.ThunderBreakerStage3EnergyDrainId,
		skill.ThunderBreakerStage3TransformationId,
		skill.ThunderBreakerStage3SpeedInfusionId,
		skill.ThunderBreakerStage3SparkId,
	},
}

var ThunderBreakerStage4 = Job{
	id:        ThunderBreakerStage4Id,
	skills:    []skill.Id{},
	fourthJob: true,
}

var Legend = Job{
	id: LegendId,
	skills: []skill.Id{
		skill.LegendThreeSnailsId,
		skill.LegendRecoveryId,
		skill.LegendNimbleFeetId,
		skill.LegendSoulOfCraftsmanId,
		skill.LegendMonsterRidingId,
		skill.LegendEchoOfHeroId,
		skill.LegendDamageMeterId,
		skill.LegendMakerId,
		skill.LegendBambooId,
		skill.LegendInvincibleId,
		skill.LegendBerserkId,
		skill.LegendBlessOfNymphId,
	},
	buffs: []skill.Id{
		skill.LegendBlessOfNymphId,
		// AGILE BODY
		skill.LegendEchoOfHeroId,
		skill.LegendNimbleFeetId,
		skill.LegendRecoveryId,
		skill.LegendMonsterRidingId,
	},
}

var AranStage1 = Job{
	id: AranStage1Id,
	skills: []skill.Id{
		skill.AranStage1ComboAbilityId,
		skill.AranStage1CombatStepId,
		skill.AranStage1DoubleSwingId,
		skill.AranStage1PolearmBoosterId,
	},
	buffs: []skill.Id{
		skill.AranStage1ComboAbilityId,
		skill.AranStage1PolearmBoosterId,
	},
}

var AranStage2 = Job{
	id: AranStage2Id,
	skills: []skill.Id{
		skill.AranStage2PolearmMasteryId,
		skill.AranStage2TripleSwingId,
		skill.AranStage2FinalChargeId,
		skill.AranStage2BodyPressureId,
		skill.AranStage2ComboSmashId,
		skill.AranStage2ComboDrainId,
	},
	buffs: []skill.Id{
		skill.AranStage2BodyPressureId,
		skill.AranStage2ComboDrainId,
	},
}

var AranStage3 = Job{
	id: AranStage3Id,
	skills: []skill.Id{
		skill.AranStage3ComboCriticalId,
		skill.AranStage3SmartKnockbackId,
		skill.AranStage3FullSwingId,
		skill.AranStage3FinalTossId,
		skill.AranStage3ComboFenrirId,
		skill.AranStage3SnowChargeId,
		skill.AranStage3RollingSpinId,
		skill.AranStage3FullSwingDoubleSwingId,
		skill.AranStage3FullSwingTripleSwingId,
	},
	buffs: []skill.Id{
		skill.AranStage3SmartKnockbackId,
		skill.AranStage3SnowChargeId,
	},
}

var AranStage4 = Job{
	id: AranStage4Id,
	skills: []skill.Id{
		skill.AranStage4MapleWarriorId,
		skill.AranStage4HighMasteryId,
		skill.AranStage4OverSwingId,
		skill.AranStage4FreezeStandingId,
		skill.AranStage4HighDefenseId,
		skill.AranStage4FinalBlowId,
		skill.AranStage4ComboTempestId,
		skill.AranStage4ComboBarrierId,
		skill.AranStage4HerosWillId,
		skill.AranStage4OverswingDoubleSwingId,
		skill.AranStage4OverswingTripleSwingId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.AranStage4MapleWarriorId,
		skill.AranStage4ComboBarrierId,
		skill.AranStage4HerosWillId,
	},
}

var Evan = Job{
	id: EvanId,
	skills: []skill.Id{
		skill.EvanThreeSnailsId,
		skill.EvanRecoveryId,
		skill.EvanNimbleFeetId,
		skill.EvanSoulOfCraftsmanId,
		skill.EvanMonsterRidingId,
		skill.EvanEchoOfHeroId,
		skill.EvanDamageMeterId,
		skill.EvanMakerId,
		skill.EvanBambooId,
		skill.EvanInvincibleId,
		skill.EvanBerserkId,
		skill.EvanBlessOfNymphId,
	},
	buffs: []skill.Id{
		skill.EvanBlessOfNymphId,
		skill.EvanRecoveryId,
		skill.EvanNimbleFeetId,
		skill.EvanEchoOfHeroId,
		skill.EvanMonsterRidingId,
	},
}

var EvanStage1 = Job{
	id: EvanStage1Id,
	skills: []skill.Id{
		skill.EvanStage1DragonSoulId,
		skill.EvanStage1MagicMissileId,
	},
}

var EvanStage2 = Job{
	id: EvanStage2Id,
	skills: []skill.Id{
		skill.EvanStage2FireCircleId,
		skill.EvanStage2TeleportId,
	},
}

var EvanStage3 = Job{
	id: EvanStage3Id,
	skills: []skill.Id{
		skill.EvanStage3LightningBoltId,
		skill.EvanStage3MagicGuardId,
	},
	buffs: []skill.Id{
		skill.EvanStage3MagicGuardId,
	},
}

var EvanStage4 = Job{
	id: EvanStage4Id,
	skills: []skill.Id{
		skill.EvanStage4IceBreathId,
		skill.EvanStage4ElementalResetId,
	},
	buffs: []skill.Id{
		skill.EvanStage4ElementalResetId,
	},
}

var EvanStage5 = Job{
	id: EvanStage5Id,
	skills: []skill.Id{
		skill.EvanStage5MagicFlareId,
		skill.EvanStage5MagicShieldId,
	},
	buffs: []skill.Id{
		skill.EvanStage5MagicShieldId,
	},
}

var EvanStage6 = Job{
	id: EvanStage6Id,
	skills: []skill.Id{
		skill.EvanStage6CriticalMagicId,
		skill.EvanStage6DragonThrustId,
		skill.EvanStage6MagicBoosterId,
		skill.EvanStage6SlowId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.EvanStage6MagicBoosterId,
		skill.EvanStage6SlowId,
	},
}

var EvanStage7 = Job{
	id: EvanStage7Id,
	skills: []skill.Id{
		skill.EvanStage7MagicAmplificationId,
		skill.EvanStage7FireBreathId,
		skill.EvanStage7KillerWingsId,
		skill.EvanStage7MagicResistanceId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.EvanStage7MagicResistanceId,
	},
}

var EvanStage8 = Job{
	id: EvanStage8Id,
	skills: []skill.Id{
		skill.EvanStage8DragonFuryId,
		skill.EvanStage8EarthquakeId,
		skill.EvanStage8PhantomImprintId,
		skill.EvanStage8RecoveryAuraId,
	},
	fourthJob: true,
}

var EvanStage9 = Job{
	id: EvanStage9Id,
	skills: []skill.Id{
		skill.EvanStage9MapleWarriorId,
		skill.EvanStage9MagicMasteryId,
		skill.EvanStage9IllusionId,
		skill.EvanStage9FlameWheelId,
		skill.EvanStage9HerosWillId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.EvanStage9MapleWarriorId,
		skill.EvanStage9HerosWillId,
	},
}

var EvanStage10 = Job{
	id: EvanStage10Id,
	skills: []skill.Id{
		skill.EvanStage10BlessingOfTheOnyxId,
		skill.EvanStage10BlazeId,
		skill.EvanStage10DarkFogId,
		skill.EvanStage10SoulStoneId,
	},
	fourthJob: true,
	buffs: []skill.Id{
		skill.EvanStage10BlessingOfTheOnyxId,
		skill.EvanStage10SoulStoneId,
	},
}

var Jobs = []Job{Beginner, Warrior, Fighter, Crusader, Hero, Page, WhiteKnight, Paladin, Spearman, DragonKnight, DarkKnight, Magician, FirePoisonWizard, FirePoisonMagician, FirePoisonArchMagician, IceLightningWizard, IceLightningMagician, IceLightningArchMagician, Cleric, Priest, Bishop, Bowman, Hunter, Ranger, Bowmaster, Crossbowman, Sniper, Marksman, Rogue, Assassin, Hermit, NightLord, Bandit, ChiefBandit, Shadower, Pirate, Brawler, Marauder, Buccaneer, Gunslinger, Outlaw, Corsair, MapleLeafBrigadier, Gm, SuperGm, Noblesse, DawnWarriorStage1, DawnWarriorStage2, DawnWarriorStage3, DawnWarriorStage4, BlazeWizardStage1, BlazeWizardStage2, BlazeWizardStage3, BlazeWizardStage4, WindArcherStage1, WindArcherStage2, WindArcherStage3, WindArcherStage4, NightWalkerStage1, NightWalkerStage2, NightWalkerStage3, NightWalkerStage4, ThunderBreakerStage1, ThunderBreakerStage2, ThunderBreakerStage3, ThunderBreakerStage4, Legend, AranStage1, AranStage2, AranStage3, AranStage4, Evan, EvanStage1, EvanStage2, EvanStage3, EvanStage4, EvanStage5, EvanStage6, EvanStage7, EvanStage8, EvanStage9, EvanStage10}

const (
	BeginnerId                 = Id(0)
	WarriorId                  = Id(100)
	FighterId                  = Id(110)
	CrusaderId                 = Id(111)
	HeroId                     = Id(112)
	PageId                     = Id(120)
	WhiteKnightId              = Id(121)
	PaladinId                  = Id(122)
	SpearmanId                 = Id(130)
	DragonKnightId             = Id(131)
	DarkKnightId               = Id(132)
	MagicianId                 = Id(200)
	FirePoisonWizardId         = Id(210)
	FirePoisonMagicianId       = Id(211)
	FirePoisonArchMagicianId   = Id(212)
	IceLightningWizardId       = Id(220)
	IceLightningMagicianId     = Id(221)
	IceLightningArchMagicianId = Id(222)
	ClericId                   = Id(230)
	PriestId                   = Id(231)
	BishopId                   = Id(232)
	BowmanId                   = Id(300)
	HunterId                   = Id(310)
	RangerId                   = Id(311)
	BowmasterId                = Id(312)
	CrossbowmanId              = Id(320)
	SniperId                   = Id(321)
	MarksmanId                 = Id(322)
	RogueId                    = Id(400)
	AssassinId                 = Id(410)
	HermitId                   = Id(411)
	NightLordId                = Id(412)
	BanditId                   = Id(420)
	ChiefBanditId              = Id(421)
	ShadowerId                 = Id(422)
	PirateId                   = Id(500)
	BrawlerId                  = Id(510)
	MarauderId                 = Id(511)
	BuccaneerId                = Id(512)
	GunslingerId               = Id(520)
	OutlawId                   = Id(521)
	CorsairId                  = Id(522)
	MapleLeafBrigadierId       = Id(800)
	GmId                       = Id(900)
	SuperGmId                  = Id(910)
	NoblesseId                 = Id(1000)
	DawnWarriorStage1Id        = Id(1100)
	DawnWarriorStage2Id        = Id(1110)
	DawnWarriorStage3Id        = Id(1111)
	DawnWarriorStage4Id        = Id(1112)
	BlazeWizardStage1Id        = Id(1200)
	BlazeWizardStage2Id        = Id(1210)
	BlazeWizardStage3Id        = Id(1211)
	BlazeWizardStage4Id        = Id(1212)
	WindArcherStage1Id         = Id(1300)
	WindArcherStage2Id         = Id(1310)
	WindArcherStage3Id         = Id(1311)
	WindArcherStage4Id         = Id(1312)
	NightWalkerStage1Id        = Id(1400)
	NightWalkerStage2Id        = Id(1410)
	NightWalkerStage3Id        = Id(1411)
	NightWalkerStage4Id        = Id(1412)
	ThunderBreakerStage1Id     = Id(1500)
	ThunderBreakerStage2Id     = Id(1510)
	ThunderBreakerStage3Id     = Id(1511)
	ThunderBreakerStage4Id     = Id(1512)
	LegendId                   = Id(2000)
	AranStage1Id               = Id(2100)
	AranStage2Id               = Id(2110)
	AranStage3Id               = Id(2111)
	AranStage4Id               = Id(2112)
	EvanId                     = Id(2001)
	EvanStage1Id               = Id(2200)
	EvanStage2Id               = Id(2210)
	EvanStage3Id               = Id(2211)
	EvanStage4Id               = Id(2212)
	EvanStage5Id               = Id(2213)
	EvanStage6Id               = Id(2214)
	EvanStage7Id               = Id(2215)
	EvanStage8Id               = Id(2216)
	EvanStage9Id               = Id(2217)
	EvanStage10Id              = Id(2218)
)

func IsA(jobId Id, referenceJobs ...Id) bool {
	is := false
	for _, referenceJobId := range referenceJobs {
		if Is(jobId, referenceJobId) {
			is = true
		}
	}
	return is
}

func Is(jobId Id, referenceJobId Id) bool {
	characterBranch := jobId / 10
	referenceBranch := referenceJobId / 10
	return characterBranch == referenceBranch && jobId >= referenceJobId || referenceBranch%10 == 0 && jobId/100 == referenceJobId/100
}

func JobFromSkillId(skillId Id) Id {
	return Id(math.Floor(float64(skillId) / float64(10000)))
}

func IsFourthJob(jobId Id) bool {
	for _, j := range Jobs {
		if j.id == jobId {
			return j.IsFourthJob()
		}
	}
	return false
}
