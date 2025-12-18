package slot

import (
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
	{Type: "shoulder", Position: -51},
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
	{Type: "tamingMob", Position: -18},
	{Type: "saddle", Position: -19},
	{Type: "mobEquip", Position: -20},
	{Type: "petEquip", Position: -14},
	{Type: "petRing1", Position: -21},
	{Type: "petPouch", Position: -22},
	{Type: "petMesoMagnet", Position: -23},
	{Type: "petHP", Position: -24},
	{Type: "petMP", Position: -25},
	{Type: "petShoes", Position: -26},
	{Type: "petBinocular", Position: -27},
	{Type: "petMagicScales", Position: -28},
	{Type: "petRing2", Position: -29},
	{Type: "pet2Equip", Position: -30},
	{Type: "pet2Ring1", Position: -31},
	{Type: "pet2Pouch", Position: -32},
	{Type: "pet2MesoMagnet", Position: -33},
	{Type: "pet2Shoes", Position: -34},
	{Type: "pet2Binocular", Position: -35},
	{Type: "pet2MagicScales", Position: -36},
	{Type: "pet2Ring2", Position: -37},
	{Type: "pet3Equip", Position: -38},
	{Type: "pet3Ring1", Position: -39},
	{Type: "pet3Pouch", Position: -40},
	{Type: "pet3MesoMagnet", Position: -41},
	{Type: "pet3Shoes", Position: -42},
	{Type: "pet3Binocular", Position: -43},
	{Type: "pet3MagicScales", Position: -44},
	{Type: "pet3Ring2", Position: -45},
	{Type: "petItemIgnore", Position: -46},
	{Type: "pet2ItemIgnore", Position: -47},
	{Type: "pet3ItemIgnore", Position: -48},
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
