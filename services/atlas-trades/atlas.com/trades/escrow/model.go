package escrow

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ItemModel is one staged asset in trade custody. Immutable: private fields,
// getters, and a builder.
type ItemModel struct {
	id      uuid.UUID
	roomId  uuid.UUID
	ownerId character.Id

	tenantId     uuid.UUID
	tenantRegion string
	tenantMajor  uint16
	tenantMinor  uint16

	tradeSlot byte

	sourceInventoryType inventory.Type
	sourceSlot          slot.Position
	assetId             asset.Id

	templateId item.Id
	quantity   asset.Quantity

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
	slots         uint16
	level         byte
	itemLevel     byte
	itemExp       uint32
	ringId        uint32
	viciousCount  uint32
	flags         uint16
	owner         string

	createdAt time.Time
}

func (m ItemModel) Id() uuid.UUID            { return m.id }
func (m ItemModel) RoomId() uuid.UUID        { return m.roomId }
func (m ItemModel) OwnerId() character.Id    { return m.ownerId }
func (m ItemModel) TenantId() uuid.UUID      { return m.tenantId }
func (m ItemModel) TradeSlot() byte          { return m.tradeSlot }
func (m ItemModel) AssetId() asset.Id        { return m.assetId }
func (m ItemModel) TemplateId() item.Id      { return m.templateId }
func (m ItemModel) Quantity() asset.Quantity { return m.quantity }
func (m ItemModel) Strength() uint16         { return m.strength }
func (m ItemModel) Dexterity() uint16        { return m.dexterity }
func (m ItemModel) Intelligence() uint16     { return m.intelligence }
func (m ItemModel) Luck() uint16             { return m.luck }
func (m ItemModel) HP() uint16               { return m.hp }
func (m ItemModel) MP() uint16               { return m.mp }
func (m ItemModel) WeaponAttack() uint16     { return m.weaponAttack }
func (m ItemModel) MagicAttack() uint16      { return m.magicAttack }
func (m ItemModel) WeaponDefense() uint16    { return m.weaponDefense }
func (m ItemModel) MagicDefense() uint16     { return m.magicDefense }
func (m ItemModel) Accuracy() uint16         { return m.accuracy }
func (m ItemModel) Avoidability() uint16     { return m.avoidability }
func (m ItemModel) Hands() uint16            { return m.hands }
func (m ItemModel) Speed() uint16            { return m.speed }
func (m ItemModel) Jump() uint16             { return m.jump }
func (m ItemModel) Slots() uint16            { return m.slots }
func (m ItemModel) Level() byte              { return m.level }
func (m ItemModel) ItemLevel() byte          { return m.itemLevel }
func (m ItemModel) ItemExp() uint32          { return m.itemExp }
func (m ItemModel) RingId() uint32           { return m.ringId }
func (m ItemModel) ViciousCount() uint32     { return m.viciousCount }
func (m ItemModel) Flags() uint16            { return m.flags }
func (m ItemModel) Owner() string            { return m.owner }
func (m ItemModel) CreatedAt() time.Time     { return m.createdAt }

func (m ItemModel) SourceInventoryType() inventory.Type { return m.sourceInventoryType }
func (m ItemModel) SourceSlot() slot.Position           { return m.sourceSlot }

// Tenant rebuilds the tenant this row belongs to. Startup reconciliation runs
// with no tenant in context and must restore one per row before it can issue
// the commands that return the item (design §5A.9).
func (m ItemModel) Tenant() (tenant.Model, error) {
	return tenant.Create(m.tenantId, m.tenantRegion, m.tenantMajor, m.tenantMinor)
}

// MesoModel is one participant's escrowed meso for one room. Amount is the
// ABSOLUTE escrowed total (design §5A.5).
type MesoModel struct {
	id      uuid.UUID
	roomId  uuid.UUID
	ownerId character.Id

	tenantId     uuid.UUID
	tenantRegion string
	tenantMajor  uint16
	tenantMinor  uint16

	amount uint32

	pendingStakeId uuid.UUID
	pendingAmount  uint32

	createdAt time.Time
}

func (m MesoModel) Id() uuid.UUID             { return m.id }
func (m MesoModel) RoomId() uuid.UUID         { return m.roomId }
func (m MesoModel) OwnerId() character.Id     { return m.ownerId }
func (m MesoModel) TenantId() uuid.UUID       { return m.tenantId }
func (m MesoModel) Amount() uint32            { return m.amount }
func (m MesoModel) PendingStakeId() uuid.UUID { return m.pendingStakeId }
func (m MesoModel) PendingAmount() uint32     { return m.pendingAmount }
func (m MesoModel) CreatedAt() time.Time      { return m.createdAt }

// Tenant rebuilds the tenant this row belongs to. See ItemModel.Tenant.
func (m MesoModel) Tenant() (tenant.Model, error) {
	return tenant.Create(m.tenantId, m.tenantRegion, m.tenantMajor, m.tenantMinor)
}

// MakeItem maps a row back onto its immutable model.
func MakeItem(e ItemEntity) (ItemModel, error) {
	return ItemModel{
		id:                  e.Id,
		roomId:              e.RoomId,
		ownerId:             e.OwnerId,
		tenantId:            e.TenantId,
		tenantRegion:        e.TenantRegion,
		tenantMajor:         e.TenantMajor,
		tenantMinor:         e.TenantMinor,
		tradeSlot:           e.TradeSlot,
		sourceInventoryType: e.SourceInventoryType,
		sourceSlot:          e.SourceSlot,
		assetId:             e.AssetId,
		templateId:          e.TemplateId,
		quantity:            e.Quantity,
		strength:            e.Strength,
		dexterity:           e.Dexterity,
		intelligence:        e.Intelligence,
		luck:                e.Luck,
		hp:                  e.HP,
		mp:                  e.MP,
		weaponAttack:        e.WeaponAttack,
		magicAttack:         e.MagicAttack,
		weaponDefense:       e.WeaponDefense,
		magicDefense:        e.MagicDefense,
		accuracy:            e.Accuracy,
		avoidability:        e.Avoidability,
		hands:               e.Hands,
		speed:               e.Speed,
		jump:                e.Jump,
		slots:               e.Slots,
		level:               e.Level,
		itemLevel:           e.ItemLevel,
		itemExp:             e.ItemExp,
		ringId:              e.RingId,
		viciousCount:        e.ViciousCount,
		flags:               e.Flags,
		owner:               e.Owner,
		createdAt:           e.CreatedAt,
	}, nil
}

// MakeMeso maps a row back onto its immutable model.
func MakeMeso(e MesoEntity) (MesoModel, error) {
	return MesoModel{
		id:             e.Id,
		roomId:         e.RoomId,
		ownerId:        e.OwnerId,
		tenantId:       e.TenantId,
		tenantRegion:   e.TenantRegion,
		tenantMajor:    e.TenantMajor,
		tenantMinor:    e.TenantMinor,
		amount:         e.Amount,
		pendingStakeId: e.PendingStakeId,
		pendingAmount:  e.PendingAmount,
		createdAt:      e.CreatedAt,
	}, nil
}
