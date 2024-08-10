package model

import (
	"atlas-channel/tenant"
	"atlas-channel/tool"
	"errors"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"time"
)

type CharacterTemporaryStatTypeName string

const (
	CharacterTemporaryStatTypeNameWeaponAttack           CharacterTemporaryStatTypeName = "WEAPON_ATTACK"
	CharacterTemporaryStatTypeNameWeaponDefense          CharacterTemporaryStatTypeName = "WEAPON_DEFENSE"
	CharacterTemporaryStatTypeNameMagicAttack            CharacterTemporaryStatTypeName = "MAGIC_ATTACK"
	CharacterTemporaryStatTypeNameMagicDefense           CharacterTemporaryStatTypeName = "MAGIC_DEFENSE"
	CharacterTemporaryStatTypeNameAccuracy               CharacterTemporaryStatTypeName = "ACCURACY"
	CharacterTemporaryStatTypeNameAvoidability           CharacterTemporaryStatTypeName = "AVOIDABILITY"
	CharacterTemporaryStatTypeNameHands                  CharacterTemporaryStatTypeName = "HANDS"
	CharacterTemporaryStatTypeNameSpeed                  CharacterTemporaryStatTypeName = "SPEED"
	CharacterTemporaryStatTypeNameJump                   CharacterTemporaryStatTypeName = "JUMP"
	CharacterTemporaryStatTypeNameMagicGuard             CharacterTemporaryStatTypeName = "MAGIC_GUARD"
	CharacterTemporaryStatTypeNameDarkSight              CharacterTemporaryStatTypeName = "DARK_SIGHT"
	CharacterTemporaryStatTypeNameBooster                CharacterTemporaryStatTypeName = "BOOSTER"
	CharacterTemporaryStatTypeNamePowerGuard             CharacterTemporaryStatTypeName = "POWER_GUARD"
	CharacterTemporaryStatTypeNameHyperBodyHP            CharacterTemporaryStatTypeName = "HYPER_BODY_HP"
	CharacterTemporaryStatTypeNameHyperBodyMP            CharacterTemporaryStatTypeName = "HYPER_BODY_MP"
	CharacterTemporaryStatTypeNameInvincible             CharacterTemporaryStatTypeName = "INVINCIBLE"
	CharacterTemporaryStatTypeNameSoulArrow              CharacterTemporaryStatTypeName = "SOUL_ARROW"
	CharacterTemporaryStatTypeNameStun                   CharacterTemporaryStatTypeName = "STUN"
	CharacterTemporaryStatTypeNamePoison                 CharacterTemporaryStatTypeName = "POISON"
	CharacterTemporaryStatTypeNameSeal                   CharacterTemporaryStatTypeName = "SEAL"
	CharacterTemporaryStatTypeNameDarkness               CharacterTemporaryStatTypeName = "DARKNESS"
	CharacterTemporaryStatTypeNameCombo                  CharacterTemporaryStatTypeName = "COMBO"
	CharacterTemporaryStatTypeNameWhiteKnightCharge      CharacterTemporaryStatTypeName = "WHITE_KNIGHT_CHARGE"
	CharacterTemporaryStatTypeNameDragonBlood            CharacterTemporaryStatTypeName = "DRAGON_BLOOD"
	CharacterTemporaryStatTypeNameHolySymbol             CharacterTemporaryStatTypeName = "HOLY_SYMBOL"
	CharacterTemporaryStatTypeNameMesoUp                 CharacterTemporaryStatTypeName = "MESO_UP"
	CharacterTemporaryStatTypeNameShadowPartner          CharacterTemporaryStatTypeName = "SHADOW_PARTNER"
	CharacterTemporaryStatTypeNamePickPocket             CharacterTemporaryStatTypeName = "PICK_POCKET"
	CharacterTemporaryStatTypeNameMesoGuard              CharacterTemporaryStatTypeName = "MESO_GUARD"
	CharacterTemporaryStatTypeNameThaw                   CharacterTemporaryStatTypeName = "THAW"
	CharacterTemporaryStatTypeNameWeaken                 CharacterTemporaryStatTypeName = "WEAKEN"
	CharacterTemporaryStatTypeNameCurse                  CharacterTemporaryStatTypeName = "CURSE"
	CharacterTemporaryStatTypeNameSlow                   CharacterTemporaryStatTypeName = "SLOW"
	CharacterTemporaryStatTypeNameMorph                  CharacterTemporaryStatTypeName = "MORPH"
	CharacterTemporaryStatTypeNameRecovery               CharacterTemporaryStatTypeName = "RECOVERY"
	CharacterTemporaryStatTypeNameMapleWarrior           CharacterTemporaryStatTypeName = "MAPLE_WARRIOR"
	CharacterTemporaryStatTypeNameStance                 CharacterTemporaryStatTypeName = "STANCE"
	CharacterTemporaryStatTypeNameSharpEyes              CharacterTemporaryStatTypeName = "SHARP_EYES"
	CharacterTemporaryStatTypeNameManaReflection         CharacterTemporaryStatTypeName = "MANA_REFLECTION"
	CharacterTemporaryStatTypeNameSeduce                 CharacterTemporaryStatTypeName = "SEDUCE"
	CharacterTemporaryStatTypeNameShadowClaw             CharacterTemporaryStatTypeName = "SHADOW_CLAW"
	CharacterTemporaryStatTypeNameInfinity               CharacterTemporaryStatTypeName = "INFINITY"
	CharacterTemporaryStatTypeNameHolyShield             CharacterTemporaryStatTypeName = "HOLY_SHIELD"
	CharacterTemporaryStatTypeNameHamstring              CharacterTemporaryStatTypeName = "HAMSTRING"
	CharacterTemporaryStatTypeNameBlind                  CharacterTemporaryStatTypeName = "BLIND"
	CharacterTemporaryStatTypeNameConcentrate            CharacterTemporaryStatTypeName = "CONCENTRATE"
	CharacterTemporaryStatTypeNameBanMap                 CharacterTemporaryStatTypeName = "BAN_MAP"
	CharacterTemporaryStatTypeNameEchoOfHero             CharacterTemporaryStatTypeName = "ECHO_OF_HERO"
	CharacterTemporaryStatTypeNameMesoUpByItem           CharacterTemporaryStatTypeName = "MESO_UP_BY_ITEM"
	CharacterTemporaryStatTypeNameGhostMorph             CharacterTemporaryStatTypeName = "GHOST_MORPH"
	CharacterTemporaryStatTypeNameBarrier                CharacterTemporaryStatTypeName = "BARRIER"
	CharacterTemporaryStatTypeNameConfuse                CharacterTemporaryStatTypeName = "CONFUSE"
	CharacterTemporaryStatTypeNameItemUpByItem           CharacterTemporaryStatTypeName = "ITEM_UP_BY_ITEM"
	CharacterTemporaryStatTypeNameRespectPImmune         CharacterTemporaryStatTypeName = "RESPECT_PIMMUNE"
	CharacterTemporaryStatTypeNameRespectMImmune         CharacterTemporaryStatTypeName = "RESPECT_MIMMUNE"
	CharacterTemporaryStatTypeNameDefenseAttack          CharacterTemporaryStatTypeName = "DEFENSE_ATTACK"
	CharacterTemporaryStatTypeNameDefenseState           CharacterTemporaryStatTypeName = "DEFENSE_STATE"
	CharacterTemporaryStatTypeNameIncreaseEffectHpPotion CharacterTemporaryStatTypeName = "INCREASE_EFFECT_HP_POTION"
	CharacterTemporaryStatTypeNameIncreaseEffectMpPotion CharacterTemporaryStatTypeName = "INCREASE_EFFECT_MP_POTION"
	CharacterTemporaryStatTypeNameBerserkFury            CharacterTemporaryStatTypeName = "BERSERK_FURY"
	CharacterTemporaryStatTypeNameDivineBody             CharacterTemporaryStatTypeName = "DIVINE_BODY"
	CharacterTemporaryStatTypeNameSpark                  CharacterTemporaryStatTypeName = "SPARK"
	CharacterTemporaryStatTypeNameDojangShield           CharacterTemporaryStatTypeName = "DOJANG_SHIELD"
	CharacterTemporaryStatTypeNameSoulMasterFinal        CharacterTemporaryStatTypeName = "SOUL_MASTER_FINAL"
	CharacterTemporaryStatTypeNameWindBreakerFinal       CharacterTemporaryStatTypeName = "WIND_BREAKER_FINAL"
	CharacterTemporaryStatTypeNameElementalReset         CharacterTemporaryStatTypeName = "ELEMENTAL_RESET"
	CharacterTemporaryStatTypeNameWindWalk               CharacterTemporaryStatTypeName = "WIND_WALK"
	CharacterTemporaryStatTypeNameEventRate              CharacterTemporaryStatTypeName = "EVENT_RATE"
	CharacterTemporaryStatTypeNameAranCombo              CharacterTemporaryStatTypeName = "ARAN_COMBO"
	CharacterTemporaryStatTypeNameComboDrain             CharacterTemporaryStatTypeName = "COMBO_DRAIN"
	CharacterTemporaryStatTypeNameComboBarrier           CharacterTemporaryStatTypeName = "COMBO_BARRIER"
	CharacterTemporaryStatTypeNameBodyPressure           CharacterTemporaryStatTypeName = "BODY_PRESSURE"
	CharacterTemporaryStatTypeNameSmartKnockBack         CharacterTemporaryStatTypeName = "SMART_KNOCK_BACK"
	CharacterTemporaryStatTypeNameRepeatEffect           CharacterTemporaryStatTypeName = "REPEAT_EFFECT"
	CharacterTemporaryStatTypeNameExpBuffRate            CharacterTemporaryStatTypeName = "EXP_BUFF_RATE"
	CharacterTemporaryStatTypeNameStopPortion            CharacterTemporaryStatTypeName = "STOP_PORTION"
	CharacterTemporaryStatTypeNameStopMotion             CharacterTemporaryStatTypeName = "STOP_MOTION"
	CharacterTemporaryStatTypeNameFear                   CharacterTemporaryStatTypeName = "FEAR"
	CharacterTemporaryStatTypeNameEvanSlow               CharacterTemporaryStatTypeName = "EVAN_SLOW"
	CharacterTemporaryStatTypeNameMagicShield            CharacterTemporaryStatTypeName = "MAGIC_SHIELD"
	CharacterTemporaryStatTypeNameMagicResist            CharacterTemporaryStatTypeName = "MAGIC_RESIST"
	CharacterTemporaryStatTypeNameSoulStone              CharacterTemporaryStatTypeName = "SOUL_STONE"
	CharacterTemporaryStatTypeNameFlying                 CharacterTemporaryStatTypeName = "FLYING"
	CharacterTemporaryStatTypeNameFrozen                 CharacterTemporaryStatTypeName = "FROZEN"
	CharacterTemporaryStatTypeNameAssistCharge           CharacterTemporaryStatTypeName = "ASSIST_CHARGE"
	CharacterTemporaryStatTypeNameMirrorImage            CharacterTemporaryStatTypeName = "MIRROR_IMAGE"
	CharacterTemporaryStatTypeNameSuddenDeath            CharacterTemporaryStatTypeName = "SUDDEN_DEATH"
	CharacterTemporaryStatTypeNameNotDamaged             CharacterTemporaryStatTypeName = "NOT_DAMAGED"
	CharacterTemporaryStatTypeNameFinalCut               CharacterTemporaryStatTypeName = "FINAL_CUT"
	CharacterTemporaryStatTypeNameThornsEffect           CharacterTemporaryStatTypeName = "THORNS_EFFECT"
	CharacterTemporaryStatTypeNameSwallowAttackDamage    CharacterTemporaryStatTypeName = "SWALLOW_ATTACK_DAMAGE"
	CharacterTemporaryStatTypeNameWildDamageUp           CharacterTemporaryStatTypeName = "WILD_DAMAGE_UP"
	CharacterTemporaryStatTypeNameMine                   CharacterTemporaryStatTypeName = "MINE"
	CharacterTemporaryStatTypeNameEMHP                   CharacterTemporaryStatTypeName = "EMHP"
	CharacterTemporaryStatTypeNameEMMP                   CharacterTemporaryStatTypeName = "EMMP"
	CharacterTemporaryStatTypeNameEPAD                   CharacterTemporaryStatTypeName = "EPAD"
	CharacterTemporaryStatTypeNameEPPD                   CharacterTemporaryStatTypeName = "EPPD"
	CharacterTemporaryStatTypeNameEMDD                   CharacterTemporaryStatTypeName = "EMDD"
	CharacterTemporaryStatTypeNameGuard                  CharacterTemporaryStatTypeName = "GUARD"
	CharacterTemporaryStatTypeNameSafetyDamage           CharacterTemporaryStatTypeName = "SAFETY_DAMAGE"
	CharacterTemporaryStatTypeNameSafetyAbsorb           CharacterTemporaryStatTypeName = "SAFETY_ABSORB"
	CharacterTemporaryStatTypeNameCyclone                CharacterTemporaryStatTypeName = "CYCLONE"
	CharacterTemporaryStatTypeNameSwallowCritical        CharacterTemporaryStatTypeName = "SWALLOW_CRITICAL"
	CharacterTemporaryStatTypeNameSwallowMaxMP           CharacterTemporaryStatTypeName = "SWALLOW_MAX_MP"
	CharacterTemporaryStatTypeNameSwallowDefense         CharacterTemporaryStatTypeName = "SWALLOW_DEFENSE"
	CharacterTemporaryStatTypeNameSwallowEvasion         CharacterTemporaryStatTypeName = "SWALLOW_EVASION"
	CharacterTemporaryStatTypeNameConversion             CharacterTemporaryStatTypeName = "CONVERSION"
	CharacterTemporaryStatTypeNameRevive                 CharacterTemporaryStatTypeName = "REVIEVE"
	CharacterTemporaryStatTypeNameSneak                  CharacterTemporaryStatTypeName = "SNEAK"
	CharacterTemporaryStatTypeNameUnknown                CharacterTemporaryStatTypeName = "UNKNOWN"
	CharacterTemporaryStatTypeNameEnergyCharge           CharacterTemporaryStatTypeName = "ENERGY_CHARGE"
	CharacterTemporaryStatTypeNameDashSpeed              CharacterTemporaryStatTypeName = "DASH_SPEED"
	CharacterTemporaryStatTypeNameDashJump               CharacterTemporaryStatTypeName = "DASH_JUMP"
	CharacterTemporaryStatTypeNameMonsterRiding          CharacterTemporaryStatTypeName = "MONSTER_RIDING"
	CharacterTemporaryStatTypeNameSpeedInfusion          CharacterTemporaryStatTypeName = "SPEED_INFUSION"
	CharacterTemporaryStatTypeNameHomingBeacon           CharacterTemporaryStatTypeName = "HOMING_BEACON"
	CharacterTemporaryStatTypeNameUndead                 CharacterTemporaryStatTypeName = "UNDEAD"
	CharacterTemporaryStatTypeNameSummon                 CharacterTemporaryStatTypeName = "SUMMON"
	CharacterTemporaryStatTypeNamePuppet                 CharacterTemporaryStatTypeName = "PUPPET"
)

