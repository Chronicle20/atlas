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

var Types = []string{TypeEquip, TypeUse, TypeSetup, TypeETC, TypeCash}

type Type int8

type Model struct {
	equipable EquipableModel
	useable   ItemModel
	setup     ItemModel
	etc       ItemModel
	cash      ItemModel
}

type EquipableModel struct {
	capacity uint32
	items    []equipable.Model
}

type ItemModel struct {
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
