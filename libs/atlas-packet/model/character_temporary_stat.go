package model

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-packet/tool"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type CharacterTemporaryStatType struct {
	name               character.TemporaryStatType
	shift              uint
	mask               tool.Uint128
	disease            bool
	foreignValueWriter ForeignValueWriter
	foreignValueReader ForeignValueReader
}

func (t CharacterTemporaryStatType) Shift() uint {
	return t.shift
}

func (t CharacterTemporaryStatType) Name() character.TemporaryStatType {
	return t.name
}

// Disease reports whether this stat is a mob-applied disease (SLOW, STUN,
// POISON, SEAL, DARKNESS, WEAKEN, CURSE, SEDUCE, CONFUSE). Diseases share
// the GIVE_BUFF opcode with regular buffs but use a different per-stat wire
// shape — the 4 bytes that buffs spend on a 32-bit player skill id are
// instead split into two shorts carrying mobSkillId + mobSkillLevel.
func (t CharacterTemporaryStatType) Disease() bool {
	return t.disease
}

func NewCharacterTemporaryStatType(name character.TemporaryStatType, shift uint, disease bool, foreignValueWriter ForeignValueWriter, foreignValueReader ForeignValueReader) CharacterTemporaryStatType {
	mask := tool.Uint128{L: 1}.ShiftLeft(shift)
	return CharacterTemporaryStatType{
		name:               name,
		shift:              shift,
		mask:               mask,
		disease:            disease,
		foreignValueWriter: foreignValueWriter,
		foreignValueReader: foreignValueReader,
	}
}

type characterTemporaryStatRegistry struct {
	byName  map[character.TemporaryStatType]CharacterTemporaryStatType
	inOrder []CharacterTemporaryStatType
}

func buildCharacterTemporaryStatRegistry(t tenant.Model) characterTemporaryStatRegistry {
	var shift uint = 0
	set := make(map[character.TemporaryStatType]CharacterTemporaryStatType)
	var ordered []CharacterTemporaryStatType

	funcCallNewAndInc := func(disease bool) func(name character.TemporaryStatType) func(w ForeignValueWriter, r ForeignValueReader) {
		return func(name character.TemporaryStatType) func(w ForeignValueWriter, r ForeignValueReader) {
			return func(w ForeignValueWriter, r ForeignValueReader) {
				st := NewCharacterTemporaryStatType(name, shift, disease, w, r)
				set[name] = st
				ordered = append(ordered, st)
				shift += 1
			}
		}
	}
	newAndIncDiseased := funcCallNewAndInc(true)
	newAndIncNonDiseased := funcCallNewAndInc(false)

	newAndIncNonDiseased(character.TemporaryStatTypeWeaponAttack)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeWeaponDefense)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicAttack)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicDefense)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeAccuracy)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeAvoidability)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeHands)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSpeed)(ValueAsByteForeignValueWriter, ByteForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeJump)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicGuard)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDarkSight)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeBooster)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypePowerGuard)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeHyperBodyHP)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeHyperBodyMP)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeInvincible)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulArrow)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeStun)(MobSkillReasonForeignValueWriter, MobSkillReasonForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypePoison)(ValueMobSkillReasonForeignValueWriter, ValueMobSkillReasonForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeSeal)(MobSkillReasonForeignValueWriter, MobSkillReasonForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeDarkness)(MobSkillReasonForeignValueWriter, MobSkillReasonForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeCombo)(ValueAsByteForeignValueWriter, ByteForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeWhiteKnightCharge)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDragonBlood)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeHolySymbol)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoUp)(NoOpForeignValueWriter, NoOpForeignValueReader)
	if t.IsRegion("GMS") && t.MajorAtLeast(87) {
		// v87+ ShadowPartner level-source field; v84..86 == v83 (off-by-one fix). delta §3.2
		newAndIncNonDiseased(character.TemporaryStatTypeShadowPartner)(LevelSourceForeignValueWriter, LevelSourceForeignValueReader)
	} else {
		newAndIncNonDiseased(character.TemporaryStatTypeShadowPartner)(NoOpForeignValueWriter, NoOpForeignValueReader)
	}
	newAndIncNonDiseased(character.TemporaryStatTypePickPocket)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoGuard)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeThaw)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeWeaken)(MobSkillReasonForeignValueWriter, MobSkillReasonForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeCurse)(MobSkillReasonForeignValueWriter, MobSkillReasonForeignValueReader)
	// SLOW is mask-only on the foreign path, and must stay that way.
	// SecondaryStat::DecodeForRemote has no CTS_Slow branch on ANY supported
	// client — proven by xref on v83 (0xbeffc0) and v95 (0xc6c9a0), by table
	// position on v87/v92, and by exhaustive block enumeration on
	// v48/v61/v72/v79/v84. Slow's nOption/rOption therefore stay zero on an
	// observer, UpdateAffectedSkillList never picks it up, and no remote
	// animation is possible. Writing the mob-skill key here (as some servers
	// do) would emit 4 bytes the client reads as nDefenseAtt/nDefenseState.
	newAndIncDiseased(character.TemporaryStatTypeSlow)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMorph)(ValueAsShortForeignValueWriter, ShortForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeRecovery)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMapleWarrior)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeStance)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSharpEyes)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeManaReflection)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeSeduce)(MobSkillReasonForeignValueWriter, MobSkillReasonForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeShadowClaw)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeInfinity)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeHolyShield)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeHamstring)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeBlind)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeConcentrate)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeBanMap)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeEchoOfHero)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoUpByItem)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeGhostMorph)(ValueAsShortForeignValueWriter, ShortForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeBarrier)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeConfuse)(MobSkillReasonForeignValueWriter, MobSkillReasonForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeItemUpByItem)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeRespectPImmune)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeRespectMImmune)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDefenseAttack)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDefenseState)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeIncreaseEffectHpPotion)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeIncreaseEffectMpPotion)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeBerserkFury)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDivineBody)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSpark)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDojangShield)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulMasterFinal)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeWindBreakerFinal)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeElementalReset)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeWindWalk)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeEventRate)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeAranCombo)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeComboDrain)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeComboBarrier)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeBodyPressure)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSmartKnockBack)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeRepeatEffect)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeExpBuffRate)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeStopPortion)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeStopMotion)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeFear)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeEvanSlow)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicShield)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicResist)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulStone)(NoOpForeignValueWriter, NoOpForeignValueReader)

	// Post-SoulStone region, enumerated as one linear sequence with per-version gates
	// instead of duplicated per-version blocks. Each slot is appended only for the
	// versions whose client has it, so `shift` stays aligned: v83/v84 add nothing
	// (two-state at 82); v87 adds 4 (two-state at 86); JMS adds 28 (two-state at 110);
	// GMS v95 adds 40 (two-state at 122, RideVehicle at 125). IDA-verified — see
	// docs/tasks/task-086-mount-system/v95_secondarystat_table.md and the v87 reset map
	// (https://github.com/Chronicle20/gms-83-dll docs/tasks/cwvscontext-port/v87_secondarystat_reset_mapping.md).
	gmsV95Plus := t.Region() == "GMS" && t.MajorVersion() >= 95
	jms := t.Region() == "JMS"
	post87 := (t.Region() == "GMS" && t.MajorAtLeast(87)) || jms // clients with any post-SoulStone stats
	extended := gmsV95Plus || jms                                // clients with the SuddenDeath..Sneak block

	if post87 {
		newAndIncNonDiseased(character.TemporaryStatTypeFlying)(NoOpForeignValueWriter, NoOpForeignValueReader)       // 82
		newAndIncNonDiseased(character.TemporaryStatTypeFrozen)(ValueAsIntForeignValueWriter, IntForeignValueReader)  // 83
		newAndIncNonDiseased(character.TemporaryStatTypeAssistCharge)(NoOpForeignValueWriter, NoOpForeignValueReader) // 84
	}
	// bit 85 diverges: GMS v95 has Enrage where v87/JMS have MirrorImage.
	if gmsV95Plus {
		newAndIncNonDiseased(character.TemporaryStatTypeEnrage)(NoOpForeignValueWriter, NoOpForeignValueReader) // 85 (v95)
	} else if post87 {
		newAndIncNonDiseased(character.TemporaryStatTypeMirrorImage)(NoOpForeignValueWriter, NoOpForeignValueReader) // 85 (v87/JMS)
	}
	if extended {
		// bits 86-108: shared by GMS v95 and JMS (same order + foreign shapes).
		newAndIncNonDiseased(character.TemporaryStatTypeSuddenDeath)(ValueAsIntForeignValueWriter, IntForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeNotDamaged)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeFinalCut)(ValueAsIntForeignValueWriter, IntForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeThornsEffect)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowAttackDamage)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeWildDamageUp)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeMine)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeEMHP)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeEMMP)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeEPAD)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeEPPD)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeEMDD)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeGuard)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyDamage)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyAbsorb)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeCyclone)(ValueAsByteForeignValueWriter, ByteForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowCritical)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowMaxMP)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowDefense)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowEvasion)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeConversion)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeRevive)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSneak)(NoOpForeignValueWriter, NoOpForeignValueReader)
	}
	if jms {
		newAndIncNonDiseased(character.TemporaryStatTypeUnknown)(NoOpForeignValueWriter, NoOpForeignValueReader) // 109 (JMS)
	}
	if gmsV95Plus {
		// bits 109-121: GMS v95 only. atlas never originates these — NoOp reserves the bit.
		newAndIncNonDiseased(character.TemporaryStatTypeMechanic)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeAura)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeDarkAura)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeBlueAura)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeYellowAura)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeSuperBody)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeWildMaxHpUp)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeDice)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeBlessingArmor)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeDamageReduce)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeTeleportMastery)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeCombatOrders)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeBeholder)(NoOpForeignValueWriter, NoOpForeignValueReader)
	}

	// Two-state group (always present, last). v95's 5th slot is PartyBooster where
	// earlier versions have SpeedInfusion. MonsterRiding/RideVehicle lands at v83=85,
	// v87=89, JMS=113, v95=125 — exactly where each client reads it.
	newAndIncNonDiseased(character.TemporaryStatTypeEnergyCharge)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDashSpeed)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDashJump)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMonsterRiding)(NoOpForeignValueWriter, NoOpForeignValueReader)
	if gmsV95Plus {
		newAndIncNonDiseased(character.TemporaryStatTypePartyBooster)(NoOpForeignValueWriter, NoOpForeignValueReader)
	} else {
		newAndIncNonDiseased(character.TemporaryStatTypeSpeedInfusion)(NoOpForeignValueWriter, NoOpForeignValueReader)
	}
	newAndIncNonDiseased(character.TemporaryStatTypeHomingBeacon)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeUndead)(NoOpForeignValueWriter, NoOpForeignValueReader)

	return characterTemporaryStatRegistry{byName: set, inOrder: ordered}
}

