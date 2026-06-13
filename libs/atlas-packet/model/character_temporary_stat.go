package model

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-packet/tool"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
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

	// GMS v95 has a wholly different post-SoulStone region (bits 82-121) and
	// two-state group from v83/v87 — IDA-verified from the v95 client's CTS flag
	// initializers (1<<N per stat); see
	// docs/tasks/task-086-mount-system/v95_secondarystat_table.md. The 40 stats
	// below put MonsterRiding/RideVehicle at bit 125 exactly where the v95 client
	// reads it. Early-return so the v83/v87/JMS appends below don't also run.
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		// bits 82-121. Foreign shapes mirror the existing registry where a stat is
		// already known; the v95-only additions use NoOp (atlas never originates
		// them — the entry exists to reserve the client's mask bit).
		newAndIncNonDiseased(character.TemporaryStatTypeFlying)(NoOpForeignValueWriter, NoOpForeignValueReader)               // 82
		newAndIncNonDiseased(character.TemporaryStatTypeFrozen)(ValueAsIntForeignValueWriter, IntForeignValueReader)         // 83
		newAndIncNonDiseased(character.TemporaryStatTypeAssistCharge)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 84
		newAndIncNonDiseased(character.TemporaryStatTypeEnrage)(NoOpForeignValueWriter, NoOpForeignValueReader)              // 85
		newAndIncNonDiseased(character.TemporaryStatTypeSuddenDeath)(ValueAsIntForeignValueWriter, IntForeignValueReader)    // 86
		newAndIncNonDiseased(character.TemporaryStatTypeNotDamaged)(NoOpForeignValueWriter, NoOpForeignValueReader)          // 87
		newAndIncNonDiseased(character.TemporaryStatTypeFinalCut)(ValueAsIntForeignValueWriter, IntForeignValueReader)       // 88
		newAndIncNonDiseased(character.TemporaryStatTypeThornsEffect)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 89
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowAttackDamage)(NoOpForeignValueWriter, NoOpForeignValueReader) // 90
		newAndIncNonDiseased(character.TemporaryStatTypeWildDamageUp)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 91
		newAndIncNonDiseased(character.TemporaryStatTypeMine)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 92
		newAndIncNonDiseased(character.TemporaryStatTypeEMHP)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 93
		newAndIncNonDiseased(character.TemporaryStatTypeEMMP)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 94
		newAndIncNonDiseased(character.TemporaryStatTypeEPAD)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 95
		newAndIncNonDiseased(character.TemporaryStatTypeEPPD)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 96
		newAndIncNonDiseased(character.TemporaryStatTypeEMDD)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 97
		newAndIncNonDiseased(character.TemporaryStatTypeGuard)(NoOpForeignValueWriter, NoOpForeignValueReader)               // 98
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyDamage)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 99
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyAbsorb)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 100
		newAndIncNonDiseased(character.TemporaryStatTypeCyclone)(ValueAsByteForeignValueWriter, ByteForeignValueReader)      // 101
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowCritical)(NoOpForeignValueWriter, NoOpForeignValueReader)     // 102
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowMaxMP)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 103
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowDefense)(NoOpForeignValueWriter, NoOpForeignValueReader)      // 104
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowEvasion)(NoOpForeignValueWriter, NoOpForeignValueReader)      // 105
		newAndIncNonDiseased(character.TemporaryStatTypeConversion)(NoOpForeignValueWriter, NoOpForeignValueReader)          // 106
		newAndIncNonDiseased(character.TemporaryStatTypeRevive)(NoOpForeignValueWriter, NoOpForeignValueReader)              // 107
		newAndIncNonDiseased(character.TemporaryStatTypeSneak)(NoOpForeignValueWriter, NoOpForeignValueReader)               // 108
		newAndIncNonDiseased(character.TemporaryStatTypeMechanic)(NoOpForeignValueWriter, NoOpForeignValueReader)            // 109
		newAndIncNonDiseased(character.TemporaryStatTypeAura)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 110
		newAndIncNonDiseased(character.TemporaryStatTypeDarkAura)(NoOpForeignValueWriter, NoOpForeignValueReader)            // 111
		newAndIncNonDiseased(character.TemporaryStatTypeBlueAura)(NoOpForeignValueWriter, NoOpForeignValueReader)            // 112
		newAndIncNonDiseased(character.TemporaryStatTypeYellowAura)(NoOpForeignValueWriter, NoOpForeignValueReader)          // 113
		newAndIncNonDiseased(character.TemporaryStatTypeSuperBody)(NoOpForeignValueWriter, NoOpForeignValueReader)           // 114
		newAndIncNonDiseased(character.TemporaryStatTypeWildMaxHpUp)(NoOpForeignValueWriter, NoOpForeignValueReader)         // 115
		newAndIncNonDiseased(character.TemporaryStatTypeDice)(NoOpForeignValueWriter, NoOpForeignValueReader)                // 116
		newAndIncNonDiseased(character.TemporaryStatTypeBlessingArmor)(NoOpForeignValueWriter, NoOpForeignValueReader)       // 117
		newAndIncNonDiseased(character.TemporaryStatTypeDamageReduce)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 118
		newAndIncNonDiseased(character.TemporaryStatTypeTeleportMastery)(NoOpForeignValueWriter, NoOpForeignValueReader)     // 119
		newAndIncNonDiseased(character.TemporaryStatTypeCombatOrders)(NoOpForeignValueWriter, NoOpForeignValueReader)        // 120
		newAndIncNonDiseased(character.TemporaryStatTypeBeholder)(NoOpForeignValueWriter, NoOpForeignValueReader)            // 121
		// two-state group, bits 122-128. PartyBooster (126) replaces v83's
		// SpeedInfusion; Undead (128) overflows the 128-bit mask on v95 and is not
		// receivable, but the entry preserves ordering parity with the client.
		newAndIncNonDiseased(character.TemporaryStatTypeEnergyCharge)(NoOpForeignValueWriter, NoOpForeignValueReader)  // 122
		newAndIncNonDiseased(character.TemporaryStatTypeDashSpeed)(NoOpForeignValueWriter, NoOpForeignValueReader)     // 123
		newAndIncNonDiseased(character.TemporaryStatTypeDashJump)(NoOpForeignValueWriter, NoOpForeignValueReader)      // 124
		newAndIncNonDiseased(character.TemporaryStatTypeMonsterRiding)(NoOpForeignValueWriter, NoOpForeignValueReader) // 125
		newAndIncNonDiseased(character.TemporaryStatTypePartyBooster)(NoOpForeignValueWriter, NoOpForeignValueReader)  // 126
		newAndIncNonDiseased(character.TemporaryStatTypeHomingBeacon)(NoOpForeignValueWriter, NoOpForeignValueReader)  // 127
		newAndIncDiseased(character.TemporaryStatTypeUndead)(NoOpForeignValueWriter, NoOpForeignValueReader)           // 128
		return characterTemporaryStatRegistry{byName: set, inOrder: ordered}
	}

	// v87+ GMS and JMS have the first 4 stats of the post-SoulStone block.
	// Verified for v87 against the client SecondaryStat::Reset disassembly
	// (86 UINT128 mask groups covering Atlas bits 0..85 — see
	// https://github.com/Chronicle20/gms-83-dll docs/tasks/cwvscontext-port/v87_secondarystat_reset_mapping.md).
	if (t.Region() == "GMS" && t.MajorVersion() >= 87) || t.Region() == "JMS" {
		newAndIncNonDiseased(character.TemporaryStatTypeFlying)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeFrozen)(ValueAsIntForeignValueWriter, IntForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeAssistCharge)(NoOpForeignValueWriter, NoOpForeignValueReader)
		newAndIncNonDiseased(character.TemporaryStatTypeMirrorImage)(NoOpForeignValueWriter, NoOpForeignValueReader)
	}
	// JMS has the remaining 24 stats (SuddenDeath through Unknown). GMS v95 is handled
	// by its own dedicated enumeration above (early return); v87 lacks these (CTS_SuddenDeath
	// etc. absent from the v87 client; including them shifts the bitmask layout and breaks
	// GIVE_BUFF). This branch is now JMS-only.
	if t.Region() == "JMS" {
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

		newAndIncNonDiseased(character.TemporaryStatTypeUnknown)(NoOpForeignValueWriter, NoOpForeignValueReader)
	}
	newAndIncNonDiseased(character.TemporaryStatTypeEnergyCharge)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDashSpeed)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeDashJump)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeMonsterRiding)(NoOpForeignValueWriter, NoOpForeignValueReader)
	newAndIncNonDiseased(character.TemporaryStatTypeSpeedInfusion)(NoOpForeignValueWriter, NoOpForeignValueReader)
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
}

