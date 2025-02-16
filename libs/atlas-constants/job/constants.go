package job

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"math"
)

type Id uint16

type Job struct {
	id        Id
	skills    []skill.Skill
	fourthJob bool
}

func (j Job) Id() Id {
	return j.id
}

func (j Job) Skills() []skill.Skill {
	return j.skills
}

func (j Job) Buffs() []skill.Skill {
	return j.skills
}

func (j Job) IsFourthJob() bool {
	return j.fourthJob
}

var Beginner = Job{
	id: BeginnerId,
	skills: []skill.Skill{
		skill.BeginnerThreeSnails,
		skill.BeginnerRecovery,
		skill.BeginnerNimbleFeet,
		skill.BeginnerSoulOfCraftsman,
		skill.BeginnerMonsterRiding,
		skill.BeginnerEchoOfHero,
		skill.BeginnerJumpDown,
		skill.BeginnerMaker,
		skill.BeginnerMultiPet,
		skill.BeginnerBamboo,
		skill.BeginnerInvincible,
		skill.BeginnerBerserk,
		skill.BeginnerBlessOfNymph,
	},
}

var Warrior = Job{
	id: WarriorId,
	skills: []skill.Skill{
		skill.WarriorImprovedHpRecovery,
		skill.WarriorImprovedMaxHpIncrease,
		skill.WarriorEndure,
		skill.WarriorIronBody,
		skill.WarriorPowerStrike,
		skill.WarriorSlashBlast,
	},
}

var Fighter = Job{
	id: FighterId,
	skills: []skill.Skill{
		skill.FighterSwordMastery,
		skill.FighterAxeMastery,
		skill.FighterFinalAttackSword,
		skill.FighterFinalAttackAxe,
		skill.FighterSwordBooster,
		skill.FighterAxeBooster,
		skill.FighterRage,
		skill.FighterPowerGuard,
	},
}

var Crusader = Job{
	id: CrusaderId,
	skills: []skill.Skill{
		skill.CrusaderImprovingMpRecovery,
		skill.CrusaderShieldMastery,
		skill.CrusaderComboAttack,
		skill.CrusaderPanicSword,
		skill.CrusaderPanicAxe,
		skill.CrusaderComaSword,
		skill.CrusaderComaAxe,
		skill.CrusaderArmorCrash,
		skill.CrusaderShout,
	},
}

var Hero = Job{
	id: HeroId,
	skills: []skill.Skill{
		skill.HeroMapleWarrior,
		skill.HeroMonsterMagnet,
		skill.HeroPowerStance,
		skill.HeroAdvancedComboAttack,
		skill.HeroAchilles,
		skill.HeroGuardian,
		skill.HeroRush,
		skill.HeroBrandish,
		skill.HeroEnrage,
		skill.HeroHerosWill,
	},
	fourthJob: true,
}

var Page = Job{
	id: PageId,
	skills: []skill.Skill{
		skill.PageSwordMastery,
		skill.PageBluntWeaponMastery,
		skill.PageSwordFinalAttack,
		skill.PageBluntFinalAttack,
		skill.PageSwordBooster,
		skill.PageBluntWeaponBooster,
		skill.PageThreaten,
		skill.PagePowerGuard,
	},
}

var WhiteKnight = Job{
	id: WhiteKnightId,
	skills: []skill.Skill{
		skill.WhiteKnightImprovingMpRecovery,
		skill.WhiteKnightShieldMastery,
		skill.WhiteKnightChargedBlow,
		skill.WhiteKnightFireChargeSword,
		skill.WhiteKnightFlameChargeBluntWeapon,
		skill.WhiteKnightIceChargeSword,
		skill.WhiteKnightBlizzardChargeBluntWeapon,
		skill.WhiteKnightThunderChargeSword,
		skill.WhiteKnightLightningChargeBluntWeapon,
		skill.WhiteKnightMagicCrash,
	},
}

var Paladin = Job{
	id: PaladinId,
	skills: []skill.Skill{
		skill.PaladinMapleWarrior,
		skill.PaladinMonsterMagnet,
		skill.PaladinPowerStance,
		skill.PaladinHolyChargeSword,
		skill.PaladinDivineChargeBluntWeapon,
		skill.PaladinAchilles,
		skill.PaladinGuardian,
		skill.PaladinRush,
		skill.PaladinBlast,
		skill.PaladinAdvancedCharge,
		skill.PaladinHeavensHammer,
		skill.PaladinHerosWill,
	},
	fourthJob: true,
}

var Spearman = Job{
	id: SpearmanId,
	skills: []skill.Skill{
		skill.SpearmanSpearMastery,
		skill.SpearmanPolearmMastery,
		skill.SpearmanFinalAttackSpear,
		skill.SpearmanFinalAttackPolearm,
		skill.SpearmanSpearBooster,
		skill.SpearmanPolearmBooster,
		skill.SpearmanIronWill,
		skill.SpearmanHyperBody,
	},
}

