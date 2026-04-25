package item

import (
	itemc "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type Compartment uint8

const (
	CompartmentUnknown   Compartment = 0
	CompartmentEquipment Compartment = 1
	CompartmentUse       Compartment = 2
	CompartmentSetup     Compartment = 3
	CompartmentEtc       Compartment = 4
	CompartmentCash      Compartment = 5
)

func (c Compartment) String() string {
	switch c {
	case CompartmentEquipment:
		return "equipment"
	case CompartmentUse:
		return "use"
	case CompartmentSetup:
		return "setup"
	case CompartmentEtc:
		return "etc"
	case CompartmentCash:
		return "cash"
	default:
		return "unknown"
	}
}

// Classify derives (compartment, subcategory) from an item id alone.
// Subcategory is best-effort: classifications that genuinely require equipment
// metadata (slot disambiguation) return a placeholder that
// UpdateEquipmentClassification overrides later.
func Classify(itemId uint32) (Compartment, string) {
	compartment := compartmentOf(itemId)
	if compartment == CompartmentUnknown {
		return CompartmentUnknown, "other"
	}

	switch compartment {
	case CompartmentEquipment:
		return CompartmentEquipment, equipmentSubcategory(itemId)
	case CompartmentUse:
		return CompartmentUse, useSubcategory(itemId)
	case CompartmentSetup:
		return CompartmentSetup, setupSubcategory(itemId)
	case CompartmentEtc:
		return CompartmentEtc, etcSubcategory(itemId)
	case CompartmentCash:
		return CompartmentCash, cashSubcategory(itemId)
	}
	return compartment, "other"
}

func compartmentOf(itemId uint32) Compartment {
	switch itemId / 1_000_000 {
	case 1:
		return CompartmentEquipment
	case 2:
		return CompartmentUse
	case 3:
		return CompartmentSetup
	case 4:
		return CompartmentEtc
	case 5:
		return CompartmentCash
	default:
		return CompartmentUnknown
	}
}

func classification(itemId uint32) uint32 { return itemId / 10_000 }

var equipmentArmorByClassification = map[uint32]string{
	100: "hat", 101: "hat",
	102: "face-accessory",
	103: "eye-accessory",
	104: "earring",
	105: "top",
	106: "bottom",
	107: "shoes",
	108: "gloves",
	109: "shield",
	110: "cape",
	111: "ring",
	112: "pendant",
	113: "belt",
	114: "medal",
	190: "tamed-mob",
	191: "saddle",
}

func equipmentSubcategory(itemId uint32) string {
	cls := classification(itemId)
	if cls >= 130 && cls <= 149 {
		wt := itemc.GetWeaponType(itemc.Id(itemId))
		return weaponTypeToken(wt)
	}
	if cls >= 180 && cls <= 189 {
		return "pet-equip"
	}
	// 4-digit prefix override for cls 104: ids in 1040xxx / 1041xxx are tops, the rest stay earrings
	prefix4 := itemId / 1_000
	if cls == 104 && (prefix4 == 1040 || prefix4 == 1041) {
		return "top"
	}
	if name, ok := equipmentArmorByClassification[cls]; ok {
		return name
	}
	return "other"
}

func weaponTypeToken(wt itemc.WeaponType) string {
	switch wt {
	case itemc.WeaponTypeOneHandedSword:
		return "one-handed-sword"
	case itemc.WeaponTypeOneHandedAxe:
		return "one-handed-axe"
	case itemc.WeaponTypeOneHandedMace:
		return "one-handed-mace"
	case itemc.WeaponTypeDagger:
		return "dagger"
	case itemc.WeaponTypeWand:
		return "wand"
	case itemc.WeaponTypeStaff:
		return "staff"
	case itemc.WeaponTypeTwoHandedSword:
		return "two-handed-sword"
	case itemc.WeaponTypeTwoHandedAxe:
		return "two-handed-axe"
	case itemc.WeaponTypeTwoHandedMace:
		return "two-handed-mace"
	case itemc.WeaponTypeSpear:
		return "spear"
	case itemc.WeaponTypePolearm:
		return "polearm"
	case itemc.WeaponTypeBow:
		return "bow"
	case itemc.WeaponTypeCrossbow:
		return "crossbow"
	case itemc.WeaponTypeClaw:
		return "claw"
	case itemc.WeaponTypeKnuckle:
		return "knuckle"
	case itemc.WeaponTypeGun:
		return "gun"
	default:
		return "other"
	}
}

var useByClassification = map[uint32]string{
	200: "potion", 201: "potion", 202: "potion",
	203: "town-warp",
	204: "scroll", 205: "scroll",
	206: "arrow",
	207: "throwing-star",
	208: "megaphone",
	210: "summoning-sack",
	212: "pet-food",
	221: "transformation",
	228: "skill-book",
	229: "mastery-book",
	233: "bullet",
	238: "monster-card",
}

func useSubcategory(itemId uint32) string {
	if name, ok := useByClassification[classification(itemId)]; ok {
		return name
	}
	return "other"
}

var setupByClassification = map[uint32]string{
	301: "chair",
	303: "hired-merchant",
}

func setupSubcategory(itemId uint32) string {
	if name, ok := setupByClassification[classification(itemId)]; ok {
		return name
	}
	return "other-setup"
}

var etcByClassification = map[uint32]string{
	400: "crafting-material",
	401: "ore", 402: "ore",
	403: "production-item",
	404: "mineral-ore",
	405: "mineral-refined",
	406: "gem-rough",
	407: "gem-cut",
	411: "monster-drop", 412: "monster-drop", 413: "monster-drop", 414: "monster-drop",
	415: "monster-drop", 416: "monster-drop", 417: "monster-drop", 418: "monster-drop",
	419: "monster-drop",
	421: "magnifying-glass",
	422: "quest-item", 423: "quest-item", 424: "quest-item", 425: "quest-item",
	426: "quest-item", 427: "quest-item", 428: "quest-item",
	430: "simulator",
	431: "book-page",
}

func etcSubcategory(itemId uint32) string {
	if name, ok := etcByClassification[classification(itemId)]; ok {
		return name
	}
	return "other-etc"
}

var cashByClassification = map[uint32]string{
	500: "pet",
	501: "pet-skill",
	502: "cosmetic-throwing-star",
	503: "hired-merchant",
	504: "teleport-rock",
	505: "point-reset",
	506: "item-imprint",
	507: "megaphone",
	508: "message-banner",
	509: "note",
	510: "song-player",
	512: "field-effect",
	513: "death-protection",
	514: "store-permit",
	515: "cosmetic-coupon",
	516: "expression",
	517: "pet-imprint",
	520: "currency-sack",
	521: "experience-coupon",
	522: "gachapon-coupon",
	523: "store-search",
	524: "pet-consumable",
	525: "wedding-ticket",
	528: "character-effect",
	529: "guild-emote",
	530: "transformation-coupon",
	533: "duey-coupon",
	536: "drop-coupon",
	537: "chalkboard",
	538: "pet-evolution",
	539: "avatar-megaphone",
	540: "character-imprint",
	542: "cosmetic-membership-coupon",
	543: "character-creation",
	545: "remote-merchant",
	546: "pet-multi-consumable",
	547: "remote-store",
}

func cashSubcategory(itemId uint32) string {
	if name, ok := cashByClassification[classification(itemId)]; ok {
		return name
	}
	return "other-cash"
}
