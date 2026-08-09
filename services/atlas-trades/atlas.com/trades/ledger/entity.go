// Package ledger holds the durable completed-trade record: the GORM entities
// for PRD §6's three tables (trade_ledger_entries / _sides / _items), the
// immutable domain model they map to, and the tenant-scoped provider and
// administrator over them.
//
// Only settled trades are recorded (FR-7.3) and rows are never updated or
// deleted (FR-7.4), so the administrator exposes a single write path: create.
package ledger

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Entry is one settled trade. The unique (tenant_id, transaction_id) index is
// the write-side idempotency guard for FR-5.7: a duplicate settle for the same
// settlement saga cannot produce a second row.
//
// RoomType lives here and is deliberately NOT denormalised onto Side — both
// sides of a trade always share it (design §9). It is a miniroom type byte,
// miniroom.Trade (3) or miniroom.CashTrade (6).
type Entry struct {
	Id            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TenantId      uuid.UUID  `gorm:"type:uuid;not null;index;uniqueIndex:idx_trade_entry_tenant_tx"`
	TransactionId uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_trade_entry_tenant_tx"`
	WorldId       world.Id   `gorm:"not null"`
	ChannelId     channel.Id `gorm:"not null"`
	MapId         _map.Id    `gorm:"not null"`
	RoomType      byte       `gorm:"not null"`
	SettledAt     time.Time  `gorm:"not null;index"`
	Sides         []Side     `gorm:"foreignKey:EntryId"`
}

func (Entry) TableName() string { return "trade_ledger_entries" }

// Side is one participant's contribution to a settled trade. Exactly two rows
// per Entry. CharacterName is denormalised because names change and the ledger
// is a point-in-time record (PRD §6).
type Side struct {
	Id            uuid.UUID    `gorm:"type:uuid;primaryKey"`
	TenantId      uuid.UUID    `gorm:"type:uuid;not null;index"`
	EntryId       uuid.UUID    `gorm:"type:uuid;not null;index"`
	CharacterId   character.Id `gorm:"not null;index"`
	CharacterName string       `gorm:"not null"`
	MesoStaged    uint32       `gorm:"not null"`
	MesoTax       uint32       `gorm:"not null"`
	MesoDelivered uint32       `gorm:"not null"`
	Items         []ItemRow    `gorm:"foreignKey:SideId"`
}

func (Side) TableName() string { return "trade_ledger_sides" }

// ItemRow is one asset a side gave. AssetId and ReferenceId are nullable
// because only identity-bearing assets (equips, pets, cash) carry them.
type ItemRow struct {
	Id          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantId    uuid.UUID      `gorm:"type:uuid;not null;index"`
	SideId      uuid.UUID      `gorm:"type:uuid;not null;index"`
	ItemId      item.Id        `gorm:"not null"`
	Quantity    asset.Quantity `gorm:"not null"`
	AssetId     *asset.Id
	ReferenceId *uint32
}

func (ItemRow) TableName() string { return "trade_ledger_items" }

// Migration creates the three ledger tables. Fresh tables, no backfill.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entry{}, &Side{}, &ItemRow{})
}
