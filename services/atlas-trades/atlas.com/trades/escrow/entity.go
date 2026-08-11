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
	itemTable      = "trade_escrow_items"
	mesoTable      = "trade_escrow_mesos"
	mesoStakeTable = "trade_escrow_meso_stakes"
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
// IN-FLIGHT debits are NOT recorded here. They live one-per-row in
// MesoStakeEntity, because more than one can be outstanding at a time and a
// single slot on this row silently dropped all but the newest — see that
// type's doc comment for why the client permits the overlap and what the slot
// destroyed.
//
// The invariant tying the two tables together, and the one every operation in
// this package is written to preserve:
//
//	Amount == the sum of the deltas award_mesos ACTUALLY MOVED
//
// Amount therefore advances only when a stake's terminal status confirms its
// delta landed, never by assignment from a stake's absolute target. What the
// player currently has typed into the box is the derived figure
// `Amount + SUM(stake deltas)`, not a stored one (see EffectiveMesoByOwner).
type MesoEntity struct {
	Id           uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_trade_escrow_mesos_tenant_id,priority:2"`
	TenantId     uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_trade_escrow_mesos_tenant_id,priority:1;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:1"`
	TenantRegion string    `gorm:"column:tenant_region;type:varchar(32);not null;default:''"`
	TenantMajor  uint16    `gorm:"column:tenant_major;not null;default:0"`
	TenantMinor  uint16    `gorm:"column:tenant_minor;not null;default:0"`

	RoomId  uuid.UUID    `gorm:"column:room_id;type:uuid;not null;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:2"`
	OwnerId character.Id `gorm:"column:owner_id;not null;uniqueIndex:idx_trade_escrow_mesos_room_owner,priority:3"`

	// Amount is SIGNED, and wider than the meso values it accumulates, because
	// it is a running sum of signed deltas rather than a directly-assigned
	// total. Stakes resolve in whatever order their terminal statuses arrive,
	// which need not be the order they were armed: a player who types 1000 and
	// then 500 arms +1000 and -500, and if the reduction's status lands first
	// the committed total passes through -500 on its way to 500. That
	// intermediate is legitimate and transient. Held unsigned it underflowed.
	//
	// Consumers that pay out against this figure (the refund and settlement
	// paths) treat any non-positive value as nothing owed, which is correct on
	// its own terms: a negative total means more reduction has been confirmed
	// than increase so far, and there is nothing yet to hand back.
	Amount int64 `gorm:"column:amount;not null"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (MesoEntity) TableName() string { return mesoTable }

// MesoStakeEntity is ONE in-flight award_mesos debit against a participant's
// escrow row — durable between the moment the saga is submitted and the moment
// its terminal status is applied.
//
// It is durable rather than in-memory on the room for the original reason the
// pending slot was: if the room is torn down while the saga is still running,
// the terminal status arrives with nowhere to land, the debit has already
// happened, and nothing names the amount it should commit — so the meso is
// silently lost. A status can always resolve by stakeId alone, room or no room.
//
// It is a TABLE rather than the three pending_* columns it replaces because
// **more than one stake can be outstanding at once, and each one moved real
// meso.** The client permits the overlap: CTradingRoomDlg::PutMoney arms
// CWvsContext's excl latch on send, but the debit's own STAT_CHANGED clears it
// before the trade-level MESO_STAGED lands, so a player retyping the box faster
// than a saga round trip submits a second stake while the first is in flight.
// With a single slot the second arm OVERWROTE the first, and the first's
// terminal status then matched no row and was discarded as "superseded" — while
// its debit had already left the player's pocket. That is meso destroyed on
// success, and (had the first failed after being superseded) meso minted on
// failure. Each stake now resolves independently against its own row, so
// Amount accumulates exactly the deltas that moved.
//
// Delta is the SIGNED movement this stake submitted — the target minus
// committed-plus-already-in-flight at arm time — and it is persisted rather
// than re-derived because Amount is not stable across the stake's lifetime: a
// teardown ZEROES Amount while deliberately leaving stakes armed (see the trade
// package's clearRefundedMesos). A refund that re-derived the delta from Amount
// at resolution time refunded the whole stake on top of the teardown's own
// refund of the committed part, minting it. It is signed because a stake that
// LOWERS the box is a credit, and its width matches the award_mesos payload's
// own Amount.
//
// Amount here is the ABSOLUTE total the player typed for this stake. It is not
// load-bearing for conservation — only Delta is — but it is what the room
// re-echoes and what makes a stranded row readable by a human.
//
// Id is the stakeId the saga was submitted with, and it is the primary key
// rather than a surrogate: resolution looks the stake up by exactly that value,
// and a surrogate would allow two rows to claim one stake.
type MesoStakeEntity struct {
	Id           uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	TenantId     uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;index:idx_trade_escrow_meso_stakes_room_owner,priority:1"`
	TenantRegion string    `gorm:"column:tenant_region;type:varchar(32);not null;default:''"`
	TenantMajor  uint16    `gorm:"column:tenant_major;not null;default:0"`
	TenantMinor  uint16    `gorm:"column:tenant_minor;not null;default:0"`

	RoomId  uuid.UUID    `gorm:"column:room_id;type:uuid;not null;index:idx_trade_escrow_meso_stakes_room_owner,priority:2"`
	OwnerId character.Id `gorm:"column:owner_id;not null;index:idx_trade_escrow_meso_stakes_room_owner,priority:3"`

	Amount uint32 `gorm:"column:amount;not null;default:0"`
	Delta  int32  `gorm:"column:delta;not null;default:0"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (MesoStakeEntity) TableName() string { return mesoStakeTable }

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

// staleMesoColumns are the single in-flight-stake slot that MesoStakeEntity
// replaced. They are dropped for the same hard reason as staleItemColumns —
// pending_amount and pending_delta were created NOT NULL, so an INSERT that no
// longer names them is rejected outright on any database already migrated to
// the old shape.
var staleMesoColumns = []string{"pending_stake_id", "pending_amount", "pending_delta"}

// Migration creates the three tables and retires the meso row's old
// single-stake slot.
//
// The slot is BACKFILLED into trade_escrow_meso_stakes before it is dropped,
// not simply discarded. A row carrying a pending stake at migration time
// describes meso that has already left a player's pocket and has no other
// record anywhere; dropping the columns would strand it with nothing to resolve
// against and nothing for the boot sweep to find. At most one stake per row can
// exist to migrate, since one slot is all the old shape could hold.
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&ItemEntity{}, &MesoEntity{}, &MesoStakeEntity{}); err != nil {
		return err
	}
	m := db.Migrator()
	if err := backfillMesoStakes(db); err != nil {
		return err
	}
	for _, c := range staleItemColumns {
		if !m.HasColumn(&ItemEntity{}, c) {
			continue
		}
		if err := m.DropColumn(&ItemEntity{}, c); err != nil {
			return err
		}
	}
	for _, c := range staleMesoColumns {
		if !m.HasColumn(&MesoEntity{}, c) {
			continue
		}
		if err := m.DropColumn(&MesoEntity{}, c); err != nil {
			return err
		}
	}
	return nil
}

// backfillMesoStakes lifts any stake still armed in the old slot into its own
// row. It is a no-op on a fresh database, where the columns never existed.
//
// The INSERT selects only rows whose slot is genuinely occupied — the old shape
// used uuid.Nil rather than NULL as its "no stake" sentinel, so both have to be
// excluded. ON CONFLICT DO NOTHING makes a re-run inert, which matters because
// Migration runs on every boot and the drop below may not have been reached if
// an earlier attempt failed partway.
func backfillMesoStakes(db *gorm.DB) error {
	m := db.Migrator()
	for _, c := range staleMesoColumns {
		if !m.HasColumn(&MesoEntity{}, c) {
			return nil
		}
	}
	return db.Exec(`
		INSERT INTO `+mesoStakeTable+` (id, tenant_id, tenant_region, tenant_major, tenant_minor, room_id, owner_id, amount, delta, created_at)
		SELECT pending_stake_id, tenant_id, tenant_region, tenant_major, tenant_minor, room_id, owner_id, pending_amount, pending_delta, ?
		FROM `+mesoTable+`
		WHERE pending_stake_id IS NOT NULL AND pending_stake_id <> ?
		ON CONFLICT (id) DO NOTHING`,
		time.Now(), uuid.Nil).Error
}
