package slot

import (
	"errors"
)

func PositionFromType(slotType Type) (Position, error) {
	switch slotType {
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
	case TypePetRing1:
		return PositionPetRing1, nil
	case TypePetPouch:
		return PositionPetPouch, nil
	case TypePetMesoMagnet:
		return PositionPetMesoMagnet, nil
	case TypePetHP:
		return PositionPetHP, nil
	case TypePetMP:
		return PositionPetMP, nil
	case TypePetShoes:
		return PositionPetShoes, nil
	case TypePetBinocular:
		return PositionPetBinocular, nil
	case TypePetMagicScales:
		return PositionPetMagicScales, nil
	case TypePetRing2:
		return PositionPetRing2, nil
	case TypePetItemIgnore:
		return PositionPetItemIgnore, nil
	}
	return PositionHat, errors.New("unable to map type to position")
}

func TypeFromPosition(position Position) (Type, error) {
	switch position {
	case PositionHat:
		return TypeHat, nil
	case PositionMedal:
		return TypeMedal, nil
	case PositionForehead:
		return TypeForehead, nil
	case PositionRing1:
		return TypeRing1, nil
	case PositionRing2:
		return TypeRing2, nil
	case PositionEye:
		return TypeEye, nil
	case PositionEarring:
		return TypeEarring, nil
	case PositionShoulder:
		return TypeShoulder, nil
	case PositionCape:
		return TypeCape, nil
	case PositionTop:
		return TypeTop, nil
	case PositionPendant:
		return TypePendant, nil
	case PositionWeapon:
		return TypeWeapon, nil
	case PositionShield:
		return TypeShield, nil
	case PositionGloves:
		return TypeGloves, nil
	case PositionBottom:
		return TypeBottom, nil
	case PositionBelt:
		return TypeBelt, nil
	case PositionRing3:
		return TypeRing3, nil
	case PositionRing4:
		return TypeRing4, nil
	case PositionShoes:
		return TypeShoes, nil
	case PositionPetRing1:
		return TypePetRing1, nil
	case PositionPetPouch:
		return TypePetPouch, nil
	case PositionPetMesoMagnet:
		return TypePetMesoMagnet, nil
	case PositionPetHP:
		return TypePetHP, nil
	case PositionPetMP:
		return TypePetMP, nil
	case PositionPetShoes:
		return TypePetShoes, nil
	case PositionPetBinocular:
		return TypePetBinocular, nil
	case PositionPetMagicScales:
		return TypePetMagicScales, nil
	case PositionPetRing2:
		return TypePetRing2, nil
	case PositionPetItemIgnore:
		return TypePetItemIgnore, nil
	}
	return "", errors.New("unable to map position to type")
}