func CharacterTemporaryStatTypeByName(t tenant.Model) func(name character.TemporaryStatType) (CharacterTemporaryStatType, error) {
	reg := buildCharacterTemporaryStatRegistry(t)
	return func(name character.TemporaryStatType) (CharacterTemporaryStatType, error) {
		if val, ok := reg.byName[name]; ok {
			return val, nil
		}
		return CharacterTemporaryStatType{}, errors.New("character temporary stat type not found")
	}
}

type ForeignValueWriter func(v CharacterTemporaryStatValue) func(w *response.Writer)

func NoOpForeignValueWriter(_ CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(_ *response.Writer) {
	}
}

func ValueAsByteForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt8(int8(v.Value()))
	}
}

func ValueAsShortForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(int16(v.Value()))
	}
}

func ValueAsIntForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt32(v.Value())
	}
}

func LevelSourceForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(int16(v.Level()))
		w.WriteInt16(int16(v.SourceId()))
	}
}

// MobSkillReasonForeignValueWriter writes a mob-applied disease's reason field
// for the FOREIGN (observer) path: Short(mobSkillId) + Short(mobSkillLevel),
// which the client consumes as a single Decode4 whose value is
// mobSkillId | (level << 16).
//
// That composite is the animation key. CUser::UpdateAffectedSkillList (v83
// @0x93e344) collects each present disease's reason — never its value — into
// the affected-skill map, and CUser::ShowAffectedSkillAni (@0x932da6) splits it
// back apart (`v8 = key >> 16` is the level; the low half indexes the MobSkill
// table via sub_7632F2) to load MobSkill[id].level[lv].affected. Writing the
// disease amount here, as the old ValueAsInt/LevelSource writers did, resolved
// to no mob skill and rendered nothing. GMS v95's PDB names the targets
// outright — rStun, rSeal, rDarkness, rWeakness, rCurse, rPoison, rAttract,
// rReverseInput (SecondaryStat::DecodeForRemote @0x72b7b0).
//
// Field order mirrors the already-correct local path (Encode writes
// Short(sourceId) + Short(level) for diseases). See
// docs/tasks/task-195-foreign-disease-mobskill/investigation.md §1-2.
func MobSkillReasonForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(int16(v.SourceId()))
		w.WriteInt16(int16(v.Level()))
	}
}

// ValueMobSkillReasonForeignValueWriter is MobSkillReasonForeignValueWriter
// prefixed with the stat value. POISON is the one disease whose foreign block
// carries a value: the client reads Decode2 (nPoison, the per-tick damage) and
// then Decode4 (rPoison, the mob-skill key) — two separate mask tests against
// CTS_Poison, back to back.
func ValueMobSkillReasonForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(int16(v.Value()))
		w.WriteInt16(int16(v.SourceId()))
		w.WriteInt16(int16(v.Level()))
	}
}

