package slot

import (
	"atlas-consumables/equipable"
	"errors"
	"sync"
)

type Position int16

type Type string

type Slot struct {
	Type     Type
	Position Position
}

var Slots = []Slot{
	{Type: "hat", Position: -1},
	{Type: "medal", Position: -49},
	{Type: "forehead", Position: -2},
	{Type: "ring1", Position: -12},
	{Type: "ring2", Position: -13},
	{Type: "eye", Position: -3},
	{Type: "earring", Position: -4},
	{Type: "shoulder", Position: 99},
	{Type: "cape", Position: -9},
	{Type: "top", Position: -5},
	{Type: "pendant", Position: -17},
	{Type: "weapon", Position: -11},
	{Type: "shield", Position: -10},
	{Type: "gloves", Position: -8},
	{Type: "pants", Position: -6},
	{Type: "belt", Position: -50},
	{Type: "ring3", Position: -15},
	{Type: "ring4", Position: -16},
	{Type: "shoes", Position: -7},
	{Type: "petRing1", Position: -21},
	{Type: "petPouch", Position: -22},
	{Type: "petMesoMagnet", Position: -23},
	{Type: "petHP", Position: -24},
	{Type: "petMP", Position: -25},
	{Type: "petShoes", Position: -26},
	{Type: "petBinocular", Position: -27},
	{Type: "petMagicScales", Position: -28},
	{Type: "petRing2", Position: -29},
	{Type: "petItemIgnore", Position: -46},
}

var (
	typeToSlot     map[Type]Slot
	positionToSlot map[Position]Slot
	once           sync.Once
)

func initializeMaps() {
	once.Do(func() {
		typeToSlot = make(map[Type]Slot)
		positionToSlot = make(map[Position]Slot)
		for _, slot := range Slots {
			typeToSlot[slot.Type] = slot
			positionToSlot[slot.Position] = slot
		}
	})
}

func GetSlotByType(slotType Type) (Slot, error) {
	initializeMaps()
	if slot, found := typeToSlot[slotType]; found {
		return slot, nil
	}
	return Slot{}, errors.New("unknown slot type")
}

func GetSlotByPosition(position Position) (Slot, error) {
	initializeMaps()
	if slot, found := positionToSlot[position]; found {
		return slot, nil
	}
	return Slot{}, errors.New("unknown position")
}

type Model struct {
	Position      Position
	Equipable     *equipable.Model
	CashEquipable *equipable.Model
}
