package escrow

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
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

	// sourceInventoryType is stored separately from the snapshot because it is
	// NOT derivable from the template id: a cash-inventory item's template
	// classifies as equip or use, and returning it to that compartment would put
	// a pet in the equip tab.
	sourceInventoryType inventory.Type
	assetId             asset.Id

	// snapshot is everything needed to re-materialise the asset, in the shared
	// shape every other custody limb and the trade renderer already speak. The
	// bespoke stat list it replaced had no expiry, cash serial or pet block, so
	// a returned cash item, pet or timed item came back degraded.
	snapshot sharedsaga.AssetSnapshot

	createdAt time.Time
}

func (m ItemModel) Id() uuid.UUID         { return m.id }
func (m ItemModel) RoomId() uuid.UUID     { return m.roomId }
func (m ItemModel) OwnerId() character.Id { return m.ownerId }
func (m ItemModel) TenantId() uuid.UUID   { return m.tenantId }
func (m ItemModel) TradeSlot() byte       { return m.tradeSlot }
func (m ItemModel) AssetId() asset.Id     { return m.assetId }
func (m ItemModel) CreatedAt() time.Time  { return m.createdAt }

// Snapshot is the asset as it stood the moment it left its owner's compartment.
func (m ItemModel) Snapshot() sharedsaga.AssetSnapshot { return m.snapshot }

// TemplateId, Quantity and SourceSlot read through to the snapshot rather than
// shadowing it, so there is exactly one place a staged item's identity lives.
// Quantity is the STAGED amount, not the source stack's.
func (m ItemModel) TemplateId() item.Id       { return item.Id(m.snapshot.TemplateId) }
func (m ItemModel) Quantity() asset.Quantity  { return asset.Quantity(m.snapshot.Quantity) }
func (m ItemModel) SourceSlot() slot.Position { return slot.Position(m.snapshot.Slot) }

func (m ItemModel) SourceInventoryType() inventory.Type { return m.sourceInventoryType }

// Tenant rebuilds the tenant this row belongs to. Startup reconciliation runs
// with no tenant in context and must restore one per row before it can issue
// the commands that return the item (design §5A.9).
func (m ItemModel) Tenant() (tenant.Model, error) {
	return tenant.Create(m.tenantId, m.tenantRegion, m.tenantMajor, m.tenantMinor)
}

// MesoModel is one participant's escrowed meso for one room. Amount is the
// CONFIRMED escrowed total — the sum of the stake deltas award_mesos actually
// moved — and is signed and transiently negative for the reason MesoEntity
// gives. What the player currently has typed is Amount plus the deltas still in
// flight (EffectiveMesoByOwner), not this figure alone.
type MesoModel struct {
	id      uuid.UUID
	roomId  uuid.UUID
	ownerId character.Id

	tenantId     uuid.UUID
	tenantRegion string
	tenantMajor  uint16
	tenantMinor  uint16

	amount int64

	createdAt time.Time
}

func (m MesoModel) Id() uuid.UUID         { return m.id }
func (m MesoModel) RoomId() uuid.UUID     { return m.roomId }
func (m MesoModel) OwnerId() character.Id { return m.ownerId }
func (m MesoModel) TenantId() uuid.UUID   { return m.tenantId }
func (m MesoModel) Amount() int64         { return m.amount }
func (m MesoModel) CreatedAt() time.Time  { return m.createdAt }

// Tenant rebuilds the tenant this row belongs to. See ItemModel.Tenant.
func (m MesoModel) Tenant() (tenant.Model, error) {
	return tenant.Create(m.tenantId, m.tenantRegion, m.tenantMajor, m.tenantMinor)
}

// MesoStakeModel is one in-flight award_mesos debit. See MesoStakeEntity.
type MesoStakeModel struct {
	id      uuid.UUID
	roomId  uuid.UUID
	ownerId character.Id

	tenantId     uuid.UUID
	tenantRegion string
	tenantMajor  uint16
	tenantMinor  uint16

	amount uint32
	delta  int32

	createdAt time.Time
}

func (m MesoStakeModel) Id() uuid.UUID         { return m.id }
func (m MesoStakeModel) RoomId() uuid.UUID     { return m.roomId }
func (m MesoStakeModel) OwnerId() character.Id { return m.ownerId }
func (m MesoStakeModel) TenantId() uuid.UUID   { return m.tenantId }

// Amount is the absolute total the player typed for this stake.
func (m MesoStakeModel) Amount() uint32 { return m.amount }

// Delta is the signed movement this stake submitted. It is the only safe basis
// for refunding an orphaned stake, because the committed Amount is zeroed out
// from under a still-armed stake by an ordinary teardown (see MesoStakeEntity).
func (m MesoStakeModel) Delta() int32         { return m.delta }
func (m MesoStakeModel) CreatedAt() time.Time { return m.createdAt }

// Tenant rebuilds the tenant this stake belongs to. See ItemModel.Tenant.
func (m MesoStakeModel) Tenant() (tenant.Model, error) {
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
		assetId:             e.AssetId,
		snapshot: sharedsaga.AssetSnapshot{
			Slot:           int16(e.SourceSlot),
			TemplateId:     uint32(e.TemplateId),
			Expiration:     e.Expiration,
			CashId:         e.CashId,
			Quantity:       uint32(e.Quantity),
			Flag:           e.Flags,
			Owner:          e.Owner,
			Rechargeable:   e.Rechargeable,
			Strength:       e.Strength,
			Dexterity:      e.Dexterity,
			Intelligence:   e.Intelligence,
			Luck:           e.Luck,
			Hp:             e.HP,
			Mp:             e.MP,
			WeaponAttack:   e.WeaponAttack,
			MagicAttack:    e.MagicAttack,
			WeaponDefense:  e.WeaponDefense,
			MagicDefense:   e.MagicDefense,
			Accuracy:       e.Accuracy,
			Avoidability:   e.Avoidability,
			Hands:          e.Hands,
			Speed:          e.Speed,
			Jump:           e.Jump,
			Slots:          e.Slots,
			LevelType:      e.LevelType,
			Level:          e.Level,
			Experience:     e.Experience,
			HammersApplied: e.HammersApplied,
			PetId:          e.PetId,
			PetName:        e.PetName,
			PetLevel:       e.PetLevel,
			Closeness:      e.Closeness,
			Fullness:       e.Fullness,
		},
		createdAt: e.CreatedAt,
	}, nil
}

// MakeMeso maps a row back onto its immutable model.
func MakeMeso(e MesoEntity) (MesoModel, error) {
	return MesoModel{
		id:           e.Id,
		roomId:       e.RoomId,
		ownerId:      e.OwnerId,
		tenantId:     e.TenantId,
		tenantRegion: e.TenantRegion,
		tenantMajor:  e.TenantMajor,
		tenantMinor:  e.TenantMinor,
		amount:       e.Amount,
		createdAt:    e.CreatedAt,
	}, nil
}

// MakeMesoStake maps an in-flight stake row back onto its immutable model.
func MakeMesoStake(e MesoStakeEntity) (MesoStakeModel, error) {
	return MesoStakeModel{
		id:           e.Id,
		roomId:       e.RoomId,
		ownerId:      e.OwnerId,
		tenantId:     e.TenantId,
		tenantRegion: e.TenantRegion,
		tenantMajor:  e.TenantMajor,
		tenantMinor:  e.TenantMinor,
		amount:       e.Amount,
		delta:        e.Delta,
		createdAt:    e.CreatedAt,
	}, nil
}