var DragonKnight = Job{
	id: DragonKnightId,
	skills: []skill.Skill{
		skill.DragonKnightElementalResistance,
		skill.DragonKnightSpearCrusher,
		skill.DragonKnightPolearmCrusher,
		skill.DragonKnightDragonFurySpear,
		skill.DragonKnightDragonFuryPolearm,
		skill.DragonKnightSacrifice,
		skill.DragonKnightDragonRoar,
		skill.DragonKnightPowerCrash,
		skill.DragonKnightDragonBlood,
	},
}

var DarkKnight = Job{
	id: DarkKnightId,
	skills: []skill.Skill{
		skill.DarkKnightMapleWarrior,
		skill.DarkKnightMonsterMagnet,
		skill.DarkKnightPowerStance,
		skill.DarkKnightRush,
		skill.DarkKnightAchilles,
		skill.DarkKnightBerserk,
		skill.DarkKnightBeholder,
		skill.DarkKnightAuraOfTheBeholder,
		skill.DarkKnightHexOfTheBeholder,
		skill.DarkKnightHerosWill,
	},
	fourthJob: true,
}

var Magician = Job{
	id: MagicianId,
	skills: []skill.Skill{
		skill.MagicianImprovedMpRecovery,
		skill.MagicianImprovedMaxMpIncrease,
		skill.MagicianMagicGuard,
		skill.MagicianMagicArmor,
		skill.MagicianEnergyBolt,
		skill.MagicianMagicClaw,
	},
}

var FirePoisonWizard = Job{
	id: FirePoisonWizardId,
	skills: []skill.Skill{
		skill.FirePoisionWizardMpEater,
		skill.FirePoisionWizardMeditation,
		skill.FirePoisionWizardTeleport,
		skill.FirePoisionWizardSlow,
		skill.FirePoisionWizardFireArrow,
		skill.FirePoisionWizardPoisonBreath,
	},
}

var FirePoisonMagician = Job{
	id: FirePoisonMagicianId,
	skills: []skill.Skill{
		skill.FirePoisonMagicianPartialResistance,
		skill.FirePoisonMagicianElementAmplification,
		skill.FirePoisonMagicianExplosion,
		skill.FirePoisonMagicianPoisonMist,
		skill.FirePoisonMagicianSeal,
		skill.FirePoisonMagicianSpellBooster,
		skill.FirePoisonMagicianElementComposition,
	},
}

var FirePoisonArchMagician = Job{
	id: FirePoisonArchMagicianId,
	skills: []skill.Skill{
		skill.FirePoisonArchMagicianMapleWarrior,
		skill.FirePoisonArchMagicianBigBang,
		skill.FirePoisonArchMagicianManaReflection,
		skill.FirePoisonArchMagicianFireDemon,
		skill.FirePoisonArchMagicianInfinity,
		skill.FirePoisonArchMagicianElquines,
		skill.FirePoisonArchMagicianParalyze,
		skill.FirePoisonArchMagicianMeteorShower,
		skill.FirePoisonArchMagicianHerosWill,
	},
	fourthJob: true,
}

var IceLightningWizard = Job{
	id: IceLightningWizardId,
	skills: []skill.Skill{
		skill.IceLightningWizardMpEater,
		skill.IceLightningWizardMeditation,
		skill.IceLightningWizardTeleport,
		skill.IceLightningWizardSlow,
		skill.IceLightningWizardColdBeam,
		skill.IceLightningWizardThunderBolt,
	},
}

var IceLightningMagician = Job{
	id: IceLightningMagicianId,
	skills: []skill.Skill{
		skill.IceLightningMagicianPartialResistance,
		skill.IceLightningMagicianElementAmplification,
		skill.IceLightningMagicianIceStrike,
		skill.IceLightningMagicianThunderSpear,
		skill.IceLightningMagicianSeal,
		skill.IceLightningMagicianSpellBooster,
		skill.IceLightningMagicianElementComposition,
	},
}

var IceLightningArchMagician = Job{
	id: IceLightningArchMagicianId,
	skills: []skill.Skill{
		skill.IceLightningArchMagicianMapleWarrior,
		skill.IceLightningArchMagicianBigBang,
		skill.IceLightningArchMagicianManaReflection,
		skill.IceLightningArchMagicianIceDemon,
		skill.IceLightningArchMagicianInfinity,
		skill.IceLightningArchMagicianIfrit,
		skill.IceLightningArchMagicianChainLightning,
		skill.IceLightningArchMagicianBlizzard,
		skill.IceLightningArchMagicianHerosWill,
	},
	fourthJob: true,
}

var Cleric = Job{
	id: ClericId,
	skills: []skill.Skill{
		skill.ClericMpEater,
		skill.ClericTeleport,
		skill.ClericHeal,
		skill.ClericInvincible,
		skill.ClericBless,
		skill.ClericHolyArrow,
	},
}