func NewCharacterTemporaryStatBase(bDynamicTermSet bool) CharacterTemporaryStatBase {
	return CharacterTemporaryStatBase{
		tLastUpdated:    time.Now().Unix(),
		bDynamicTermSet: bDynamicTermSet,
	}
}

func NewCharacterTemporaryStatBaseWithOptions(bDynamicTermSet bool, nOption int32, rOption int32) CharacterTemporaryStatBase {
	return CharacterTemporaryStatBase{
		tLastUpdated:    time.Now().Unix(),
		bDynamicTermSet: bDynamicTermSet,
		nOption:         nOption,
		rOption:         rOption,
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
		writeTime(m.tLastUpdated)(w)
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
		m.tLastUpdated = readTime(r)
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
		writeTime(int64(m.tCurrentTime))(w)
		w.WriteInt16(m.usExpireItem)
		return w.Bytes()
	}
}

func (m *SpeedInfusionTemporaryStat) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.CharacterTemporaryStatBase.Decode(l, ctx)(r, options)
		m.tCurrentTime = int32(readTime(r))
		m.usExpireItem = r.ReadInt16()
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
	stats map[character.TemporaryStatType]CharacterTemporaryStatValue
}

func NewCharacterTemporaryStat() *CharacterTemporaryStat {
	return &CharacterTemporaryStat{
		stats: make(map[character.TemporaryStatType]CharacterTemporaryStatValue),
	}
}