type ForeignValueReader func(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue

func NoOpForeignValueReader(_ *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	return CharacterTemporaryStatValue{statType: st}
}

func ByteForeignValueReader(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	return CharacterTemporaryStatValue{statType: st, value: int32(r.ReadInt8())}
}

func ShortForeignValueReader(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	return CharacterTemporaryStatValue{statType: st, value: int32(r.ReadInt16())}
}

func IntForeignValueReader(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	return CharacterTemporaryStatValue{statType: st, value: r.ReadInt32()}
}

func LevelSourceForeignValueReader(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	level := byte(r.ReadInt16())
	sourceId := int32(r.ReadInt16())
	return CharacterTemporaryStatValue{statType: st, level: level, sourceId: sourceId}
}

func MobSkillReasonForeignValueReader(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	sourceId := int32(r.ReadInt16())
	level := byte(r.ReadInt16())
	return CharacterTemporaryStatValue{statType: st, sourceId: sourceId, level: level}
}

func ValueMobSkillReasonForeignValueReader(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	value := int32(r.ReadInt16())
	sourceId := int32(r.ReadInt16())
	level := byte(r.ReadInt16())
	return CharacterTemporaryStatValue{statType: st, value: value, sourceId: sourceId, level: level}
}

type CharacterTemporaryStatValue struct {
	statType  CharacterTemporaryStatType
	sourceId  int32
	level     byte
	value     int32
	expiresAt time.Time
}

func (v CharacterTemporaryStatValue) Value() int32 {
	return v.value
}

func (v CharacterTemporaryStatValue) SourceId() int32 {
	return v.sourceId
}

func (v CharacterTemporaryStatValue) Level() byte {
	return v.level
}

func (v CharacterTemporaryStatValue) ExpiresAt() time.Time {
	return v.expiresAt
}

func (v CharacterTemporaryStatValue) Write(w *response.Writer) {
	v.statType.foreignValueWriter(v)(w)
}

type CharacterTemporaryStatBase struct {
	bDynamicTermSet bool
	nOption         int32
	rOption         int32
	tLastUpdated    int64
	usExpireItem    int16
	// narrowTimeField selects GMS v61's base-block shape: the third field is a
	// bare, unprefixed Decode4 (4 bytes) instead of the bool-prefixed
	// writeTime()/readTime() pair (5 bytes) every other in-scope version uses.
	// IDA-verified: sub_66E9B6 has zero Decode1 (bool) calls, and its third
	// field is a plain Decode4 structurally identical to the first two —
	// task-167, docs/tasks/task-167-homing-beacon-bullseye/evidence/per-version/gms_v61_composition.md.
	narrowTimeField bool
}

func NewCharacterTemporaryStatBase(bDynamicTermSet bool, narrowTimeField bool) CharacterTemporaryStatBase {
	return CharacterTemporaryStatBase{
		tLastUpdated:    time.Now().Unix(),
		bDynamicTermSet: bDynamicTermSet,
		narrowTimeField: narrowTimeField,
	}
}

func NewCharacterTemporaryStatBaseWithOptions(bDynamicTermSet bool, nOption int32, rOption int32, narrowTimeField bool) CharacterTemporaryStatBase {
	return CharacterTemporaryStatBase{
		tLastUpdated:    time.Now().Unix(),
		bDynamicTermSet: bDynamicTermSet,
		nOption:         nOption,
		rOption:         rOption,
		narrowTimeField: narrowTimeField,
	}
}

func readTime(r *request.Reader) int64 {
	interval := r.ReadBool()
	delta := int64(r.ReadInt32()) * 1000
	cur := time.Now().Unix()
	if interval {
		return cur - delta
	}
	return cur + delta
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

func (m CharacterTemporaryStatBase) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.nOption)
		w.WriteInt32(m.rOption)
		if m.narrowTimeField {
			w.WriteInt32(int32(m.tLastUpdated))
		} else {
			writeTime(m.tLastUpdated)(w)
		}
		if m.bDynamicTermSet {
			w.WriteInt16(m.usExpireItem)
		}
		return w.Bytes()
	}
}

func (m *CharacterTemporaryStatBase) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.nOption = r.ReadInt32()
		m.rOption = r.ReadInt32()
		if m.narrowTimeField {
			m.tLastUpdated = int64(r.ReadInt32())
		} else {
			m.tLastUpdated = readTime(r)
		}
		if m.bDynamicTermSet {
			m.usExpireItem = r.ReadInt16()
		}
	}
}

type SpeedInfusionTemporaryStat struct {
	CharacterTemporaryStatBase
	tCurrentTime int32
}

func (m SpeedInfusionTemporaryStat) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.CharacterTemporaryStatBase.Encode(l, ctx)(options))
		if m.narrowTimeField {
			// GMS v61: SpeedInfusion's extra field is a bare Decode4 (4 bytes),
			// not the 5-byte bool+delta writeTime() pair — IDA sub_66E8EF,
			// task-167.
			w.WriteInt32(m.tCurrentTime)
		} else {
			writeTime(int64(m.tCurrentTime))(w)
		}
		w.WriteInt16(m.usExpireItem)
		return w.Bytes()
	}
}

func (m *SpeedInfusionTemporaryStat) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.CharacterTemporaryStatBase.Decode(l, ctx)(r, options)
		if m.narrowTimeField {
			m.tCurrentTime = r.ReadInt32()
		} else {
			m.tCurrentTime = int32(readTime(r))
		}
		m.usExpireItem = r.ReadInt16()
	}
}

func NewSpeedInfusionTemporaryStat(narrowTimeField bool) SpeedInfusionTemporaryStat {
	return SpeedInfusionTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         0,
			rOption:         0,
			tLastUpdated:    time.Now().Unix(),
			usExpireItem:    0,
			narrowTimeField: narrowTimeField,
		},
		tCurrentTime: 0,
	}
}

type GuidedBulletTemporaryStat struct {
	CharacterTemporaryStatBase
	dwMobId uint32
}

func (m GuidedBulletTemporaryStat) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.CharacterTemporaryStatBase.Encode(l, ctx)(options))
		w.WriteInt(m.dwMobId)
		return w.Bytes()
	}
}

func (m *GuidedBulletTemporaryStat) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.CharacterTemporaryStatBase.Decode(l, ctx)(r, options)
		m.dwMobId = r.ReadUint32()
	}
}

func NewGuidedBulletTemporaryStat(narrowTimeField bool) GuidedBulletTemporaryStat {
	return GuidedBulletTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         0,
			rOption:         0,
			tLastUpdated:    time.Now().Unix(),
			usExpireItem:    0,
			narrowTimeField: narrowTimeField,
		},
		dwMobId: 0,
	}
}

// NewGuidedBulletTemporaryStatWithOptions builds a populated GuidedBullet
// block for an active HOMING_BEACON lock. nOption must be nonzero — the
// client's set path gates on IsActivated (nValue != 0) before calling
// CMob::SetGuided (IDA v83 @0xA202BE, v95 @0xA02FC0; design.md §2.3/§2.4).
func NewGuidedBulletTemporaryStatWithOptions(nOption int32, rOption int32, dwMobId uint32, narrowTimeField bool) GuidedBulletTemporaryStat {
	return GuidedBulletTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         nOption,
			rOption:         rOption,
			tLastUpdated:    time.Now().Unix(),
			narrowTimeField: narrowTimeField,
		},
		dwMobId: dwMobId,
	}
}

type CharacterTemporaryStat struct {
	stats map[character.TemporaryStatType]CharacterTemporaryStatValue
}

func NewCharacterTemporaryStat() *CharacterTemporaryStat {
	return &CharacterTemporaryStat{
		stats: make(map[character.TemporaryStatType]CharacterTemporaryStatValue),
	}
}

// HasDisease reports whether any stat held by this CTS is a mob-applied
// disease. Used by BuffGive to pick the correct trailer bytes — diseases
// require the debuff trailer (Short delay=900, Byte apply=1) for
// the v83 client to actually render the debuff icon and apply
// flag-gated effects (e.g. WEAKEN's jump-block).
func (m *CharacterTemporaryStat) HasDisease() bool {
	for _, v := range m.stats {
		if v.statType.Disease() {
			return true
		}
	}
	return false
}

