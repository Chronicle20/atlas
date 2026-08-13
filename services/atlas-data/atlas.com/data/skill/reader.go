package skill

import (
	"atlas-data/skill/effect"
	"atlas-data/skill/effect/statup"
	"atlas-data/xml"
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ParseJobId derives a job id from a Skill.wz image name. The name reaches
// this helper as the root <imgdir name="…"> attribute (e.g. "112.img");
// non-numeric per-mob images like "MobSkill.img" or "BFSkill.img" return an
// error. Exported so the `job` package can share this logic rather than
// duplicate it.
func ParseJobId(filePath string) (uint32, error) {
	baseName := filepath.Base(filePath)
	if !strings.HasSuffix(baseName, ".img") {
		return 0, fmt.Errorf("file does not match expected format: %s", filePath)
	}
	idStr := strings.TrimSuffix(baseName, ".img")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[Derivation] {
	return func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[Derivation] {
		t := tenant.MustFromContext(ctx)
		return func(np model.Provider[xml.Node]) model.Provider[Derivation] {
			exml, err := np()
			if err != nil {
				return model.ErrorProvider[Derivation](err)
			}

			jobId, err := ParseJobId(exml.Name)
			if err != nil {
				return model.ErrorProvider[Derivation](err)
			}
			l.Debugf("Processing skills for job [%d].", jobId)

			ssxml, err := exml.ChildByName("skill")
			if err != nil {
				return model.ErrorProvider[Derivation](err)
			}

			ms := make([]RestModel, 0)
			stats := Stats{}
			for _, sxml := range ssxml.ChildNodes {
				skillId, err := strconv.Atoi(sxml.Name)
				if err != nil {
					return model.ErrorProvider[Derivation](err)
				}
				l.Debugf("Processing skill [%d] for job [%d].", skillId, jobId)

				m, s, err := produceSkill(l, t, jobId, skill.Id(skillId), sxml)
				if err != nil {
					return model.ErrorProvider[Derivation](err)
				}
				stats.Add(s)
				ms = append(ms, m)
			}
			l.Debugf("Derived %d skills for job [%d].", stats.Processed, jobId)
			return model.FixedProvider(Derivation{Models: ms, Stats: stats})
		}
	}
}

// isSuperGmHolySymbol reports whether skillId is this tenant version's
// SuperGM Holy Symbol skill. SuperGmHolySymbolId is a DIVERGENT wire id
// (canonical 9101002, but 5101002 at v0.48) so a raw skill.Is compare would
// silently miss it at v0.48; resolve skillId to its version-independent
// Identity first (task-187).
func isSuperGmHolySymbol(t tenant.Model, skillId skill.Id) bool {
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skillId)
	return ok && id == skill.SuperGmHolySymbol
}

// isSuperGmHealDispel reports whether skillId is this tenant version's
// SuperGM Heal + Dispel skill. SuperGmHealDispelId is a DIVERGENT wire id
// (canonical 9101000, but 5101000 at v0.48) so a raw skill.Is compare would
// silently miss it at v0.48; resolve skillId to its version-independent
// Identity first (task-187).
func isSuperGmHealDispel(t tenant.Model, skillId skill.Id) bool {
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skillId)
	return ok && id == skill.SuperGmHealDispel
}

func produceSkill(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, xml xml.Node) (RestModel, Stats, error) {
	element := readElement(xml)
	action := false
	buff := false

	skillType := xml.GetIntegerWithDefault("skillType", 0)
	if skillType == 2 {
		buff = true
	} else {
		action = hasAnyAction(xml)
		buff = getBuff(xml)

		if isCategory1(skillId) || isSuperGmHealDispel(t, skillId) {
			buff = false
		} else if skill.IsBuff(skillId) {
			buff = true
		}

	}

	stats := Stats{Processed: 1}
	es := make([]effect.RestModel, 0)
	maxLevel := uint8(0)

	// FR-1.2: COMMON WINS UNCONDITIONALLY. When `common` is present the
	// `level` subtree is not read at all. Detection is structural — never
	// gated on region/major/minor (FR-1.4).
	if common, err := xml.ChildByName("common"); err == nil {
		var commonStats Stats
		es, maxLevel, commonStats = expandCommon(l, t, jobId, skillId, buff, common)
		stats.Add(commonStats)
		stats.FromCommon = 1
	} else if level, err := xml.ChildByName("level"); err == nil {
		es = getEffects(t, skillId, buff, level.ChildNodes)
		if n := len(es); n > 255 {
			maxLevel = 255
		} else {
			maxLevel = uint8(n)
		}
		stats.FromLevel = 1
	} else {
		// FR-1.3: neither subtree. Zero effects, maxLevel 0, no panic.
		stats.Neither = 1
	}

	name, desc := "", ""
	ss, err := GetSkillStringRegistry().Get(t, strconv.Itoa(int(skillId)))
	if err == nil {
		name = ss.Name()
		desc = ss.Desc()
	}

	m := RestModel{
		Id:            uint32(skillId),
		Name:          name,
		Description:   desc,
		Action:        action,
		Element:       element,
		AnimationTime: 0,
		MaxLevel:      maxLevel,
		Effects:       es,
	}

	return m, stats, nil
}

