package slot

import (
	"atlas-character/equipable"
)

type Position int16

const (
	PositionHat            Position = -1
	PositionMedal          Position = -49
	PositionForehead       Position = -2
	PositionRing1          Position = -12
	PositionRing2          Position = -13
	PositionEye            Position = -3
	PositionEarring        Position = -4
	PositionShoulder       Position = 99
	PositionCape           Position = -9
	PositionTop            Position = -5
	PositionPendant        Position = -17
	PositionWeapon         Position = -11
	PositionShield         Position = -10
	PositionGloves         Position = -8
	PositionBottom         Position = -6
	PositionBelt           Position = -50
	PositionRing3          Position = -15
	PositionRing4          Position = -16
	PositionShoes          Position = -7
	PositionPetRing1       Position = -21
	PositionPetPouch       Position = -22
	PositionPetMesoMagnet  Position = -23
	PositionPetHP          Position = -24
	PositionPetMP          Position = -25
	PositionPetShoes       Position = -26
	PositionPetBinocular   Position = -27
	PositionPetMagicScales Position = -28
	PositionPetRing2       Position = -29
	PositionPetItemIgnore  Position = -46
)

type Type string

const (
	TypeHat            = Type("hat")
	TypeMedal          = Type("medal")
	TypeForehead       = Type("forehead")
	TypeRing1          = Type("ring1")
	TypeRing2          = Type("ring2")
	TypeEye            = Type("eye")
	TypeEarring        = Type("earring")
	TypeShoulder       = Type("shoulder")
	TypeCape           = Type("cape")
	TypeTop            = Type("top")
	TypePendant        = Type("pendant")
	TypeWeapon         = Type("weapon")
	TypeShield         = Type("shield")
	TypeGloves         = Type("gloves")
	TypeBottom         = Type("pants")
	TypeBelt           = Type("belt")
	TypeRing3          = Type("ring3")
	TypeRing4          = Type("ring4")
	TypeShoes          = Type("shoes")
	TypePetRing1       = Type("petRing1")
	TypePetPouch       = Type("petPouch")
	TypePetMesoMagnet  = Type("petMesoMagnet")
	TypePetHP          = Type("petHP")
	TypePetMP          = Type("petMP")
	TypePetShoes       = Type("petShoes")
	TypePetBinocular   = Type("petBinocular")
	TypePetMagicScales = Type("petMagicScales")
	TypePetRing2       = Type("petRing2")
	TypePetItemIgnore  = Type("petItemIgnore")
)

var Types = []Type{TypeHat, TypeMedal, TypeForehead, TypeRing1, TypeRing2, TypeEye, TypeEarring, TypeShoulder, TypeCape,
	TypeTop, TypePendant, TypeWeapon, TypeShield, TypeGloves, TypeBottom, TypeBelt, TypeRing3, TypeRing4, TypeShoes,
	TypePetRing1, TypePetPouch, TypePetMesoMagnet, TypePetHP, TypePetMP, TypePetShoes, TypePetBinocular, TypePetMagicScales, TypePetRing2, TypePetItemIgnore}

type Model struct {
	Position      Position
	Equipable     *equipable.Model
	CashEquipable *equipable.Model
}

func (m Model) Clone() Model {
	return Model{
		Position:      m.Position,
		Equipable:     m.Equipable,
		CashEquipable: m.CashEquipable,
	}
}

func (m Model) SetEquipable(e *equipable.Model) Model {
	rm := m.Clone()
	rm.Equipable = e
	return rm
}

func (m Model) SetCashEquipable(e *equipable.Model) Model {
	rm := m.Clone()
	rm.CashEquipable = e
	return rm
}