var Priest = Job{
	id: PriestId,
	skills: []skill.Skill{
		skill.PriestElementalResistance,
		skill.PriestDispel,
		skill.PriestMysticDoor,
		skill.PriestHolySymbol,
		skill.PriestShiningRay,
		skill.PriestDoom,
		skill.PriestSummonDragon,
	},
}

var Bishop = Job{
	id: BishopId,
	skills: []skill.Skill{
		skill.BishopMapleWarrior,
		skill.BishopBigBang,
		skill.BishopManaReflection,
		skill.BishopBahamut,
		skill.BishopInfinity,
		skill.BishopHolyShield,
		skill.BishopResurrection,
		skill.BishopAngelRay,
		skill.BishopGenesis,
		skill.BishopHerosWill,
	},
	fourthJob: true,
}

var Bowman = Job{
	id: BowmanId,
	skills: []skill.Skill{
		skill.BowmanTheBlessingOfAmazon,
		skill.BowmanCriticalShot,
		skill.BowmanTheEyeOfAmazon,
		skill.BowmanFocus,
		skill.BowmanArrowBlow,
		skill.BowmanDoubleShot,
	},
}

var Hunter = Job{
	id: HunterId,
	skills: []skill.Skill{
		skill.HunterBowMastery,
		skill.HunterFinalAttackBow,
		skill.HunterBowBooster,
		skill.HunterPowerKnockBack,
		skill.HunterSoulArrowBow,
		skill.HunterArrowBombBow,
	},
}

var Ranger = Job{
	id: RangerId,
	skills: []skill.Skill{
		skill.RangerThrust,
		skill.RangerMortalBlow,
		skill.RangerPuppet,
		skill.RangerInferno,
		skill.RangerArrowRain,
		skill.RangerSilverHawk,
		skill.RangerStrafe,
	},
}

var Bowmaster = Job{
	id: BowmasterId,
	skills: []skill.Skill{
		skill.BowmasterMapleWarrior,
		skill.BowmasterSharpEyes,
		skill.BowmasterDragonsBreath,
		skill.BowmasterHurricane,
		skill.BowmasterBowExpert,
		skill.BowmasterPhoenix,
		skill.BowmasterHamstring,
		skill.BowmasterConcentrate,
		skill.BowmasterHerosWill,
	},
	fourthJob: true,
}

var Crossbowman = Job{
	id: CrossbowmanId,
	skills: []skill.Skill{
		skill.CrossbowmanCrossbowMastery,
		skill.CrossbowmanFinalAttackCrossbow,
		skill.CrossbowmanCrossbowBooster,
		skill.CrossbowmanPowerKnockBack,
		skill.CrossbowmanSoulArrowCrossbow,
		skill.CrossbowmanIronArrowCrossbow,
	},
}

var Sniper = Job{
	id: SniperId,
	skills: []skill.Skill{
		skill.SniperThrust,
		skill.SniperMortalBlow,
		skill.SniperPuppet,
		skill.SniperBlizzard,
		skill.SniperArrowEruption,
		skill.SniperGoldenEagle,
		skill.SniperStrafe,
	},
}

var Marksman = Job{
	id: MarksmanId,
	skills: []skill.Skill{
		skill.MarksmanMapleWarrior,
		skill.MarksmanPiercingArrow,
		skill.MarksmanSharpEyes,
		skill.MarksmanDragonsBreath,
		skill.MarksmanMarksmanBoost,
		skill.MarksmanFrostprey,
		skill.MarksmanBlind,
		skill.MarksmanSnipe,
		skill.MarksmanHerosWill,
	},
	fourthJob: true,
}

var Rogue = Job{
	id: RogueId,
	skills: []skill.Skill{
		skill.RogueNimbleBody,
		skill.RogueKeenEyes,
		skill.RogueDisorder,
		skill.RogueDarkSight,
		skill.RogueDoubleStab,
		skill.RogueLuckySeven,
	},
}

var Assassin = Job{
	id: AssassinId,
	skills: []skill.Skill{
		skill.AssassinClawMastery,
		skill.AssassinCriticalThrow,
		skill.AssassinEndure,
		skill.AssassinClawBooster,
		skill.AssassinHaste,
		skill.AssassinDrain,
	},
}

var Hermit = Job{
	id: HermitId,
	skills: []skill.Skill{
		skill.HermitAlchemist,
		skill.HermitMesoUp,
		skill.HermitShadowPartner,
		skill.HermitShadowWeb,
		skill.HermitShadowMeso,
		skill.HermitAvenger,
		skill.HermitFlashJump,
	},
}