func getEffects(t tenant.Model, skillId skill.Id, buff bool, nodes []xml.Node) []effect.RestModel {
	results := make([]effect.RestModel, 0)
	for _, node := range nodes {
		result := getEffect(t, skillId, buff, node)
		results = append(results, result)
	}
	return results
}

func getEffect(t tenant.Model, skillId skill.Id, overTime bool, node xml.Node) effect.RestModel {
	e := effect.NewModelBuilder().
		SetDuration(node.GetIntegerWithDefault("time", -1)).
		SetHp(uint16(node.GetIntegerWithDefault("hp", 0))).
		SetHPRecovery(float64(node.GetIntegerWithDefault("hpR", 0)) / 100.0).
		SetMp(uint16(node.GetIntegerWithDefault("mp", 0))).
		SetMPRecovery(float64(node.GetIntegerWithDefault("mpR", 0)) / 100.0).
		SetHPCon(uint16(node.GetIntegerWithDefault("hpCon", 0))).
		SetMPCon(uint16(node.GetIntegerWithDefault("mpCon", 0))).
		SetProp(float64(node.GetIntegerWithDefault("prop", 0)) / 100.0).
		SetCP(uint32(node.GetIntegerWithDefault("cp", 0))).
		SetCureAbnormalStatuses(getAbnormalStatuses(node)).
		SetNuffSkill(uint32(node.GetIntegerWithDefault("nuffSkill", 0))).
		SetMobCount(uint32(node.GetIntegerWithDefault("mobCount", 1))).
		SetCooldown(uint32(node.GetIntegerWithDefault("cooltime", 0))).
		SetMorphId(uint32(node.GetIntegerWithDefault("morph", 0))).
		SetGhost(uint32(node.GetIntegerWithDefault("ghost", 0))).
		SetFatigue(uint32(node.GetIntegerWithDefault("incFatigue", 0))).
		SetRepeatEffect(node.GetIntegerWithDefault("repeatEffect", 0) > 0)

	applyMobInformation(e, node)

	e.SetMob(getMob(node))
	e.SetSkill(true)
	// Why ms: the wz `time` attribute is in seconds; convert here so
	// downstream consumers (atlas-buffs, atlas-monsters) interpret
	// effect.Duration() uniformly as time.Millisecond. See task-054.
	if e.Duration() > -1 {
		e.SetDuration(e.Duration() * 1000)
		e.SetOverTime(true)
	} else {
		e.SetOverTime(overTime)
	}

	e.SetWeaponAttack(int16(node.GetIntegerWithDefault("pad", 0))).
		SetWeaponDefense(int16(node.GetIntegerWithDefault("pdd", 0))).
		SetMagicAttack(int16(node.GetIntegerWithDefault("mad", 0))).
		SetMagicDefense(int16(node.GetIntegerWithDefault("mdd", 0))).
		SetAccuracy(int16(node.GetIntegerWithDefault("acc", 0))).
		SetAvoidability(int16(node.GetIntegerWithDefault("eva", 0))).
		SetSpeed(int16(node.GetIntegerWithDefault("speed", 0))).
		SetJump(int16(node.GetIntegerWithDefault("jump", 0))).
		SetBarrier(node.GetIntegerWithDefault("barrier", 0))

	statups := make([]statup.RestModel, 0)
	statups = produceBuffStatAmount(statups, character.TemporaryStatTypeBarrier, e.Barrier())

	e.SetMapProtection(getMapProtection(item.Id(skillId)))
	// TODO does below this work right?
	statups = produceBuffStatAmount(statups, character.TemporaryStatTypeThaw, int32(e.MapProtection()))

	if e.OverTime() && getSummonMovementType(skillId) == skill.SummonMovementTypeNone {
		// TODO handle map chairs?
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWeaponAttack, int32(e.WeaponAttack()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWeaponDefense, int32(e.WeaponDefense()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMagicAttack, int32(e.MagicAttack()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMagicDefense, int32(e.MagicDefense()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeAccuracy, int32(e.Accuracy()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeAvoidability, int32(e.Avoidability()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSpeed, int32(e.Speed()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeJump, int32(e.Jump()))
	}

	ltX, ltY := node.GetPoint("lt", 0, 0)
	rbX, rbY := node.GetPoint("rb", 0, 0)
	e.SetLT(point.NewModel(point.X(int16(ltX)), point.Y(int16(ltY)))).
		SetRB(point.NewModel(point.X(int16(rbX)), point.Y(int16(rbY))))

	// DoT fields. `dot` is a raw damage-per-tick integer, forwarded unscaled.
	// `dotInterval` and `dotTime` are WZ SECONDS -- converted to milliseconds
	// HERE, the single conversion point, matching the `time` treatment above
	// (task-054). No downstream service may re-scale.
	//
	// Read here rather than in the `common`-key block below (task-192) because
	// this is the ONLY site that may scale them: that block forwards its keys
	// unscaled, and re-reading dot* there would silently undo the conversion.
	// Both blocks live in the shared getEffect, so the `common` and `level`
	// paths are covered either way. See task-200 design §2.1.
	e.SetDot(node.GetIntegerWithDefault("dot", 0)).
		SetDotInterval(node.GetIntegerWithDefault("dotInterval", 0) * 1000).
		SetDotTime(node.GetIntegerWithDefault("dotTime", 0) * 1000)

	e.SetX(int16(node.GetIntegerWithDefault("x", 0))).
		SetY(int16(node.GetIntegerWithDefault("y", 0))).
		SetDamage(uint32(node.GetIntegerWithDefault("damage", 100))).
		SetFixDamage(node.GetIntegerWithDefault("fixdamage", -1)).
		SetAttackCount(uint32(node.GetIntegerWithDefault("attackCount", 1))).
		SetBulletCount(uint16(node.GetIntegerWithDefault("bulletCount", 1))).
		SetBulletConsume(uint16(node.GetIntegerWithDefault("bulletConsume", 0))).
		SetMoneyConsume(uint32(node.GetIntegerWithDefault("moneyCon", 0))).
		SetItemConsume(uint32(node.GetIntegerWithDefault("itemCon", 0))).
		SetItemConsumeNumber(uint32(node.GetIntegerWithDefault("itemConNo", 0))).
		SetMoveTo(node.GetIntegerWithDefault("moveTo", -1))

	// Skill.wz `common` keys (task-192). Read here, in the single shared
	// getEffect, so they populate on BOTH the `common` and `level` paths
	// (FR-6.1). Absent keys default to 0, matching the `level` path's rule
	// for every key it does not find.
	e.SetRange(node.GetIntegerWithDefault("range", 0)).
		SetMastery(node.GetIntegerWithDefault("mastery", 0)).
		SetZ(node.GetIntegerWithDefault("z", 0)).
		SetCr(node.GetIntegerWithDefault("cr", 0)).
		SetDamR(node.GetIntegerWithDefault("damR", 0)).
		SetCriticaldamageMin(node.GetIntegerWithDefault("criticaldamageMin", 0)).
		SetMHPRRate(uint16(node.GetIntegerWithDefault("mhpR", 0))).
		SetV(node.GetIntegerWithDefault("v", 0)).
		SetIgnoreMobpdpR(node.GetIntegerWithDefault("ignoreMobpdpR", 0)).
		SetEpad(node.GetIntegerWithDefault("epad", 0)).
		SetW(node.GetIntegerWithDefault("w", 0)).
		SetU(node.GetIntegerWithDefault("u", 0)).
		SetEpdd(node.GetIntegerWithDefault("epdd", 0)).
		SetEmdd(node.GetIntegerWithDefault("emdd", 0)).
		SetSelfDestruction(node.GetIntegerWithDefault("selfDestruction", 0)).
		SetAsrR(node.GetIntegerWithDefault("asrR", 0)).
		SetMMPRRate(uint16(node.GetIntegerWithDefault("mmpR", 0))).
		SetT(node.GetIntegerWithDefault("t", 0)).
		SetEr(node.GetIntegerWithDefault("er", 0)).
		SetPddR(node.GetIntegerWithDefault("pddR", 0)).
		SetTerR(node.GetIntegerWithDefault("terR", 0)).
		SetMadX(node.GetIntegerWithDefault("madX", 0)).
		SetSubProp(node.GetIntegerWithDefault("subProp", 0)).
		SetEmhp(node.GetIntegerWithDefault("emhp", 0)).
		SetCriticaldamageMax(node.GetIntegerWithDefault("criticaldamageMax", 0)).
		SetExpR(node.GetIntegerWithDefault("expR", 0)).
		SetEmmp(node.GetIntegerWithDefault("emmp", 0)).
		SetConsumeItemId(node.GetIntegerWithDefault("itemConsume", 0)).
		SetMddR(node.GetIntegerWithDefault("mddR", 0)).
		SetSubTime(node.GetIntegerWithDefault("subTime", 0)).
		SetPadX(node.GetIntegerWithDefault("padX", 0)).
		SetMesoR(node.GetIntegerWithDefault("mesoR", 0))

	ms := make(map[string]uint32)

	if skill.Is(skillId, skill.BeginnerRecoveryId, skill.NoblesseRecoveryId, skill.LegendRecoveryId, skill.EvanRecoveryId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeRecovery, int32(e.X()))
	} else if skill.Is(skillId, skill.BeginnerEchoOfHeroId, skill.NoblesseEchoOfHeroId, skill.LegendEchoOfHeroId, skill.EvanEchoOfHeroId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeEchoOfHero, int32(e.X()))
	} else if mountStatups := mountStatupsForSkill(skillId, levelFromNode(node)); len(mountStatups) > 0 {
		// Mount skills. Tamed mounts (and Battleship) emit the skill id as a
		// placeholder (the channel overrides it with the equipped taming-mob
		// item id); skill-only mounts (Broomstick/Yeti/Balrog/SpaceShip) emit
		// their fixed vehicle item id, SpaceShip per-level (1932000+level).
		statups = append(statups, mountStatups...)
	} else if skill.Is(skillId, skill.BeginnerInvincibleId, skill.NoblesseInvincibleId, skill.LegendInvincibleId, skill.EvanInvincibleId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeDivineBody, 1)
	} else if skill.Is(skillId, skill.FighterPowerGuardId, skill.PagePowerGuardId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypePowerGuard, int32(e.X()))
	} else if skill.Is(skillId, skill.SpearmanHyperBodyId, skill.SuperGmHyperBodyId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeHyperBodyHP, int32(e.X()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeHyperBodyMP, int32(e.Y()))
	} else if skill.Is(skillId, skill.CrusaderComboAttackId, skill.DawnWarriorStage3ComboAttackId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeCombo, 1)
	} else if skill.Is(skillId,
		skill.WhiteKnightFlameChargeBluntWeaponId, skill.WhiteKnightBlizzardChargeBluntWeaponId, skill.WhiteKnightLightningChargeBluntWeaponId,
		skill.WhiteKnightFireChargeSwordId, skill.WhiteKnightIceChargeSwordId, skill.WhiteKnightThunderChargeSwordId,
		skill.PaladinDivineChargeBluntWeaponId, skill.PaladinHolyChargeSwordId, skill.DawnWarriorStage3SoulChargeId, skill.ThunderBreakerStage2LightningChargeId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWhiteKnightCharge, int32(e.X()))
	} else if skill.Is(skillId, skill.DragonKnightDragonBloodId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeDragonBlood, int32(e.X()))
	} else if skill.Is(skillId, skill.HeroPowerStanceId, skill.PaladinPowerStanceId, skill.DarkKnightPowerStanceId, skill.AranStage4FreezeStandingId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeStance, int32(math.Floor(e.Prop()*100)))
	} else if skill.Is(skillId, skill.DawnWarriorStage2FinalAttackSwordId, skill.WindArcherStage2FinalAttackBowId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWindBreakerFinal, int32(e.X())) // TODO this may not be right
	} else if skill.Is(skillId, skill.MagicianMagicGuardId, skill.BlazeWizardStage1MagicGuardId, skill.EvanStage3MagicGuardId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMagicGuard, int32(e.X()))
	} else if skill.Is(skillId, skill.ClericInvincibleId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeInvincible, int32(e.X()))
	} else if skill.Is(skillId, skill.PriestHolySymbolId) || isSuperGmHolySymbol(t, skillId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeHolySymbol, int32(e.X()))
	} else if skill.Is(skillId, skill.FirePoisonArchMagicianInfinityId, skill.IceLightningArchMagicianInfinityId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeInfinity, int32(e.X()))
	} else if skill.Is(skillId, skill.FirePoisonArchMagicianManaReflectionId, skill.IceLightningArchMagicianManaReflectionId, skill.BishopManaReflectionId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeManaReflection, 1)
	} else if skill.Is(skillId, skill.BishopHolyShieldId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeHolyShield, int32(e.X()))
	} else if skill.Is(skillId, skill.BlazeWizardStage2ElementalResetId, skill.EvanStage4ElementalResetId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeElementalReset, int32(e.X()))
	} else if skill.Is(skillId, skill.EvanStage5MagicShieldId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMagicShield, int32(e.X()))
	} else if skill.Is(skillId, skill.EvanStage6SlowId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMagicResist, int32(e.X()))
	} else if skill.Is(skillId, skill.PriestMysticDoorId, skill.HunterSoulArrowBowId, skill.CrossbowmanSoulArrowCrossbowId, skill.WindArcherStage2SoulArrowId) {
		// TODO this is weird right?
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSoulArrow, int32(e.X()))
	} else if skill.Is(skillId, skill.RangerPuppetId, skill.SniperPuppetId, skill.WindArcherStage3PuppetId, skill.OutlawOctopusId, skill.CorsairWrathOfTheOctopiId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypePuppet, 1)
	} else if skill.Is(skillId, skill.BowmasterConcentrateId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeConcentrate, int32(e.X()))
	} else if skill.Is(skillId, skill.BowmasterHamstringId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeHamstring, int32(e.X()))
		ms[monster.StatusSpeed] = uint32(e.X())
	} else if skill.Is(skillId, skill.MarksmanBlindId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeBlind, int32(e.X()))
		ms[monster.StatusAccuracy] = uint32(e.X())
	} else if skill.Is(skillId, skill.BowmasterSharpEyesId, skill.MarksmanSharpEyesId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSharpEyes, int32(e.X()<<8|e.Y()))
	} else if skill.Is(skillId, skill.WindArcherStage2WindWalkId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWindWalk, int32(e.X()))
	} else if skill.Is(skillId, skill.RogueDarkSightId, skill.NightWalkerStage1DarkSightId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeDarkSight, int32(e.X()))
	} else if skill.Is(skillId, skill.HermitMesoUpId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMesoUp, int32(e.X()))
	} else if skill.Is(skillId, skill.HermitShadowPartnerId, skill.NightWalkerStage3ShadowPartnerId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeShadowPartner, int32(e.X()))
	} else if skill.Is(skillId, skill.ChiefBanditMesoGuardId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMesoGuard, int32(e.X()))
	} else if skill.Is(skillId, skill.ChiefBanditPickpocketId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypePickPocket, int32(e.X()))
	} else if skill.Is(skillId, skill.NightLordShadowStarsId) {
		// The SHADOW_CLAW placeholder value MUST be nonzero: produceBuffStatAmount
		// drops the statup entirely when value == 0 (its `if value != 0` guard),
		// so a literal 0 here means SHADOW_CLAW never reaches atlas-channel at
		// all. atlas-channel overwrites this placeholder with the client-chosen
		// throwing-star item id at cast time (rewriteShadowClawStatups); the
		// value here only needs to survive the guard.
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeShadowClaw, 1)
	} else if skill.Is(skillId, skill.PirateDashId, skill.ThunderBreakerStage1DashId) {
		// TODO space dash
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeDashSpeed, int32(e.X()))
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeDashJump, int32(e.Y()))
	} else if skill.Is(skillId, skill.CorsairSpeedInfusionId, skill.BuccaneerSpeedInfusionId, skill.ThunderBreakerStage3SpeedInfusionId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSpeedInfusion, int32(e.X()))
	} else if skill.Is(skillId, skill.OutlawHomingBeaconId, skill.CorsairBullseyeId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeHomingBeacon, int32(e.X()))
	} else if skill.Is(skillId, skill.ThunderBreakerStage3SparkId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSpark, int32(e.X()))
	} else if skill.Is(skillId, skill.AranStage1PolearmBoosterId, skill.FighterAxeBoosterId, skill.FighterSwordBoosterId, skill.PageSwordBoosterId, skill.PageBluntWeaponBoosterId,
		skill.SpearmanSpearBoosterId, skill.SpearmanPolearmBoosterId, skill.HunterBowBoosterId, skill.CrossbowmanCrossbowBoosterId, skill.AssassinClawBoosterId, skill.BanditDaggerBoosterId,
		skill.FirePoisonMagicianSpellBoosterId, skill.IceLightningMagicianSpellBoosterId, skill.BrawlerKnucklerBoosterId, skill.GunslingerGunBoosterId, skill.DawnWarriorStage2SwordBoosterId,
		skill.BlazeWizardStage2SpellBoosterId, skill.WindArcherStage2BowBoosterId, skill.NightWalkerStage2ClawBoosterId, skill.ThunderBreakerStage2KnuckleBoosterId, skill.EvanStage6MagicBoosterId) {
		// TODO power explosion
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeBooster, int32(e.X()))
	} else if skill.Is(skillId, skill.HeroMapleWarriorId, skill.PaladinMapleWarriorId, skill.DarkKnightMapleWarriorId, skill.FirePoisonArchMagicianMapleWarriorId, skill.IceLightningArchMagicianMapleWarriorId,
		skill.BishopMapleWarriorId, skill.BowmasterMapleWarriorId, skill.MarksmanMapleWarriorId, skill.NightLordMapleWarriorId, skill.ShadowerMapleWarriorId, skill.CorsairMapleWarriorId, skill.BuccaneerMapleWarriorId,
		skill.AranStage4MapleWarriorId, skill.EvanStage9MapleWarriorId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMapleWarrior, int32(e.X()))
	} else if skill.Is(skillId, skill.RangerSilverHawkId, skill.SniperGoldenEagleId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSummon, 1)
		ms[monster.StatusStun] = 1
	} else if skill.Is(skillId, skill.FirePoisonArchMagicianElquinesId, skill.MarksmanFrostpreyId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSummon, 1)
		ms[monster.StatusFreeze] = 1
	} else if skill.Is(skillId, skill.PriestSummonDragonId, skill.BowmasterPhoenixId, skill.IceLightningArchMagicianIfritId, skill.BishopBahamutId, skill.DarkKnightBeholderId, skill.OutlawGaviotaId,
		skill.DawnWarriorStage1SoulId, skill.BlazeWizardStage1FlameId, skill.WindArcherStage1StormId, skill.NightWalkerStage1DarknessId, skill.ThunderBreakerStage1LightningSpriteId, skill.BlazeWizardStage3IfritId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSummon, 1)
	} else if skill.Is(skillId, skill.CrusaderArmorCrashId, skill.DragonKnightPowerCrashId, skill.WhiteKnightMagicCrashId) {
		ms[monster.StatusSeal] = 1
	} else if skill.Is(skillId, skill.RogueDisorderId, skill.PageThreatenId) {
		ms[monster.StatusWeaponAttack] = uint32(e.X())
		ms[monster.StatusWeaponDefense] = uint32(e.X())
	} else if skill.Is(skillId, skill.CorsairHypnotizeId) {
		ms[monster.StatusInertMob] = 1
	} else if skill.Is(skillId, skill.NightLordNinjaAmbushId, skill.ShadowerNinjaAmbushId) {
		ms[monster.StatusNinjaAmbush] = e.Damage()
	} else if skill.Is(skillId, skill.DragonKnightDragonRoarId) {
		e.SetHPRecovery(float64(-e.X()) / 100.0)
		ms[monster.StatusStun] = 1
	} else if skill.Is(skillId, skill.CrusaderComaAxeId, skill.CrusaderComaSwordId, skill.CrusaderShoutId, skill.WhiteKnightChargedBlowId, skill.HunterArrowBombBowId,
		skill.ChiefBanditAssaulterId, skill.ShadowerBoomerangStepId, skill.BrawlerBackspinBlowId, skill.BrawlerDoubleUppercutId, skill.BuccaneerDemolitionId, skill.BuccaneerSnatchId,
		skill.BuccaneerBarrageId, skill.GunslingerBlankShotId, skill.DawnWarriorStage3ComaId, skill.ThunderBreakerStage3BarrageId, skill.AranStage3RollingSpinId,
		skill.EvanStage7FireBreathId, skill.EvanStage10BlazeId) {
		ms[monster.StatusStun] = 1
	} else if skill.Is(skillId, skill.NightLordTauntId, skill.ShadowerTauntId) {
		ms[monster.StatusShowdown] = uint32(e.X())
		ms[monster.StatusMagicDefense] = uint32(e.X())
		ms[monster.StatusWeaponDefense] = uint32(e.X())
	} else if skill.Is(skillId, skill.IceLightningWizardColdBeamId, skill.IceLightningMagicianIceStrikeId, skill.IceLightningArchMagicianBlizzardId, skill.IceLightningMagicianElementCompositionId, skill.SniperBlizzardId,
		skill.OutlawIceSplitterId, skill.FirePoisonArchMagicianParalyzeId, skill.AranStage4ComboTempestId, skill.EvanStage4IceBreathId) {
		ms[monster.StatusFreeze] = 1
		e.SetDuration(e.Duration() * 2)
	} else if skill.Is(skillId, skill.FirePoisonWizardSlowId, skill.IceLightningWizardSlowId, skill.BlazeWizardStage2SlowId) {
		ms[monster.StatusSpeed] = uint32(e.X())
	} else if skill.Is(skillId, skill.FirePoisonWizardPoisonBreathId, skill.FirePoisonMagicianElementCompositionId) {
		ms[monster.StatusPoison] = 1
	} else if skill.Is(skillId, skill.PriestDoomId) {
		ms[monster.StatusDoom] = 1
	} else if skill.Is(skillId, skill.IceLightningMagicianSealId, skill.FirePoisonMagicianSealId, skill.BlazeWizardStage3SealId) {
		ms[monster.StatusSeal] = 1
	} else if skill.Is(skillId, skill.HermitShadowWebId, skill.NightWalkerStage3ShadowWebId) {
		ms[monster.StatusShadowWeb] = 1
	} else if skill.Is(skillId, skill.FirePoisonArchMagicianFireDemonId, skill.IceLightningArchMagicianIceDemonId) {
		ms[monster.StatusPoison] = 1
		ms[monster.StatusFreeze] = 1
	} else if skill.Is(skillId, skill.EvanStage8PhantomImprintId) {
		ms[monster.StatusPhantomImprint] = uint32(e.X())
	} else if skill.Is(skillId, skill.AranStage1ComboAbilityId, skill.LegendComboAbilityId) {
		// ARAN_COMBO is a damage-calculation input decoded by the client as a
		// signed short (SecondaryStat::DecodeForLocal), NOT the combo count --
		// the count is delivered by SHOW_COMBO and never touches this stat
		// (task-217 design.md §2.3). Combo Ability's WZ node carries no field
		// whose value is 100; x is the level-scaled magnitude, matching every
		// sibling Aran statup below.
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeAranCombo, int32(e.X()))
	} else if skill.Is(skillId, skill.AranStage4ComboBarrierId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeComboBarrier, int32(e.X()))
	} else if skill.Is(skillId, skill.AranStage2ComboDrainId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeComboDrain, int32(e.X()))
	} else if skill.Is(skillId, skill.AranStage3SmartKnockbackId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeSmartKnockBack, int32(e.X()))
	} else if skill.Is(skillId, skill.AranStage2BodyPressureId) {
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeBodyPressure, int32(e.X()))
	} else if skill.Is(skillId, skill.AranStage3SnowChargeId) {
		// TODO(post-task-054): SnowCharge passes Duration as the
		// WhiteKnightCharge stat amount. After task-054 this is 1000x
		// larger (now ms, was raw seconds). The right fix is to pass
		// a charge-amount field (likely e.X()), not Duration. Tracked
		// in docs/TODO.md.
		statups = produceBuffStatAmount(statups, character.TemporaryStatTypeWhiteKnightCharge, e.Duration())
	}

	statups = produceBuffStatAmount(statups, character.TemporaryStatTypeMorph, int32(e.MorphId()))
	m := e.SetMonsterStatus(ms).
		SetStatups(statups).
		Build()
	// Transform is a pure struct copy over an already-built Model; it never
	// errors in practice (mirrors the no-op-error Transform convention used
	// elsewhere, e.g. atlas-query-aggregator's transport.Transform).
	rm, _ := effect.Transform(m)
	return rm
}

