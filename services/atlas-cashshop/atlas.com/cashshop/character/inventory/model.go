package inventory

import (
	"atlas-cashshop/character/inventory/equipable"
	"atlas-cashshop/character/inventory/item"
)

type Type int8

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

var TypeValues = []Type{TypeValueEquip, TypeValueUse, TypeValueSetup, TypeValueETC, TypeValueCash}
var Types = []string{TypeEquip, TypeUse, TypeSetup, TypeETC, TypeCash}

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

func (m Model) Use() ItemModel {
	return m.useable
}

func (m Model) Setup() ItemModel {
	return m.setup
}

func (m Model) Etc() ItemModel {
	return m.etc
}

func (m Model) Cash() ItemModel {
	return m.cash
}

type EquipableModel struct {
	capacity uint32
	items    []equipable.Model
}

func (m EquipableModel) Capacity() uint32 {
	return m.capacity
}

func (m EquipableModel) Items() []equipable.Model {
	return m.items
}

type ItemModel struct {
	capacity uint32
	items    []item.Model
}

func (m ItemModel) Capacity() uint32 {
	return m.capacity
}

func (m ItemModel) Items() []item.Model {
	return m.items
}

type EquippedItem struct {
	itemId        uint32
	slot          int16
	strength      uint16
	dexterity     uint16
	intelligence  uint16
	luck          uint16
	hp            uint16
	mp            uint16
	weaponAttack  uint16
	magicAttack   uint16
	weaponDefense uint16
	magicDefense  uint16
	accuracy      uint16
	avoidability  uint16
	hands         uint16
	speed         uint16
	jump          uint16
	slots         byte
}

func (i EquippedItem) NotInWeaponSlot() bool {
	if i.slot != -111 {
		return true
	}
	return false
}

func (i EquippedItem) InvertSlot() EquippedItem {
	return Clone(i).SetSlot(i.Slot() * -1).Build()
}

func (i EquippedItem) Slot() int16 {
	return i.slot
}

func (i EquippedItem) ItemId() uint32 {
	return i.itemId
}

func (i EquippedItem) InWeaponSlot() bool {
	if i.slot == -111 {
		return true
	}
	return false
}

func (i EquippedItem) IsRegularEquipment() bool {
	return i.slot > -100
}

func (i EquippedItem) Expiration() int64 {
	return -1
}

func (i EquippedItem) Slots() byte {
	return i.slots
}

func (i EquippedItem) Level() byte {
	return 0
}

func (i EquippedItem) Strength() uint16 {
	return i.strength
}

func (i EquippedItem) Dexterity() uint16 {
	return i.dexterity
}

func (i EquippedItem) Intelligence() uint16 {
	return i.intelligence
}

func (i EquippedItem) Luck() uint16 {
	return i.luck
}

func (i EquippedItem) Hp() uint16 {
	return i.hp
}

func (i EquippedItem) Mp() uint16 {
	return i.mp
}

func (i EquippedItem) WeaponAttack() uint16 {
	return i.weaponAttack
}

func (i EquippedItem) MagicAttack() uint16 {
	return i.magicAttack
}

func (i EquippedItem) WeaponDefense() uint16 {
	return i.weaponDefense
}

func (i EquippedItem) MagicDefense() uint16 {
	return i.magicDefense
}

func (i EquippedItem) Accuracy() uint16 {
	return i.accuracy
}

func (i EquippedItem) Avoidability() uint16 {
	return i.avoidability
}

func (i EquippedItem) Hands() uint16 {
	return i.hands
}

func (i EquippedItem) Speed() uint16 {
	return i.speed
}

func (i EquippedItem) Jump() uint16 {
	return i.jump
}

func (i EquippedItem) OwnerName() string {
	return ""
}

func (i EquippedItem) Flags() uint16 {
	return 0
}

func (i EquippedItem) IsEquippedCashItem() bool {
	return i.slot <= -100
}

func (i EquippedItem) Quantity() uint16 {
	return 1
}

type equippedItemBuilder struct {
	itemId        uint32
	slot          int16
	strength      uint16
	dexterity     uint16
	intelligence  uint16
	luck          uint16
	hp            uint16
	mp            uint16
	weaponAttack  uint16
	magicAttack   uint16
	weaponDefense uint16
	magicDefense  uint16
	accuracy      uint16
	avoidability  uint16
	hands         uint16
	speed         uint16
	jump          uint16
	slots         byte
}