var NightLord = Job{
	id: NightLordId,
	skills: []skill.Skill{
		skill.NightLordMapleWarrior,
		skill.NightLordShadowShifter,
		skill.NightLordTaunt,
		skill.NightLordNinjaAmbush,
		skill.NightLordVenomousStar,
		skill.NightLordShadowStars,
		skill.NightLordTripleThrow,
		skill.NightLordNinjaStorm,
		skill.NightLordHerosWill,
	},
	fourthJob: true,
}

var Bandit = Job{
	id: BanditId,
	skills: []skill.Skill{
		skill.BanditDaggerMastery,
		skill.BanditEndure,
		skill.BanditDaggerBooster,
		skill.BanditHaste,
		skill.BanditSteal,
		skill.BanditSavageBlow,
	},
}

var ChiefBandit = Job{
	id: ChiefBanditId,
	skills: []skill.Skill{
		skill.ChiefBanditShieldMastery,
		skill.ChiefBanditChakra,
		skill.ChiefBanditAssaulter,
		skill.ChiefBanditPickpocket,
		skill.ChiefBanditBandOfThieves,
		skill.ChiefBanditMesoGuard,
		skill.ChiefBanditMesoExplosion,
	},
}

var Shadower = Job{
	id: ShadowerId,
	skills: []skill.Skill{
		skill.ShadowerMapleWarrior,
		skill.ShadowerAssassinate,
		skill.ShadowerShadowShifter,
		skill.ShadowerTaunt,
		skill.ShadowerNinjaAmbush,
		skill.ShadowerVenomousStab,
		skill.ShadowerSmokescreen,
		skill.ShadowerBoomerangStep,
		skill.ShadowerHerosWill,
	},
	fourthJob: true,
}

var Pirate = Job{
	id: PirateId,
	skills: []skill.Skill{
		skill.PirateBulletTime,
		skill.PirateFlashFist,
		skill.PirateSommersaultKick,
		skill.PirateDoubleShot,
		skill.PirateDash,
	},
}

var Brawler = Job{
	id: BrawlerId,
	skills: []skill.Skill{
		skill.BrawlerImproveMaxHp,
		skill.BrawlerKnucklerMastery,
		skill.BrawlerBackspinBlow,
		skill.BrawlerDoubleUppercut,
		skill.BrawlerCorkscrewBlow,
		skill.BrawlerMPRecovery,
		skill.BrawlerKnucklerBooster,
		skill.BrawlerOakBarrel,
	},
}

var Marauder = Job{
	id: MarauderId,
	skills: []skill.Skill{
		skill.MarauderStunMastery,
		skill.MarauderEnergyCharge,
		skill.MarauderEnergyBlast,
		skill.MarauderEnergyDrain,
		skill.MarauderTransformation,
		skill.MarauderShockwave,
	},
}

var Buccaneer = Job{
	id: BuccaneerId,
	skills: []skill.Skill{
		skill.BuccaneerMapleWarrior,
		skill.BuccaneerDragonStrike,
		skill.BuccaneerEnergyOrb,
		skill.BuccaneerSuperTransformation,
		skill.BuccaneerDemolition,
		skill.BuccaneerSnatch,
		skill.BuccaneerBarrage,
		skill.BuccaneerPiratesRage,
		skill.BuccaneerSpeedInfusion,
		skill.BuccaneerTimeLeap,
	},
	fourthJob: true,
}

var Gunslinger = Job{
	id: GunslingerId,
	skills: []skill.Skill{
		skill.GunslingerGunMastery,
		skill.GunslingerInvisibleShot,
		skill.GunslingerGrenade,
		skill.GunslingerGunBooster,
		skill.GunslingerBlankShot,
		skill.GunslingerWings,
		skill.GunslingerRecoilShot,
	},
}

var Outlaw = Job{
	id: OutlawId,
	skills: []skill.Skill{
		skill.OutlawBurstFire,
		skill.OutlawOctopus,
		skill.OutlawGaviota,
		skill.OutlawFlamethrower,
		skill.OutlawIceSplitter,
		skill.OutlawHomingBeacon,
	},
}

var Corsair = Job{
	id: CorsairId,
	skills: []skill.Skill{
		skill.CorsairMapleWarrior,
		skill.CorsairElementalBoost,
		skill.CorsairWrathOfTheOctopi,
		skill.CorsairAerialStrike,
		skill.CorsairRapidFire,
		skill.CorsairBattleship,
		skill.CorsairBattleshipCannon,
		skill.CorsairBattleshipTorpedo,
		skill.CorsairHypnotize,
		skill.CorsairSpeedInfusion,
		skill.CorsairBullseye,
	},
	fourthJob: true,
}

var MapleLeafBrigadier = Job{
	id: MapleLeafBrigadierId,
	skills: []skill.Skill{
		skill.MapleLeafBrigadierAntiMacro,
		skill.MapleLeafBrigadierTeleport,
	},
}