func getMob(node xml.Node) uint32 {
	c, err := node.ChildByName("mob")
	if err != nil || len(c.ChildNodes) <= 0 {
		return 0
	}
	return uint32(c.GetIntegerWithDefault("mob", 0))
}

func applyMobInformation(e *effect.ModelBuilder, node xml.Node) {
	c, err := node.ChildByName("0")
	if err == nil && len(c.ChildNodes) > 0 {
		e.SetMobSkill(uint16(node.GetIntegerWithDefault("mobSkill", 0))).
			SetMobSkillLevel(uint16(node.GetIntegerWithDefault("level", 0))).
			SetTarget(uint32(node.GetIntegerWithDefault("target", 0)))
	}
}

func getAbnormalStatuses(node xml.Node) []string {
	statuses := make([]string, 0)
	if node.GetIntegerWithDefault("poison", 0) > 0 {
		statuses = append(statuses, "POISON")
	}
	if node.GetIntegerWithDefault("seal", 0) > 0 {
		statuses = append(statuses, "SEAL")
	}
	if node.GetIntegerWithDefault("darkness", 0) > 0 {
		statuses = append(statuses, "DARKNESS")
	}
	if node.GetIntegerWithDefault("weakness", 0) > 0 {
		statuses = append(statuses, "WEAKEN", "SLOW")
	}
	if node.GetIntegerWithDefault("curse", 0) > 0 {
		statuses = append(statuses, "CURSE")
	}
	return statuses
}

