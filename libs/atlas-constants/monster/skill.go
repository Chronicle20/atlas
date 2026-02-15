package monster

import "sort"

const (
	// Skill type categories
	SkillCategoryStatBuff    = "STAT_BUFF"
	SkillCategoryDebuff      = "DEBUFF"
	SkillCategoryImmunity    = "IMMUNITY"
	SkillCategoryReflect     = "REFLECT"
	SkillCategoryHeal        = "HEAL"
	SkillCategorySummon      = "SUMMON"
	SkillCategoryCarnivalBuf = "CARNIVAL_BUFF"

	// MobSkill type IDs (from MobSkill.img.xml)
	SkillTypeWeaponAttackUp     = 100
	SkillTypeMagicAttackUp      = 101
	SkillTypeWeaponDefenseUp    = 102
	SkillTypeMagicDefenseUp     = 103
	SkillTypeWeaponAttackUpAoe  = 110
	SkillTypeMagicAttackUpAoe   = 111
	SkillTypeWeaponDefenseUpAoe = 112
	SkillTypeMagicDefenseUpAoe  = 113
	SkillTypeHeal               = 114
	SkillTypeSpeedUp            = 115
	SkillTypeSeal               = 120
	SkillTypeDarkness           = 121
	SkillTypeWeakness           = 122
	SkillTypeStun               = 123
	SkillTypeCurse              = 124
	SkillTypePoison             = 125
	SkillTypeSlow               = 126
	SkillTypeDispel             = 127
	SkillTypeSeduce             = 128
	SkillTypeBanish             = 129
	SkillTypeAreaPoison         = 131
	SkillTypeReverseInput       = 132
	SkillTypeUndead             = 133
	SkillTypeStopPotion         = 134
	SkillTypeStopMotion         = 135
	SkillTypeFear               = 136
	SkillTypePhysicalImmune     = 140
	SkillTypeMagicImmune        = 141
	SkillTypeHardSkin           = 142
	SkillTypePhysicalCounter    = 143
	SkillTypeMagicCounter       = 144
	SkillTypePhysicalMagicCounter = 145
	SkillTypeCarnivalPAD        = 150
	SkillTypeCarnivalMAD        = 151
	SkillTypeCarnivalPDR        = 152
	SkillTypeCarnivalMDR        = 153
	SkillTypeCarnivalACC        = 154
	SkillTypeCarnivalEVA        = 155
	SkillTypeCarnivalSpeed      = 156
	SkillTypeCarnivalSealSkill  = 157
	SkillTypeSummon             = 200
)

// SkillTypeToStatusName maps a mob skill type to the monster temporary stat name
// used for status effect tracking and client broadcast.
func SkillTypeToStatusName(skillType uint16) TemporaryStatType {
	switch skillType {
	case SkillTypeWeaponAttackUp, SkillTypeWeaponAttackUpAoe:
		return TemporaryStatTypePowerUp
	case SkillTypeMagicAttackUp, SkillTypeMagicAttackUpAoe:
		return TemporaryStatTypeMagicUp
	case SkillTypeWeaponDefenseUp, SkillTypeWeaponDefenseUpAoe:
		return TemporaryStatTypePowerGuardUp
	case SkillTypeMagicDefenseUp, SkillTypeMagicDefenseUpAoe:
		return TemporaryStatTypeMagicGuardUp
	case SkillTypeSpeedUp:
		return TemporaryStatTypeSpeed
	case SkillTypePhysicalImmune:
		return TemporaryStatTypeWeaponAttackImmune
	case SkillTypeMagicImmune:
		return TemporaryStatTypeMagicAttackImmune
	case SkillTypeHardSkin:
		return TemporaryStatTypeHardSkin
	case SkillTypePhysicalCounter:
		return TemporaryStatTypeWeaponCounter
	case SkillTypeMagicCounter:
		return TemporaryStatTypeMagicCounter
	default:
		return ""
	}
}

// IsAoeSkill returns true if the skill type is an AoE variant that affects nearby monsters.
func IsAoeSkill(skillType uint16) bool {
	switch skillType {
	case SkillTypeWeaponAttackUpAoe, SkillTypeMagicAttackUpAoe,
		SkillTypeWeaponDefenseUpAoe, SkillTypeMagicDefenseUpAoe,
		SkillTypeHeal:
		return true
	default:
		return false
	}
}

// SkillTypeToDiseaseName maps a debuff mob skill type to the character temporary stat name
// used for disease application via atlas-buffs.
func SkillTypeToDiseaseName(skillType uint16) string {
	switch skillType {
	case SkillTypeSeal:
		return "SEAL"
	case SkillTypeDarkness:
		return "DARKNESS"
	case SkillTypeWeakness:
		return "WEAKEN"
	case SkillTypeStun:
		return "STUN"
	case SkillTypeCurse:
		return "CURSE"
	case SkillTypePoison:
		return "POISON"
	case SkillTypeSlow:
		return "SLOW"
	case SkillTypeSeduce:
		return "SEDUCE"
	case SkillTypeReverseInput:
		return "CONFUSE"
	case SkillTypeUndead:
		return "UNDEAD"
	case SkillTypeStopPotion:
		return "STOP_PORTION"
	case SkillTypeStopMotion:
		return "STOP_MOTION"
	case SkillTypeFear:
		return "FEAR"
	default:
		return ""
	}
}

