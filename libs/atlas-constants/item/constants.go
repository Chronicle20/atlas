package item

import "math"

type Id uint32

type Classification uint32

type WeaponType byte

const (
	ClassificationThrowingStar = Classification(207)
	ClassificationBullet       = Classification(233)

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