func getMapProtection(sourceId item.Id) byte {
	if item.Is(sourceId, item.UseRedBeanPorridge, item.UseSoftWhiteBun) {
		return 1 // elnath cold
	} else if sourceId == item.UseAirBubble {
		return 2 // aqua road underwater
	} else {
		return 0
	}
}

func produceBuffStatAmount(existing []statup.RestModel, buff character.TemporaryStatType, value int32) []statup.RestModel {
	if value != 0 {
		return append(existing, statup.RestModel{
			Type:   string(buff),
			Amount: value,
		})
	}
	return existing
}

// levelFromNode parses the per-level effect node's name (e.g. "1", "2") to its
// integer level. Each child of the WZ "level" imgdir is named by its level
// number; SkillOnlyMountVehicleId needs this for the per-level SpaceShip id.
// Returns 0 when the name is not a number (level-independent mounts ignore it).
func levelFromNode(node xml.Node) int {
	level, err := strconv.Atoi(node.Name)
	if err != nil {
		return 0
	}
	return level
}

// mountStatupsForSkill returns the MONSTER_RIDING statups produced for a mount
// skill at the given level. Tamed mounts (and Battleship) emit the skill id as
// a placeholder (the channel overrides it with the equipped taming-mob id);
// skill-only mounts emit their fixed vehicle item id (SpaceShip per-level).
// Non-mount skills produce no statups.
func mountStatupsForSkill(skillId skill.Id, level int) []statup.RestModel {
	statups := make([]statup.RestModel, 0)
	if skill.Is(skillId, skill.BeginnerMonsterRidingId, skill.NoblesseMonsterRidingId, skill.LegendMonsterRidingId, skill.EvanMonsterRidingId, skill.CorsairBattleshipId) {
		return produceBuffStatAmount(statups, character.TemporaryStatTypeMonsterRiding, int32(skillId))
	}
	if vid, ok := skill.SkillOnlyMountVehicleId(skillId, level); ok {
		return produceBuffStatAmount(statups, character.TemporaryStatTypeMonsterRiding, vid)
	}
	return statups
}