// serverOnlyStatNames are temporary stats that exist only for server-side
// lifecycle bookkeeping (Odin lineage). No supported client has a
// SecondaryStat bit for them — IDA-verified across every version Atlas holds
// a binary for (GMS v48/v61/v72/v79/v83/v84/v87/v92/v95, JMS v185), see
// docs/tasks/task-164-summon-temp-stats/prd.md §1.1 — so they are never
// encoded into any CTS mask or payload, on any tenant version. Summon
// visibility for observers is carried by the summon object packets
// (task-088/106), not by a buff. Adding a name here requires the same
// IDA evidence trail.
var serverOnlyStatNames = map[character.TemporaryStatType]bool{
	character.TemporaryStatTypePuppet: true,
	character.TemporaryStatTypeSummon: true,
}

func (m *CharacterTemporaryStat) AddStat(l logrus.FieldLogger) func(t tenant.Model) func(n string, sourceId int32, amount int32, level byte, expiresAt time.Time) {
	return func(t tenant.Model) func(n string, sourceId int32, amount int32, level byte, expiresAt time.Time) {
		return func(n string, sourceId int32, amount int32, level byte, expiresAt time.Time) {
			name := character.TemporaryStatType(n)
			if serverOnlyStatNames[name] {
				l.Debugf("Skipping server-only temporary stat [%s]; it has no client wire representation.", name)
				return
			}
			st, err := CharacterTemporaryStatTypeByName(t)(name)
			if err != nil {
				l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)
				return
			}
			v := CharacterTemporaryStatValue{
				statType:  st,
				sourceId:  sourceId,
				level:     level,
				value:     amount,
				expiresAt: expiresAt,
			}
			if e, ok := m.stats[name]; ok {
				if v.Value() > e.Value() {
					m.stats[name] = v
				}
			} else {
				m.stats[name] = v
			}
		}
	}
}

// legacyGmsMask reports the pre-v61 GMS clients whose CharacterTemporaryStat
// (SecondaryStat) mask is a plain 64-bit little-endian value read via
// CInPacket::DecodeBuffer(&v, 8) — NOT the 128-bit UINT128 (DecodeBuffer 16) the
// v61+ anchor codec emits. IDA-verified on GMS_v48_1_DEVM.exe (port 13337):
//   - local  CWvsContext::OnTemporaryStatSet   @0x71af4b → sub_5CA524 DecodeBuffer(8) @0x5ca539
//   - reset  CWvsContext::OnTemporaryStatReset  @0x71b054 → DecodeBuffer(8) @0x71b06e
//   - foreign CUserRemote::Init CTS decode       sub_5CBA1F  DecodeBuffer(8) @0x5cba33
//
// All three test bits 0-46, and the foreign per-bit value shapes match this
// registry's shift order stat-for-stat (bit7=Speed byte, bit21=Combo byte,
// bit33=Morph short, bit17/19/20/22=int, bit10/16/26=flag), so bits 0-46 map
// identically to the shared registry — only the mask WIDTH differs. The
// two-state base stats sit at shifts 81-87 (mask.H), which pre-v61 clients do
// not read; WriteLong(mask.L) drops them, matching an empty v48 mask of 8 zero
// bytes. v61+ (v61/72/79/83/84/87/95/JMS anchors) are untouched. v28 (also < 61,
// no IDB) inherits this path — round-trip-only, previously the equally-unverified
// 128-bit path; an 8-byte mask is more plausible for a pre-v61 client.
func legacyGmsMask(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 61
}

// EncodeMask writes the 128-bit SecondaryStat mask: one bit per stat this CTS
// actually holds, and nothing else. It is shared by the SET (GIVE_BUFF) and
// RESET (CANCEL_BUFF) paths, which differ only in whether value/base blocks
// follow — never in which bits are claimed.
//
// This used to OR in the entire TwoState/base group (EnergyCharge, DashSpeed,
// DashJump, MonsterRiding, SpeedInfusion, HomingBeacon, Undead) unconditionally,
// on the theory that the client reads a fixed base-stat group. It does not: it
// reads one base block per SET base bit, so claiming a base stat the CTS does
// not hold is both a lie and a wire cost.
//
// It was also a live bug (task-190). Both CWvsContext::OnTemporaryStatSet and
// ::OnTemporaryStatReset branch on the received mask:
//
//	if (mask & CTS_RideVehicle)  CUser::ShowRideVehicleEffect(...)   // or SendSkillCancelRequest
//	if (mask & CTS_GuidedBullet) CMobPool::ResetGuidedMob(...)       // CMob::SetGuided on set
//
// so every buff give and every buff cancel — a mob's Slow expiring, say — drove
// the ride-vehicle and guided-bullet paths for a player who was simply standing
// there, desyncing client and server about whether they were mounted. Verified
// on GMS v61 (@0x84353a), v72 (@0x918f3c), v83 (@0xa2071f / @0xa202be) and v95
// (@0x9f2ab0, whose PDB-backed symbols name CTS_RideVehicle_2 and
// CTS_GuidedBullet_0 outright), plus the foreign path
// CUserRemote::OnResetTemporaryStat (v83 @0x983921).
//
// The per-version shift already places each bit where the client reads it: on
// v83 RideVehicle is shift 85 -> wire bytes 4-7, matching
// SecondaryStat::DecodeForLocal's flag 1<<(i+82). No version-specific mask
// placement is needed.
//
// getBaseTemporaryStats is gated on the same presence test, so bits and blocks
// cannot drift apart.
func (m *CharacterTemporaryStat) EncodeMask(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		mask := m.activeMask()

		if legacyGmsMask(t) {
			// Pre-v61 GMS: 8-byte little-endian mask (DecodeBuffer 8). Bits 0-46
			// live in mask.L; the two-state base bits (shifts 81-87) fall in
			// mask.H and are intentionally dropped — the pre-v61 client never
			// reads them (sub_5CBA1F / sub_5CA524 test only bits 0-46).
			w.WriteLong(mask.L)
			return
		}

		writeMask(w, mask)
	}
}

// activeMask is the bit set EncodeMask claims: one bit per stat this CTS holds,
// and nothing else. Split out so tests can assert the invariant directly rather
// than by reading it back off the wire.
func (m *CharacterTemporaryStat) activeMask() tool.Uint128 {
	mask := tool.Uint128{}
	for _, v := range m.stats {
		mask = mask.Or(v.statType.mask)
	}
	return mask
}

func writeMask(w *response.Writer, mask tool.Uint128) {
	w.WriteInt(uint32(mask.H >> 32))
	w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
	w.WriteInt(uint32(mask.L >> 32))
	w.WriteInt(uint32(mask.L & 0xFFFFFFFF))
}

// task-167 carried a separate CancelMask/EncodeCancelMask pair here, whose whole
// job was to give the RESET path a present-stats-only mask while EncodeMask kept
// asserting the two-state group for the SET path. EncodeMask now claims only the
// stats the CTS holds on both paths (task-190), so the cancel-specific pair was
// exactly EncodeMask and has been dropped rather than kept as a second spelling
// of it.