// HasDisease reports whether any stat held by this CTS is a mob-applied
// disease. Used by BuffGive to pick the correct trailer bytes — diseases
// require Cosmic's giveDebuff trailer (Short delay=900, Byte apply=1) for
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

func (m *CharacterTemporaryStat) EncodeMask(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		mask := tool.Uint128{}
		// The 7 TwoState/base stats (EnergyCharge, DashSpeed, DashJump, RideVehicle/
		// MonsterRiding, SpeedInfusion, GuidedBullet, Undead) are always present and
		// always encoded as base-stat blocks (see getBaseTemporaryStats), so their mask
		// bits are set unconditionally. The registry's per-version shift assignment
		// already places them where the client reads them: on v83 RideVehicle is shift
		// 85 -> wire bytes 4-7, matching SecondaryStat::DecodeForLocal's flag 1<<(i+82)
		// (IDA @0x781D0E). No version-specific mask placement is needed.
		applyMask := func(name character.TemporaryStatType) {
			if val, err := CharacterTemporaryStatTypeByName(t)(name); err == nil {
				mask = mask.Or(val.mask)
			}
		}
		for _, name := range twoStateBaseStats(t) {
			applyMask(name)
		}

		for _, v := range m.stats {
			mask = mask.Or(v.statType.mask)
		}

		w.WriteInt(uint32(mask.H >> 32))
		w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
		w.WriteInt(uint32(mask.L >> 32))
		w.WriteInt(uint32(mask.L & 0xFFFFFFFF))
	}
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

		w.WriteByte(0) // nDefenseAtt
		w.WriteByte(0) // nDefenseState

		var baseTemporaryStats = m.getBaseTemporaryStats(t)
		for _, bts := range baseTemporaryStats {
			w.WriteByteArray(bts.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
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

		w.WriteByte(0) // nDefenseAtt
		w.WriteByte(0) // nDefenseState

		var baseTemporaryStats = m.getBaseTemporaryStats(t)
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

// twoStateBaseStats returns the two-state/base stat group for this tenant, in the
// exact order the client reads their trailing base-stat blocks. These stats are
// always encoded as base-stat blocks (never per-stat value blocks) and their mask
// bits are set unconditionally. v83/v84/v87/JMS use the classic 7-member group.
//
// GMS v95 differs: its two-state group is EnergyCharge, DashSpeed, DashJump,
// RideVehicle(125), PartyBooster(126), GuidedBullet(127), Undead(128) — but Undead
// overflows the 128-bit mask and PartyBooster/GuidedBullet base-stat wire sizes are
// not yet IDA-verified (Task 41b). The client reads base blocks per set mask bit,
// sequentially, so we emit only the verified leading four (EnergyCharge..RideVehicle);
// RideVehicle/MonsterRiding is the mount payload and renders correctly as slot 3.
func twoStateBaseStats(t tenant.Model) []character.TemporaryStatType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return []character.TemporaryStatType{
			character.TemporaryStatTypeEnergyCharge,
			character.TemporaryStatTypeDashSpeed,
			character.TemporaryStatTypeDashJump,
			character.TemporaryStatTypeMonsterRiding,
		}
	}
	return []character.TemporaryStatType{
		character.TemporaryStatTypeEnergyCharge,
		character.TemporaryStatTypeDashSpeed,
		character.TemporaryStatTypeDashJump,
		character.TemporaryStatTypeMonsterRiding,
		character.TemporaryStatTypeSpeedInfusion,
		character.TemporaryStatTypeHomingBeacon,
		character.TemporaryStatTypeUndead,
	}
}

