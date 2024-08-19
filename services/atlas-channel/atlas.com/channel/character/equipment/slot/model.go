package slot

import "atlas-channel/character/inventory/equipable"

type Position int16

type Model struct {
	Position      Position
	Equipable     *equipable.Model
	CashEquipable *equipable.Model
}
