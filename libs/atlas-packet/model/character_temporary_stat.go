package model

import (
	"context"
	"errors"
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
	newAndIncDiseased(character.TemporaryStatTypeStun)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypePoison)(ValueSourceLevelForeignValueWriter, ValueSourceLevelForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeSeal)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeDarkness)(ValueAsIntForeignValueWriter, IntForeignValueReader)
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
	newAndIncDiseased(character.TemporaryStatTypeWeaken)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeCurse)(ValueAsIntForeignValueWriter, IntForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeSlow)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMorph)(ValueAsShortForeignValueWriter, ShortForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeRecovery)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMapleWarrior)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeStance)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSharpEyes)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeManaReflection)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncDiseased(character.TemporaryStatTypeSeduce)(LevelSourceForeignValueWriter, LevelSourceForeignValueReader)
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
	newAndIncDiseased(character.TemporaryStatTypeConfuse)(LevelSourceForeignValueWriter, LevelSourceForeignValueReader)
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

func ValueSourceLevelForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(int16(v.Value()))
		w.WriteInt16(int16(v.Level()))
		w.WriteInt16(int16(v.SourceId()))
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

func ValueSourceLevelForeignValueReader(r *request.Reader, st CharacterTemporaryStatType) CharacterTemporaryStatValue {
	value := int32(r.ReadInt16())
	level := byte(r.ReadInt16())
	sourceId := int32(r.ReadInt16())
	return CharacterTemporaryStatValue{statType: st, value: value, level: level, sourceId: sourceId}
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

func (m *CharacterTemporaryStat) AddStat(l logrus.FieldLogger) func(t tenant.Model) func(n string, sourceId int32, amount int32, level byte, expiresAt time.Time) {
	return func(t tenant.Model) func(n string, sourceId int32, amount int32, level byte, expiresAt time.Time) {
		return func(n string, sourceId int32, amount int32, level byte, expiresAt time.Time) {
			name := character.TemporaryStatType(n)
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

func (m *CharacterTemporaryStat) EncodeMask(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		reg := buildCharacterTemporaryStatRegistry(t)
		mask := tool.Uint128{}
		// The TwoState/base stats are always present and always encoded as base-stat
		// blocks (see getBaseTemporaryStats), so their mask bits are set
		// unconditionally. The registry's per-version shift already places them where
		// the client reads them: on v83 RideVehicle is shift 85 -> wire bytes 4-7,
		// matching SecondaryStat::DecodeForLocal's flag 1<<(i+82) (IDA @0x781D0E). No
		// version-specific mask placement is needed.
		for _, bs := range twoStateBaseStats(t) {
			if bs.conditional {
				// Conditional members' bits are set by the active-stats loop
				// below only when the stat is present (v95 mask-gated read).
				continue
			}
			if st, ok := reg.byName[bs.name]; ok {
				mask = mask.Or(st.mask)
			}
		}

		if isGmsV61(t) {
			// GMS v61 has no Undead two-state slot (task-167) - twoStateBaseStats
			// omits it from the 6-member block list above so no base-stat block is
			// written for it. The registry still assigns Undead a mask bit for
			// wire-numbering continuity with the pre-95 scheme, and existing
			// BuffGive/BuffCancel fixtures pin that bit as always set on v61; this
			// is harmless (the v61 client's two-state loop bound is hard-coded to 6
			// and never tests a bit outside that range), so it is preserved here
			// rather than treated as a block-membership signal.
			if st, ok := reg.byName[character.TemporaryStatTypeUndead]; ok {
				mask = mask.Or(st.mask)
			}
		}

		for _, v := range m.stats {
			mask = mask.Or(v.statType.mask)
		}

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

func writeMask(w *response.Writer, mask tool.Uint128) {
	w.WriteInt(uint32(mask.H >> 32))
	w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
	w.WriteInt(uint32(mask.L >> 32))
	w.WriteInt(uint32(mask.L & 0xFFFFFFFF))
}

// CancelMask returns the mask of ONLY the stats present on this CTS — never
// the unconditional two-state group bits EncodeMask always sets for the give
// shape. Cancel packets must use this instead: the client's
// TemporaryStatReset clears EVERY masked stat (v83 @0xA2071F, v95 @0x9F2AB0),
// so a cancel carrying the unconditional two-state bits (RideVehicle,
// DashSpeed/DashJump, SpeedInfusion/PartyBooster, HomingBeacon, Undead)
// destroys any active mount/dash/energy-charge/beacon even when the caller
// only meant to cancel one unrelated buff (design.md §3 F1).
func (m *CharacterTemporaryStat) CancelMask(t tenant.Model) tool.Uint128 {
	mask := tool.Uint128{}
	for _, v := range m.stats {
		mask = mask.Or(v.statType.mask)
	}
	return mask
}

// EncodeCancelMask writes CancelMask in the same wire layout EncodeMask uses
// for its mask (8-byte legacy pre-v61 GMS / 16-byte UINT128 everywhere else)
// — only the bits differ, never the width, so DecodeMask (shared by both
// paths) stays a single implementation.
func (m *CharacterTemporaryStat) EncodeCancelMask(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		mask := m.CancelMask(t)
		if legacyGmsMask(t) {
			w.WriteLong(mask.L)
			return
		}
		writeMask(w, mask)
	}
}

// movementAffectingStatNames is the version-gated mirror of the client's
// movement filter: the reset/give trailing byte is read ONLY when the
// packet's mask intersects this set. Evidence:
// docs/tasks/task-167-homing-beacon-bullseye/evidence/movement-filter.md
// (v83 sub_77DC78; v61 0x660B44; v72 0x6c87b6; v79 0x6f852f; v84 sub_7a07e7;
// v87 sub_7cc3e2; v92 sub_705080; v95 SecondaryStat::IsMovementAffectingStat
// @0x7208C0; JMS sub_7f76d1).
//
// v61/v72/v79/v83 all test the identical 12-constant shape (v61 fully
// name-resolved; v72/v79 confirmed by count+structure with some individual
// names positional-only — see the evidence file's per-version caveats).
//
// v92's own filter is confirmed to DIFFER (13 constants at different raw
// shifts, and notably does NOT include shift 7/Speed — evidence: gms_v92.md)
// but not one of its 13 bits is name-resolved in that IDB, so per the
// "do not invent a mapping" rule no v92-specific list can be constructed
// here. v92 falls through to the base 12-name list below (the pre-existing
// assumption) pending a dedicated v92 audit; all 13 raw bits are reported
// unmapped in task-8-report.md. Same fallthrough applies to any GMS version
// with no dedicated evidence (v28, v48, v86).
func movementAffectingStatNames(t tenant.Model) []character.TemporaryStatType {
	if t.Region() == "JMS" {
		// JMS v185 (sub_7f76d1) tests a wholly different 13-constant set —
		// NOT "the v83 list plus/minus extras". Only Stun, GhostMorph, and
		// MonsterRiding(RideVehicle) overlap v83's 12-stat meaning; the
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

	switch {
	case t.Region() == "GMS" && t.MajorVersion() == 84:
		// v84 (sub_7a07e7) tests 14 constants, not 12: the 12 above plus two
		// raw shift-82/83 constants that are NOT independently name-resolved
		// in v84's own IDB — no dedicated OnTemporaryStatReset side-effect
		// block references either one individually (evidence: gms_v84.md).
		// v87 independently resolves the identically-positioned raw 82/83
		// constants as Flying/Frozen, and cross-version corroboration
		// (two-state-group-per-version.md) places v84's two new constants
		// at that same slot. Included here anyway even though v84's own
		// evidence never names them: the failure mode is one-directional —
		// if v84's client really does gate the trailing byte on
		// Flying/Frozen and this list omitted them, a cancel of either stat
		// would silently drop the movement byte the client expects and the
		// packet would desync; including them when the client doesn't check
		// costs nothing.
		names = append(names,
			character.TemporaryStatTypeFlying,
			character.TemporaryStatTypeFrozen,
		)
	case t.Region() == "GMS" && t.MajorVersion() == 87:
		// v87 (sub_7cc3e2) independently resolves Flying(82)/Frozen(83)
		// inside its own IDB, cross-checked bit-for-bit against the atlas
		// registry's MajorAtLeast(87) block (evidence: gms_v87.md).
		names = append(names,
			character.TemporaryStatTypeFlying,
			character.TemporaryStatTypeFrozen,
		)
	case t.Region() == "GMS" && t.MajorVersion() >= 95:
		// v95 (SecondaryStat::IsMovementAffectingStat @0x7208C0) resolves
		// all 15 constants by symbol name: the base 12 plus Flying, Frozen,
		// and YellowAura (design.md §2.4).
		names = append(names,
			character.TemporaryStatTypeFlying,
			character.TemporaryStatTypeFrozen,
			character.TemporaryStatTypeYellowAura,
		)
	}

	return names
}

// MovementAffectingMask returns the movement filter as a mask for this
// tenant's registry layout. Writers AND their packet mask against it to
// decide whether the client will read the trailing movement byte. A name
// with no entry in this tenant's registry (e.g. Flying/Frozen on a version
// whose registry doesn't allocate them) is silently skipped — see
// movementAffectingStatNames.
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
			et := int32(v.ExpiresAt().Sub(time.Now()).Milliseconds())
			w.WriteInt32(et)
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

// legacyDurationUnits converts an absolute expiry into the pre-v61 wire duration
// short: the v48 client reads Decode2 and multiplies by 500 (sub_5CA524 @0x5ca58d
// `500 * Decode2`), so the wire carries remaining-ms / 500.
func legacyDurationUnits(expiresAt time.Time) int16 {
	ms := expiresAt.Sub(time.Now()).Milliseconds()
	if ms <= 0 {
		return 0
	}
	return int16(ms / 500)
}

func (m *CharacterTemporaryStat) EncodeForeign(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.EncodeMask(l, t, options)(w)

		keys := make([]CharacterTemporaryStatType, 0)
		for _, v := range m.stats {
			if baseStatNames[v.statType.name] {
				continue // TwoState/base stats are encoded only as base stats below
			}
			keys = append(keys, v.statType)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Shift() < keys[j].Shift()
		})

		sortedValues := make([]CharacterTemporaryStatValue, 0)
		for _, v := range keys {
			sortedValues = append(sortedValues, m.stats[v.name])
		}

		for _, v := range sortedValues {
			v.Write(w)
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
	// conditional members (v95 PartyBooster/GuidedBullet) set their mask bit and
	// write their block ONLY when the stat is active. The v95 client's two-state
	// trailer read is mask-gated per member (IDA @0x73DBA0), so absent members are
	// simply skipped; pre-95 clients read all 7 blocks unconditionally, so pre-95
	// members are never conditional. design.md §2.4/§4.4.
	conditional bool
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
	return t.Region() == "GMS" && t.MajorVersion() == 61
}

// twoStateBaseStats returns the two-state/base stat group for this tenant, in the
// exact order the client reads their trailing base-stat blocks. These stats are
// always encoded as base-stat blocks (never per-stat value blocks). v72/v79/v83/
// v84/v87/v92/JMS use the classic 7-member group, whose 7 members are all
// unconditional (mask bit always set, block always written) because pre-95
// clients read all 7 blocks unconditionally.
//
// GMS v95's two-state group is IDA-verified as 6 members (design.md §2.4):
// EnergyCharge(122)/DashSpeed(123)/DashJump(124)/RideVehicle(125) stay
// unconditional (status quo, fixture-locked), and PartyBooster(126,
// twoStatePartyBooster, 20B)/GuidedBullet(127, twoStateGuidedBullet, 17B) are
// conditional — the v95 client's trailer read is mask-gated per member (IDA
// @0x73DBA0), so these two only appear when active. Undead has no v95 wire slot
// (bit 128 would overflow the 128-bit mask). Block sizes 15/15/15/13/20/17.
//
// GMS v61 is a second, independently-verified 6-member group (isGmsV61,
// task-167): same 4 leading members plus SpeedInfusion and GuidedBullet
// (both unconditional, unlike v95's conditional pair), no Undead slot. Block
// sizes are 14/14/14/12/18/16 — narrower than pre-95 because the base block
// (and SpeedInfusion's own extra field) drop the leading bool-prefixed time
// byte; see narrowTimeField.
func twoStateBaseStats(t tenant.Model) []twoStateStat {
	stats := []twoStateStat{
		{character.TemporaryStatTypeEnergyCharge, twoStateDynamic, false},
		{character.TemporaryStatTypeDashSpeed, twoStateDynamic, false},
		{character.TemporaryStatTypeDashJump, twoStateDynamic, false},
		{character.TemporaryStatTypeMonsterRiding, twoStateMonsterRiding, false},
	}
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		// v95 verified 6-member group (design.md §2.4): the 4 unconditional
		// members above stay always-written (status quo, fixture-locked);
		// PartyBooster(126) and GuidedBullet(127) are conditional. Undead has
		// no v95 wire slot (bit 128 overflows the mask).
		return append(stats,
			twoStateStat{character.TemporaryStatTypePartyBooster, twoStatePartyBooster, true},
			twoStateStat{character.TemporaryStatTypeHomingBeacon, twoStateGuidedBullet, true},
		)
	}
	if isGmsV61(t) {
		// GMS v61 verified 6-member group (task-167): no Undead slot. Member
		// order/kinds otherwise match the pre-95 shape; block sizes differ via
		// the narrowTimeField base threaded in by getBaseTemporaryStats /
		// decodeBaseTemporaryStats.
		return append(stats,
			twoStateStat{character.TemporaryStatTypeSpeedInfusion, twoStateSpeedInfusion, false},
			twoStateStat{character.TemporaryStatTypeHomingBeacon, twoStateGuidedBullet, false},
		)
	}
	return append(stats,
		twoStateStat{character.TemporaryStatTypeSpeedInfusion, twoStateSpeedInfusion, false},
		twoStateStat{character.TemporaryStatTypeHomingBeacon, twoStateGuidedBullet, false},
		twoStateStat{character.TemporaryStatTypeUndead, twoStateDynamic, false},
	)
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

		m.decodeBaseTemporaryStats(l, ctx)(r, options, mask)
	}
}

func (m *CharacterTemporaryStat) DecodeForeign(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		mask := m.DecodeMask(r, t)
		reg := buildCharacterTemporaryStatRegistry(t)

		for _, st := range reg.inOrder {
			if mask.And(st.mask).IsZero() {
				continue
			}
			if baseStatNames[st.name] {
				continue
			}
			v := st.foreignValueReader(r, st)
			m.stats[st.name] = v
		}

		if legacyGmsMask(t) {
			// Pre-v61 foreign read (sub_5CBA1F) stops after the value blocks.
			return
		}

		_ = r.ReadByte() // nDefenseAtt
		_ = r.ReadByte() // nDefenseState

		m.decodeBaseTemporaryStats(l, ctx)(r, options, mask)
	}
}

func (m *CharacterTemporaryStat) decodeBaseTemporaryStats(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}, mask tool.Uint128) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}, mask tool.Uint128) {
		reg := buildCharacterTemporaryStatRegistry(t)
		narrow := isGmsV61(t)
		// Mirror getBaseTemporaryStats exactly (same version-specific group + order)
		// so the bytes consumed match the bytes emitted, boundary-for-boundary.
		for _, bs := range twoStateBaseStats(t) {
			if bs.conditional {
				st, ok := reg.byName[bs.name]
				if !ok || mask.And(st.mask).IsZero() {
					continue
				}
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

func (m *CharacterTemporaryStat) getBaseTemporaryStats(t tenant.Model) []packet.Encoder {
	list := make([]packet.Encoder, 0)
	narrow := isGmsV61(t)
	for _, bs := range twoStateBaseStats(t) {
		if bs.conditional {
			if _, ok := m.stats[bs.name]; !ok {
				continue
			}
		}
		switch bs.kind {
		case twoStateMonsterRiding:
			// Monster Riding: nOption = vehicle/taming-mob item id, rOption = source
			// skill id. Wire contract IDA-confirmed — context.md §2, design.md §1.1.
			if s, ok := m.stats[bs.name]; ok {
				list = append(list, NewCharacterTemporaryStatBaseWithOptions(false, s.Value(), s.SourceId(), narrow))
			} else {
				list = append(list, NewCharacterTemporaryStatBase(false, narrow)) // 13 (12 on GMS v61)
			}
		case twoStateSpeedInfusion:
			list = append(list, NewSpeedInfusionTemporaryStat(narrow)) // 20 (18 on GMS v61)
		case twoStateGuidedBullet:
			// GuidedBullet / HOMING_BEACON: nOption = locked monster object id
			// (allocator range guarantees nonzero — IsActivated gate), rOption =
			// source skill id (SetGuided reason + icon), dwMobId = monster object
			// id. Absent stat -> empty block, byte-identical to the pre-task
			// encode. design.md §5.5.1.
			if s, ok := m.stats[bs.name]; ok {
				list = append(list, NewGuidedBulletTemporaryStatWithOptions(s.Value(), s.SourceId(), uint32(s.Value()), narrow))
			} else {
				list = append(list, NewGuidedBulletTemporaryStat(narrow)) // 17 (16 on GMS v61)
			}
		case twoStatePartyBooster:
			// v95 PartyBooster (bit 126): 20-byte block, same wire shape as the
			// pre-95 SpeedInfusion block (base + tCurrentTime + usExpireTerm),
			// IDA DecodeForClient @0x72C600. Reached only when active
			// (conditional member).
			s := m.stats[bs.name]
			list = append(list, SpeedInfusionTemporaryStat{
				CharacterTemporaryStatBase: CharacterTemporaryStatBase{
					bDynamicTermSet: false,
					nOption:         s.Value(),
					rOption:         s.SourceId(),
					tLastUpdated:    time.Now().Unix(),
				},
			})
		default: // twoStateDynamic
			list = append(list, NewCharacterTemporaryStatBase(true, narrow)) // dynamic, 15 (14 on GMS v61)
		}
	}
	return list
}
