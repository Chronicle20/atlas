package slot

import (
	"errors"
	"strings"
)

const (
	TypeHat      = "hat"
	TypeMedal    = "medal"
	TypeForehead = "forehead"
	TypeRing1    = "ring1"
	TypeRing2    = "ring2"
	TypeEye      = "eye"
	TypeEarring  = "earring"
	TypeShoulder = "shoulder"
	TypeCape     = "cape"
	TypeTop      = "top"
	TypePendant  = "pendant"
	TypeWeapon   = "weapon"
	TypeShield   = "shield"
	TypeGloves   = "gloves"
	TypeBottom   = "bottom"
	TypeBelt     = "belt"
	TypeRing3    = "ring3"
	TypeRing4    = "ring4"
	TypeShoes    = "shoes"
	TypeOverall  = "overall"
)

func PositionFromType(slotType string) (Position, error) {
	switch strings.ToLower(slotType) {
	case TypeHat:
		return PositionHat, nil
	case TypeMedal:
		return PositionMedal, nil
	case TypeForehead:
		return PositionForehead, nil
	case TypeRing1:
		return PositionRing1, nil
	case TypeRing2:
		return PositionRing2, nil
	case TypeEye:
		return PositionEye, nil
	case TypeEarring:
		return PositionEarring, nil
	case TypeShoulder:
		return PositionShoulder, nil
	case TypeCape:
		return PositionCape, nil
	case TypeOverall:
		return PositionOverall, nil
	case TypeTop:
		return PositionTop, nil
	case TypePendant:
		return PositionPendant, nil
	case TypeWeapon:
		return PositionWeapon, nil
	case TypeShield:
		return PositionShield, nil
	case TypeGloves:
		return PositionGloves, nil
	case TypeBottom:
		return PositionBottom, nil
	case TypeBelt:
		return PositionBelt, nil
	case TypeRing3:
		return PositionRing3, nil
	case TypeRing4:
		return PositionRing4, nil
	case TypeShoes:
		return PositionShoes, nil
	}
	return PositionHat, errors.New("unable to map type to position")
}