var Gm = Job{
	id: GmId,
	skills: []skill.Skill{
		skill.GmHaste,
		skill.GmSuperDragonRoar,
		skill.GmTeleport,
		skill.GmHide,
	},
}

var SuperGm = Job{
	id: SuperGmId,
	skills: []skill.Skill{
		skill.SuperGmHealDispel,
		skill.SuperGmHaste,
		skill.SuperGmHolySymbol,
		skill.SuperGmBless,
		skill.SuperGmHide,
		skill.SuperGmResurrection,
		skill.SuperGmDragonRoar,
		skill.SuperGmTeleport,
		skill.SuperGmHyperBody,
	},
}

var Noblesse = Job{
	id: NoblesseId,
	skills: []skill.Skill{
		skill.NoblesseThreeSnails,
		skill.NoblesseRecovery,
		skill.NoblesseNimbleFeet,
		skill.NoblesseSoulOfCraftsman,
		skill.NoblesseMonsterRiding,
		skill.NoblesseEchoOfHero,
		skill.NoblesseJumpDown,
		skill.NoblesseMaker,
		skill.NoblesseMultiPet,
		skill.NoblesseBamboo,
		skill.NoblesseInvincible,
		skill.NoblesseBerserk,
		skill.NoblesseBlessOfNymph,
	},
}

var DawnWarriorStage1 = Job{
	id: DawnWarriorStage1Id,
	skills: []skill.Skill{
		skill.DawnWarriorStage1IronBody,
		skill.DawnWarriorStage1PowerStrike,
		skill.DawnWarriorStage1SlashBlast,
		skill.DawnWarriorStage1Soul,
		skill.DawnWarriorStage1ImprovedMaxHpIncrease,
	},
}

var DawnWarriorStage2 = Job{
	id: DawnWarriorStage2Id,
	skills: []skill.Skill{
		skill.DawnWarriorStage2SwordMastery,
		skill.DawnWarriorStage2SwordBooster,
		skill.DawnWarriorStage2FinalAttackSword,
		skill.DawnWarriorStage2Rage,
		skill.DawnWarriorStage2SoulBlade,
		skill.DawnWarriorStage2SoulRush,
	},
}

var DawnWarriorStage3 = Job{
	id: DawnWarriorStage3Id,
	skills: []skill.Skill{
		skill.DawnWarriorStage3ImprovedMpRecovery,
		skill.DawnWarriorStage3ComboAttack,
		skill.DawnWarriorStage3Panic,
		skill.DawnWarriorStage3Coma,
		skill.DawnWarriorStage3Brandish,
		skill.DawnWarriorStage3AdvancedCombo,
		skill.DawnWarriorStage3SoulDriver,
		skill.DawnWarriorStage3SoulCharge,
	},
}

var DawnWarriorStage4 = Job{
	id:        DawnWarriorStage4Id,
	skills:    []skill.Skill{},
	fourthJob: true,
}

var BlazeWizardStage1 = Job{
	id: BlazeWizardStage1Id,
	skills: []skill.Skill{
		skill.BlazeWizardStage1MagicGuard,
		skill.BlazeWizardStage1MagicArmor,
		skill.BlazeWizardStage1MagicClaw,
		skill.BlazeWizardStage1Flame,
		skill.BlazeWizardStage1ImprovedMaxMpIncrease,
	},
}

var BlazeWizardStage2 = Job{
	id: BlazeWizardStage2Id,
	skills: []skill.Skill{
		skill.BlazeWizardStage2Meditation,
		skill.BlazeWizardStage2Slow,
		skill.BlazeWizardStage2FireArrow,
		skill.BlazeWizardStage2Teleport,
		skill.BlazeWizardStage2SpellBooster,
		skill.BlazeWizardStage2ElementalReset,
		skill.BlazeWizardStage2FirePillar,
	},
}

var BlazeWizardStage3 = Job{
	id: BlazeWizardStage3Id,
	skills: []skill.Skill{
		skill.BlazeWizardStage3SpellMastery,
		skill.BlazeWizardStage3ElementResistance,
		skill.BlazeWizardStage3ElementAmplification,
		skill.BlazeWizardStage3Seal,
		skill.BlazeWizardStage3MeteorShower,
		skill.BlazeWizardStage3Ifrit,
		skill.BlazeWizardStage3FlameGear,
		skill.BlazeWizardStage3FireStrike,
	},
}

var BlazeWizardStage4 = Job{
	id:        BlazeWizardStage4Id,
	skills:    []skill.Skill{},
	fourthJob: true,
}

var WindArcherStage1 = Job{
	id: WindArcherStage1Id,
	skills: []skill.Skill{
		skill.WindArcherStage1CriticalShot,
		skill.WindArcherStage1TheEyeOfAmazon,
		skill.WindArcherStage1Focus,
		skill.WindArcherStage1DoubleShot,
		skill.WindArcherStage1Storm,
	},
}

