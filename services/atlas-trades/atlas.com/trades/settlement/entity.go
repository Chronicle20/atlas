// Package settlement holds the DURABLE record of a settlement saga that
// atlas-trades has submitted but not yet seen the outcome of.
//
// It exists because the live trade room is process-local in-memory state
// (design §9) while the settlement saga is not: the saga lives in
// atlas-saga-orchestrator and keeps running across an atlas-trades restart.
// Without this record a restart between submission and the terminal status lost
// the room, so a trade that FULLY EXECUTED wrote no ledger row and emitted no
// SETTLED — contradicting FR-7.1 ("every settled trade writes one durable ledger
// row").
//
// The row is written in the SAME transaction that publishes the saga command,
// so the two cannot diverge, and is deleted once the terminal status has been
// handled, so it never accumulates. It is therefore a work record, not an audit
// record — the audit record is the ledger. It carries everything needed to
// produce that ledger entry and the client-visible outcome WITHOUT the room:
// both participants, their staged items at their re-resolved slots, the frozen
// tax split, and the room identity the status event's envelope needs.
package settlement

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// The three table names. Constants rather than literals inside each
// TableName(), matching the ledger's reasoning: a rename that missed one would
// still compile.
const (
	entryTable = "trade_settlements"
	sideTable  = "trade_settlement_sides"
	itemTable  = "trade_settlement_items"
)

// Entry is one in-flight settlement.
//
// TenantRegion / TenantMajor / TenantMinor are stored alongside TenantId
// because startup reconciliation runs with NO tenant in context: it reads every
// unfinished row across every tenant and has to rebuild each row's tenant to
// scope the ledger write, the REST reads and the Kafka headers that follow.
// atlas-saga-orchestrator's own saga Entity carries the same four fields for
// the same reason (saga/entity.go:17-20).
//
// The room identity is denormalised here because the room is gone by the time
// this row is read: the status event envelope needs the room id, wire handle,
// room type, field and both participants, and none of it is recoverable from
// anywhere else after a restart.
type Entry struct {
	Id            uuid.UUID    `gorm:"type:uuid;primaryKey"`
	TenantId      uuid.UUID    `gorm:"type:uuid;not null;index;uniqueIndex:idx_trade_settlement_tenant_tx"`
	TenantRegion  string       `gorm:"type:varchar(32);not null;default:''"`
	TenantMajor   uint16       `gorm:"not null;default:0"`
	TenantMinor   uint16       `gorm:"not null;default:0"`
	TransactionId uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_trade_settlement_tenant_tx"`
	RoomId        uuid.UUID    `gorm:"type:uuid;not null"`
	Handle        uint32       `gorm:"not null"`
	RoomType      byte         `gorm:"not null"`
	WorldId       world.Id     `gorm:"not null"`
	ChannelId     channel.Id   `gorm:"not null"`
	MapId         _map.Id      `gorm:"not null"`
	Instance      uuid.UUID    `gorm:"type:uuid;not null"`
	OwnerId       character.Id `gorm:"not null"`
	VisitorId     character.Id `gorm:"not null"`
	SubmittedAt   time.Time    `gorm:"not null;index"`
	Sides         []Side       `gorm:"foreignKey:EntryId"`
}

func (Entry) TableName() string { return entryTable }

// Side is one participant's contribution, with the tax split already resolved
// — re-deriving it at reconciliation time would read a tenant configuration
// that may have changed since the saga was submitted, and record figures the
// orchestrator never moved.
type Side struct {
	Id            uuid.UUID    `gorm:"type:uuid;primaryKey"`
	TenantId      uuid.UUID    `gorm:"type:uuid;not null;index"`
	EntryId       uuid.UUID    `gorm:"type:uuid;not null;index"`
	Position      byte         `gorm:"not null"`
	CharacterId   character.Id `gorm:"not null;index"`
	CharacterName string       `gorm:"not null"`
	MesoStaged    uint32       `gorm:"not null"`
	MesoTax       uint32       `gorm:"not null"`
	MesoDelivered uint32       `gorm:"not null"`
	Items         []ItemRow    `gorm:"foreignKey:SideId"`
}

func (Side) TableName() string { return sideTable }

// ItemRow is one escrowed asset. EscrowId names the custody row the settlement
// saga releases from; InventoryType and SourceSlot are the item's provenance,
// recorded for the ledger and for diagnosing a stuck row rather than replayed.
type ItemRow struct {
	Id            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantId      uuid.UUID      `gorm:"type:uuid;not null;index"`
	SideId        uuid.UUID      `gorm:"type:uuid;not null;index"`
	EscrowId      uuid.UUID      `gorm:"column:escrow_id;type:uuid;not null"`
	InventoryType inventory.Type `gorm:"not null"`
	SourceSlot    slot.Position  `gorm:"not null"`
	AssetId       asset.Id       `gorm:"not null"`
	TemplateId    item.Id        `gorm:"not null"`
	Quantity      asset.Quantity `gorm:"not null"`
}

func (ItemRow) TableName() string { return itemTable }

// Migration creates the three tables. Fresh tables, no backfill.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entry{}, &Side{}, &ItemRow{})
}
