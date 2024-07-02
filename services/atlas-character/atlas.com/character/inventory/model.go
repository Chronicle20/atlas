package inventory

import (
	"atlas-character/equipable"
	"atlas-character/inventory/item"
)

const (
	TypeValueEquip Type = 1
	TypeValueUse   Type = 2
	TypeValueSetup Type = 3
	TypeValueETC   Type = 4
	TypeValueCash  Type = 5
	TypeEquip           = "EQUIP"
	TypeUse             = "USE"
	TypeSetup           = "SETUP"
	TypeETC             = "ETC"
	TypeCash            = "CASH"
)

var Types = []Type{TypeValueEquip, TypeValueUse, TypeValueSetup, TypeValueETC, TypeValueCash}

type Type int8

type Model struct {
	equipable EquipableModel
	useable   ItemModel
	setup     ItemModel
	etc       ItemModel
	cash      ItemModel
}

func (m Model) Equipable() EquipableModel {
	return m.equipable
}

func NewModel(defaultCapacity uint32) Model {
	return Model{
		equipable: EquipableModel{capacity: defaultCapacity},
		useable:   ItemModel{capacity: defaultCapacity},
		setup:     ItemModel{capacity: defaultCapacity},
		etc:       ItemModel{capacity: defaultCapacity},
		cash:      ItemModel{capacity: defaultCapacity},
	}
}

type EquipableModel struct {
	id       uint32
	capacity uint32
	items    []equipable.Model
}

func (m EquipableModel) Id() uint32 {
	return m.id
}

type ItemModel struct {
	id       uint32
	capacity uint32
	items    []item.Model
}

func GetInventoryType(itemId uint32) (int8, bool) {
	t := int8(itemId / 1000000)
	if t >= 1 && t <= 5 {
		return t, true
	}
	return 0, false
}
