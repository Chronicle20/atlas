package inventory

import (
	"atlas-character/equipable"
	"atlas-character/inventory/item"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
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
	equipable ItemHolder
	useable   ItemHolder
	setup     ItemHolder
	etc       ItemHolder
	cash      ItemHolder
}

func (m Model) Equipable() EquipableModel {
	if m.equipable == nil {
		return EquipableModel{}
	}
	return m.equipable.(EquipableModel)
}

func (m Model) SetEquipable(em EquipableModel) Model {
	m.equipable = em
	return m
}

func (m Model) Useable() ItemModel {
	if m.useable == nil {
		return ItemModel{}
	}
	return m.useable.(ItemModel)
}

func (m Model) SetUseable(um ItemModel) Model {
	m.useable = um
	return m
}

func (m Model) Setup() ItemModel {
	if m.setup == nil {
		return ItemModel{}
	}
	return m.setup.(ItemModel)
}

func (m Model) SetSetup(um ItemModel) Model {
	m.setup = um
	return m
}

func (m Model) Etc() ItemModel {
	if m.etc == nil {
		return ItemModel{}
	}
	return m.etc.(ItemModel)
}

func (m Model) SetEtc(um ItemModel) Model {
	m.etc = um
	return m
}

func (m Model) Cash() ItemModel {
	if m.cash == nil {
		return ItemModel{}
	}
	return m.cash.(ItemModel)
}

func (m Model) SetCash(um ItemModel) Model {
	m.cash = um
	return m
}

func NewModel(defaultCapacity uint32) Model {
	return Model{
		equipable: EquipableModel{capacity: defaultCapacity},
		useable:   ItemModel{mType: TypeValueUse, capacity: defaultCapacity},
		setup:     ItemModel{mType: TypeValueSetup, capacity: defaultCapacity},
		etc:       ItemModel{mType: TypeValueETC, capacity: defaultCapacity},
		cash:      ItemModel{mType: TypeValueCash, capacity: defaultCapacity},
	}
}

type EquipableModel struct {
	id       uint32
	capacity uint32
	items    []equipable.Model
}

func NewEquipableModel(id uint32, capacity uint32) model.Provider[EquipableModel] {
	return func() (EquipableModel, error) {
		return EquipableModel{id: id, capacity: capacity}, nil
	}
}

func (m EquipableModel) Id() uint32 {
	return m.id
}

func (m EquipableModel) SetId(id uint32) ItemHolder {
	m.id = id
	return m
}

func (m EquipableModel) Capacity() uint32 {
	return m.capacity
}

func (m EquipableModel) SetCapacity(capacity uint32) ItemHolder {
	m.capacity = capacity
	return m
}

func (m EquipableModel) Items() []equipable.Model {
	return m.items
}

func (m EquipableModel) SetItems(items []equipable.Model) ItemHolder {
	m.items = items
	return m
}

type ItemModel struct {
	id       uint32
	mType    Type
	capacity uint32
	items    []item.Model
}

func NewItemModel(id uint32, mType Type, capacity uint32) model.Provider[ItemModel] {
	return func() (ItemModel, error) {
		return ItemModel{id: id, mType: mType, capacity: capacity}, nil
	}
}

func (m ItemModel) Id() uint32 {
	return m.id
}

func (m ItemModel) SetId(id uint32) ItemHolder {
	m.id = id
	return m
}

func (m ItemModel) Type() Type {
	return m.mType
}

func (m ItemModel) Capacity() uint32 {
	return m.capacity
}

func (m ItemModel) SetCapacity(capacity uint32) ItemHolder {
	m.capacity = capacity
	return m
}

func (m ItemModel) Items() []item.Model {
	return m.items
}

func (m ItemModel) SetItems(items []item.Model) ItemHolder {
	m.items = items
	return m
}

func GetInventoryType(itemId uint32) (int8, bool) {
	t := int8(itemId / 1000000)
	if t >= 1 && t <= 5 {
		return t, true
	}
	return 0, false
}

func (m Model) GetHolderByType(inventoryType Type) (ItemHolder, error) {
	switch inventoryType {
	case TypeValueEquip:
		return m.equipable, nil
	case TypeValueUse:
		return m.useable, nil
	case TypeValueSetup:
		return m.setup, nil
	case TypeValueETC:
		return m.etc, nil
	case TypeValueCash:
		return m.cash, nil
	}
	return nil, errors.New("invalid inventory type")
}

type ItemHolder interface {
	Id() uint32
	SetId(id uint32) ItemHolder
	Capacity() uint32
	SetCapacity(capacity uint32) ItemHolder
}
