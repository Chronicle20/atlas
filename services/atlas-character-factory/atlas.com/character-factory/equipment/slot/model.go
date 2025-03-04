package slot

import (
	"atlas-character-factory/equipable"
)

type Position int16

const (
	PositionHat      Position = -1
	PositionMedal    Position = -49
	PositionForehead Position = -2
	PositionRing1    Position = -12
	PositionRing2    Position = -13
	PositionEye      Position = -3
	PositionEarring  Position = -4
	PositionShoulder Position = 99
	PositionCape     Position = -9
	PositionTop      Position = -5
	PositionPendant  Position = -17
	PositionWeapon   Position = -11
	PositionShield   Position = -10
	PositionGloves   Position = -8
	PositionBottom   Position = -6
	PositionBelt     Position = -50
	PositionRing3    Position = -15
	PositionRing4    Position = -16
	PositionShoes    Position = -7
)

type Type string

const (
	TypeHat      = Type("hat")
	TypeMedal    = Type("medal")
	TypeForehead = Type("forehead")
	TypeRing1    = Type("ring1")
	TypeRing2    = Type("ring2")
	TypeEye      = Type("eye")
	TypeEarring  = Type("earring")
	TypeShoulder = Type("shoulder")
	TypeCape     = Type("cape")
	TypeTop      = Type("top")
	TypePendant  = Type("pendant")
	TypeWeapon   = Type("weapon")
	TypeShield   = Type("shield")
	TypeGloves   = Type("gloves")
	TypeBottom   = Type("pants")
	TypeBelt     = Type("belt")
	TypeRing3    = Type("ring3")
	TypeRing4    = Type("ring4")
	TypeShoes    = Type("shoes")
)

var Types = []Type{TypeHat, TypeMedal, TypeForehead, TypeRing1, TypeRing2, TypeEye, TypeEarring, TypeShoulder, TypeCape, TypeTop, TypePendant, TypeWeapon, TypeShield, TypeGloves, TypeBottom, TypeBelt, TypeRing3, TypeRing4, TypeShoes}

type Model struct {
	Position      Position
	Equipable     *equipable.Model
	CashEquipable *equipable.Model
}
