package item

import "math"

type Id uint32

type Classification uint32

type WeaponType byte

const (
	ClassificationOverall                  = Classification(105)
	ClassificationConsumableTownWarp       = Classification(203)
	ClassificationConsumableScroll         = Classification(204)
	ClassificationConsumableArrow          = Classification(206)
	ClassificationConsumableThrowingStar   = Classification(207)
	ClassificationConsumableMegaphone      = Classification(208)
	ClassificationConsumableSummoningSack  = Classification(210)
	ClassificationConsumablePetFood        = Classification(212)
	ClassificationConsumableTransformation = Classification(221)
	ClassificationConsumableSkillBook      = Classification(228)
	ClassificationConsumableMasteryBook    = Classification(229)
	ClassificationBullet                   = Classification(233)
	ClassificationConsumableMonsterCard    = Classification(238)
	ClassificationPet                      = Classification(500)
	ClassificationTeleportRock             = Classification(504)
	ClassificationPointReset               = Classification(505)
	ClassificationItemImprints             = Classification(506)
	ClassificationMegaphones               = Classification(507)
	ClassificationMessageBanner            = Classification(508)
	ClassificationNote                     = Classification(509)
	ClassificationSongPlayer               = Classification(510)
	ClassificationMapEffect                = Classification(512)
	ClassificationStorePermit              = Classification(514)
	ClassificationCosmeticCoupon           = Classification(515)
	ClassificationExpression               = Classification(516)
	ClassificationPetImprints              = Classification(517)
	ClassificationCurrencySack             = Classification(520)
	ClassificationExperienceCoupon         = Classification(521)
	ClassificationGachaponCoupon           = Classification(522)
	ClassificationStoreSearch              = Classification(523)
	ClassificationPetConsumable            = Classification(524)
	ClassificationWeddingTicket            = Classification(525)
	ClassificationGuildEmote               = Classification(529)
	ClassificationTransformationCoupon     = Classification(530)
	ClassificationDueyCoupon               = Classification(533)
	ClassificationDropCoupon               = Classification(536)
	ClassificationChalkboard               = Classification(537)
	ClassificationPetEvolution             = Classification(538)
	ClassificationAvatarMegaphone          = Classification(539)
	ClassificationCharacterImprints        = Classification(540)
	ClassificationCosmeticMembershipCoupon = Classification(542)
	ClassificationCharacterCreation        = Classification(543)
	ClassificationRemoteMerchant           = Classification(545)
	ClassificationPetMultiConsumable       = Classification(546)
	ClassificationRemoteStore              = Classification(547)

	WeaponTypeOneHandedSword = WeaponType(0)
	WeaponTypeOneHandedAxe   = WeaponType(1)
	WeaponTypeOneHandedMace  = WeaponType(2)
	WeaponTypeDagger         = WeaponType(3)
	WeaponTypeWand           = WeaponType(7)
	WeaponTypeStaff          = WeaponType(8)

	WeaponTypeTwoHandedSword = WeaponType(10)
	WeaponTypeTwoHandedAxe   = WeaponType(11)
	WeaponTypeTwoHandedMace  = WeaponType(12)
	WeaponTypeSpear          = WeaponType(13)
	WeaponTypePolearm        = WeaponType(14)
	WeaponTypeBow            = WeaponType(15)
	WeaponTypeCrossbow       = WeaponType(16)
	WeaponTypeClaw           = WeaponType(17)
	WeaponTypeKnuckle        = WeaponType(18)
	WeaponTypeGun            = WeaponType(19)
	WeaponTypeNone           = WeaponType(99)
)

func GetClassification(itemId Id) Classification {
	return Classification(math.Floor(float64(itemId) / float64(10000)))
}

func GetWeaponType(itemId Id) WeaponType {
	cat := (itemId / 10000) % 100
	if cat < 30 || cat > 49 {
		return WeaponTypeNone
	}
	switch cat - 30 {
	case 0:
		return WeaponTypeOneHandedSword
	case 1:
		return WeaponTypeOneHandedAxe
	case 2:
		return WeaponTypeOneHandedMace
	case 3:
		return WeaponTypeDagger
	case 7:
		return WeaponTypeWand
	case 8:
		return WeaponTypeStaff
	case 10:
		return WeaponTypeTwoHandedSword
	case 11:
		return WeaponTypeTwoHandedAxe
	case 12:
		return WeaponTypeTwoHandedMace
	case 13:
		return WeaponTypeSpear
	case 14:
		return WeaponTypePolearm
	case 15:
		return WeaponTypeBow
	case 16:
		return WeaponTypeCrossbow
	case 17:
		return WeaponTypeClaw
	case 18:
		return WeaponTypeKnuckle
	case 19:
		return WeaponTypeGun
	}
	return WeaponTypeNone
}

func IsThrowingStar(itemId Id) bool {
	return GetClassification(itemId) == ClassificationConsumableThrowingStar
}

func IsBullet(itemId Id) bool {
	return GetClassification(itemId) == ClassificationBullet
}

func IsScrollCleanSlate(itemId Id) bool {
	return Is(itemId, CleanSlateOnePercent, CleanSlateThreePercent, CleanSlateFivePercent, CleanSlateTwentyPercent)
}

func IsScrollColdProtection(itemId Id) bool {
	return Is(itemId, ScrollForCapeColdProtectionTenPercent)
}

func IsScrollSpikes(itemId Id) bool {
	return Is(itemId, ScrollForSpikesOnShoesTenPercent)
}

func IsChaosScroll(itemId Id) bool {
	return Is(itemId, ChaosScrollSixtyPercent, LiarTreeSapOneHundredPercent, MapleSyrupOneHundredPercent, AgentEquipmentScrollOneHundredPercent)
}

func Is(itemId Id, references ...Id) bool {
	for _, r := range references {
		if itemId == r {
			return true
		}
	}
	return false
}

const (
	UseRedBeanPorridge                    = Id(2022001)
	UseAirBubble                          = Id(2022040)
	UseSoftWhiteBun                       = Id(2022186)
	CleanSlateOnePercent                  = Id(2049000)
	CleanSlateThreePercent                = Id(2049001)
	CleanSlateFivePercent                 = Id(2049002)
	CleanSlateTwentyPercent               = Id(2049003)
	WhiteScroll                           = Id(2340000)
	ScrollForCapeColdProtectionTenPercent = Id(2041058)
	ScrollForSpikesOnShoesTenPercent      = Id(2040727)
	ChaosScrollSixtyPercent               = Id(2049100)
	LiarTreeSapOneHundredPercent          = Id(2049101)
	MapleSyrupOneHundredPercent           = Id(2049102)
	AgentEquipmentScrollOneHundredPercent = Id(2049104)
)