var WindArcherStage2 = Job{
	id: WindArcherStage2Id,
	skills: []skill.Skill{
		skill.WindArcherStage2BowMastery,
		skill.WindArcherStage2BowBooster,
		skill.WindArcherStage2FinalAttackBow,
		skill.WindArcherStage2SoulArrow,
		skill.WindArcherStage2Thrust,
		skill.WindArcherStage2StormBreak,
		skill.WindArcherStage2WindWalk,
	},
}

var WindArcherStage3 = Job{
	id: WindArcherStage3Id,
	skills: []skill.Skill{
		skill.WindArcherStage3ArrowRain,
		skill.WindArcherStage3Strafe,
		skill.WindArcherStage3Hurricane,
		skill.WindArcherStage3BowExpert,
		skill.WindArcherStage3Puppet,
		skill.WindArcherStage3EagleEye,
		skill.WindArcherStage3WindPiercing,
		skill.WindArcherStage3WindShot,
	},
}

var WindArcherStage4 = Job{
	id:        WindArcherStage4Id,
	skills:    []skill.Skill{},
	fourthJob: true,
}

var NightWalkerStage1 = Job{
	id: NightWalkerStage1Id,
	skills: []skill.Skill{
		skill.NightWalkerStage1NimbleBody,
		skill.NightWalkerStage1KeenEyes,
		skill.NightWalkerStage1Disorder,
		skill.NightWalkerStage1DarkSight,
		skill.NightWalkerStage1LuckySeven,
		skill.NightWalkerStage1Darkness,
	},
}

var NightWalkerStage2 = Job{
	id: NightWalkerStage2Id,
	skills: []skill.Skill{
		skill.NightWalkerStage2ClawMastery,
		skill.NightWalkerStage2CriticalThrow,
		skill.NightWalkerStage2ClawBooster,
		skill.NightWalkerStage2Haste,
		skill.NightWalkerStage2FlashJump,
		skill.NightWalkerStage2Vanish,
		skill.NightWalkerStage2Vampire,
	},
}

var NightWalkerStage3 = Job{
	id: NightWalkerStage3Id,
	skills: []skill.Skill{
		skill.NightWalkerStage3ShadowPartner,
		skill.NightWalkerStage3ShadowWeb,
		skill.NightWalkerStage3Avenger,
		skill.NightWalkerStage3Alchemist,
		skill.NightWalkerStage3Venom,
		skill.NightWalkerStage3TripleThrow,
		skill.NightWalkerStage3PoisonBomb,
	},
}

var NightWalkerStage4 = Job{
	id:        NightWalkerStage4Id,
	skills:    []skill.Skill{},
	fourthJob: true,
}

var ThunderBreakerStage1 = Job{
	id: ThunderBreakerStage1Id,
	skills: []skill.Skill{
		skill.ThunderBreakerStage1QuickMotion,
		skill.ThunderBreakerStage1FirstStrike,
		skill.ThunderBreakerStage1SomersaultKick,
		skill.ThunderBreakerStage1Dash,
		skill.ThunderBreakerStage1LightningSprite,
	},
}

var ThunderBreakerStage2 = Job{
	id: ThunderBreakerStage2Id,
	skills: []skill.Skill{
		skill.ThunderBreakerStage2KnuckleMastery,
		skill.ThunderBreakerStage2KnuckleBooster,
		skill.ThunderBreakerStage2CorkscrewBlow,
		skill.ThunderBreakerStage2EnergyCharge,
		skill.ThunderBreakerStage2EnergyBlast,
		skill.ThunderBreakerStage2LightningCharge,
		skill.ThunderBreakerStage2ImprovedMaxHpIncrease,
	},
}

var ThunderBreakerStage3 = Job{
	id: ThunderBreakerStage3Id,
	skills: []skill.Skill{
		skill.ThunderBreakerStage3CriticalPunch,
		skill.ThunderBreakerStage3EnergyDrain,
		skill.ThunderBreakerStage3Transformation,
		skill.ThunderBreakerStage3ShockWave,
		skill.ThunderBreakerStage3Barrage,
		skill.ThunderBreakerStage3SpeedInfusion,
		skill.ThunderBreakerStage3Spark,
		skill.ThunderBreakerStage3SharkWave,
	},
}

var ThunderBreakerStage4 = Job{
	id:        ThunderBreakerStage4Id,
	skills:    []skill.Skill{},
	fourthJob: true,
}

var Legend = Job{
	id: LegendId,
	skills: []skill.Skill{
		skill.LegendThreeSnails,
		skill.LegendRecovery,
		skill.LegendNimbleFeet,
		skill.LegendSoulOfCraftsman,
		skill.LegendMonsterRiding,
		skill.LegendEchoOfHero,
		skill.LegendDamageMeter,
		skill.LegendMaker,
		skill.LegendBamboo,
		skill.LegendInvincible,
		skill.LegendBerserk,
		skill.LegendBlessOfNymph,
	},
}