type CharacterTemporaryStatType struct {
	shift   uint
	mask    tool.Uint128
	disease bool
}

func NewCharacterTemporaryStatType(shift uint, disease bool) CharacterTemporaryStatType {
	mask := tool.Uint128{L: 1}.ShiftLeft(shift)
	return CharacterTemporaryStatType{
		shift:   shift,
		mask:    mask,
		disease: disease,
	}
}

func CharacterTemporaryStatTypeByName(tenant tenant.Model) func(name CharacterTemporaryStatTypeName) (CharacterTemporaryStatType, error) {
	var shift uint = 0
	set := make(map[CharacterTemporaryStatTypeName]CharacterTemporaryStatType)

	funcCallNewAndInc := func(disease bool) func(name CharacterTemporaryStatTypeName) {
		return func(name CharacterTemporaryStatTypeName) {
			set[name] = NewCharacterTemporaryStatType(shift, disease)
			shift += 1
		}
	}
	newAndIncDiseased := funcCallNewAndInc(true)
	newAndIncNonDiseased := funcCallNewAndInc(false)

	newAndIncNonDiseased(CharacterTemporaryStatTypeNameWeaponAttack)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameWeaponDefense)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMagicAttack)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMagicDefense)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameAccuracy)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameAvoidability)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameHands)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSpeed)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameJump)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMagicGuard)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDarkSight)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameBooster)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNamePowerGuard)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameHyperBodyHP)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameHyperBodyMP)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameInvincible)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSoulArrow)
	newAndIncDiseased(CharacterTemporaryStatTypeNameStun)
	newAndIncDiseased(CharacterTemporaryStatTypeNamePoison)
	newAndIncDiseased(CharacterTemporaryStatTypeNameSeal)
	newAndIncDiseased(CharacterTemporaryStatTypeNameDarkness)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameCombo)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameWhiteKnightCharge)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDragonBlood)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameHolySymbol)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMesoUp)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameShadowPartner)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNamePickPocket)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMesoGuard)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameThaw)
	newAndIncDiseased(CharacterTemporaryStatTypeNameWeaken)
	newAndIncDiseased(CharacterTemporaryStatTypeNameCurse)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSlow)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMorph)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameRecovery)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMapleWarrior)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameStance)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSharpEyes)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameManaReflection)
	newAndIncDiseased(CharacterTemporaryStatTypeNameSeduce)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameShadowClaw)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameInfinity)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameHolyShield)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameHamstring)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameBlind)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameConcentrate)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameBanMap)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameEchoOfHero)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMesoUpByItem)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameGhostMorph)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameBarrier)
	newAndIncDiseased(CharacterTemporaryStatTypeNameConfuse)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameItemUpByItem)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameRespectPImmune)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameRespectMImmune)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDefenseAttack)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDefenseState)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameIncreaseEffectHpPotion)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameIncreaseEffectMpPotion)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameBerserkFury)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDivineBody)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSpark)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDojangShield)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSoulMasterFinal)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameWindBreakerFinal)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameElementalReset)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameWindWalk)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameEventRate)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameAranCombo)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameComboDrain)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameComboBarrier)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameBodyPressure)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSmartKnockBack)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameRepeatEffect)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameExpBuffRate)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameStopPortion)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameStopMotion)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameFear)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameEvanSlow)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMagicShield)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMagicResist)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSoulStone)
	if (tenant.Region == "GMS" && tenant.MajorVersion > 83) || tenant.Region == "JMS" {
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameFlying)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameFrozen)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameAssistCharge)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameMirrorImage)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSuddenDeath)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameNotDamaged)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameFinalCut)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameThornsEffect)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSwallowAttackDamage)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameWildDamageUp)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameMine)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameEMHP)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameEMMP)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameEPAD)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameEPPD)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameEMDD)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameGuard)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSafetyDamage)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSafetyAbsorb)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameCyclone)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSwallowCritical)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSwallowMaxMP)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSwallowDefense)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSwallowEvasion)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameConversion)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameRevive)
		newAndIncNonDiseased(CharacterTemporaryStatTypeNameSneak)

		newAndIncNonDiseased(CharacterTemporaryStatTypeNameUnknown)
	}
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameEnergyCharge)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDashSpeed)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameDashJump)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameMonsterRiding)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameSpeedInfusion)
	newAndIncNonDiseased(CharacterTemporaryStatTypeNameHomingBeacon)
	newAndIncDiseased(CharacterTemporaryStatTypeNameUndead)

	return func(name CharacterTemporaryStatTypeName) (CharacterTemporaryStatType, error) {
		if val, ok := set[name]; ok {
			return val, nil
		}
		return CharacterTemporaryStatType{}, errors.New("character temporary stat type not found")
	}
}

