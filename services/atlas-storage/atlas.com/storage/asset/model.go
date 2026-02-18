package asset

import (
	"encoding/json"
	"time"

	af "github.com/Chronicle20/atlas-constants/asset"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/google/uuid"
)

type Model struct {
	id        uint32
	storageId uuid.UUID
	slot      int16
	templateId uint32
	expiration time.Time
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
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	// cash fields
	cashId      int64
	commodityId uint32
	purchaseBy  uint32
	// pet reference
	petId uint32
}

func (m Model) Id() uint32               { return m.id }
func (m Model) StorageId() uuid.UUID     { return m.storageId }
func (m Model) Slot() int16              { return m.slot }
func (m Model) TemplateId() uint32       { return m.templateId }
func (m Model) Expiration() time.Time    { return m.expiration }
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
func (m Model) Locked() bool             { return af.HasFlag(m.flag, af.FlagLock) }
func (m Model) Spikes() bool             { return af.HasFlag(m.flag, af.FlagSpikes) }
func (m Model) KarmaUsed() bool          { return af.HasFlag(m.flag, af.FlagKarmaUse) }
func (m Model) Cold() bool               { return af.HasFlag(m.flag, af.FlagCold) }
func (m Model) CanBeTraded() bool        { return !af.HasFlag(m.flag, af.FlagUntradeable) }
func (m Model) LevelType() byte          { return m.levelType }
func (m Model) Level() byte              { return m.level }
func (m Model) Experience() uint32       { return m.experience }
func (m Model) HammersApplied() uint32   { return m.hammersApplied }
func (m Model) CashId() int64            { return m.cashId }
func (m Model) CommodityId() uint32      { return m.commodityId }
func (m Model) PurchaseBy() uint32       { return m.purchaseBy }
func (m Model) PetId() uint32            { return m.petId }

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

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id             uint32    `json:"id"`
		StorageId      uuid.UUID `json:"storageId"`
		Slot           int16     `json:"slot"`
		TemplateId     uint32    `json:"templateId"`
		Expiration     time.Time `json:"expiration"`
		Quantity       uint32    `json:"quantity"`
		OwnerId        uint32    `json:"ownerId"`
		Flag           uint16    `json:"flag"`
		Rechargeable   uint64    `json:"rechargeable"`
		Strength       uint16    `json:"strength"`
		Dexterity      uint16    `json:"dexterity"`
		Intelligence   uint16    `json:"intelligence"`
		Luck           uint16    `json:"luck"`
		Hp             uint16    `json:"hp"`
		Mp             uint16    `json:"mp"`
		WeaponAttack   uint16    `json:"weaponAttack"`
		MagicAttack    uint16    `json:"magicAttack"`
		WeaponDefense  uint16    `json:"weaponDefense"`
		MagicDefense   uint16    `json:"magicDefense"`
		Accuracy       uint16    `json:"accuracy"`
		Avoidability   uint16    `json:"avoidability"`
		Hands          uint16    `json:"hands"`
		Speed          uint16    `json:"speed"`
		Jump           uint16    `json:"jump"`
		Slots          uint16    `json:"slots"`
		LevelType      byte      `json:"levelType"`
		Level          byte      `json:"level"`
		Experience     uint32    `json:"experience"`
		HammersApplied uint32    `json:"hammersApplied"`
		CashId         int64     `json:"cashId"`
		CommodityId    uint32    `json:"commodityId"`
		PurchaseBy     uint32    `json:"purchaseBy"`
		PetId          uint32    `json:"petId"`
	}{
		Id: m.id, StorageId: m.storageId, Slot: m.slot, TemplateId: m.templateId,
		Expiration: m.expiration, Quantity: m.quantity, OwnerId: m.ownerId, Flag: m.flag,
		Rechargeable: m.rechargeable, Strength: m.strength, Dexterity: m.dexterity,
		Intelligence: m.intelligence, Luck: m.luck, Hp: m.hp, Mp: m.mp,
		WeaponAttack: m.weaponAttack, MagicAttack: m.magicAttack, WeaponDefense: m.weaponDefense,
		MagicDefense: m.magicDefense, Accuracy: m.accuracy, Avoidability: m.avoidability,
		Hands: m.hands, Speed: m.speed, Jump: m.jump, Slots: m.slots,
		LevelType: m.levelType, Level: m.level, Experience: m.experience,
		HammersApplied: m.hammersApplied, CashId: m.cashId, CommodityId: m.commodityId,
		PurchaseBy: m.purchaseBy, PetId: m.petId,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id             uint32    `json:"id"`
		StorageId      uuid.UUID `json:"storageId"`
		Slot           int16     `json:"slot"`
		TemplateId     uint32    `json:"templateId"`
		Expiration     time.Time `json:"expiration"`
		Quantity       uint32    `json:"quantity"`
		OwnerId        uint32    `json:"ownerId"`
		Flag           uint16    `json:"flag"`
		Rechargeable   uint64    `json:"rechargeable"`
		Strength       uint16    `json:"strength"`
		Dexterity      uint16    `json:"dexterity"`
		Intelligence   uint16    `json:"intelligence"`
		Luck           uint16    `json:"luck"`
		Hp             uint16    `json:"hp"`
		Mp             uint16    `json:"mp"`
		WeaponAttack   uint16    `json:"weaponAttack"`
		MagicAttack    uint16    `json:"magicAttack"`
		WeaponDefense  uint16    `json:"weaponDefense"`
		MagicDefense   uint16    `json:"magicDefense"`
		Accuracy       uint16    `json:"accuracy"`
		Avoidability   uint16    `json:"avoidability"`
		Hands          uint16    `json:"hands"`
		Speed          uint16    `json:"speed"`
		Jump           uint16    `json:"jump"`
		Slots          uint16    `json:"slots"`
		LevelType      byte      `json:"levelType"`
		Level          byte      `json:"level"`
		Experience     uint32    `json:"experience"`
		HammersApplied uint32    `json:"hammersApplied"`
		CashId         int64     `json:"cashId"`
		CommodityId    uint32    `json:"commodityId"`
		PurchaseBy     uint32    `json:"purchaseBy"`
		PetId          uint32    `json:"petId"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.id = aux.Id
	m.storageId = aux.StorageId
	m.slot = aux.Slot
	m.templateId = aux.TemplateId
	m.expiration = aux.Expiration
	m.quantity = aux.Quantity
	m.ownerId = aux.OwnerId
	m.flag = aux.Flag
	m.rechargeable = aux.Rechargeable
	m.strength = aux.Strength
	m.dexterity = aux.Dexterity
	m.intelligence = aux.Intelligence
	m.luck = aux.Luck
	m.hp = aux.Hp
	m.mp = aux.Mp
	m.weaponAttack = aux.WeaponAttack
	m.magicAttack = aux.MagicAttack
	m.weaponDefense = aux.WeaponDefense
	m.magicDefense = aux.MagicDefense
	m.accuracy = aux.Accuracy
	m.avoidability = aux.Avoidability
	m.hands = aux.Hands
	m.speed = aux.Speed
	m.jump = aux.Jump
	m.slots = aux.Slots
	m.levelType = aux.LevelType
	m.level = aux.Level
	m.experience = aux.Experience
	m.hammersApplied = aux.HammersApplied
	m.cashId = aux.CashId
	m.commodityId = aux.CommodityId
	m.purchaseBy = aux.PurchaseBy
	m.petId = aux.PetId
	return nil
}