// findStatup returns the first statup matching the given type.
func findStatup(statups []statup.RestModel, statType string) (statup.RestModel, bool) {
	for _, s := range statups {
		if s.Type == statType {
			return s, true
		}
	}
	return statup.RestModel{}, false
}

// TODO find better name for this
func isCategory1(id skill.Id) bool {
	return skill.Is(id,
		skill.HeroRushId, skill.PaladinRushId, skill.DarkKnightRushId, skill.DragonKnightSacrificeId, skill.HeroMonsterMagnetId, skill.PaladinMonsterMagnetId, skill.DarkKnightMonsterMagnetId,
		skill.FirePoisonMagicianExplosionId, skill.FirePoisonMagicianPoisonMistId, skill.ClericHealId,
		skill.RangerMortalBlowId, skill.SniperMortalBlowId,
		skill.AssassinDrainId, skill.HermitShadowWebId, skill.BanditStealId, skill.ShadowerSmokescreenId, skill.ChiefBanditChakraId,
		skill.GunslingerRecoilShotId, skill.MarauderEnergyDrainId,
		skill.AranStage1CombatStepId,
		skill.EvanStage4IceBreathId, skill.EvanStage7FireBreathId, skill.EvanStage8RecoveryAuraId,
		skill.BlazeWizardStage3FlameGearId, skill.NightWalkerStage3ShadowWebId, skill.NightWalkerStage3PoisonBombId, skill.NightWalkerStage2VampireId,
	)
}