type CharacterTemporaryStatValue struct {
}

type CharacterTemporaryStatBase struct {
	bDynamicTermSet bool
	nOption         int32
	rOption         int32
	tLastUpdated    int64
	usExpireItem    int16
}

func NewCharacterTemporaryStatBase(bDynamicTermSet bool) CharacterTemporaryStatBase {
	return CharacterTemporaryStatBase{
		tLastUpdated:    time.Now().Unix(),
		bDynamicTermSet: bDynamicTermSet,
	}
}

func writeTime(t int64) func(w *response.Writer) {
	return func(w *response.Writer) {
		cur := time.Now().Unix()
		interval := false
		if t >= cur {
			t -= cur
		} else {
			interval = true
			t = cur - t
		}
		t /= 1000
		w.WriteBool(interval)
		w.WriteInt32(int32(t))
	}
}

func (m CharacterTemporaryStatBase) Encode(_ logrus.FieldLogger, t tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt32(m.nOption)
		w.WriteInt32(m.rOption)
		writeTime(m.tLastUpdated)(w)
		if m.bDynamicTermSet {
			w.WriteInt16(m.usExpireItem)
		}
	}
}

type SpeedInfusionTemporaryStat struct {
	CharacterTemporaryStatBase
	tCurrentTime int32
}

