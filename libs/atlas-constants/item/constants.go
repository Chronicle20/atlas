package item

import "math"

type Id uint32

type Classification uint32

type WeaponType byte

const (
	ClassificationThrowingStar = Classification(207)
	ClassificationBullet       = Classification(233)

	WeaponTypeNone           = WeaponType(0)
	WeaponTypeOneHandedSword = WeaponType(1)
	WeaponTypeOneHandedAxe   = WeaponType(2)
	WeaponTypeOneHandedMace  = WeaponType(3)
	WeaponTypeDagger         = WeaponType(4)
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
	case 1:
		return WeaponTypeOneHandedSword
	case 2:
		return WeaponTypeOneHandedAxe
	case 3:
		return WeaponTypeOneHandedMace
	case 4:
		return WeaponTypeDagger
	case 5:
		return WeaponTypeWand
	case 6:
		return WeaponTypeStaff
	case 9:
		return WeaponTypeTwoHandedSword
	case 10:
		return WeaponTypeTwoHandedAxe
	case 11:
		return WeaponTypeTwoHandedMace
	case 12:
		return WeaponTypeSpear
	case 13:
		return WeaponTypePolearm
	case 14:
		return WeaponTypeBow
	case 15:
		return WeaponTypeCrossbow
	case 16:
		return WeaponTypeClaw
	case 17:
		return WeaponTypeKnuckle
	case 18:
		return WeaponTypeGun
	}
	return WeaponTypeNone
}

func IsThrowingStar(itemId Id) bool {
	return GetClassification(itemId) == ClassificationThrowingStar
}

func IsBullet(itemId Id) bool {
	return GetClassification(itemId) == ClassificationBullet
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
	UseRedBeanPorridge = Id(2022001)
	UseAirBubble       = Id(2022040)
	UseSoftWhiteBun    = Id(2022186)
)
