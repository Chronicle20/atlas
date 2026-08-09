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

// The ledger's three table names. They are constants rather than string
// literals inside each TableName() because provider.go's EXISTS subquery
// names two of them in raw SQL, and a rename that missed one would compile.
const (
	entryTable = "trade_ledger_entries"
	sideTable  = "trade_ledger_sides"
	itemTable  = "trade_ledger_items"
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

func (Entry) TableName() string { return entryTable }

// Side is one participant's contribution to a settled trade. Exactly two rows
// per Entry. CharacterName is denormalised because names change and the ledger
// is a point-in-time record (PRD §6).
//
// tenant_id and character_id share one composite index rather than having one
// each: the only query that reaches this table by character (the FR-7.2 GM
// lookup) always filters on both, and a tenant_id-leading composite serves the
// tenant-only reads too.
type Side struct {
	Id            uuid.UUID    `gorm:"type:uuid;primaryKey"`
	TenantId      uuid.UUID    `gorm:"type:uuid;not null;index:idx_trade_side_tenant_char,priority:1"`
	EntryId       uuid.UUID    `gorm:"type:uuid;not null;index"`
	CharacterId   character.Id `gorm:"not null;index:idx_trade_side_tenant_char,priority:2"`
	CharacterName string       `gorm:"not null"`
	MesoStaged    uint32       `gorm:"not null"`
	MesoTax       uint32       `gorm:"not null"`
	MesoDelivered uint32       `gorm:"not null"`
	Items         []ItemRow    `gorm:"foreignKey:SideId"`
}

func (Side) TableName() string { return sideTable }

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

func (ItemRow) TableName() string { return itemTable }

// Migration creates the three ledger tables. Fresh tables, no backfill.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entry{}, &Side{}, &ItemRow{})
}