var AranStage1 = Job{
	id: AranStage1Id,
	skills: []skill.Skill{
		skill.AranStage1ComboAbility,
		skill.AranStage1CombatStep,
		skill.AranStage1DoubleSwing,
		skill.AranStage1PolearmBooster,
	},
}

var AranStage2 = Job{
	id: AranStage2Id,
	skills: []skill.Skill{
		skill.AranStage2PolearmMastery,
		skill.AranStage2TripleSwing,
		skill.AranStage2FinalCharge,
		skill.AranStage2BodyPressure,
		skill.AranStage2ComboSmash,
		skill.AranStage2ComboDrain,
	},
}

var AranStage3 = Job{
	id: AranStage3Id,
	skills: []skill.Skill{
		skill.AranStage3ComboCritical,
		skill.AranStage3SmartKnockback,
		skill.AranStage3FullSwing,
		skill.AranStage3FinalToss,
		skill.AranStage3ComboFenrir,
		skill.AranStage3SnowCharge,
		skill.AranStage3RollingSpin,
		skill.AranStage3FullSwingDoubleSwing,
		skill.AranStage3FullSwingTripleSwing,
	},
}

var AranStage4 = Job{
	id: AranStage4Id,
	skills: []skill.Skill{
		skill.AranStage4MapleWarrior,
		skill.AranStage4HighMastery,
		skill.AranStage4OverSwing,
		skill.AranStage4FreezeStanding,
		skill.AranStage4HighDefense,
		skill.AranStage4FinalBlow,
		skill.AranStage4ComboTempest,
		skill.AranStage4ComboBarrier,
		skill.AranStage4HerosWill,
		skill.AranStage4OverswingDoubleSwing,
		skill.AranStage4OverswingTripleSwing,
	},
	fourthJob: true,
}

var Evan = Job{
	id: EvanId,
	skills: []skill.Skill{
		skill.EvanThreeSnails,
		skill.EvanRecovery,
		skill.EvanNimbleFeet,
		skill.EvanSoulOfCraftsman,
		skill.EvanMonsterRiding,
		skill.EvanEchoOfHero,
		skill.EvanDamageMeter,
		skill.EvanMaker,
		skill.EvanBamboo,
		skill.EvanInvincible,
		skill.EvanBerserk,
		skill.EvanBlessOfNymph,
	},
}

var EvanStage1 = Job{
	id: EvanStage1Id,
	skills: []skill.Skill{
		skill.EvanStage1DragonSoul,
		skill.EvanStage1MagicMissile,
	},
}

var EvanStage2 = Job{
	id: EvanStage2Id,
	skills: []skill.Skill{
		skill.EvanStage2FireCircle,
		skill.EvanStage2Teleport,
	},
}

var EvanStage3 = Job{
	id: EvanStage3Id,
	skills: []skill.Skill{
		skill.EvanStage3LightningBolt,
		skill.EvanStage3MagicGuard,
	},
}

var EvanStage4 = Job{
	id: EvanStage4Id,
	skills: []skill.Skill{
		skill.EvanStage4IceBreath,
		skill.EvanStage4ElementalReset,
	},
}

var EvanStage5 = Job{
	id: EvanStage5Id,
	skills: []skill.Skill{
		skill.EvanStage5MagicFlare,
		skill.EvanStage5MagicShield,
	},
}

var EvanStage6 = Job{
	id: EvanStage6Id,
	skills: []skill.Skill{
		skill.EvanStage6CriticalMagic,
		skill.EvanStage6DragonThrust,
		skill.EvanStage6MagicBooster,
		skill.EvanStage6Slow,
	},
	fourthJob: true,
}

var EvanStage7 = Job{
	id: EvanStage7Id,
	skills: []skill.Skill{
		skill.EvanStage7MagicAmplification,
		skill.EvanStage7FireBreath,
		skill.EvanStage7KillerWings,
		skill.EvanStage7MagicResistance,
	},
	fourthJob: true,
}

var EvanStage8 = Job{
	id: EvanStage8Id,
	skills: []skill.Skill{
		skill.EvanStage8DragonFury,
		skill.EvanStage8Earthquake,
		skill.EvanStage8PhantomImprint,
		skill.EvanStage8RecoveryAura,
	},
	fourthJob: true,
}

var EvanStage9 = Job{
	id: EvanStage9Id,
	skills: []skill.Skill{
		skill.EvanStage9MapleWarrior,
		skill.EvanStage9MagicMastery,
		skill.EvanStage9Illusion,
		skill.EvanStage9FlameWheel,
		skill.EvanStage9HerosWill,
	},
	fourthJob: true,
}