// movementAffectingStatNames is the version-gated mirror of the client's
// movement filter: the reset/give trailing byte is read ONLY when the
// packet's mask intersects this set. Evidence:
// docs/tasks/task-167-homing-beacon-bullseye/evidence/movement-filter.md
// (v83 sub_77DC78; v61 0x660B44; v72 0x6c87b6; v79 0x6f852f; v84 sub_7a07e7;
// v87 sub_7cc3e2; v92 sub_705080; v95 SecondaryStat::IsMovementAffectingStat
// @0x7208C0; JMS sub_7f76d1).
//
// Over-inclusion is the safe direction, which is why the gates below are
// cumulative >= bounds rather than a per-version switch: the failure mode is
// one-directional. If a client gates the trailing byte on a stat this list
// omits, cancelling that stat drops a byte the client expects and the packet
// desyncs; naming a stat the client never tests costs nothing, and
// MovementAffectingMask drops any name the tenant's registry does not allocate.
func movementAffectingStatNames(t tenant.Model) []character.TemporaryStatType {
	if t.IsRegion("JMS") {
		// JMS v185 (sub_7f76d1) tests a wholly different 13-constant set —
		// NOT "the v83 list plus/minus extras". Only Stun and
		// MonsterRiding(RideVehicle) overlap v83's 12-stat meaning; GhostMorph
		// is present but occupies a different semantic slot. The
		// other 9 v83 stats (Speed, Jump, Weaken, Slow, Morph, MapleWarrior,
		// Seduce, DashSpeed, DashJump) are absent, and JMS's remaining bits
		// map to unrelated stats (Invincible, SoulArrow, MesoUpByItem,
		// WindBreakerFinal, ElementalReset, EventRate, BodyPressure,
		// SoulStone, SwallowDefense). Raw shift 126 has no atlas registry
		// entry (the JMS registry branch only defines shifts 0-116) and is
		// therefore omitted — reported unmapped in task-8-report.md.
		return []character.TemporaryStatType{
			character.TemporaryStatTypeInvincible,
			character.TemporaryStatTypeSoulArrow,
			character.TemporaryStatTypeStun,
			character.TemporaryStatTypeMesoUpByItem,
			character.TemporaryStatTypeGhostMorph,
			character.TemporaryStatTypeWindBreakerFinal,
			character.TemporaryStatTypeElementalReset,
			character.TemporaryStatTypeEventRate,
			character.TemporaryStatTypeBodyPressure,
			character.TemporaryStatTypeSoulStone,
			character.TemporaryStatTypeSwallowDefense,
			character.TemporaryStatTypeMonsterRiding,
		}
	}

	// The base 12, tested identically by v61/v72/v79/v83 (v61 fully
	// name-resolved; v72/v79 confirmed by count+structure with some individual
	// names positional-only — see the evidence file's per-version caveats).
	// v92 is the one known subtraction: it does not test Speed (gms_v92.md).
	// Speed stays here anyway, per the over-inclusion rule above.
	names := []character.TemporaryStatType{
		character.TemporaryStatTypeSpeed,
		character.TemporaryStatTypeJump,
		character.TemporaryStatTypeStun,
		character.TemporaryStatTypeWeaken,
		character.TemporaryStatTypeSlow,
		character.TemporaryStatTypeMorph,
		character.TemporaryStatTypeGhostMorph,
		character.TemporaryStatTypeMapleWarrior,
		character.TemporaryStatTypeSeduce,
		character.TemporaryStatTypeMonsterRiding,
		character.TemporaryStatTypeDashSpeed,
		character.TemporaryStatTypeDashJump,
	}

	// Flying(82)/Frozen(83) join the filter from v84 on. Only v87 resolves them
	// by name (sub_7cc3e2, raw shift == registry shift, cross-checked against
	// the registry's declaration order — gms_v87.md). v84 (sub_7a07e7, 14
	// constants) and v92 (sub_705080, 13 constants) test the
	// identically-positioned raw 82/83 constants without naming either in their
	// own IDBs; two-state-group-per-version.md places them at that same slot.
	//
	// The registry allocates the pair only from v87 (the post87 block), so on
	// v84 these two names resolve to no bit at all — carried here for intent and
	// dropped by MovementAffectingMask.
	if t.IsRegion("GMS") && t.MajorAtLeast(84) {
		names = append(names,
			character.TemporaryStatTypeFlying,
			character.TemporaryStatTypeFrozen,
		)
	}

	// v95 (SecondaryStat::IsMovementAffectingStat @0x7208C0) is the only version
	// that resolves its whole filter by symbol name: the 14 above plus
	// YellowAura (design.md §2.4).
	if t.IsRegion("GMS") && t.MajorAtLeast(95) {
		names = append(names, character.TemporaryStatTypeYellowAura)
	}

	return names
}

// MovementAffectingMask returns the movement filter as a mask for this
// tenant's registry layout. A name with no entry in this tenant's registry
// (e.g. Flying/Frozen on a version whose registry doesn't allocate them) is
// silently skipped — see movementAffectingStatNames.
//
// NOT currently wired into any writer. BuffCancel/BuffCancelForeign write the
// trailing nSecondaryStatChangedPoint byte unconditionally, which is the
// fail-safe direction: an unread trailing byte is slack the client ignores,
// while omitting one the client does read runs it off the end of the packet.
// Gating the write on this mask is only safe once every version's filter is
// name-resolved, and it is not — v72/v79/v92/JMS bits are positional or
// inferred (see the per-version caveats in movement-filter.md). This stays
// here, with its membership test, so that re-verification flips a switch
// instead of redoing the derivation.
func MovementAffectingMask(t tenant.Model) tool.Uint128 {
	reg := buildCharacterTemporaryStatRegistry(t)
	mask := tool.Uint128{}
	for _, n := range movementAffectingStatNames(t) {
		if st, ok := reg.byName[n]; ok {
			mask = mask.Or(st.mask)
		}
	}
	return mask
}