func (m SpeedInfusionTemporaryStat) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.CharacterTemporaryStatBase.Encode(l, t, options)(w)
		writeTime(int64(m.tCurrentTime))(w)
		w.WriteInt16(m.usExpireItem)
	}
}

func NewSpeedInfusionTemporaryStat() SpeedInfusionTemporaryStat {
	return SpeedInfusionTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         0,
			rOption:         0,
			tLastUpdated:    time.Now().Unix(),
			usExpireItem:    0,
		},
		tCurrentTime: 0,
	}
}

type GuidedBulletTemporaryStat struct {
	CharacterTemporaryStatBase
	dwMobId uint32
}

func (m GuidedBulletTemporaryStat) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.CharacterTemporaryStatBase.Encode(l, t, options)(w)
		w.WriteInt(m.dwMobId)
	}
}

func NewGuidedBulletTemporaryStat() GuidedBulletTemporaryStat {
	return GuidedBulletTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         0,
			rOption:         0,
			tLastUpdated:    time.Now().Unix(),
			usExpireItem:    0,
		},
		dwMobId: 0,
	}
}

type CharacterTemporaryStat struct {
	stats map[CharacterTemporaryStatType]CharacterTemporaryStatValue
}

func (m *CharacterTemporaryStat) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	temporaryStatGetter := CharacterTemporaryStatTypeByName(t)
	return func(w *response.Writer) {
		mask := tool.Uint128{}
		applyMask := func(name CharacterTemporaryStatTypeName) {
			if val, err := temporaryStatGetter(name); err == nil {
				mask = mask.Or(val.mask)
			}
		}
		applyMask(CharacterTemporaryStatTypeNameEnergyCharge)
		applyMask(CharacterTemporaryStatTypeNameDashSpeed)
		applyMask(CharacterTemporaryStatTypeNameDashJump)
		applyMask(CharacterTemporaryStatTypeNameMonsterRiding)
		applyMask(CharacterTemporaryStatTypeNameSpeedInfusion)
		applyMask(CharacterTemporaryStatTypeNameHomingBeacon)
		applyMask(CharacterTemporaryStatTypeNameUndead)

		// TODO gather active buffs
		w.WriteInt(uint32(mask.H >> 32))
		w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
		w.WriteInt(uint32(mask.L >> 32))
		w.WriteInt(uint32(mask.L & 0xFFFFFFFF))

		// TODO write active buffs

		w.WriteByte(0) // nDefenseAtt
		w.WriteByte(0) // nDefenseState

		var baseTemporaryStats = m.getBaseTemporaryStats()
		for _, bts := range baseTemporaryStats {
			bts.Encode(l, t, options)(w)
		}
	}
}

func (m *CharacterTemporaryStat) getBaseTemporaryStats() []Encoder {
	var list = make([]Encoder, 0)
	list = append(list, NewCharacterTemporaryStatBase(true)) // Energy Charge 15
	list = append(list, NewCharacterTemporaryStatBase(true)) // Dash Speed 15
	list = append(list, NewCharacterTemporaryStatBase(true)) // Dash Jump 15
	// TODO look up actual buff values if riding mount.
	list = append(list, NewCharacterTemporaryStatBase(false)) // Monster Riding 13
	list = append(list, NewSpeedInfusionTemporaryStat())      // 17
	list = append(list, NewGuidedBulletTemporaryStat())       // 17
	list = append(list, NewCharacterTemporaryStatBase(true))  // Undead 15
	return list
}
