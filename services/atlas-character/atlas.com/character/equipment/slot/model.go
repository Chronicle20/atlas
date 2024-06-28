package slot

import (
	"atlas-character/equipable"
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
	PositionShoes             = -7
)

type Model struct {
	Position  Position
	Equipable *equipable.Model
}
