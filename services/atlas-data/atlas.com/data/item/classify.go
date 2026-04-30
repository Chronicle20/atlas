package item

import (
	itemc "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// Classify derives (compartment, subcategory) from an item id alone.
func Classify(itemId uint32) (inventory.Type, string) {
	compartment, ok := inventory.TypeFromItemId(itemc.Id(itemId))
	if !ok {
		return inventory.Type(0), "other"
	}

	switch compartment {
	case inventory.TypeValueEquip:
		return compartment, equipmentSubcategory(itemId)
	case inventory.TypeValueUse:
		return compartment, useSubcategory(itemId)
	case inventory.TypeValueSetup:
		return compartment, setupSubcategory(itemId)
	case inventory.TypeValueETC:
		return compartment, etcSubcategory(itemId)
	case inventory.TypeValueCash:
		return compartment, cashSubcategory(itemId)
	}
	return compartment, "other"
}

func classOf(itemId uint32) itemc.Classification {
	return itemc.GetClassification(itemc.Id(itemId))
}

var equipmentArmorByClassification = map[itemc.Classification]string{
	itemc.ClassificationHat:           "hat",
	itemc.ClassificationFaceAccessory: "face-accessory",
	itemc.ClassificationEyeAccessory:  "eye-accessory",
	itemc.ClassificationEarring:       "earring",
	itemc.ClassificationTop:           "top",
	itemc.ClassificationOverall:       "overall",
	itemc.ClassificationBottom:        "bottom",
	itemc.ClassificationShoes:         "shoes",
	itemc.ClassificationGloves:        "gloves",
	itemc.ClassificationShield:        "shield",
	itemc.ClassificationCape:          "cape",
	itemc.ClassificationRing:          "ring",
	itemc.ClassificationPendant:       "pendant",
	itemc.ClassificationBelt:          "belt",
	itemc.ClassificationMedal:         "medal",
	itemc.ClassificationTamedMob:      "tamed-mob",
	itemc.ClassificationSaddle:        "saddle",
}

func equipmentSubcategory(itemId uint32) string {
	cls := classOf(itemId)
	if cls >= 130 && cls <= 149 {
		wt := itemc.GetWeaponType(itemc.Id(itemId))
		return weaponTypeToken(wt)
	}
	if cls >= 180 && cls <= 189 {
		return "pet-equip"
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

var useByClassification = map[itemc.Classification]string{
	itemc.ClassificationConsumableTownWarp:       "town-warp",
	itemc.ClassificationConsumableArrow:          "arrow",
	itemc.ClassificationConsumableThrowingStar:   "throwing-star",
	itemc.ClassificationConsumableMegaphone:      "megaphone",
	itemc.ClassificationConsumableSummoningSack:  "summoning-sack",
	itemc.ClassificationConsumablePetFood:        "pet-food",
	itemc.ClassificationConsumableTransformation: "transformation",
	itemc.ClassificationConsumableSkillBook:      "skill-book",
	itemc.ClassificationConsumableMasteryBook:    "mastery-book",
	itemc.ClassificationBullet:                   "bullet",
	itemc.ClassificationConsumableMonsterCard:    "monster-card",
}

func useSubcategory(itemId uint32) string {
	cls := classOf(itemId)
	if cls >= 200 && cls <= 202 {
		return "potion"
	}
	if cls == itemc.ClassificationConsumableScroll || cls == 205 {
		return "scroll"
	}
	if name, ok := useByClassification[cls]; ok {
		return name
	}
	return "other"
}

var setupByClassification = map[itemc.Classification]string{
	itemc.ClassificationChair:              "chair",
	itemc.ClassificationSetupHiredMerchant: "hired-merchant",
}

func setupSubcategory(itemId uint32) string {
	if name, ok := setupByClassification[classOf(itemId)]; ok {
		return name
	}
	return "other-setup"
}

var etcByClassification = map[itemc.Classification]string{
	itemc.ClassificationCraftingMaterial: "crafting-material",
	401:                                  "ore",
	402:                                  "ore",
	itemc.ClassificationProductionItem:   "production-item",
	itemc.ClassificationMineralOre:       "mineral-ore",
	itemc.ClassificationMineralRefined:   "mineral-refined",
	itemc.ClassificationGemRough:         "gem-rough",
	itemc.ClassificationGemCut:           "gem-cut",
	itemc.ClassificationMagnifyingGlass:  "magnifying-glass",
	itemc.ClassificationSimulator:        "simulator",
	itemc.ClassificationBookPage:         "book-page",
}

func etcSubcategory(itemId uint32) string {
	cls := classOf(itemId)
	if cls >= 411 && cls <= 419 {
		return "monster-drop"
	}
	if cls >= 422 && cls <= 428 {
		return "quest-item"
	}
	if name, ok := etcByClassification[cls]; ok {
		return name
	}
	return "other-etc"
}

var cashByClassification = map[itemc.Classification]string{
	itemc.ClassificationPet:                      "pet",
	itemc.ClassificationCharacterEffect:          "character-effect",
	itemc.ClassificationCosmeticThrowingStar:     "cosmetic-throwing-star",
	itemc.ClassificationHiredMerchant:            "hired-merchant",
	itemc.ClassificationTeleportRock:             "teleport-rock",
	itemc.ClassificationPointReset:               "point-reset",
	itemc.ClassificationItemImprints:             "item-imprint",
	itemc.ClassificationMegaphones:               "megaphone",
	itemc.ClassificationMessageBanner:            "message-banner",
	itemc.ClassificationNote:                     "note",
	itemc.ClassificationSongPlayer:               "song-player",
	itemc.ClassificationFieldEffect:              "field-effect",
	itemc.ClassificationDeathProtection:          "death-protection",
	itemc.ClassificationStorePermit:              "store-permit",
	itemc.ClassificationCosmeticCoupon:           "cosmetic-coupon",
	itemc.ClassificationExpression:               "expression",
	itemc.ClassificationPetImprints:              "pet-imprint",
	itemc.ClassificationCurrencySack:             "currency-sack",
	itemc.ClassificationExperienceCoupon:         "experience-coupon",
	itemc.ClassificationGachaponCoupon:           "gachapon-coupon",
	itemc.ClassificationStoreSearch:              "store-search",
	itemc.ClassificationPetConsumable:            "pet-consumable",
	itemc.ClassificationWeddingTicket:            "wedding-ticket",
	itemc.ClassificationCharacterEffect2:         "character-effect",
	itemc.ClassificationGuildEmote:               "guild-emote",
	itemc.ClassificationTransformationCoupon:     "transformation-coupon",
	itemc.ClassificationDueyCoupon:               "duey-coupon",
	itemc.ClassificationDropCoupon:               "drop-coupon",
	itemc.ClassificationChalkboard:               "chalkboard",
	itemc.ClassificationPetEvolution:             "pet-evolution",
	itemc.ClassificationAvatarMegaphone:          "avatar-megaphone",
	itemc.ClassificationCharacterImprints:        "character-imprint",
	itemc.ClassificationCosmeticMembershipCoupon: "cosmetic-membership-coupon",
	itemc.ClassificationCharacterCreation:        "character-creation",
	itemc.ClassificationRemoteMerchant:           "remote-merchant",
	itemc.ClassificationPetMultiConsumable:       "pet-multi-consumable",
	itemc.ClassificationRemoteStore:              "remote-store",
}

func cashSubcategory(itemId uint32) string {
	if name, ok := cashByClassification[classOf(itemId)]; ok {
		return name
	}
	return "other-cash"
}