func (m *CharacterTemporaryStat) DecodeMask(r *request.Reader) tool.Uint128 {
	h1 := uint64(r.ReadUint32()) << 32
	h2 := uint64(r.ReadUint32())
	l1 := uint64(r.ReadUint32()) << 32
	l2 := uint64(r.ReadUint32())
	return tool.Uint128{H: h1 | h2, L: l1 | l2}
}

func (m *CharacterTemporaryStat) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		mask := m.DecodeMask(r)
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

		_ = r.ReadByte() // nDefenseAtt
		_ = r.ReadByte() // nDefenseState

		m.decodeBaseTemporaryStats(l, ctx)(r, options)
	}
}

func (m *CharacterTemporaryStat) DecodeForeign(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		mask := m.DecodeMask(r)
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

		_ = r.ReadByte() // nDefenseAtt
		_ = r.ReadByte() // nDefenseState

		m.decodeBaseTemporaryStats(l, ctx)(r, options)
	}
}

func (m *CharacterTemporaryStat) decodeBaseTemporaryStats(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		// Mirror getBaseTemporaryStats exactly (same version-specific group + order)
		// so the bytes consumed match the bytes emitted, boundary-for-boundary.
		for _, name := range twoStateBaseStats(t) {
			switch name {
			case character.TemporaryStatTypeSpeedInfusion:
				si := SpeedInfusionTemporaryStat{CharacterTemporaryStatBase: CharacterTemporaryStatBase{bDynamicTermSet: false}}
				si.Decode(l, ctx)(r, options)
			case character.TemporaryStatTypeHomingBeacon:
				gb := GuidedBulletTemporaryStat{CharacterTemporaryStatBase: CharacterTemporaryStatBase{bDynamicTermSet: false}}
				gb.Decode(l, ctx)(r, options)
			case character.TemporaryStatTypeMonsterRiding:
				base := CharacterTemporaryStatBase{bDynamicTermSet: false}
				base.Decode(l, ctx)(r, options)
			default:
				base := CharacterTemporaryStatBase{bDynamicTermSet: true}
				base.Decode(l, ctx)(r, options)
			}
		}
	}
}

func (m *CharacterTemporaryStat) getBaseTemporaryStats(t tenant.Model) []packet.Encoder {
	var list = make([]packet.Encoder, 0)
	for _, name := range twoStateBaseStats(t) {
		switch name {
		case character.TemporaryStatTypeMonsterRiding:
			// Monster Riding: nOption = vehicle/taming-mob item id, rOption = source
			// skill id. Wire contract IDA-confirmed — context.md §2, design.md §1.1.
			if s, ok := m.stats[name]; ok {
				list = append(list, NewCharacterTemporaryStatBaseWithOptions(false, s.Value(), s.SourceId()))
			} else {
				list = append(list, NewCharacterTemporaryStatBase(false)) // 13
			}
		case character.TemporaryStatTypeSpeedInfusion:
			list = append(list, NewSpeedInfusionTemporaryStat()) // 20
		case character.TemporaryStatTypeHomingBeacon:
			list = append(list, NewGuidedBulletTemporaryStat()) // 17
		default:
			list = append(list, NewCharacterTemporaryStatBase(true)) // dynamic, 15
		}
	}
	return list
}