func NewEquippedItemBuilder() *equippedItemBuilder {
	return &equippedItemBuilder{}
}

func Clone(o EquippedItem) *equippedItemBuilder {
	return &equippedItemBuilder{
		itemId:        o.itemId,
		slot:          o.slot,
		strength:      o.strength,
		dexterity:     o.dexterity,
		intelligence:  o.intelligence,
		luck:          o.luck,
		hp:            o.hp,
		mp:            o.mp,
		weaponAttack:  o.weaponAttack,
		magicAttack:   o.magicAttack,
		weaponDefense: o.weaponDefense,
		magicDefense:  o.magicDefense,
		accuracy:      o.accuracy,
		avoidability:  o.avoidability,
		hands:         o.hands,
		speed:         o.speed,
		jump:          o.jump,
		slots:         o.slots,
	}
}

func (e *equippedItemBuilder) SetItemId(itemId uint32) *equippedItemBuilder {
	e.itemId = itemId
	return e
}

func (e *equippedItemBuilder) SetSlot(slot int16) *equippedItemBuilder {
	e.slot = slot
	return e
}

func (e *equippedItemBuilder) SetStrength(strength uint16) *equippedItemBuilder {
	e.strength = strength
	return e
}

func (e *equippedItemBuilder) SetDexterity(dexterity uint16) *equippedItemBuilder {
	e.dexterity = dexterity
	return e
}

func (e *equippedItemBuilder) SetIntelligence(intelligence uint16) *equippedItemBuilder {
	e.intelligence = intelligence
	return e
}

func (e *equippedItemBuilder) SetLuck(luck uint16) *equippedItemBuilder {
	e.luck = luck
	return e
}

func (e *equippedItemBuilder) SetHp(hp uint16) *equippedItemBuilder {
	e.hp = hp
	return e
}

func (e *equippedItemBuilder) SetMp(mp uint16) *equippedItemBuilder {
	e.mp = mp
	return e
}

func (e *equippedItemBuilder) SetWeaponAttack(weaponAttack uint16) *equippedItemBuilder {
	e.weaponAttack = weaponAttack
	return e
}

func (e *equippedItemBuilder) SetMagicAttack(magicAttack uint16) *equippedItemBuilder {
	e.magicAttack = magicAttack
	return e
}

func (e *equippedItemBuilder) SetWeaponDefense(weaponDefense uint16) *equippedItemBuilder {
	e.weaponDefense = weaponDefense
	return e
}

func (e *equippedItemBuilder) SetMagicDefense(magicDefense uint16) *equippedItemBuilder {
	e.magicDefense = magicDefense
	return e
}

func (e *equippedItemBuilder) SetAccuracy(accuracy uint16) *equippedItemBuilder {
	e.accuracy = accuracy
	return e
}

func (e *equippedItemBuilder) SetAvoidability(avoidability uint16) *equippedItemBuilder {
	e.avoidability = avoidability
	return e
}

func (e *equippedItemBuilder) SetHands(hands uint16) *equippedItemBuilder {
	e.hands = hands
	return e
}

func (e *equippedItemBuilder) SetSpeed(speed uint16) *equippedItemBuilder {
	e.speed = speed
	return e
}

func (e *equippedItemBuilder) SetJump(jump uint16) *equippedItemBuilder {
	e.jump = jump
	return e
}

func (e *equippedItemBuilder) SetSlots(slots byte) *equippedItemBuilder {
	e.slots = slots
	return e
}

func (e *equippedItemBuilder) Build() EquippedItem {
	return EquippedItem{
		itemId:        e.itemId,
		slot:          e.slot,
		strength:      e.strength,
		dexterity:     e.dexterity,
		intelligence:  e.intelligence,
		luck:          e.luck,
		hp:            e.hp,
		mp:            e.mp,
		weaponAttack:  e.weaponAttack,
		magicAttack:   e.magicAttack,
		weaponDefense: e.weaponDefense,
		magicDefense:  e.magicDefense,
		accuracy:      e.accuracy,
		avoidability:  e.avoidability,
		hands:         e.hands,
		speed:         e.speed,
		jump:          e.jump,
		slots:         e.slots,
	}
}
