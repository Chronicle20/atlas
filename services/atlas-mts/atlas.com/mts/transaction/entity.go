package transaction

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration creates the mts_transactions table. It is a brand-new table (no
// legacy primary-key rewrite), so AutoMigrate alone produces the correct
// surrogate-key shape and the composite index declared on the entity tags.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is the GORM row for a settled MTS transaction-history record.
//
// A single listing settle records TWO rows: one for the buyer (Kind=purchase)
// and one for the seller (Kind=sale), each owned by its own CharacterId with
// the other party in CounterpartyId. The primary key is a surrogate UUID; a
// (tenant_id, character_id) index backs the My Page -> History read query.
type entity struct {
	Id       uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_mts_transactions_tenant_id,priority:2"`
	TenantId uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_mts_transactions_tenant_id,priority:1;index:idx_mts_transactions_character,priority:1"`
	WorldId  byte      `gorm:"column:world_id;not null"`

	CharacterId    uint32 `gorm:"column:character_id;not null;index:idx_mts_transactions_character,priority:2"`
	CounterpartyId uint32 `gorm:"column:counterparty_id;not null"`

	ItemId     uint32 `gorm:"column:item_id;not null"`
	Quantity   uint32 `gorm:"column:quantity;not null"`
	TotalPrice uint32 `gorm:"column:total_price;not null"`

	Kind string `gorm:"column:kind;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (e entity) TableName() string {
	return "mts_transactions"
}
