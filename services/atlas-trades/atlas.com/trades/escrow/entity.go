// Package escrow is atlas-trades' durable CUSTODY store for staged items and
// staged meso (task-205 design §5A).
//
// It exists because a staged item genuinely LEAVES its owner's compartment at
// stage time. That is not an implementation preference: the reference client
// arms m_bExclRequestSent when it sends PUT_ITEM / PUT_MONEY and refuses both
// until a server packet clears it, and the only packets that clear it are the
// inventory and stat deltas a real mutation produces (design §5A.1). Reserving
// the item instead — the model this package replaced — left the trade dialog
// permanently unable to stage anything after the first item.
//
// Because the asset is genuinely gone, something durable has to name its owner
// or a crash would strand it. That is this package. It is the trade limb of the
// accept/release custody family (atlas-storage, atlas-cashshop, atlas-mts), and
// its row shape follows atlas-mts's holdings table: a surrogate UUID key, a
// tenant-scoped unique index, and the item snapshot as explicit name-keyed
// columns rather than a JSON blob, so a binary COPY/restore is column-order
// safe.
package escrow

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// The two table names. Constants rather than literals inside each TableName(),
// matching the ledger's and settlement's reasoning: a rename that missed one
// would still compile.
const (
	itemTable = "trade_escrow_items"
	mesoTable = "trade_escrow_mesos"
)

// ItemEntity is one staged asset held in trade custody.
//
// TenantRegion / TenantMajor / TenantMinor sit alongside TenantId for the same
// reason settlement.Entry carries them: startup reconciliation runs with NO
// tenant in context, reads every row across every tenant, and has to rebuild
// each row's tenant before it can scope the commands that return the item.
//
// DeletedAt is a GORM soft-delete column. A release soft-deletes so the
// compensating restore can bring the row back; a spurious accept is HARD
// deleted (see RemoveItem) precisely because a restorable row could resurrect
// an item its owner already holds again.
//
// RoomId is not a foreign key. Rooms are process-local in-memory state (design
// §9) and do not survive a restart, which is exactly why every surviving row is
// an orphan by definition and the reconciler can simply return them all.
type ItemEntity struct {
	Id           uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_trade_escrow_items_tenant_id,priority:2"`
	TenantId     uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_trade_escrow_items_tenant_id,priority:1;index:idx_trade_escrow_items_room,priority:1;index:idx_trade_escrow_items_owner,priority:1"`
	TenantRegion string    `gorm:"column:tenant_region;type:varchar(32);not null;default:''"`
	TenantMajor  uint16    `gorm:"column:tenant_major;not null;default:0"`
	TenantMinor  uint16    `gorm:"column:tenant_minor;not null;default:0"`

	RoomId  uuid.UUID    `gorm:"column:room_id;type:uuid;not null;index:idx_trade_escrow_items_room,priority:2"`
	OwnerId character.Id `gorm:"column:owner_id;not null;index:idx_trade_escrow_items_owner,priority:2"`

	TradeSlot byte `gorm:"column:trade_slot;not null"`

	// Provenance. A return does NOT replay these — it accepts to the owner's
	// compartment and lets atlas-inventory pick the slot, because the original
	// slot may be occupied by the time the trade unwinds. They are recorded for
	// the ledger and for diagnosing a stuck row.
	SourceInventoryType inventory.Type `gorm:"column:source_inventory_type;not null"`
	SourceSlot          slot.Position  `gorm:"column:source_slot;not null"`
	AssetId             asset.Id       `gorm:"column:asset_id;not null"`

	TemplateId item.Id        `gorm:"column:template_id;not null"`
	Quantity   asset.Quantity `gorm:"column:quantity;not null"`

	Strength      uint16 `gorm:"column:strength;not null"`
	Dexterity     uint16 `gorm:"column:dexterity;not null"`
	Intelligence  uint16 `gorm:"column:intelligence;not null"`
	Luck          uint16 `gorm:"column:luck;not null"`
	HP            uint16 `gorm:"column:hp;not null"`
	MP            uint16 `gorm:"column:mp;not null"`
	WeaponAttack  uint16 `gorm:"column:weapon_attack;not null"`
	MagicAttack   uint16 `gorm:"column:magic_attack;not null"`
	WeaponDefense uint16 `gorm:"column:weapon_defense;not null"`
	MagicDefense  uint16 `gorm:"column:magic_defense;not null"`
	Accuracy      uint16 `gorm:"column:accuracy;not null"`
	Avoidability  uint16 `gorm:"column:avoidability;not null"`
	Hands         uint16 `gorm:"column:hands;not null"`
	Speed         uint16 `gorm:"column:speed;not null"`
	Jump          uint16 `gorm:"column:jump;not null"`
	Slots         uint16 `gorm:"column:slots;not null"`
	Level         byte   `gorm:"column:level;not null"`
	ItemLevel     byte   `gorm:"column:item_level;not null"`
	ItemExp       uint32 `gorm:"column:item_exp;not null"`
	RingId        uint32 `gorm:"column:ring_id;not null"`
	ViciousCount  uint32 `gorm:"column:vicious_count;not null"`
	Flags         uint16 `gorm:"column:flags;not null"`
	Owner         string `gorm:"column:owner;not null;default:''"`

	CreatedAt time.Time      `gorm:"column:created_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (ItemEntity) TableName() string { return itemTable }

// MesoEntity is one participant's escrowed meso for one room.
//
// It is a separate table rather than a `kind` discriminator on ItemEntity: the
// two share nothing but the room and the owner, and a discriminated table would
// leave every stat column null for half its rows.
//
// Amount is the ABSOLUTE escrowed total, not a running delta. Clientbound mode
// 16 assigns rather than accumulates (design §1.6), so staging works in deltas
// against this figure and the row is REPLACED on each stage — see UpsertMeso.
// A row that accumulated would refund more than was ever debited.
type MesoEntity struct {
	Id           uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_trade_escrow_mesos_tenant_id,priority:2"`
	TenantId     uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_trade_escrow_mesos_tenant_id,priority:1;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:1"`
	TenantRegion string    `gorm:"column:tenant_region;type:varchar(32);not null;default:''"`
	TenantMajor  uint16    `gorm:"column:tenant_major;not null;default:0"`
	TenantMinor  uint16    `gorm:"column:tenant_minor;not null;default:0"`

	RoomId  uuid.UUID    `gorm:"column:room_id;type:uuid;not null;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:2"`
	OwnerId character.Id `gorm:"column:owner_id;not null;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:3"`

	Amount uint32 `gorm:"column:amount;not null"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (MesoEntity) TableName() string { return mesoTable }

// Migration creates both tables. Fresh tables, no backfill.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&ItemEntity{}, &MesoEntity{})
}