func (m *CharacterTemporaryStat) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.EncodeMask(l, t, options)(w)

		keys := make([]CharacterTemporaryStatType, 0)
		for _, v := range m.stats {
			if baseStatNames[v.statType.name] {
				// Base/TwoState stats (e.g. MonsterRiding) are encoded only as
				// base-stat blocks below — never as a per-stat value block. The v83
				// client reads them in its 7-iteration base loop, so a per-stat block
				// here would desync the entire tail. Version-independent.
				continue
			}
			keys = append(keys, v.statType)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Shift() < keys[j].Shift()
		})

		sortedValues := make([]CharacterTemporaryStatValue, 0)
		for _, k := range keys {
			sortedValues = append(sortedValues, m.stats[k.name])
		}

		for _, v := range sortedValues {
			if legacyGmsMask(t) {
				// Pre-v61 GMS local value block (sub_5CA524, per set bit):
				// Decode2(value) + Decode4(reason) + Decode2(duration/500). No
				// disease split (the helper reads Decode4 for every stat), no
				// nDefenseAtt/nDefenseState bytes, no trailing base-stat blocks —
				// OnTemporaryStatSet reads only the delay short + optional byte
				// afterward (emitted by BuffGive).
				w.WriteInt16(int16(v.Value()))
				w.WriteInt32(v.SourceId())
				w.WriteInt16(legacyDurationUnits(v.ExpiresAt()))
				continue
			}
			w.WriteInt16(int16(v.Value()))
			if v.statType.Disease() {
				// Mob-applied disease: bytes 4-5 carry mobSkillLevel, not the
				// high half of sourceId. The v83 client otherwise looks up
				// MobSkill(skill, 0), gets nothing, and crashes the render path.
				w.WriteInt16(int16(v.SourceId()))
				w.WriteInt16(int16(v.Level()))
			} else {
				w.WriteInt32(v.SourceId())
			}
			w.WriteInt32(remainingMillis(v.ExpiresAt()))
		}

		if legacyGmsMask(t) {
			return w.Bytes()
		}

		w.WriteByte(0) // nDefenseAtt
		w.WriteByte(0) // nDefenseState

		baseTemporaryStats := m.getBaseTemporaryStats(t)
		for _, bts := range baseTemporaryStats {
			w.WriteByteArray(bts.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

// remainingMillis converts an absolute expiry into the client-facing remaining
// duration.
//
// The zero time is atlas-buffs' marker for a buff that never expires on its own
// (buff.NewNoExpiryBuff — GM hide, mounts, the beacon lock). No client has a
// no-expiry concept: every one of them reads a concrete duration here, so the
// sentinel has to be reintroduced at the wire boundary. Doing it here rather
// than at the caller is what keeps it out of the domain model and off the Kafka
// and REST contracts.
//
// Without this the zero time underflows spectacularly rather than harmlessly:
// year 1 to now exceeds what an int64 nanosecond Duration can hold, so
// time.Until saturates and the int32 truncation lands on an arbitrary negative
// (-2077252342 as of this writing) — a stat the client tears down immediately.
func remainingMillis(expiresAt time.Time) int32 {
	if expiresAt.IsZero() {
		return math.MaxInt32
	}
	return int32(time.Until(expiresAt).Milliseconds())
}

// legacyDurationUnits converts an absolute expiry into the pre-v61 wire duration
// short: the v48 client reads Decode2 and multiplies by 500 (sub_5CA524 @0x5ca58d
// `500 * Decode2`), so the wire carries remaining-ms / 500. A no-expiry buff
// (zero expiry — see remainingMillis) saturates the short rather than reading as
// already-expired.
func legacyDurationUnits(expiresAt time.Time) int16 {
	if expiresAt.IsZero() {
		return math.MaxInt16
	}
	ms := time.Until(expiresAt).Milliseconds()
	if ms <= 0 {
		return 0
	}
	if ms/500 > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(ms / 500)
}

// foreignReadOrder is SecondaryStat::DecodeForRemote's per-stat sequence, in the
// order the client reads it.
//
// It is deliberately NOT the registry's shift order. The LOCAL decoder
// (DecodeForLocal) walks the mask in bit order — v83 code positions ascend with
// shift: Stun@0x782d9d < Poison@0x782ea2 < Combo@0x783157 < Weaken@0x783976 <
// Slow@0x783b44 — which is why Encode can sort by shift. The REMOTE decoder is a
// hand-written sequence that hoists Combo/WeaponCharge ahead of the diseases and
// drops Poison in after Curse: Combo@0x7881b0 < Stun@0x788234 < Weaken@0x78830f
// < Poison@0x7883a1. EncodeForeign used to sort by shift here too, so any two
// value-carrying stats present at once had their payloads swapped — the common
// case for diseases (SEAL+DARKNESS, POISON+WEAKEN, STUN+anything).
//
// The same relative order holds on every supported client; later versions only
// append to the tail. Verified gms_v48 (sub_5CBA1F, 8-byte mask), v61
// (@0x667c5f), v72 (@0x6cfe78), v79 (@0x701539), v83 (@0x788156), v84
// (@0x7ac409), v87 (@0x7d8533), v92 (@0x711240), v95 (@0x72b7b0). One list
// therefore serves all of them: a stat a version lacks never appears in its
// mask, and a stat whose foreign shape is NoOp on a version contributes no
// bytes. See docs/tasks/task-195-foreign-disease-mobskill/investigation.md §3.
//
// SLOW is absent on purpose — no client's remote decoder reads it (§4).
var foreignReadOrder = []character.TemporaryStatType{
	character.TemporaryStatTypeSpeed,             // Decode1
	character.TemporaryStatTypeCombo,             // Decode1
	character.TemporaryStatTypeWhiteKnightCharge, // Decode4
	character.TemporaryStatTypeStun,              // Decode4 (disease reason)
	character.TemporaryStatTypeDarkness,          // Decode4 (disease reason)
	character.TemporaryStatTypeSeal,              // Decode4 (disease reason)
	character.TemporaryStatTypeWeaken,            // Decode4 (disease reason)
	character.TemporaryStatTypeCurse,             // Decode4 (disease reason)
	character.TemporaryStatTypePoison,            // Decode2 value + Decode4 reason
	character.TemporaryStatTypeShadowPartner,     // Decode4 on v87+, flag-only below
	character.TemporaryStatTypeDarkSight,         // flag only
	character.TemporaryStatTypeSoulArrow,         // flag only
	character.TemporaryStatTypeMorph,             // Decode2
	character.TemporaryStatTypeGhostMorph,        // Decode2
	character.TemporaryStatTypeSeduce,            // Decode4 (disease reason)
	character.TemporaryStatTypeShadowClaw,        // Decode4
	character.TemporaryStatTypeBanMap,            // Decode4 (v61+; v48 reads none)
	character.TemporaryStatTypeBarrier,           // Decode4
	character.TemporaryStatTypeDojangShield,      // Decode4
	character.TemporaryStatTypeConfuse,           // Decode4 (disease reason; absent on v61)
	character.TemporaryStatTypeRespectPImmune,    // Decode4
	character.TemporaryStatTypeRespectMImmune,    // Decode4
	character.TemporaryStatTypeDefenseAttack,     // Decode4
	character.TemporaryStatTypeDefenseState,      // Decode4
	character.TemporaryStatTypeWindWalk,          // flag only
	character.TemporaryStatTypeRepeatEffect,      // Decode4
	character.TemporaryStatTypeStopPortion,       // Decode4
	character.TemporaryStatTypeStopMotion,        // Decode4
	character.TemporaryStatTypeFear,              // Decode4
	character.TemporaryStatTypeMagicShield,       // Decode4
	character.TemporaryStatTypeFlying,            // flag only
	character.TemporaryStatTypeFrozen,            // Decode4
	character.TemporaryStatTypeSuddenDeath,       // Decode4
	character.TemporaryStatTypeFinalCut,          // Decode4
	character.TemporaryStatTypeCyclone,           // Decode1
	character.TemporaryStatTypeSneak,             // flag only
	character.TemporaryStatTypeWildDamageUp,      // flag only
	character.TemporaryStatTypeMechanic,          // Decode4 on v95 (registry NoOp — never originated)
	character.TemporaryStatTypeDarkAura,          // Decode4 on v95 (registry NoOp — never originated)
	character.TemporaryStatTypeBlueAura,          // Decode4 on v95 (registry NoOp — never originated)
	character.TemporaryStatTypeYellowAura,        // Decode4 on v95 (registry NoOp — never originated)
	character.TemporaryStatTypeBlessingArmor,     // flag only
}

// sortForeign orders stat types the way SecondaryStat::DecodeForRemote reads
// them: foreignReadOrder first, then anything that sequence does not name, by
// shift. EncodeForeign and DecodeForeign both use it, so the two stay in step.
//
// The shift-ordered tail is a safety net, not a wire contract. Every stat that
// carries foreign bytes today is named in foreignReadOrder — pinned by
// TestForeignReadOrderCoversEveryValueCarryingStat — so the tail only ever holds
// NoOp stats, which contribute nothing. Keeping it means a future stat that
// gains a foreign shape without being added to the list encodes in the wrong
// place rather than vanishing, and the test says which one.
func sortForeign(types []CharacterTemporaryStatType) {
	sort.Slice(types, func(i, j int) bool {
		ri, rj := foreignRank(types[i]), foreignRank(types[j])
		if ri != rj {
			return ri < rj
		}
		return types[i].Shift() < types[j].Shift()
	})
}

// foreignRank is a stat's position in foreignReadOrder, or one past the end for
// stats the client's remote sequence does not name — see sortForeign.
func foreignRank(st CharacterTemporaryStatType) int {
	if i, ok := foreignReadOrderIndex[st.name]; ok {
		return i
	}
	return len(foreignReadOrder)
}

var foreignReadOrderIndex = func() map[character.TemporaryStatType]int {
	m := make(map[character.TemporaryStatType]int, len(foreignReadOrder))
	for i, n := range foreignReadOrder {
		m[n] = i
	}
	return m
}()

func (m *CharacterTemporaryStat) EncodeForeign(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.EncodeMask(l, t, options)(w)

		keys := make([]CharacterTemporaryStatType, 0, len(m.stats))
		for _, v := range m.stats {
			if baseStatNames[v.statType.name] {
				continue // TwoState/base stats are encoded only as base stats below
			}
			keys = append(keys, v.statType)
		}
		sortForeign(keys)

		for _, k := range keys {
			m.stats[k.name].Write(w)
		}

		if legacyGmsMask(t) {
			// Pre-v61 foreign CTS (sub_5CBA1F) ends after the per-bit value
			// blocks — no nDefenseAtt/nDefenseState bytes, no trailing base-stat
			// blocks. SPAWN_PLAYER (sub_6BBC17 @0x6bbcde) and BuffGiveForeign
			// both consume exactly this shape.
			return w.Bytes()
		}

		w.WriteByte(0) // nDefenseAtt
		w.WriteByte(0) // nDefenseState

		baseTemporaryStats := m.getBaseTemporaryStats(t)
		for _, bts := range baseTemporaryStats {
			w.WriteByteArray(bts.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

var baseStatNames = map[character.TemporaryStatType]bool{
	character.TemporaryStatTypeEnergyCharge:  true,
	character.TemporaryStatTypeDashSpeed:     true,
	character.TemporaryStatTypeDashJump:      true,
	character.TemporaryStatTypeMonsterRiding: true,
	character.TemporaryStatTypeSpeedInfusion: true,
	character.TemporaryStatTypeHomingBeacon:  true,
	character.TemporaryStatTypeUndead:        true,
	character.TemporaryStatTypePartyBooster:  true, // v95 two-state member (replaces SpeedInfusion)
}

// twoStateKind is the wire shape of a two-state base-stat block. Tagging each
// group member with its kind lets EncodeMask, getBaseTemporaryStats, and
// decodeBaseTemporaryStats all drive off one ordered list instead of repeating a
// name→behaviour switch.
type twoStateKind int

const (
	twoStateDynamic       twoStateKind = iota // dynamic base block (15B): EnergyCharge, DashSpeed, DashJump, Undead
	twoStateMonsterRiding                     // non-dynamic base (13B): nOption=vehicle id, rOption=skill id
	twoStateSpeedInfusion                     // SpeedInfusion special block (20B)
	twoStateGuidedBullet                      // GuidedBullet special block (17B)
	twoStatePartyBooster                      // v95 PartyBooster block (20B: base 13 + tCurrentTime 5 + usExpireTerm 2)
)

type twoStateStat struct {
	name character.TemporaryStatType
	kind twoStateKind
}

// isGmsV61 gates GMS v61's structurally distinct two-state group: 6 members
// (no Undead slot — the client's decode/reset loop bound is hard-coded 6,
// IDA-confirmed three independent ways: SecondaryStat::DecodeForLocal's tail
// loop, DecodeForRemote's tail loop, and the constructor's allocation loop),
// with a 12-byte base block (3 plain 4-byte fields, no leading bool-prefixed
// time field) instead of the 13-byte pre-95 base every other in-scope
// version uses. task-167,
// docs/tasks/task-167-homing-beacon-bullseye/evidence/per-version/gms_v61_composition.md.
func isGmsV61(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorVersion() == 61
}

// twoStateBaseStats returns the two-state/base stat group this tenant's client
// knows, in the exact order it reads their trailing base-stat blocks. Membership
// and order are what this list fixes; whether a given member appears on the wire
// is decided per packet by presence (EncodeMask / getBaseTemporaryStats), not
// here. These stats are always encoded as base-stat blocks, never as per-stat
// value blocks.
//
// Three shapes, all IDA-verified, differing only in slots 5 and 7:
//
//	v72/v79/v83/v84/v87/v92/JMS  7 members, block sizes 15/15/15/13/20/17/15
//	GMS v95 (design.md §2.4)     6 members, block sizes 15/15/15/13/20/17
//	GMS v61 (isGmsV61, task-167) 6 members, block sizes 14/14/14/12/18/16
//
// v61's blocks are narrower because its base block — and SpeedInfusion's own
// extra field — drop the leading bool-prefixed time byte; see narrowTimeField.
// v95's trailer read is mask-gated per member (IDA @0x73DBA0).
func twoStateBaseStats(t tenant.Model) []twoStateStat {
	gmsV95Plus := t.IsRegion("GMS") && t.MajorAtLeast(95)

	// Slots 1-4 are the same stat, in the same order, on every version.
	stats := []twoStateStat{
		{character.TemporaryStatTypeEnergyCharge, twoStateDynamic},
		{character.TemporaryStatTypeDashSpeed, twoStateDynamic},
		{character.TemporaryStatTypeDashJump, twoStateDynamic},
		{character.TemporaryStatTypeMonsterRiding, twoStateMonsterRiding},
	}

	// Slot 5 is the one substitution in the group: v95 replaced SpeedInfusion
	// with PartyBooster. Both are 20-byte blocks of the same shape.
	if gmsV95Plus {
		stats = append(stats, twoStateStat{character.TemporaryStatTypePartyBooster, twoStatePartyBooster})
	} else {
		stats = append(stats, twoStateStat{character.TemporaryStatTypeSpeedInfusion, twoStateSpeedInfusion})
	}

	// Slot 6 is GuidedBullet everywhere.
	stats = append(stats, twoStateStat{character.TemporaryStatTypeHomingBeacon, twoStateGuidedBullet})

	// Slot 7, Undead, closes the classic group. Both 6-member versions drop it,
	// for unrelated reasons: v61's client loop bound is hard-coded to 6, and on
	// v95 its bit (128) would overflow the 128-bit mask.
	if !isGmsV61(t) && !gmsV95Plus {
		stats = append(stats, twoStateStat{character.TemporaryStatTypeUndead, twoStateDynamic})
	}

	return stats
}

func (m *CharacterTemporaryStat) DecodeMask(r *request.Reader, t tenant.Model) tool.Uint128 {
	if legacyGmsMask(t) {
		// Pre-v61 GMS: plain 8-byte little-endian mask (DecodeBuffer 8).
		return tool.Uint128{L: r.ReadUint64()}
	}
	h1 := uint64(r.ReadUint32()) << 32
	h2 := uint64(r.ReadUint32())
	l1 := uint64(r.ReadUint32()) << 32
	l2 := uint64(r.ReadUint32())
	return tool.Uint128{H: h1 | h2, L: l1 | l2}
}

func (m *CharacterTemporaryStat) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		mask := m.DecodeMask(r, t)
		reg := buildCharacterTemporaryStatRegistry(t)

		for _, st := range reg.inOrder {
			if mask.And(st.mask).IsZero() {
				continue
			}
			if baseStatNames[st.name] {
				// Base/TwoState stats carry no per-stat block; they are read by
				// decodeBaseTemporaryStats below. Skip regardless of version.
				continue
			}
			if legacyGmsMask(t) {
				// Pre-v61 local block: short value + int reason + short duration.
				value := r.ReadInt16()
				sourceId := r.ReadInt32()
				_ = r.ReadInt16() // duration units
				m.stats[st.name] = CharacterTemporaryStatValue{
					statType: st,
					sourceId: sourceId,
					value:    int32(value),
				}
				continue
			}
			value := r.ReadInt16()
			var sourceId int32
			var level byte
			if st.Disease() {
				sourceId = int32(r.ReadInt16())
				level = byte(r.ReadInt16())
			} else {
				sourceId = r.ReadInt32()
			}
			_ = r.ReadInt32() // expiresAt (relative ms)
			m.stats[st.name] = CharacterTemporaryStatValue{
				statType: st,
				sourceId: sourceId,
				level:    level,
				value:    int32(value),
			}
		}

		if legacyGmsMask(t) {
			// Pre-v61 local read stops after the value blocks (no defense bytes,
			// no base-stat blocks). The delay short + optional byte are consumed
			// by BuffGive.Decode.
			return
		}

		_ = r.ReadByte() // nDefenseAtt
		_ = r.ReadByte() // nDefenseState

		m.decodeBaseTemporaryStats(l, ctx, mask)(r, options)
	}
}

func (m *CharacterTemporaryStat) DecodeForeign(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		mask := m.DecodeMask(r, t)
		reg := buildCharacterTemporaryStatRegistry(t)

		keys := make([]CharacterTemporaryStatType, 0, len(reg.inOrder))
		for _, st := range reg.inOrder {
			if mask.And(st.mask).IsZero() || baseStatNames[st.name] {
				continue
			}
			keys = append(keys, st)
		}
		sortForeign(keys)

		for _, st := range keys {
			m.stats[st.name] = st.foreignValueReader(r, st)
		}

		if legacyGmsMask(t) {
			// Pre-v61 foreign read (sub_5CBA1F) stops after the value blocks.
			return
		}

		_ = r.ReadByte() // nDefenseAtt
		_ = r.ReadByte() // nDefenseState

		m.decodeBaseTemporaryStats(l, ctx, mask)(r, options)
	}
}

func (m *CharacterTemporaryStat) decodeBaseTemporaryStats(l logrus.FieldLogger, ctx context.Context, mask tool.Uint128) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		// Mirror getBaseTemporaryStats exactly (same version-specific group, same
		// order, same presence gate) so the bytes consumed match the bytes
		// emitted, boundary-for-boundary. The gate is the mask: a base block is
		// on the wire only when its bit is set.
		reg := buildCharacterTemporaryStatRegistry(t)
		narrow := isGmsV61(t)
		for _, bs := range twoStateBaseStats(t) {
			st, known := reg.byName[bs.name]
			if !known || mask.And(st.mask).IsZero() {
				continue
			}
			switch bs.kind {
			case twoStateSpeedInfusion, twoStatePartyBooster:
				si := SpeedInfusionTemporaryStat{CharacterTemporaryStatBase: CharacterTemporaryStatBase{bDynamicTermSet: false, narrowTimeField: narrow}}
				si.Decode(l, ctx)(r, options)
			case twoStateGuidedBullet:
				gb := GuidedBulletTemporaryStat{CharacterTemporaryStatBase: CharacterTemporaryStatBase{bDynamicTermSet: false, narrowTimeField: narrow}}
				gb.Decode(l, ctx)(r, options)
			case twoStateMonsterRiding:
				base := CharacterTemporaryStatBase{bDynamicTermSet: false, narrowTimeField: narrow}
				base.Decode(l, ctx)(r, options)
			default: // twoStateDynamic
				base := CharacterTemporaryStatBase{bDynamicTermSet: true, narrowTimeField: narrow}
				base.Decode(l, ctx)(r, options)
			}
		}
	}
}

