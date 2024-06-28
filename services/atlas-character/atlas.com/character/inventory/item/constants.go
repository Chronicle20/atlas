package item

const (
	WeaponTypeInvalid               = 0
	WeaponTypeGeneralOneHandedSwing = 1
	WeaponTypeGeneralOneHandedStab  = 2
	WeaponTypeGeneralTwoHandedSwing = 3
	WeaponTypeGeneralTwoHandedStab  = 4
	WeaponTypeBow                   = 5
	WeaponTypeClaw                  = 6
	WeaponTypeCrossbow              = 7
	WeaponTypeDaggerThieves         = 8
	WeaponTypeDaggerOther           = 9
	WeaponTypeGun                   = 10
	WeaponTypeKnuckle               = 11
	WeaponTypePoleArmSwing          = 12
	WeaponTypePoleArmStab           = 13
	WeaponTypeSpearStab             = 14
	WeaponTypeSpearSwing            = 15
	WeaponTypeStaff                 = 16
	WeaponTypeSwordOneHanded        = 17
	WeaponTypeSwordTwoHanded        = 18
	WeaponTypeWand                  = 19
)

func GetWeaponType(itemId uint32) int {
	cat := (itemId / 10000) % 100
	if cat < 30 || cat > 49 {
		return WeaponTypeInvalid
	}
	var types = []int{
		WeaponTypeSwordOneHanded,
		WeaponTypeGeneralOneHandedSwing,
		WeaponTypeGeneralOneHandedSwing,
		WeaponTypeDaggerOther,
		WeaponTypeInvalid,
		WeaponTypeInvalid,
		WeaponTypeInvalid,
		WeaponTypeWand,
		WeaponTypeStaff,
		WeaponTypeInvalid,
		WeaponTypeSwordTwoHanded,
		WeaponTypeGeneralTwoHandedSwing,
		WeaponTypeGeneralTwoHandedSwing,
		WeaponTypeSpearStab,
		WeaponTypePoleArmSwing,
		WeaponTypeBow,
		WeaponTypeCrossbow,
		WeaponTypeClaw,
		WeaponTypeKnuckle,
		WeaponTypeGun,
	}
	return types[cat-30]
}

func GetWeaponDamageMultiplier(weaponType int) float64 {
	switch weaponType {
	case WeaponTypeInvalid:
		return 0
	case WeaponTypeGeneralOneHandedSwing:
		return 4.4
	case WeaponTypeGeneralOneHandedStab:
		return 3.2
	case WeaponTypeGeneralTwoHandedSwing:
		return 4.8
	case WeaponTypeGeneralTwoHandedStab:
		return 3.4
	case WeaponTypeBow:
		return 3.4
	case WeaponTypeClaw:
		return 3.6
	case WeaponTypeCrossbow:
		return 3.6
	case WeaponTypeDaggerThieves:
		return 3.6
	case WeaponTypeDaggerOther:
		return 4
	case WeaponTypeGun:
		return 3.6
	case WeaponTypeKnuckle:
		return 4.8
	case WeaponTypePoleArmSwing:
		return 5.0
	case WeaponTypePoleArmStab:
		return 3.0
	case WeaponTypeSpearStab:
		return 5.0
	case WeaponTypeSpearSwing:
		return 3.0
	case WeaponTypeStaff:
		return 3.6
	case WeaponTypeSwordOneHanded:
		return 4.0
	case WeaponTypeSwordTwoHanded:
		return 4.6
	case WeaponTypeWand:
		return 3.6
	}
	return 0
}