func getBuff(exml xml.Node) bool {
	buff := hasEffect(exml) && noHit(exml) && noBall(exml)
	if buff {
		return true
	}

	a, err := exml.ChildByName("action")
	if err != nil {
		return false
	}
	astr := a.GetString("0", "")
	return astr == "alert2"
}

func noBall(exml xml.Node) bool {
	_, err := exml.ChildByName("ball")
	return err != nil
}

func noHit(exml xml.Node) bool {
	_, err := exml.ChildByName("hit")
	return err != nil
}

func hasEffect(exml xml.Node) bool {
	_, err := exml.ChildByName("effect")
	return err == nil
}

func hasAnyAction(exml xml.Node) bool {
	_, err := exml.ChildByName("action")
	if err == nil {
		return true
	}
	_, err = exml.ChildByName("prepare/action")
	if err == nil {
		return true
	}
	skillId, err := strconv.Atoi(exml.Name)
	if err != nil {
		return false
	}
	return actionBySkill(skill.Id(skillId))
}

func actionBySkill(skillId skill.Id) bool {
	switch skillId {
	case skill.GunslingerInvisibleShotId, skill.CorsairHypnotizeId:
		return true
	default:
		return false
	}
}

func readElement(node xml.Node) string {
	i := node.GetString("elemAttr", "P")
	r, err := ElementFromChar(i)
	if err != nil {
		return ElementNeutral
	}
	return r
}

