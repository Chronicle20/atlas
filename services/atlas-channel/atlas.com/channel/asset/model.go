package asset

import (
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/google/uuid"
)

type Model struct {
	id            uint32
	compartmentId uuid.UUID
	slot          int16
	templateId    uint32
	expiration    time.Time
	createdAt     time.Time
	// stackable fields
	quantity     uint32
	ownerId      uint32
	flag         uint16
	rechargeable uint64
	// equipment fields
	strength       uint16
	dexterity      uint16
	intelligence   uint16
	luck           uint16
	hp             uint16
	mp             uint16
	weaponAttack   uint16
	magicAttack    uint16
	weaponDefense  uint16
	magicDefense   uint16
	accuracy       uint16
	avoidability   uint16
	hands          uint16
	speed          uint16
	jump           uint16
	slots          uint16
	locked         bool
	spikes         bool
	karmaUsed      bool
	cold           bool
	canBeTraded    bool
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	equippedSince  *time.Time
	// cash fields
	cashId      int64
	commodityId uint32
	purchaseBy  uint32
	// pet fields
	petId     uint32
	petName   string
	petLevel  byte
	closeness uint16
	fullness  byte
	petSlot   int8
}

func (m Model) Id() uint32               { return m.id }
func (m Model) CompartmentId() uuid.UUID  { return m.compartmentId }
func (m Model) Slot() int16              { return m.slot }
func (m Model) TemplateId() uint32       { return m.templateId }
func (m Model) Expiration() time.Time    { return m.expiration }
func (m Model) CreatedAt() time.Time     { return m.createdAt }
func (m Model) OwnerId() uint32          { return m.ownerId }
func (m Model) Flag() uint16             { return m.flag }
func (m Model) Rechargeable() uint64     { return m.rechargeable }
func (m Model) Strength() uint16         { return m.strength }
func (m Model) Dexterity() uint16        { return m.dexterity }
func (m Model) Intelligence() uint16     { return m.intelligence }
func (m Model) Luck() uint16             { return m.luck }
func (m Model) Hp() uint16               { return m.hp }
func (m Model) Mp() uint16               { return m.mp }
func (m Model) WeaponAttack() uint16     { return m.weaponAttack }
func (m Model) MagicAttack() uint16      { return m.magicAttack }
func (m Model) WeaponDefense() uint16    { return m.weaponDefense }
func (m Model) MagicDefense() uint16     { return m.magicDefense }
func (m Model) Accuracy() uint16         { return m.accuracy }
func (m Model) Avoidability() uint16     { return m.avoidability }
func (m Model) Hands() uint16            { return m.hands }
func (m Model) Speed() uint16            { return m.speed }
func (m Model) Jump() uint16             { return m.jump }
func (m Model) Slots() uint16            { return m.slots }
func (m Model) Locked() bool             { return m.locked }
func (m Model) Spikes() bool             { return m.spikes }
func (m Model) KarmaUsed() bool          { return m.karmaUsed }
func (m Model) Cold() bool               { return m.cold }
func (m Model) CanBeTraded() bool        { return m.canBeTraded }
func (m Model) LevelType() byte          { return m.levelType }
func (m Model) Level() byte              { return m.level }
func (m Model) Experience() uint32       { return m.experience }
func (m Model) HammersApplied() uint32   { return m.hammersApplied }
func (m Model) EquippedSince() *time.Time { return m.equippedSince }
func (m Model) CashId() int64            { return m.cashId }
func (m Model) CommodityId() uint32      { return m.commodityId }
func (m Model) PurchaseBy() uint32       { return m.purchaseBy }
func (m Model) PetId() uint32            { return m.petId }
func (m Model) PetName() string          { return m.petName }
func (m Model) PetLevel() byte           { return m.petLevel }
func (m Model) Closeness() uint16        { return m.closeness }
func (m Model) Fullness() byte           { return m.fullness }
func (m Model) PetSlot() int8            { return m.petSlot }

func (m Model) InventoryType() inventory.Type {
	t, _ := inventory.TypeFromItemId(item.Id(m.templateId))
	return t
}

func (m Model) IsEquipment() bool {
	return m.InventoryType() == inventory.TypeValueEquip
}

func (m Model) IsCashEquipment() bool {
	return m.IsEquipment() && m.cashId != 0
}

func (m Model) IsConsumable() bool {
	return m.InventoryType() == inventory.TypeValueUse
}

func (m Model) IsSetup() bool {
	return m.InventoryType() == inventory.TypeValueSetup
}

func (m Model) IsEtc() bool {
	return m.InventoryType() == inventory.TypeValueETC
}

func (m Model) IsCash() bool {
	return m.InventoryType() == inventory.TypeValueCash
}

func (m Model) IsPet() bool {
	return m.IsCash() && m.petId > 0
}

func (m Model) IsStackable() bool {
	t := m.InventoryType()
	return t == inventory.TypeValueUse || t == inventory.TypeValueSetup || t == inventory.TypeValueETC
}

func (m Model) HasQuantity() bool {
	return m.IsStackable() || (m.IsCash() && !m.IsPet())
}

func (m Model) Quantity() uint32 {
	if m.HasQuantity() {
		return m.quantity
	}
	return 1
}

// InventoryType aliases for backward compatibility (previously in reference_data.go)
type InventoryType = inventory.Type

var (
	InventoryTypeEquip = inventory.TypeValueEquip
	InventoryTypeUse   = inventory.TypeValueUse
	InventoryTypeSetup = inventory.TypeValueSetup
	InventoryTypeEtc   = inventory.TypeValueETC
	InventoryTypeCash  = inventory.TypeValueCash
)
