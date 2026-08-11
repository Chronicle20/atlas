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

	// Expiration / CashId / Rechargeable / the pet block are what make a cash
	// item, a pet or a timed item survive escrow. Cash items and pets ARE
	// stageable — checkRestrictions (trade/restriction.go) blocks only equipped
	// items, the untradeable flags and the WZ tradeBlock — so a row that stored
	// nothing but equip stats handed back a degraded asset: no cash serial, no
	// expiry, no pet identity.
	Expiration   time.Time `gorm:"column:expiration"`
	CashId       int64     `gorm:"column:cash_id;not null;default:0"`
	Rechargeable uint64    `gorm:"column:rechargeable;not null;default:0"`

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

	// Named for the snapshot fields they hold, not for the packet-side aliases
	// the first cut used (item_level / item_exp / vicious_count). One concept
	// under two names across a service boundary is what let the original stat
	// list drift from the asset it was supposed to reproduce.
	LevelType      byte   `gorm:"column:level_type;not null;default:0"`
	Level          byte   `gorm:"column:level;not null"`
	Experience     uint32 `gorm:"column:experience;not null;default:0"`
	HammersApplied uint32 `gorm:"column:hammers_applied;not null;default:0"`
	Flags          uint16 `gorm:"column:flags;not null"`
	Owner          string `gorm:"column:owner;not null;default:''"`

	PetId     uint32 `gorm:"column:pet_id;not null;default:0"`
	PetName   string `gorm:"column:pet_name;not null;default:''"`
	PetLevel  byte   `gorm:"column:pet_level;not null;default:0"`
	Closeness uint16 `gorm:"column:closeness;not null;default:0"`
	Fullness  byte   `gorm:"column:fullness;not null;default:0"`

	// ReturningAt is the row's SINGLE-CLAIMANT latch for the return path, and it
	// is the item twin of MesoEntity.PendingStakeId.
	//
	// Two independent code paths can each decide to return the same row: a
	// teardown reads ItemsByRoom and unwinds everything escrowed for the room,
	// while a stage's saga-completed status that finds no dialog slot reads
	// ItemById and unwinds that one row (trade/settlement.go emitUnwind,
	// trade/processor.go returnOrphanedStage). Both are reachable for one row at
	// the same time, because the row is written by the custody consumer long
	// before atlas-trades learns the stage succeeded. Nothing downstream dedupes
	// them: accept_to_character grants unconditionally, DeleteItem treats a
	// no-match as success, and each unwind mints its own transaction id. Two
	// submissions therefore granted the item twice. ClaimItemForReturn stamps
	// this column in the same UPDATE that decides whether the caller may submit,
	// so exactly one of them can.
	//
	// It is a column of its own rather than a reuse of DeletedAt because the two
	// answer different questions: DeletedAt means "the item has LEFT custody"
	// (release_from_trade), and reusing it would both hide the row from ItemById
	// — making returnOrphanedStage report "this was never a stage" and mis-route
	// the caller — and make an in-flight claim indistinguishable from a
	// completed release to the compensating restore.
	//
	// A pointer, so "unclaimed" is SQL NULL and the claim's `IS NULL` predicate
	// is a real compare-and-set rather than a comparison against a zero time
	// that a claim could legitimately write.
	ReturningAt *time.Time `gorm:"column:returning_at"`

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
//
// PendingStakeId / PendingAmount durably record an IN-FLIGHT award_mesos debit
// between the moment the saga is submitted and the moment its terminal status
// is applied. Staging used to keep that bookkeeping only in the room's
// in-memory state; if the room was torn down while the saga was still running,
// its terminal status arrived with nowhere to land — the debit had already
// happened, but no durable record named the amount it should commit into
// Amount, so the meso was silently lost. Recording the pending stake in the
// row itself (rather than the room) means a terminal status can always resolve
// against the row by stakeId alone, room or no room — see ArmMesoStake,
// CommitMesoStake, AbandonMesoStake, and MesoStakeById.
//
// PendingStakeId is uuid.Nil when no stake is in flight; that is the "none"
// sentinel rather than a nullable column, so the compare-and-set in
// CommitMesoStake/AbandonMesoStake is a single ordinary equality check.
//
// PendingDelta is the SIGNED movement the in-flight award_mesos actually
// submitted — the stake minus whatever was escrowed when it was armed — and it
// is persisted rather than re-derived because Amount is not stable across the
// stake's lifetime: a teardown ZEROES Amount while deliberately leaving the
// stake armed (see the trade package's clearRefundedMesos). A refund that
// re-derived the delta from Amount at resolution time therefore refunded the
// whole stake on top of the teardown's own refund of the committed part, minting
// the committed amount. It is signed because a stake that LOWERS the box is a
// credit, and its width matches the award_mesos payload's own Amount.
type MesoEntity struct {
	Id           uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_trade_escrow_mesos_tenant_id,priority:2"`
	TenantId     uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_trade_escrow_mesos_tenant_id,priority:1;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:1"`
	TenantRegion string    `gorm:"column:tenant_region;type:varchar(32);not null;default:''"`
	TenantMajor  uint16    `gorm:"column:tenant_major;not null;default:0"`
	TenantMinor  uint16    `gorm:"column:tenant_minor;not null;default:0"`

	RoomId  uuid.UUID    `gorm:"column:room_id;type:uuid;not null;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:2"`
	OwnerId character.Id `gorm:"column:owner_id;not null;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:3"`

	Amount uint32 `gorm:"column:amount;not null"`

	PendingStakeId uuid.UUID `gorm:"column:pending_stake_id;type:uuid"`
	PendingAmount  uint32    `gorm:"column:pending_amount;not null;default:0"`
	PendingDelta   int32     `gorm:"column:pending_delta;not null;default:0"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (MesoEntity) TableName() string { return mesoTable }

// staleItemColumns are columns an earlier shape of ItemEntity created that
// nothing writes any more: ring_id was never sourced (no asset projection
// anywhere in the fleet carries a ring id), and item_level / item_exp /
// vicious_count were renamed to the snapshot's own vocabulary.
//
// AutoMigrate adds and widens columns but never drops them, so leaving these
// behind is not cosmetic: each was created NOT NULL with no default, and an
// INSERT that no longer names the column would be rejected by Postgres on any
// database that had already been migrated to the old shape.
var staleItemColumns = []string{"ring_id", "item_level", "item_exp", "vicious_count"}

// Migration creates both tables. Fresh tables, no backfill.
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&ItemEntity{}, &MesoEntity{}); err != nil {
		return err
	}
	m := db.Migrator()
	for _, c := range staleItemColumns {
		if !m.HasColumn(&ItemEntity{}, c) {
			continue
		}
		if err := m.DropColumn(&ItemEntity{}, c); err != nil {
			return err
		}
	}
	return nil
}