// getBaseTemporaryStats returns one base-stat block per two-state stat this CTS
// actually holds, in the client's read order. Presence-gated to match
// EncodeMask: the client reads one block per SET base bit, so emitting a block
// whose bit is clear would leave unread bytes and desync the tail, and setting a
// bit with no block would run it off the end.
//
// Absent base stats are simply skipped. They used to be emitted as empty
// placeholder blocks alongside an unconditional mask bit — self-consistent, but
// it made every buff packet claim a mount and a guided bullet. See EncodeMask.
func (m *CharacterTemporaryStat) getBaseTemporaryStats(t tenant.Model) []packet.Encoder {
	list := make([]packet.Encoder, 0)
	narrow := isGmsV61(t)
	for _, bs := range twoStateBaseStats(t) {
		s, ok := m.stats[bs.name]
		if !ok {
			continue
		}
		switch bs.kind {
		case twoStateMonsterRiding:
			// Monster Riding: nOption = vehicle/taming-mob item id, rOption = source
			// skill id. Wire contract IDA-confirmed — context.md §2, design.md §1.1.
			list = append(list, NewCharacterTemporaryStatBaseWithOptions(false, s.Value(), s.SourceId(), narrow))
		case twoStateSpeedInfusion:
			list = append(list, NewSpeedInfusionTemporaryStat(narrow)) // 20 (18 on GMS v61)
		case twoStateGuidedBullet:
			// GuidedBullet / HOMING_BEACON: nOption = locked monster object id
			// (allocator range guarantees nonzero — IsActivated gate), rOption =
			// source skill id (SetGuided reason + icon), dwMobId = monster object
			// id. design.md §5.5.1.
			list = append(list, NewGuidedBulletTemporaryStatWithOptions(s.Value(), s.SourceId(), uint32(s.Value()), narrow)) // 17 (16 on GMS v61)
		case twoStatePartyBooster:
			// v95 PartyBooster (bit 126): 20-byte block, same wire shape as the
			// pre-95 SpeedInfusion block (base + tCurrentTime + usExpireTerm),
			// IDA DecodeForClient @0x72C600.
			list = append(list, SpeedInfusionTemporaryStat{
				CharacterTemporaryStatBase: CharacterTemporaryStatBase{
					bDynamicTermSet: false,
					nOption:         s.Value(),
					rOption:         s.SourceId(),
					tLastUpdated:    time.Now().Unix(),
					narrowTimeField: narrow,
				},
			})
		default: // twoStateDynamic
			// ENERGY_CHARGE's nOption IS the client's energy-bar reading:
			// GMS v83 sub_7F9BAD computes the fill as this[364]/this[365],
			// where this[364] is the block's first int32 (task-216
			// design.md §1.1). rOption carries the source skill id, matching
			// every other populated two-state block.
			//
			// The group's other dynamic members (DASH_SPEED, DASH_JUMP,
			// UNDEAD) keep the zeroed block deliberately: no evidence was
			// gathered for what their clients read, and their matrix cells
			// are already verified against the zeros.
			if bs.name == character.TemporaryStatTypeEnergyCharge {
				list = append(list, NewCharacterTemporaryStatBaseWithOptions(true, s.Value(), s.SourceId(), narrow)) // 15 (14 on GMS v61)
				continue
			}
			list = append(list, NewCharacterTemporaryStatBase(true, narrow)) // dynamic, 15 (14 on GMS v61)
		}
	}
	return list
}