var skillNameMap = map[string]uint16{
	"WEAPON_ATTACK_UP":       SkillTypeWeaponAttackUp,
	"MAGIC_ATTACK_UP":        SkillTypeMagicAttackUp,
	"WEAPON_DEFENSE_UP":      SkillTypeWeaponDefenseUp,
	"MAGIC_DEFENSE_UP":       SkillTypeMagicDefenseUp,
	"WEAPON_ATTACK_UP_AOE":   SkillTypeWeaponAttackUpAoe,
	"MAGIC_ATTACK_UP_AOE":    SkillTypeMagicAttackUpAoe,
	"WEAPON_DEFENSE_UP_AOE":  SkillTypeWeaponDefenseUpAoe,
	"MAGIC_DEFENSE_UP_AOE":   SkillTypeMagicDefenseUpAoe,
	"HEAL":                   SkillTypeHeal,
	"SPEED_UP":               SkillTypeSpeedUp,
	"SEAL":                   SkillTypeSeal,
	"DARKNESS":               SkillTypeDarkness,
	"WEAKNESS":               SkillTypeWeakness,
	"STUN":                   SkillTypeStun,
	"CURSE":                  SkillTypeCurse,
	"POISON":                 SkillTypePoison,
	"SLOW":                   SkillTypeSlow,
	"DISPEL":                 SkillTypeDispel,
	"SEDUCE":                 SkillTypeSeduce,
	"BANISH":                 SkillTypeBanish,
	"AREA_POISON":            SkillTypeAreaPoison,
	"REVERSE_INPUT":          SkillTypeReverseInput,
	"UNDEAD":                 SkillTypeUndead,
	"STOP_POTION":            SkillTypeStopPotion,
	"STOP_MOTION":            SkillTypeStopMotion,
	"FEAR":                   SkillTypeFear,
	"PHYSICAL_IMMUNE":        SkillTypePhysicalImmune,
	"MAGIC_IMMUNE":           SkillTypeMagicImmune,
	"HARD_SKIN":              SkillTypeHardSkin,
	"PHYSICAL_COUNTER":       SkillTypePhysicalCounter,
	"MAGIC_COUNTER":          SkillTypeMagicCounter,
	"PHYSICAL_MAGIC_COUNTER": SkillTypePhysicalMagicCounter,
	"SUMMON":                 SkillTypeSummon,
}

// SkillNameToId maps a human-readable skill name to its skill type ID.
func SkillNameToId(name string) (uint16, bool) {
	id, ok := skillNameMap[name]
	return id, ok
}

// SkillTypeNames returns a sorted list of all valid skill name strings.
func SkillTypeNames() []string {
	names := make([]string, 0, len(skillNameMap))
	for k := range skillNameMap {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func SkillCategory(skillType uint16) string {
	switch skillType {
	case SkillTypeWeaponAttackUp, SkillTypeWeaponAttackUpAoe,
		SkillTypeMagicAttackUp, SkillTypeMagicAttackUpAoe,
		SkillTypeWeaponDefenseUp, SkillTypeWeaponDefenseUpAoe,
		SkillTypeMagicDefenseUp, SkillTypeMagicDefenseUpAoe,
		SkillTypeSpeedUp:
		return SkillCategoryStatBuff
	case SkillTypeSeal, SkillTypeDarkness, SkillTypeWeakness,
		SkillTypeStun, SkillTypeCurse, SkillTypePoison,
		SkillTypeSlow, SkillTypeDispel, SkillTypeSeduce,
		SkillTypeBanish, SkillTypeAreaPoison,
		SkillTypeReverseInput, SkillTypeUndead,
		SkillTypeStopPotion, SkillTypeStopMotion, SkillTypeFear:
		return SkillCategoryDebuff
	case SkillTypePhysicalImmune, SkillTypeMagicImmune, SkillTypeHardSkin:
		return SkillCategoryImmunity
	case SkillTypePhysicalCounter, SkillTypeMagicCounter, SkillTypePhysicalMagicCounter:
		return SkillCategoryReflect
	case SkillTypeHeal:
		return SkillCategoryHeal
	case SkillTypeSummon:
		return SkillCategorySummon
	case SkillTypeCarnivalPAD, SkillTypeCarnivalMAD,
		SkillTypeCarnivalPDR, SkillTypeCarnivalMDR,
		SkillTypeCarnivalACC, SkillTypeCarnivalEVA,
		SkillTypeCarnivalSpeed, SkillTypeCarnivalSealSkill:
		return SkillCategoryCarnivalBuf
	default:
		return ""
	}
}