var EvanStage10 = Job{
	id: EvanStage10Id,
	skills: []skill.Skill{
		skill.EvanStage10BlessingOfTheOnyx,
		skill.EvanStage10Blaze,
		skill.EvanStage10DarkFog,
		skill.EvanStage10SoulStone,
	},
	fourthJob: true,
}

var Jobs = map[Id]Job{
	BeginnerId:                 Beginner,
	WarriorId:                  Warrior,
	FighterId:                  Fighter,
	CrusaderId:                 Crusader,
	HeroId:                     Hero,
	PageId:                     Page,
	WhiteKnightId:              WhiteKnight,
	PaladinId:                  Paladin,
	SpearmanId:                 Spearman,
	DragonKnightId:             DragonKnight,
	DarkKnightId:               DarkKnight,
	MagicianId:                 Magician,
	FirePoisonWizardId:         FirePoisonWizard,
	FirePoisonMagicianId:       FirePoisonMagician,
	FirePoisonArchMagicianId:   FirePoisonArchMagician,
	IceLightningWizardId:       IceLightningWizard,
	IceLightningMagicianId:     IceLightningMagician,
	IceLightningArchMagicianId: IceLightningArchMagician,
	ClericId:                   Cleric,
	PriestId:                   Priest,
	BishopId:                   Bishop,
	BowmanId:                   Bowman,
	HunterId:                   Hunter,
	RangerId:                   Ranger,
	BowmasterId:                Bowmaster,
	CrossbowmanId:              Crossbowman,
	SniperId:                   Sniper,
	MarksmanId:                 Marksman,
	RogueId:                    Rogue,
	AssassinId:                 Assassin,
	HermitId:                   Hermit,
	NightLordId:                NightLord,
	BanditId:                   Bandit,
	ChiefBanditId:              ChiefBandit,
	ShadowerId:                 Shadower,
	PirateId:                   Pirate,
	BrawlerId:                  Brawler,
	MarauderId:                 Marauder,
	BuccaneerId:                Buccaneer,
	GunslingerId:               Gunslinger,
	OutlawId:                   Outlaw,
	CorsairId:                  Corsair,
	MapleLeafBrigadierId:       MapleLeafBrigadier,
	GmId:                       Gm,
	SuperGmId:                  SuperGm,
	NoblesseId:                 Noblesse,
	DawnWarriorStage1Id:        DawnWarriorStage1,
	DawnWarriorStage2Id:        DawnWarriorStage2,
	DawnWarriorStage3Id:        DawnWarriorStage3,
	DawnWarriorStage4Id:        DawnWarriorStage4,
	BlazeWizardStage1Id:        BlazeWizardStage1,
	BlazeWizardStage2Id:        BlazeWizardStage2,
	BlazeWizardStage3Id:        BlazeWizardStage3,
	BlazeWizardStage4Id:        BlazeWizardStage4,
	WindArcherStage1Id:         WindArcherStage1,
	WindArcherStage2Id:         WindArcherStage2,
	WindArcherStage3Id:         WindArcherStage3,
	WindArcherStage4Id:         WindArcherStage4,
	NightWalkerStage1Id:        NightWalkerStage1,
	NightWalkerStage2Id:        NightWalkerStage2,
	NightWalkerStage3Id:        NightWalkerStage3,
	NightWalkerStage4Id:        NightWalkerStage4,
	ThunderBreakerStage1Id:     ThunderBreakerStage1,
	ThunderBreakerStage2Id:     ThunderBreakerStage2,
	ThunderBreakerStage3Id:     ThunderBreakerStage3,
	ThunderBreakerStage4Id:     ThunderBreakerStage4,
	LegendId:                   Legend,
	AranStage1Id:               AranStage1,
	AranStage2Id:               AranStage2,
	AranStage3Id:               AranStage3,
	AranStage4Id:               AranStage4,
	EvanId:                     Evan,
	EvanStage1Id:               EvanStage1,
	EvanStage2Id:               EvanStage2,
	EvanStage3Id:               EvanStage3,
	EvanStage4Id:               EvanStage4,
	EvanStage5Id:               EvanStage5,
	EvanStage6Id:               EvanStage6,
	EvanStage7Id:               EvanStage7,
	EvanStage8Id:               EvanStage8,
	EvanStage9Id:               EvanStage9,
	EvanStage10Id:              EvanStage10,
}

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

func FromSkillId(skillId skill.Id) (Job, bool) {
	jobId := IdFromSkillId(skillId)
	j, ok := Jobs[jobId]
	return j, ok
}

func IdFromSkillId(skillId skill.Id) Id {
	return Id(math.Floor(float64(skillId) / float64(10000)))
}

func IsFourthJob(jobId Id) bool {
	j, ok := Jobs[jobId]
	if !ok {
		return false
	}
	return j.IsFourthJob()
}

func IsBeginner(jobId Id) bool {
	return IsA(jobId, BeginnerId, NoblesseId, LegendId, EvanId)
}