func getSummonMovementType(skillId skill.Id) int8 {
	if skill.Is(skillId, skill.RangerPuppetId, skill.SniperPuppetId, skill.WindArcherStage3PuppetId, skill.OutlawOctopusId, skill.CorsairWrathOfTheOctopiId) {
		return skill.SummonMovementTypeStationary
	}
	if skill.Is(skillId, skill.RangerSilverHawkId, skill.SniperGoldenEagleId, skill.PriestSummonDragonId, skill.MarksmanFrostpreyId, skill.BowmasterPhoenixId, skill.OutlawGaviotaId) {
		return skill.SummonMovementTypeCircleFollow
	}
	if skill.Is(skillId, skill.DarkKnightBeholderId, skill.FirePoisonArchMagicianElquinesId, skill.IceLightningArchMagicianIfritId, skill.BishopBahamutId, skill.DawnWarriorStage1SoulId, skill.BlazeWizardStage1FlameId, skill.WindArcherStage1StormId, skill.NightWalkerStage1DarknessId, skill.ThunderBreakerStage1LightningSpriteId) {
		return skill.SummonMovementTypeFollow
	}
	return skill.SummonMovementTypeNone
}

const (
	ElementNeutral  string = "NEUTRAL"
	ElementPhysical string = "PHYSICAL"
	ElementFire     string = "FIRE"
	ElementIce      string = "ICE"
	ElementLighting string = "LIGHTING"
	ElementPoison   string = "POISON"
	ElementHoly     string = "HOLY"
	ElementDarkness string = "DARKNESS"
)

func ElementFromChar(char string) (string, error) {
	switch strings.ToUpper(char) {
	case "F":
		return ElementFire, nil
	case "I":
		return ElementIce, nil
	case "L":
		return ElementLighting, nil
	case "S":
		return ElementPoison, nil
	case "H":
		return ElementHoly, nil
	case "D":
		return ElementDarkness, nil
	case "P":
		return ElementNeutral, nil
	}
	return "", errors.New("unknown element")
}
